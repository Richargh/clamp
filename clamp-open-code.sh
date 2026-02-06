#!/bin/bash
set -e

# Resolve the real path of this script (handles symlinks)
CLAMP_SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "$0")")" && pwd)"

# shellcheck source=clamp-lib.sh
source "$CLAMP_SCRIPT_DIR/clamp-lib.sh"

# OpenCode specific settings
CLAMP_TOOL_NAME="OpenCode"
CLAMP_VOLUME_PREFIX="opencode-clamp"
CLAMP_CONFIG_DIR="/home/dev/.local/share/opencode"
CLAMP_CONFIG_ENV="OPENCODE_CONFIG_DIR"
CLAMP_COMMAND="opencode"
CLAMP_DANGER_FLAG=""  # OpenCode equivalent TBD
CLAMP_HARNESS="opencode"
CLAMP_WORKFLOWS_ENV="OPENCODE_CLAMP_ADD_WORKFLOWS"

clamp_main "$@"
