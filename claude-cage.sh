#!/bin/bash
set -e

REBUILD=false
NO_CACHE=""

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
        -*)
            echo "Unknown option: $1"
            echo "Usage: $(basename "$0") [--rebuild] [--no-cache] <folder>"
            echo "  --rebuild: Force rebuild of the Docker image"
            echo "  --no-cache: Rebuild without Docker layer cache"
            echo "  folder: Path to the project folder to run in"
            exit 1
            ;;
        *)
            FOLDER="$1"
            shift
            ;;
    esac
done

if [ -z "$FOLDER" ]; then
    echo "Usage: $(basename "$0") [--rebuild] [--no-cache] <folder>"
    echo "  --rebuild: Force rebuild of the Docker image"
    echo "  --no-cache: Rebuild without Docker layer cache"
    echo "  folder: Path to the project folder to run in"
    exit 1
fi

WORKSPACE="$(cd "$FOLDER" && pwd)"
IMAGE_NAME="claude-cage"
CONTAINER_NAME="claude-cage-$(date +%Y%m%d-%H%M%S)"
PROJECT_NAME=$(basename "$WORKSPACE")

# Build if image doesn't exist or --rebuild flag is set
if [ "$REBUILD" = true ] || ! docker image inspect "$IMAGE_NAME" &>/dev/null; then
    echo "Building $IMAGE_NAME..."
    docker build $NO_CACHE -t "$IMAGE_NAME" "$(dirname "$0")"
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
    -v "$HOME/.claude/.credentials.json:/home/dev/.claude/.credentials.json:delegated" \
    -v "$HOME/.claude.json:/home/dev/.claude.json:delegated" \
    "$IMAGE_NAME" \
    bash -c 'sudo /usr/local/bin/init-firewall.sh && claude'
