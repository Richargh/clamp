#!/bin/bash
# clamp-lib.sh - Shared library for clamp wrapper scripts
# This file is sourced by clamp-claude.sh and clamp-open-code.sh

# Required variables to be set by wrapper before calling clamp_main:
#   CLAMP_TOOL_NAME      - Display name ("Claude Code" or "OpenCode")
#   CLAMP_VOLUME_PREFIX  - Volume naming prefix ("claude-clamp" or "opencode-clamp")
#   CLAMP_CONFIG_DIR     - Config path in container ("/home/dev/.claude" or "/home/dev/.local/share/opencode")
#   CLAMP_CONFIG_ENV     - Environment variable name ("CLAUDE_CONFIG_DIR" or "OPENCODE_CONFIG_DIR")
#   CLAMP_COMMAND        - Binary to execute ("claude" or "opencode")
#   CLAMP_DANGER_FLAG    - Permission skip flag ("--dangerously-skip-permissions" or "")
#   CLAMP_HARNESS        - Harness identifier for startup script ("claude" or "opencode")
#   CLAMP_SCRIPT_DIR     - Directory containing the wrapper script
#   CLAMP_WORKFLOWS_ENV  - Environment variable for workflows ("CLAUDE_CLAMP_ADD_WORKFLOWS" or "OPENCODE_CLAMP_ADD_WORKFLOWS")

# Image names / Dockerfiles
CLAMP_BASE_IMAGE_NAME="clamp-base"
CLAMP_IMAGE_NAME=""

# Unprivileged user inside the Clamp images. Wrappers derive config paths from
# this value and pass it into the container; startup scripts fail if it is not
# present in the container environment.
CLAMP_CONTAINER_USER="dev"

# Parsed argument flags (set by clamp_parse_args)
CLAMP_REBUILD=false
CLAMP_NO_CACHE=""
CLAMP_PER_PROJECT_AUTH=false
CLAMP_PER_PROJECT_AUTH_EXPLICIT=false
CLAMP_NO_FIREWALL=false
CLAMP_DANGER_MODE=false
CLAMP_SHELL_MODE=false
CLAMP_FOLDER=""

# User config
CLAMP_CONFIG_HOME="${XDG_CONFIG_HOME:-$HOME/.config}/clamp"
CLAMP_AUTH_CONFIG="$CLAMP_CONFIG_HOME/auth-projects.tsv"

clamp_usage() {
    echo "Usage: $(basename "$0") [-h|--help] [--rebuild] [--no-cache] [--per-project-auth] [--no-firewall] [--danger] [--shell] <folder>"
    echo "  -h, --help: Show this help message"
    echo "  --rebuild: Force rebuild of the Docker image"
    echo "  --no-cache: Rebuild without Docker layer cache"
    echo "  --per-project-auth: Use separate credentials for this project"
    echo "  --no-firewall: Disable the network firewall (allow all outbound traffic). Useful for constrained research."
    echo "  --danger: Run ${CLAMP_TOOL_NAME} with auto-accept permissions (no confirmations)"
    echo "  --shell: Run startup then drop to shell instead of launching ${CLAMP_TOOL_NAME} (for debugging)"
    echo "  folder: Path to the project folder to run in"
    echo ""
    echo "Environment variables:"
    echo "  ${CLAMP_WORKFLOWS_ENV}: Set to 'true' to copy bundled workflows into container"
}

clamp_parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --rebuild)
                CLAMP_REBUILD=true
                shift
                ;;
            --no-cache)
                CLAMP_REBUILD=true
                CLAMP_NO_CACHE="--no-cache"
                shift
                ;;
            --per-project-auth)
                CLAMP_PER_PROJECT_AUTH=true
                CLAMP_PER_PROJECT_AUTH_EXPLICIT=true
                shift
                ;;
            --no-firewall)
                CLAMP_NO_FIREWALL=true
                shift
                ;;
            --danger)
                CLAMP_DANGER_MODE=true
                shift
                ;;
            --shell)
                CLAMP_SHELL_MODE=true
                shift
                ;;
            -h|--help)
                clamp_usage
                exit 0
                ;;
            -*)
                echo "Unknown option: $1"
                clamp_usage
                exit 1
                ;;
            *)
                CLAMP_FOLDER="$1"
                shift
                ;;
        esac
    done

    if [ -z "$CLAMP_FOLDER" ]; then
        clamp_usage
        exit 1
    fi
}

clamp_config_get_auth_policy() {
    local workspace="$1"

    [ -f "$CLAMP_AUTH_CONFIG" ] || return 1

    awk -F '\t' -v path="$workspace" '$1 == path { print $2; found=1 } END { exit !found }' "$CLAMP_AUTH_CONFIG"
}

clamp_config_set_auth_policy() {
    local workspace="$1"
    local policy="$2"

    mkdir -p "$CLAMP_CONFIG_HOME"

    if [ -f "$CLAMP_AUTH_CONFIG" ]; then
        awk -F '\t' -v path="$workspace" '$1 != path' "$CLAMP_AUTH_CONFIG" > "${CLAMP_AUTH_CONFIG}.tmp"
        mv "${CLAMP_AUTH_CONFIG}.tmp" "$CLAMP_AUTH_CONFIG"
    fi

    printf '%s\t%s\n' "$workspace" "$policy" >> "$CLAMP_AUTH_CONFIG"
}

clamp_prompt_auth_policy() {
    local workspace="$1"
    local choice

    echo "No Clamp auth preference found for:" >&2
    echo "  $workspace" >&2
    echo "" >&2
    echo "Use:" >&2
    echo "  1) global/shared auth" >&2
    echo "  2) per-project auth" >&2
    echo "" >&2

    while true; do
        printf "Choice [1/2]: " >&2
        if ! read -r choice; then
            echo "Unable to read auth preference." >&2
            exit 1
        fi
        case "$choice" in
            1|"")
                echo "global"
                return
                ;;
            2)
                echo "project"
                return
                ;;
            *)
                echo "Please enter 1 or 2." >&2
                ;;
        esac
    done
}

clamp_resolve_auth_policy() {
    local workspace policy

    workspace="$(cd "$CLAMP_FOLDER" && pwd)"

    if [ "$CLAMP_PER_PROJECT_AUTH_EXPLICIT" = true ]; then
        clamp_config_set_auth_policy "$workspace" "project"
        return
    fi

    policy="$(clamp_config_get_auth_policy "$workspace")" || {
        policy="$(clamp_prompt_auth_policy "$workspace")"
        clamp_config_set_auth_policy "$workspace" "$policy"
    }

    case "$policy" in
        project)
            CLAMP_PER_PROJECT_AUTH=true
            ;;
        global)
            CLAMP_PER_PROJECT_AUTH=false
            ;;
        *)
            echo "Invalid Clamp auth policy for $workspace: $policy"
            exit 1
            ;;
    esac
}

clamp_log_config() {
    local auth firewall header_color text_color reset_color

    if [ -t 1 ]; then
        header_color=$'\033[1;36m'
        text_color=$'\033[90m'
        reset_color=$'\033[0m'
    else
        header_color=""
        text_color=""
        reset_color=""
    fi

    if [ "$CLAMP_PER_PROJECT_AUTH" = true ]; then
        auth="Project"
    else
        auth="Global"
    fi

    if [ "$CLAMP_NO_FIREWALL" = true ]; then
        firewall="Off"
    else
        firewall="On"
    fi

    printf '%s[Clamp User Config]%s\n' "$header_color" "$reset_color"
    printf '%s  Auth: %s%s\n' "$text_color" "$auth" "$reset_color"
    echo ""
    printf '%s[Clamp CLI Config]%s\n' "$header_color" "$reset_color"
    printf '%s  Coding Agent: %s%s\n' "$text_color" "$CLAMP_TOOL_NAME" "$reset_color"
    printf '%s  Firewall: %s%s\n' "$text_color" "$firewall" "$reset_color"
    echo ""
}

clamp_create_project_dockerfile() {
    local workspace project_dockerfile base_image project_allowed_domains_dir default_allowed_domains_dir header_color text_color reset_color printed_project_setup

    if [ -t 1 ]; then
        header_color=$'\033[1;36m'
        text_color=$'\033[90m'
        reset_color=$'\033[0m'
    else
        header_color=""
        text_color=""
        reset_color=""
    fi

    printed_project_setup=false

    workspace="$1"
    project_dockerfile="$workspace/.clamp/clamp.Dockerfile"
    project_allowed_domains_dir="$workspace/.clamp/allowed-domains.d"
    default_allowed_domains_dir="$CLAMP_SCRIPT_DIR/clamp-shared/allowed-domains.d"
    base_image="clamp-${CLAMP_HARNESS}"

    mkdir -p "$workspace/.clamp"

    if [ ! -d "$project_allowed_domains_dir" ]; then
        mkdir -p "$project_allowed_domains_dir"
        if [ -d "$default_allowed_domains_dir" ]; then
            cp -a "$default_allowed_domains_dir/." "$project_allowed_domains_dir/"
        fi
        printf '%s[Clamp Project Setup]%s\n' "$header_color" "$reset_color"
        printed_project_setup=true
        printf '%s  Created allowed domains directory: %s%s\n' "$text_color" "$project_allowed_domains_dir" "$reset_color"
        printf '%s  Review this directory and delete domain files this project should not allow.%s\n' "$text_color" "$reset_color"
    fi

    if [ -f "$project_dockerfile" ]; then
        if [ "$printed_project_setup" = false ]; then
            printf '%s[Clamp Project Setup]%s\n' "$header_color" "$reset_color"
            printed_project_setup=true
        fi
        printf '%s  Using project Dockerfile: %s%s\n' "$text_color" "$project_dockerfile" "$reset_color"
        if [ -f "$workspace/mise.toml" ] && ! grep -q 'mise install -C /tmp' "$project_dockerfile"; then
            printf '%s  Note: mise.toml exists, but this Dockerfile does not install mise tools.%s\n' "$text_color" "$reset_color"
            printf '%s  Add the mise install block to the Dockerfile, or delete it to regenerate.%s\n' "$text_color" "$reset_color"
        fi
        echo ""
        return
    fi

    {
        # shellcheck disable=SC2016 # Keep Dockerfile ARG references literal.
        printf 'ARG CLAMP_TOOL_IMAGE=%s
FROM ${CLAMP_TOOL_IMAGE}

ARG USERNAME=dev

COPY .clamp/allowed-domains.d/ /opt/clamp-shared/allowed-domains.d/

USER $USERNAME
WORKDIR /workspace
' "$base_image"

        if [ -f "$workspace/mise.toml" ]; then
            # shellcheck disable=SC2016 # Keep Dockerfile ARG references literal.
            printf '
COPY --chown=$USERNAME:$USERNAME mise.toml /tmp/mise.toml
RUN mise install -C /tmp
'
        fi
    } > "$project_dockerfile"

    if [ "$printed_project_setup" = false ]; then
        printf '%s[Clamp Project Setup]%s\n' "$header_color" "$reset_color"
    fi
    printf '%s  Created project Dockerfile: %s%s\n' "$text_color" "$project_dockerfile" "$reset_color"
    echo ""
}

clamp_build_image() {
    local tool_dockerfile base_dockerfile workspace project_key project_dockerfile project_image_name tool_image_name

    workspace="$(cd "$CLAMP_FOLDER" && pwd)"
    project_key="$(printf '%s' "$workspace" | cksum | awk '{print $1}')"
    tool_image_name="clamp-${CLAMP_HARNESS}"
    project_image_name="${tool_image_name}-${project_key}"
    CLAMP_IMAGE_NAME="$project_image_name"
    base_dockerfile="$CLAMP_SCRIPT_DIR/clamp-base.Dockerfile"
    tool_dockerfile="$CLAMP_SCRIPT_DIR/clamp-${CLAMP_HARNESS}.Dockerfile"
    project_dockerfile="$workspace/.clamp/clamp.Dockerfile"

    if [ ! -f "$base_dockerfile" ]; then
        echo "Missing Dockerfile: $base_dockerfile"
        exit 1
    fi
    if [ ! -f "$tool_dockerfile" ]; then
        echo "Missing Dockerfile: $tool_dockerfile"
        exit 1
    fi

    # Build base if needed
    if [ "$CLAMP_REBUILD" = true ] || ! docker image inspect "$CLAMP_BASE_IMAGE_NAME" &>/dev/null; then
        echo "Building $CLAMP_BASE_IMAGE_NAME from clamp-base.Dockerfile..."
        docker build $CLAMP_NO_CACHE -f "$base_dockerfile" -t "$CLAMP_BASE_IMAGE_NAME" "$CLAMP_SCRIPT_DIR"
    fi

    # Build tool image if needed
    if [ "$CLAMP_REBUILD" = true ] || ! docker image inspect "$tool_image_name" &>/dev/null; then
        echo "Building $tool_image_name from clamp-${CLAMP_HARNESS}.Dockerfile..."
        docker build $CLAMP_NO_CACHE -f "$tool_dockerfile" -t "$tool_image_name" "$CLAMP_SCRIPT_DIR"
    fi

    clamp_create_project_dockerfile "$workspace"

    # Always run the project build so edits to .clamp/clamp.Dockerfile are picked up;
    # Docker's layer cache keeps the no-change path fast.
    echo "Building $project_image_name from .clamp/clamp.Dockerfile..."
    docker build $CLAMP_NO_CACHE --build-arg "CLAMP_TOOL_IMAGE=$tool_image_name" -f "$project_dockerfile" -t "$project_image_name" "$workspace"
}

clamp_detect_timezone() {
    local timezone="${TZ:-}"

    if [ -z "$timezone" ] && [ -r /etc/timezone ]; then
        timezone="$(head -n 1 /etc/timezone)"
    fi

    if [ -z "$timezone" ] && [ -L /etc/localtime ]; then
        timezone="$(readlink /etc/localtime | sed 's|^.*zoneinfo/||')"
    fi

    if [[ ! "$timezone" =~ ^[A-Za-z0-9_+./-]+$ ]]; then
        return 1
    else
        printf '%s\n' "$timezone"
    fi
}

clamp_run() {
    local workspace container_name project_name project_key config_volume cap_opts proxy_env timezone final_cmd workflows_enabled session_log_dir session_log_file session_log_container
    local startup_args docker_run_status
    local -a timezone_env=()

    workspace="$(cd "$CLAMP_FOLDER" && pwd)"
    container_name="${CLAMP_VOLUME_PREFIX}-$(date +%Y%m%d-%H%M%S)"
    project_name=$(basename "$workspace")
    project_key="$(printf '%s' "$workspace" | cksum | awk '{print $1}')"
    session_log_dir="$workspace/.clamp/sessions"
    session_log_file="$session_log_dir/$(date +%Y%m%d-%H%M%S)-network.log"
    session_log_container="/workspace/.clamp/sessions/${session_log_file##*/}"
    mkdir -p "$session_log_dir"

    # Determine config volume name based on auth policy
    if [ "$CLAMP_PER_PROJECT_AUTH" = true ]; then
        config_volume="${CLAMP_VOLUME_PREFIX}-${project_name}-${project_key}"
    else
        config_volume="${CLAMP_VOLUME_PREFIX}"
    fi

    # Set startup options based on flags
    startup_args=("--harness=${CLAMP_HARNESS}")
    if [ "$CLAMP_NO_FIREWALL" = true ]; then
        startup_args+=("--no-firewall")
        cap_opts=""
        proxy_env=""
    else
        cap_opts="--cap-add=NET_ADMIN"
        proxy_env="-e HTTP_PROXY=http://127.0.0.1:8888 -e HTTPS_PROXY=http://127.0.0.1:8888 -e http_proxy=http://127.0.0.1:8888 -e https_proxy=http://127.0.0.1:8888 -e NO_PROXY=localhost,127.0.0.1,::1 -e no_proxy=localhost,127.0.0.1,::1"
    fi

    # Keep container-side session log timestamps in the host's local time.
    if timezone="$(clamp_detect_timezone)"; then
        timezone_env=(-e "TZ=$timezone")
    fi
    # Check workflows environment variable
    workflows_enabled="${!CLAMP_WORKFLOWS_ENV:-false}"
    if [ "$workflows_enabled" = true ]; then
        startup_args+=("--add-workflows")
    fi

    # Set final command based on --shell and --danger flags
    final_cmd="$CLAMP_COMMAND"
    if [ "$CLAMP_DANGER_MODE" = true ] && [ -n "$CLAMP_DANGER_FLAG" ]; then
        final_cmd="$CLAMP_COMMAND $CLAMP_DANGER_FLAG"
    fi
    if [ "$CLAMP_SHELL_MODE" = true ]; then
        final_cmd="bash"
    fi

    # Run with:
    # - delegated mount for macOS performance
    # - tmpfs over /workspace/.clamp so containerized agents cannot see Clamp's project config
    # - bind mount only /workspace/.clamp/sessions back to the host for session logs
    # - Named volumes for heavy I/O directories (build artifacts, caches)
    # - Credential volumes (config copied fresh from image on each start)
    # - startup runs as root, then container-startup.sh execs the agent as dev
    # - NET_ADMIN capability for firewall setup/DNS ipset updates (unless --no-firewall)
    # - no-new-privileges prevents the dev agent from gaining privileges later
    # - Interactive TTY
    # - Auto-remove on exit
    if docker run -it --rm \
        --name "$container_name" \
        --user root \
        --security-opt no-new-privileges \
        $cap_opts \
        ${proxy_env:+$proxy_env} \
        "${timezone_env[@]}" \
        -v "$workspace:/workspace:delegated" \
        --tmpfs /workspace/.clamp:rw,noexec,nosuid,nodev,mode=700 \
        -v "$session_log_dir:/workspace/.clamp/sessions" \
        -v "${project_name}-node-modules:/workspace/node_modules" \
        -v "${project_name}-gradle-build:/workspace/build" \
        -v "${project_name}-gradle-cache:/home/dev/.gradle" \
        -v "${config_volume}:${CLAMP_CONFIG_DIR}" \
        -e "${CLAMP_CONFIG_ENV}=${CLAMP_CONFIG_DIR}" \
        -e "CLAMP_CONTAINER_USER=$CLAMP_CONTAINER_USER" \
        -e "CLAMP_BLOCKED_LOG_FILE=$session_log_container" \
        "$CLAMP_IMAGE_NAME" \
        /usr/local/bin/container-startup.sh "${startup_args[@]}" -- bash -c "$final_cmd"; then
        docker_run_status=0
    else
        docker_run_status=$?
    fi

    "$CLAMP_SCRIPT_DIR/clamp-shutdown.sh" "$workspace" || true
    return "$docker_run_status"
}

clamp_main() {
    clamp_parse_args "$@"
    clamp_resolve_auth_policy
    clamp_log_config
    clamp_build_image
    clamp_run
}
