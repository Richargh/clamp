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

clamp_print_header() {
    if [ -t 1 ]; then
        printf '\033[1;36m[%s]\033[0m\n' "$1"
    else
        printf '[%s]\n' "$1"
    fi
}

clamp_print_text() {
    if [ -t 1 ]; then
        printf '\033[90m  '
        printf '%s' "$@"
        printf '\033[0m\n'
    else
        printf '  '
        printf '%s' "$@"
        printf '\n'
    fi
}

clamp_print_warning() {
    if [ -t 2 ]; then
        printf '\033[1;33m[Warning]\033[0m ' >&2
    else
        printf '[Warning] ' >&2
    fi
    printf '%s' "$@" >&2
    printf '\n' >&2
}

CLAMP_SHELL_MODE=false
CLAMP_FOLDER=""
CLAMP_RUNTIME="docker"

# User config
CLAMP_CONFIG_HOME="${XDG_CONFIG_HOME:-$HOME/.config}/clamp"
CLAMP_AUTH_CONFIG="$CLAMP_CONFIG_HOME/auth-projects.tsv"

clamp_usage() {
    echo "Usage: $(basename "$0") [-h|--help] [-v|--version] [--runtime docker|container] [--rebuild] [--no-cache] [--per-project-auth] [--no-firewall] [--danger] [--shell] <folder>"
    echo "  -h, --help: Show this help message"
    echo "  -v, --version: Show the Clamp version"
    echo "  --rebuild: Force rebuild of the container images"
    echo "  --no-cache: Rebuild without the image layer cache"
    echo "  --runtime: Container runtime to use: docker (default) or container (Apple container CLI)"
    echo "  --per-project-auth: Use separate credentials for this project"
    echo "  --no-firewall: Disable the network firewall (allow all outbound traffic). Useful for constrained research."
    echo "  --danger: Run ${CLAMP_TOOL_NAME} with auto-accept permissions (no confirmations)"
    echo "  --shell: Run startup then drop to shell instead of launching ${CLAMP_TOOL_NAME} (for debugging)"
    echo "  folder: Path to the project folder to run in"
    echo ""
    echo "Environment variables:"
    echo "  ${CLAMP_WORKFLOWS_ENV}: Set to 'true' to copy bundled workflows into container"
    echo "  CLAMP_CONTAINER_DNS: DNS server for Apple container builds/runs (default: 1.1.1.1)"
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
            --runtime)
                if [ $# -lt 2 ]; then
                    echo "Missing value for --runtime" >&2
                    exit 1
                fi
                CLAMP_RUNTIME="$2"
                shift 2
                ;;
            --runtime=*)
                CLAMP_RUNTIME="${1#*=}"
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
            -v|--version)
                if [ ! -r "$CLAMP_SCRIPT_DIR/VERSION" ]; then
                    echo "Missing version file: $CLAMP_SCRIPT_DIR/VERSION" >&2
                    exit 1
                fi
                printf 'Clamp %s (%s)\n' "$(head -n 1 "$CLAMP_SCRIPT_DIR/VERSION")" "$CLAMP_TOOL_NAME"
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

    case "$CLAMP_RUNTIME" in
        docker|container) ;;
        *)
            echo "Unsupported runtime: $CLAMP_RUNTIME (expected docker or container)" >&2
            exit 1
            ;;
    esac

    if ! command -v "$CLAMP_RUNTIME" >/dev/null 2>&1; then
        echo "Container runtime not found: $CLAMP_RUNTIME" >&2
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
    local auth firewall runtime

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

    if [ "$CLAMP_RUNTIME" = container ]; then
        runtime="Apple container"
    else
        runtime="Docker"
    fi

    clamp_print_header "Clamp User Config"
    clamp_print_text "Auth: $auth"
    clamp_print_text "Auth Config: $CLAMP_AUTH_CONFIG"
    echo ""
    clamp_print_header "Clamp CLI Config"
    clamp_print_text "Coding Agent: $CLAMP_TOOL_NAME"
    clamp_print_text "Runtime: $runtime"
    clamp_print_text "Firewall: $firewall"
    echo ""
}

clamp_write_project_dockerfile_header() {
    local base_image="$1"

    # shellcheck disable=SC2016 # Keep Dockerfile ARG references literal.
    printf 'ARG CLAMP_TOOL_IMAGE=%s
FROM ${CLAMP_TOOL_IMAGE}

ARG CONTAINER_USER=dev

COPY .clamp/allowed-domains.d/ /opt/clamp-shared/allowed-domains.d/

USER $CONTAINER_USER
WORKDIR /workspace

' "$base_image"
}

clamp_write_project_dockerignore_additions() {
    local workspace="$1"

    printf '# Include build-context files referenced by .clamp/clamp.Dockerfile.
# Clamp already includes .clamp/**.
#
# Examples:
# !mise.toml
# !requirements.txt
# !src/
# !src/**
'

    if [ -f "$workspace/mise.toml" ]; then
        printf '\n!mise.toml\n'
    fi
}

clamp_create_project_dockerfile() {
    local workspace project_dockerfile project_dockerignore project_allowed_domains_dir default_allowed_domains_dir printed_project_setup

    printed_project_setup=false

    workspace="$1"
    project_dockerfile="$workspace/.clamp/clamp.Dockerfile"
    project_dockerignore="$workspace/.clamp/clamp.Dockerfile.dockerignore"
    project_allowed_domains_dir="$workspace/.clamp/allowed-domains.d"
    default_allowed_domains_dir="$CLAMP_SCRIPT_DIR/clamp-shared/allowed-domains.d"

    mkdir -p "$workspace/.clamp"

    if [ ! -d "$project_allowed_domains_dir" ]; then
        mkdir -p "$project_allowed_domains_dir"
        if [ -d "$default_allowed_domains_dir" ]; then
            cp -a "$default_allowed_domains_dir/." "$project_allowed_domains_dir/"
        fi
        clamp_print_header "Clamp Project Setup"
        printed_project_setup=true
        clamp_print_text "Created allowed domains directory: $project_allowed_domains_dir"
        clamp_print_text 'Review this directory and delete domain files this project should not allow.'
    fi

    if [ ! -f "$project_dockerignore" ]; then
        clamp_write_project_dockerignore_additions "$workspace" > "$project_dockerignore"
        if [ "$printed_project_setup" = false ]; then
            clamp_print_header "Clamp Project Setup"
            printed_project_setup=true
        fi
        clamp_print_text "Created project Dockerignore additions: $project_dockerignore"
    fi

    if [ -f "$project_dockerfile" ]; then
        if [ "$printed_project_setup" = false ]; then
            clamp_print_header "Clamp Project Setup"
            printed_project_setup=true
        fi
        clamp_print_text "Using project Dockerfile additions: $project_dockerfile"
        if grep -Eq '^[[:space:]]*FROM[[:space:]]+' "$project_dockerfile"; then
            clamp_print_warning 'This is a legacy complete Dockerfile; remove its standard Clamp header to use the dynamic header.'
        fi
        if [ -f "$workspace/mise.toml" ] && ! grep -q 'mise install -C /tmp' "$project_dockerfile"; then
            clamp_print_text 'Note: mise.toml exists, but this Dockerfile does not install mise tools.'
            clamp_print_text 'Add the mise install block to the Dockerfile, or delete it to regenerate.'
        fi
        echo ""
        return
    fi

    # shellcheck disable=SC2016 # Keep Dockerfile ARG references literal.
    printf '# Clamp prepends the base image automatically.
#
USER 0
# Add system packages as needed here, for example:
# RUN apt-get update && apt-get install -y --no-install-recommends \
#     <package> \
#     && rm -rf /var/lib/apt/lists/*
USER $CONTAINER_USER

# Other examples:
# COPY --chown=$CONTAINER_USER:$CONTAINER_USER mise.toml /tmp/mise.toml
# RUN mise install -C /tmp
# COPY --chown=$CONTAINER_USER:$CONTAINER_USER requirements.txt /tmp/requirements.txt
# RUN pip install --user -r /tmp/requirements.txt
# RUN npm install --global <package>
' > "$project_dockerfile"

    if [ -f "$workspace/mise.toml" ]; then
        # shellcheck disable=SC2016 # Keep Dockerfile ARG references literal.
        printf '
COPY --chown=$CONTAINER_USER:$CONTAINER_USER mise.toml /tmp/mise.toml
RUN mise install -C /tmp
' >> "$project_dockerfile"
    fi

    if [ "$printed_project_setup" = false ]; then
        clamp_print_header "Clamp Project Setup"
    fi
    clamp_print_text "Created project Dockerfile: $project_dockerfile"
    echo ""
}

clamp_build_project_image() (
    local project_dockerfile="$1"
    local tool_image_name="$2"
    local project_image_name="$3"
    local workspace="$4"
    local project_dockerignore="$workspace/.clamp/clamp.Dockerfile.dockerignore"
    local project_build_dockerfile project_build_dockerignore
    shift 4
    local -a runtime_build_opts=("$@")

    # A FROM instruction identifies the legacy complete-Dockerfile format.
    if grep -Eq '^[[:space:]]*FROM[[:space:]]+' "$project_dockerfile"; then
        project_build_dockerfile="$project_dockerfile"
    else
        project_build_dockerfile="$(mktemp "$workspace/.clamp/clamp.generated.XXXXXX")"
        project_build_dockerignore="${project_build_dockerfile}.dockerignore"
        trap 'rm -f "$project_build_dockerfile" "$project_build_dockerignore"' EXIT
        clamp_write_project_dockerfile_header "$tool_image_name" > "$project_build_dockerfile"
        cat "$project_dockerfile" >> "$project_build_dockerfile"

        # Restrict the build context to Clamp's files plus the paths explicitly
        # included alongside the user's project Dockerfile additions.
        printf '*\n!.clamp/\n!.clamp/**\n' > "$project_build_dockerignore"
        if [ -f "$project_dockerignore" ]; then
            cat "$project_dockerignore" >> "$project_build_dockerignore"
        fi
    fi

    "$CLAMP_RUNTIME" build "${runtime_build_opts[@]}" $CLAMP_NO_CACHE --build-arg "CLAMP_TOOL_IMAGE=$tool_image_name" -f "$project_build_dockerfile" -t "$project_image_name" "$workspace"
)

clamp_build_image() {
    local tool_dockerfile base_dockerfile workspace project_key project_dockerfile project_image_name tool_image_name
    local build_tool
    local -a runtime_build_opts=()

    workspace="$(cd "$CLAMP_FOLDER" && pwd)"
    project_key="$(printf '%s' "$workspace" | cksum | awk '{print $1}')"
    tool_image_name="clamp-${CLAMP_HARNESS}"
    project_image_name="${tool_image_name}-${project_key}"
    CLAMP_IMAGE_NAME="$project_image_name"
    base_dockerfile="$CLAMP_SCRIPT_DIR/clamp-base.Dockerfile"
    tool_dockerfile="$CLAMP_SCRIPT_DIR/clamp-${CLAMP_HARNESS}.Dockerfile"
    project_dockerfile="$workspace/.clamp/clamp.Dockerfile"

    if [ "$CLAMP_RUNTIME" = container ]; then
        runtime_build_opts=(--dns "${CLAMP_CONTAINER_DNS:-1.1.1.1}")
        build_tool="Apple container"
    else
        build_tool="Docker"
    fi

    if [ ! -f "$base_dockerfile" ]; then
        echo "Missing Dockerfile: $base_dockerfile"
        exit 1
    fi
    if [ ! -f "$tool_dockerfile" ]; then
        echo "Missing Dockerfile: $tool_dockerfile"
        exit 1
    fi

    # Build base if needed
    if [ "$CLAMP_REBUILD" = true ] || ! "$CLAMP_RUNTIME" image inspect "$CLAMP_BASE_IMAGE_NAME" &>/dev/null; then
        clamp_print_header "Clamp Image Build"
        clamp_print_text "Tool: $build_tool"
        clamp_print_text "Image: $CLAMP_BASE_IMAGE_NAME"
        clamp_print_text 'Dockerfile: clamp-base.Dockerfile'
        "$CLAMP_RUNTIME" build "${runtime_build_opts[@]}" $CLAMP_NO_CACHE -f "$base_dockerfile" -t "$CLAMP_BASE_IMAGE_NAME" "$CLAMP_SCRIPT_DIR"
        echo ""
    fi

    # Build tool image if needed
    if [ "$CLAMP_REBUILD" = true ] || ! "$CLAMP_RUNTIME" image inspect "$tool_image_name" &>/dev/null; then
        clamp_print_header "Clamp Image Build"
        clamp_print_text "Tool: $build_tool"
        clamp_print_text "Image: $tool_image_name"
        clamp_print_text "Dockerfile: clamp-${CLAMP_HARNESS}.Dockerfile"
        "$CLAMP_RUNTIME" build "${runtime_build_opts[@]}" $CLAMP_NO_CACHE -f "$tool_dockerfile" -t "$tool_image_name" "$CLAMP_SCRIPT_DIR"
        echo ""
    fi

    clamp_create_project_dockerfile "$workspace"

    # Always run the project build so edits to .clamp/clamp.Dockerfile are picked up;
    # The runtime's layer cache keeps the no-change path fast.
    clamp_print_header "Clamp Image Build"
    clamp_print_text "Tool: $build_tool"
    clamp_print_text "Image: $project_image_name"
    clamp_print_text 'Dockerfile: .clamp/clamp.Dockerfile'
    clamp_print_text 'Why: Clamp builds a per-project image so project Dockerfile changes, allowed-domain files, and tool setup are applied before the agent starts.'
    clamp_build_project_image "$project_dockerfile" "$tool_image_name" "$project_image_name" "$workspace" "${runtime_build_opts[@]}" || return $?
    echo ""
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
    local startup_args runtime_status
    local -a timezone_env=() runtime_security_opts=() runtime_network_opts=() workspace_volume=()

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
    proxy_env="\
-e HTTP_PROXY=http://127.0.0.1:8888 \
-e HTTPS_PROXY=http://127.0.0.1:8888 \
-e NO_PROXY="
    cap_opts="--cap-add=NET_ADMIN"
    if [ "$CLAMP_NO_FIREWALL" = true ]; then
        startup_args+=("--no-firewall")
    fi

    # Keep container-side session log timestamps in the host's local time.
    if timezone="$(clamp_detect_timezone)"; then
        timezone_env=(-e "TZ=$timezone")
    fi

    if [ "$CLAMP_RUNTIME" = docker ]; then
        runtime_security_opts=(--security-opt no-new-privileges)
        workspace_volume=(-v "$workspace:/workspace:delegated")
    else
        # Apple's CLI does not support Docker's --security-opt or :delegated suffix.
        runtime_network_opts=(--dns "${CLAMP_CONTAINER_DNS:-1.1.1.1}")
        workspace_volume=(-v "$workspace:/workspace")
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
    # - NET_ADMIN capability for firewall setup/DNS ipset updates
    # - Docker's no-new-privileges prevents the dev agent from gaining privileges later
    # - Interactive TTY
    # - Auto-remove on exit
    if "$CLAMP_RUNTIME" run -it --rm \
        --name "$container_name" \
        --user root \
        "${runtime_security_opts[@]}" \
        "${runtime_network_opts[@]}" \
        $cap_opts \
        ${proxy_env:+$proxy_env} \
        "${timezone_env[@]}" \
        "${workspace_volume[@]}" \
        --tmpfs /workspace/.clamp:rw,noexec,nosuid,nodev,mode=700 \
        -v "$session_log_dir:/workspace/.clamp/sessions" \
        -v "${project_name}-node-modules:/workspace/node_modules" \
        -v "${project_name}-gradle-build:/workspace/build" \
        -v "${project_name}-gradle-cache:/home/dev/.gradle" \
        -v "${config_volume}:${CLAMP_CONFIG_DIR}" \
        -e "${CLAMP_CONFIG_ENV}=${CLAMP_CONFIG_DIR}" \
        -e "CLAMP_CONFIG_DIR=${CLAMP_CONFIG_DIR}" \
        -e "CLAMP_CONTAINER_USER=$CLAMP_CONTAINER_USER" \
        -e "CLAMP_NETWORK_LOG_FILE=$session_log_container" \
        "$CLAMP_IMAGE_NAME" \
        /opt/clamp/startup/container-startup.sh "${startup_args[@]}" -- bash -c "$final_cmd"; then
        runtime_status=0
    else
        runtime_status=$?
    fi

    "$CLAMP_SCRIPT_DIR/shutdown-scripts/clamp-shutdown.sh" "$workspace" "${session_log_file##*/}" || true
    return "$runtime_status"
}

clamp_main() {
    clamp_parse_args "$@"
    clamp_resolve_auth_policy
    clamp_log_config
    clamp_build_image
    clamp_run
}
