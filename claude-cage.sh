#!/bin/bash
set -e

if [ -z "$1" ]; then
    echo "Usage: $(basename "$0") <folder>"
    echo "  folder: Path to the project folder to run in"
    exit 1
fi

WORKSPACE="$(cd "$1" && pwd)"
IMAGE_NAME="claude-cage"
CONTAINER_NAME="claude-cage-$(date +%Y%m%d-%H%M%S)"
PROJECT_NAME=$(basename "$WORKSPACE")

# Build if image doesn't exist
if ! docker image inspect "$IMAGE_NAME" &>/dev/null; then
    echo "Building $IMAGE_NAME..."
    docker build -t "$IMAGE_NAME" "$(dirname "$0")"
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
    -v "$HOME/.claude/.credentials.json:/home/dev/.claude/.credentials.json:ro" \
    -v "$HOME/.claude.json:/home/dev/.claude.json:delegated" \
    "$IMAGE_NAME" \
    bash -c 'sudo /usr/local/bin/init-firewall.sh && claude'
