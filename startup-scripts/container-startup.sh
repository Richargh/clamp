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

NO_FIREWALL=false
ADD_WORKFLOWS=false
HARNESS=""
AGENT_CMD=()
while [ $# -gt 0 ]; do
    case $1 in
        --no-firewall) NO_FIREWALL=true; shift ;;
        --add-workflows) ADD_WORKFLOWS=true; shift ;;
        --harness=*) HARNESS="${1#--harness=}"; shift ;;
        --)
            shift
            AGENT_CMD=("$@")
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

# Copy workflows only when --add-workflows is set
if [ "$ADD_WORKFLOWS" = true ]; then
    if [ "$HARNESS" = "claude" ]; then
        cp -a /opt/claude-workflows/* "$CLAMP_CONTAINER_HOME/.claude/"
        WORKFLOW_COUNT=$(find /opt/claude-workflows -type f | wc -l)
        echo "Added $WORKFLOW_COUNT workflow files from container image"
    fi
fi

if [ "$NO_FIREWALL" = true ]; then
    if [ "$HARNESS" = "claude" ]; then
        # Remove web permissions (no firewall = no restrictions)
        sed -i '/"WebFetch(domain:\*)",/d' "$CLAMP_CONTAINER_HOME/.claude/settings.json"
        sed -i '/"WebSearch"/d' "$CLAMP_CONTAINER_HOME/.claude/settings.json"
    fi
    # OpenCode: no special handling needed for no-firewall mode
else
    if ! find /opt/clamp-shared/allowed-domains.d -maxdepth 1 -type f -name '*.txt' 2>/dev/null | grep -q .; then
        echo "No allowed-domain files found in /opt/clamp-shared/allowed-domains.d."
        echo "The firewall will block outbound traffic except DNS, loopback, and established connections."
        read -r -p "Continue? [y/N] " confirm
        case "$confirm" in
            [yY]|[yY][eE][sS]) ;;
            *) echo "Startup cancelled."; exit 1 ;;
        esac
    fi
    /usr/local/bin/init-firewall.sh /opt/clamp-shared/allowed-domains.d
fi

if [ ${#AGENT_CMD[@]} -eq 0 ]; then
    exit 0
fi

# Permanently hand off to the coding agent as the container user. Remove
# inheritable/ambient capabilities and remove NET_ADMIN from the bounding set so
# the agent and its children cannot alter the firewall even though the container
# needed NET_ADMIN during startup.
exec setpriv \
    --reuid="$CLAMP_CONTAINER_USER" \
    --regid="$CLAMP_CONTAINER_USER" \
    --init-groups \
    --inh-caps=-all \
    --ambient-caps=-all \
    --bounding-set=-net_admin \
    --reset-env \
    -- \
    env HOME="$CLAMP_CONTAINER_HOME" USER="$CLAMP_CONTAINER_USER" LOGNAME="$CLAMP_CONTAINER_USER" PATH="$PATH" "${AGENT_CMD[@]}"
