#!/bin/bash
set -e

# Resolve the real path of this script (handles symlinks)
CLAMP_SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "$0")")" && pwd)"

# shellcheck source=clamp-lib.sh
source "$CLAMP_SCRIPT_DIR/clamp-lib.sh"

# Claude Code specific settings
CLAMP_TOOL_NAME="Pi"
CLAMP_VOLUME_PREFIX="pi"
CLAMP_CONFIG_DIR="/home/${CLAMP_CONTAINER_USER}/.pi"
CLAMP_CONFIG_ENV="CLAUDE_CONFIG_DIR"
CLAMP_COMMAND="pi"
CLAMP_DANGER_FLAG=""
CLAMP_HARNESS="pi"
CLAMP_WORKFLOWS_ENV="PI_CLAMP_ADD_WORKFLOWS"

clamp_main "$@"
