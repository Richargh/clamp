#!/bin/bash
set -e

# Resolve the real path of this script (handles symlinks)
SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "$0")")" && pwd)"

usage() {
    echo "Usage: $(basename "$0") [-h|--help] [--rebuild] [--no-cache] [--per-project-auth] <folder>"
    echo "  -h, --help: Show this help message"
    echo "  --rebuild: Force rebuild of the Docker image"
    echo "  --no-cache: Rebuild without Docker layer cache"
    echo "  --per-project-auth: Use separate credentials for this project"
    echo "  folder: Path to the project folder to run in"
}

REBUILD=false
NO_CACHE=""
PER_PROJECT_AUTH=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --rebuild)
            REBUILD=true
            shift
            ;;
        --no-cache)
            REBUILD=true
            NO_CACHE="--no-cache"
            shift
            ;;
        --per-project-auth)
            PER_PROJECT_AUTH=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        -*)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
        *)
            FOLDER="$1"
            shift
            ;;
    esac
done

if [ -z "$FOLDER" ]; then
    usage
    exit 1
fi

WORKSPACE="$(cd "$FOLDER" && pwd)"
IMAGE_NAME="claude-clamp"
CONTAINER_NAME="claude-clamp-$(date +%Y%m%d-%H%M%S)"
PROJECT_NAME=$(basename "$WORKSPACE")

# Determine claude config volume name based on flag
if [ "$PER_PROJECT_AUTH" = true ]; then
    CLAUDE_VOLUME="claude-clamp-${PROJECT_NAME}"
else
    CLAUDE_VOLUME="claude-clamp"
fi

# Build if image doesn't exist or --rebuild flag is set
if [ "$REBUILD" = true ] || ! docker image inspect "$IMAGE_NAME" &>/dev/null; then
    echo "Building $IMAGE_NAME..."
    docker build $NO_CACHE -t "$IMAGE_NAME" "$SCRIPT_DIR"
fi

# Run with:
# - delegated mount for macOS performance
# - Named volumes for heavy I/O directories (build artifacts, caches)
# - NET_ADMIN capability for firewall
# - Interactive TTY
# - Auto-remove on exit
docker run -it --rm \
    --name "$CONTAINER_NAME" \
    --cap-add=NET_ADMIN \
    -v "$WORKSPACE:/workspace:delegated" \
    -v "${PROJECT_NAME}-node-modules:/workspace/node_modules" \
    -v "${PROJECT_NAME}-gradle-build:/workspace/build" \
    -v "${PROJECT_NAME}-gradle-cache:/home/dev/.gradle" \
    -v "${CLAUDE_VOLUME}:/home/dev/.claude" \
    -e "CLAUDE_CONFIG_DIR=/home/dev/.claude" \
    "$IMAGE_NAME" \
    bash -c 'sudo /usr/local/bin/init-firewall.sh && claude'
