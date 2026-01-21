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

# Copy fresh config from template (settings and hooks)
# Credentials (.credentials.json, .claude.json) persist in the volume and are not overwritten
cp -a /opt/claude-config/* /home/dev/.claude/

# Parse options
NO_FIREWALL=false
COPY_WORKFLOWS=true
for arg in "$@"; do
    case $arg in
        --no-firewall) NO_FIREWALL=true ;;
        --no-workflows) COPY_WORKFLOWS=false ;;
    esac
done

# Copy workflows unless --no-workflows is set
if [ "$COPY_WORKFLOWS" = true ]; then
    cp -a /opt/claude-workflows/* /home/dev/.claude/
fi

if [ "$NO_FIREWALL" = true ]; then
    # Swap to no-firewall hook and remove web permissions (no firewall = no restrictions)
    sed -i 's|/firewall-preflight\.mjs|/no-firewall-preflight.mjs|g' /home/dev/.claude/settings.json
    sed -i '/"WebFetch(domain:\*)",/d' /home/dev/.claude/settings.json
    sed -i '/"WebSearch"/d' /home/dev/.claude/settings.json
else
    # Initialize iptables firewall
    /usr/local/bin/init-firewall.sh
fi
