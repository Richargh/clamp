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

# Parse options
NO_FIREWALL=false
ADD_WORKFLOWS=false
HARNESS=""
for arg in "$@"; do
    case $arg in
        --no-firewall) NO_FIREWALL=true ;;
        --add-workflows) ADD_WORKFLOWS=true ;;
        --harness=*) HARNESS="${arg#--harness=}" ;;
    esac
done

# Validate harness
if [ "$HARNESS" != "claude" ] && [ "$HARNESS" != "opencode" ]; then
    echo "Error: --harness must be either 'claude' or 'opencode', got '$HARNESS'"
    exit 1
fi

# Copy fresh config from template based on harness (settings and hooks)
# Credentials persist in the volume and are not overwritten
if [ "$HARNESS" = "claude" ]; then
    cp -a /opt/claude-config/* /home/dev/.claude/
    cp /opt/clamp-shared/allowed-domains.txt /home/dev/.claude/hooks/
else
    cp -a /opt/opencode-config/* /home/dev/.local/share/opencode/
    mkdir -p /home/dev/.local/share/opencode/hooks
    cp /opt/clamp-shared/allowed-domains.txt /home/dev/.local/share/opencode/hooks/
fi

# Copy workflows only when --add-workflows is set
if [ "$ADD_WORKFLOWS" = true ]; then
    if [ "$HARNESS" = "claude" ]; then
        cp -a /opt/claude-workflows/* /home/dev/.claude/
        WORKFLOW_COUNT=$(find /opt/claude-workflows -type f | wc -l)
        echo "Added $WORKFLOW_COUNT workflow files from container image"
    fi
fi

if [ "$NO_FIREWALL" = true ]; then
    if [ "$HARNESS" = "claude" ]; then
        # claude has a special firewall hook to give faster feedback why a domain does not work
        # Swap to no-firewall hook and remove web permissions (no firewall = no restrictions)
        sed -i 's|/firewall-preflight\.mjs|/no-firewall-preflight.mjs|g' /home/dev/.claude/settings.json
        sed -i '/"WebFetch(domain:\*)",/d' /home/dev/.claude/settings.json
        sed -i '/"WebSearch"/d' /home/dev/.claude/settings.json
    fi
    # OpenCode: no special handling needed for no-firewall mode
else
    /usr/local/bin/init-firewall.sh /opt/clamp-shared/allowed-domains.txt
fi
