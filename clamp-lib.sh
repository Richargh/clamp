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

# Image name is shared between all tools
CLAMP_IMAGE_NAME="claude-clamp"

# Parsed argument flags (set by clamp_parse_args)
CLAMP_REBUILD=false
CLAMP_NO_CACHE=""
CLAMP_PER_PROJECT_AUTH=false
CLAMP_NO_FIREWALL=false
CLAMP_DANGER_MODE=false
CLAMP_SHELL_MODE=false
CLAMP_FOLDER=""

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

clamp_build_image() {
    # Build if image doesn't exist or --rebuild flag is set
    if [ "$CLAMP_REBUILD" = true ] || ! docker image inspect "$CLAMP_IMAGE_NAME" &>/dev/null; then
        echo "Building $CLAMP_IMAGE_NAME..."
        docker build $CLAMP_NO_CACHE -t "$CLAMP_IMAGE_NAME" "$CLAMP_SCRIPT_DIR"
    fi
}

clamp_run() {
    local workspace container_name project_name config_volume startup_opts cap_opts final_cmd workflows_enabled

    workspace="$(cd "$CLAMP_FOLDER" && pwd)"
    container_name="${CLAMP_VOLUME_PREFIX}-$(date +%Y%m%d-%H%M%S)"
    project_name=$(basename "$workspace")

    # Determine config volume name based on flag
    if [ "$CLAMP_PER_PROJECT_AUTH" = true ]; then
        config_volume="${CLAMP_VOLUME_PREFIX}-${project_name}"
    else
        config_volume="${CLAMP_VOLUME_PREFIX}"
    fi

    # Set startup options based on flags
    startup_opts="--harness=${CLAMP_HARNESS}"
    if [ "$CLAMP_NO_FIREWALL" = true ]; then
        startup_opts="$startup_opts --no-firewall"
        cap_opts=""
    else
        cap_opts="--cap-add=NET_ADMIN"
    fi

    # Check workflows environment variable
    workflows_enabled="${!CLAMP_WORKFLOWS_ENV:-false}"
    if [ "$workflows_enabled" = true ]; then
        startup_opts="$startup_opts --add-workflows"
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
    # - Named volumes for heavy I/O directories (build artifacts, caches)
    # - Credential volumes (config copied fresh from image on each start)
    # - NET_ADMIN capability for firewall (unless --no-firewall)
    # - Interactive TTY
    # - Auto-remove on exit
    docker run -it --rm \
        --name "$container_name" \
        $cap_opts \
        -v "$workspace:/workspace:delegated" \
        -v "${project_name}-node-modules:/workspace/node_modules" \
        -v "${project_name}-gradle-build:/workspace/build" \
        -v "${project_name}-gradle-cache:/home/dev/.gradle" \
        -v "${config_volume}:${CLAMP_CONFIG_DIR}" \
        -e "${CLAMP_CONFIG_ENV}=${CLAMP_CONFIG_DIR}" \
        "$CLAMP_IMAGE_NAME" \
        bash -c "sudo /usr/local/bin/container-startup.sh $startup_opts && $final_cmd"
}

clamp_main() {
    clamp_parse_args "$@"
    clamp_build_image
    clamp_run
}
