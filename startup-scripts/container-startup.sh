#!/bin/bash
set -e

### Notes on how Claude reads files in the user directory
# Statusline is not captured but rerun
# Hooks are captured at startup and are used throughout the session
## https://code.claude.com/docs/en/hooks#configuration-safety
# Skill descriptions are loaded at startup, but full skill content only loads when invoked
## https://code.claude.com/docs/en/skills
# Subagents are loaded at session start.
## https://code.claude.com/docs/en/sub-agents#write-subagent-files
# Rules are automatically loaded as project memory when launched
## https://code.claude.com/docs/en/memory

# Parse startup options. Arguments after -- are the coding-agent command to run
# as the unprivileged container user.
if [ -z "${CLAMP_HOST_PROXY_HOST:-}" ]; then
    echo "Error: CLAMP_HOST_PROXY_HOST must be set" >&2
    exit 1
fi
if [ -z "${CLAMP_HOST_PROXY_PORT:-}" ]; then
    echo "Error: CLAMP_HOST_PROXY_PORT must be set" >&2
    exit 1
fi
if [ -z "${CLAMP_CONTAINER_USER:-}" ]; then
    echo "Error: CLAMP_CONTAINER_USER must be set" >&2
    exit 1
fi
USER_ENTRY="$(getent passwd "$CLAMP_CONTAINER_USER")"
if [ -z "$USER_ENTRY" ]; then
    echo "Error: user '$CLAMP_CONTAINER_USER' does not exist" >&2
    exit 1
fi
IFS=: read -r _user_name _password _uid _gid _gecos CLAMP_CONTAINER_HOME _shell <<< "$USER_ENTRY"
if [ -z "$CLAMP_CONTAINER_HOME" ]; then
    echo "Error: user '$CLAMP_CONTAINER_USER' has no home directory" >&2
    exit 1
fi
if [ -z "${CLAMP_CONFIG_DIR:-}" ]; then
    echo "Error: CLAMP_CONFIG_DIR must be set" >&2
    exit 1
fi
case "$CLAMP_CONFIG_DIR" in
    "$CLAMP_CONTAINER_HOME"/*) ;;
    *)
        echo "Error: CLAMP_CONFIG_DIR must be inside $CLAMP_CONTAINER_HOME" >&2
        exit 1
        ;;
esac

# Docker initializes a new named volume from the image directory, preserving its
# owner. Apple's container CLI creates an empty root-owned volume instead. Make
# every writable named-volume mount usable by the unprivileged coding-agent user.
for writable_dir in \
    "$CLAMP_CONFIG_DIR" \
    /workspace/node_modules \
    /workspace/build \
    "$CLAMP_CONTAINER_HOME/.gradle"
do
    if [ -L "$writable_dir" ]; then
        echo "Error: writable volume path must not be a symlink: $writable_dir" >&2
        exit 1
    fi
    mkdir -p "$writable_dir"
    chown --no-dereference "$_uid:$_gid" "$writable_dir"
done

# Domain policy is enforced by the host proxy.
# The container must never get a direct outbound path.
# shellcheck source=startup-scripts/init-host-proxy-firewall.sh
source /opt/clamp/startup/init-host-proxy-firewall.sh

HARNESS=""
HARNESS_CMD=()
# Use the resolved address: direct DNS is now blocked.
HARNESS_ENV=(
    "HOME=$CLAMP_CONTAINER_HOME"
    "USER=$CLAMP_CONTAINER_USER"
    "LOGNAME=$CLAMP_CONTAINER_USER"
    "PATH=$PATH"
    "HTTP_PROXY=http://$CLAMP_HOST_PROXY_HOST:$CLAMP_HOST_PROXY_PORT"
    "HTTPS_PROXY=http://$CLAMP_HOST_PROXY_HOST:$CLAMP_HOST_PROXY_PORT"
    "NO_PROXY=${NO_PROXY?Error: NO_PROXY must be set}"
    "http_proxy=http://$CLAMP_HOST_PROXY_HOST:$CLAMP_HOST_PROXY_PORT"
    "https_proxy=http://$CLAMP_HOST_PROXY_HOST:$CLAMP_HOST_PROXY_PORT"
    "no_proxy=$NO_PROXY"
)
while [ $# -gt 0 ]; do
    case $1 in
        --harness=*) HARNESS="${1#--harness=}"; shift ;;
        --)
            shift
            HARNESS_CMD=("$@")
            break
            ;;
        *)
            echo "Error: unknown startup option '$1'"
            exit 1
            ;;
    esac
done

# Validate harness
if [ "$HARNESS" != "claude" ] && [ "$HARNESS" != "opencode" ] && [ "$HARNESS" != "pi" ]; then
    echo "Error: --harness must be either 'claude' or 'opencode', got '$HARNESS'"
    exit 1
fi

# Copy fresh config from template based on harness (settings and hooks)
# Credentials persist in the volume and are not overwritten
if [ "$HARNESS" = "claude" ]; then
    cp -a /opt/claude-config/* "$CLAMP_CONTAINER_HOME/.claude/"
elif [ "$HARNESS" = "opencode" ]; then
    cp -a /opt/opencode-config/* "$CLAMP_CONTAINER_HOME/.local/share/opencode/"
fi

if [ ${#HARNESS_CMD[@]} -eq 0 ]; then
    exit 0
fi

# Permanently hand off to the coding agent as the container user. Remove
# inheritable/ambient capabilities and clear the bounding set so the agent
# cannot alter or bypass the firewall with raw sockets.
exec setpriv \
    --reuid="$CLAMP_CONTAINER_USER" \
    --regid="$CLAMP_CONTAINER_USER" \
    --init-groups \
    --inh-caps=-all \
    --ambient-caps=-all \
    --bounding-set=-all \
    --reset-env \
    -- \
    env "${HARNESS_ENV[@]}" "${HARNESS_CMD[@]}"
