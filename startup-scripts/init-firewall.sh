#!/bin/bash
# Firewall initialization script for clamp containers
# Based on Anthropic's init-firewall.sh, extended with JVM repository domains

set -e

# Accept domains file path as argument, default to Claude location for backwards compatibility
DOMAINS_FILE="${1:-/home/dev/.claude/hooks/allowed-domains.txt}"

# Read domains from file (skip comments and empty lines)
ALLOWED_DOMAINS=()
while IFS= read -r line || [[ -n "$line" ]]; do
    # Trim whitespace and skip comments/empty lines
    line="${line%%#*}"
    line="${line// /}"
    [[ -z "$line" ]] && continue
    ALLOWED_DOMAINS+=("$line")
done < "$DOMAINS_FILE"

echo "Initializing firewall with domain whitelist..."

# Flush existing rules
iptables -F OUTPUT 2>/dev/null || true
ipset destroy allowed_ips 2>/dev/null || true

# Create ipset for allowed IPs
ipset create allowed_ips hash:ip family inet hashsize 4096 maxelem 65536 2>/dev/null || true

# Always allow loopback
iptables -A OUTPUT -o lo -j ACCEPT

# Allow established connections
iptables -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Resolve and add IPs for each domain
for domain in "${ALLOWED_DOMAINS[@]}"; do
    echo "  Resolving $domain..."
    # Get all IPs for domain (IPv4 only)
    ips=$(dig +short A "$domain" 2>/dev/null | grep -E '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$' || true)
    for ip in $ips; do
        ipset add allowed_ips "$ip" 2>/dev/null || true
    done

    # Also try to resolve via host for CNAME chains
    ips=$(host -t A "$domain" 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+' || true)
    for ip in $ips; do
        ipset add allowed_ips "$ip" 2>/dev/null || true
    done
done

# Allow DNS (needed for domain resolution)
iptables -A OUTPUT -p udp --dport 53 -j ACCEPT
iptables -A OUTPUT -p tcp --dport 53 -j ACCEPT

# Allow whitelisted IPs
iptables -A OUTPUT -m set --match-set allowed_ips dst -j ACCEPT

# Log and drop everything else
iptables -A OUTPUT -j LOG --log-prefix "BLOCKED: " --log-level 4
iptables -A OUTPUT -j DROP

echo "Firewall initialized. Allowed domains:"
printf '  - %s\n' "${ALLOWED_DOMAINS[@]}"
echo ""
echo "Total IPs whitelisted: $(ipset list allowed_ips | grep -c '^[0-9]' || echo 0)"
