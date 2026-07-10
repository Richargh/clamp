#!/bin/bash
set -e

# Resolve the real path of this script (handles symlinks)
CLAMP_SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "$0")")" && pwd)"

# shellcheck source=clamp-lib.sh
source "$CLAMP_SCRIPT_DIR/clamp-lib.sh"

# Claude Code specific settings
CLAMP_TOOL_NAME="Claude Code"
CLAMP_VOLUME_PREFIX="claude-clamp"
CLAMP_CONFIG_DIR="/home/${CLAMP_CONTAINER_USER}/.claude"
CLAMP_CONFIG_ENV="CLAUDE_CONFIG_DIR"
CLAMP_COMMAND="claude"
CLAMP_DANGER_FLAG="--dangerously-skip-permissions"
CLAMP_HARNESS="claude"
CLAMP_WORKFLOWS_ENV="CLAUDE_CLAMP_ADD_WORKFLOWS"

clamp_main "$@"
