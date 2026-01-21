#!/bin/bash
set -e

# Copy fresh config from template (settings and hooks)
# Credentials (.credentials.json, .claude.json) persist in the volume and are not overwritten
cp -r /opt/claude-config/* /home/dev/.claude/
chown -R dev:dev /home/dev/.claude

# Parse options
NO_FIREWALL=false
for arg in "$@"; do
    case $arg in
        --no-firewall) NO_FIREWALL=true ;;
    esac
done

if [ "$NO_FIREWALL" = true ]; then
    # Swap to no-firewall hook
    sed -i 's|/firewall-preflight\.mjs|/no-firewall-preflight.mjs|g' /home/dev/.claude/settings.json
else
    # Initialize iptables firewall
    /usr/local/bin/init-firewall.sh
fi
