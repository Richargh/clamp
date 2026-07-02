#!/bin/bash
# Firewall initialization script for clamp containers
# Based on Anthropic's init-firewall.sh, extended with JVM repository domains

set -e

# Accept one or more domain files/directories as arguments. Directory arguments load
# all *.txt files directly inside the directory. If no arguments are provided,
# default to the Claude hook domains directory, plus the old single-file location
# for backwards compatibility.
DOMAIN_SOURCES=("$@")
if [ ${#DOMAIN_SOURCES[@]} -eq 0 ]; then
    DOMAIN_SOURCES=("/home/dev/.claude/hooks/allowed-domains.d" "/home/dev/.claude/hooks/allowed-domains.txt")
fi

DOMAIN_FILES=()
add_domain_file() {
    local path="$1"
    [ -f "$path" ] && DOMAIN_FILES+=("$path")
}

for source in "${DOMAIN_SOURCES[@]}"; do
    if [ -d "$source" ]; then
        while IFS= read -r -d '' file; do
            DOMAIN_FILES+=("$file")
        done < <(find "$source" -maxdepth 1 -type f -name '*.txt' -print0 | sort -z)
    else
        add_domain_file "$source"
    fi
done

if [ ${#DOMAIN_FILES[@]} -eq 0 ]; then
    echo "Warning: no allowed domain files found in: ${DOMAIN_SOURCES[*]}" >&2
fi

# Read domains from files (skip comments and empty lines)
declare -A SEEN_DOMAINS=()
ALLOWED_DOMAINS=()
for domain_file in "${DOMAIN_FILES[@]}"; do
    while IFS= read -r line || [[ -n "$line" ]]; do
        # Trim comments, whitespace, and normalize to lowercase
        line="${line%%#*}"
        line="${line//[[:space:]]/}"
        line="${line,,}"
        [[ -z "$line" ]] && continue
        if [[ -z "${SEEN_DOMAINS[$line]:-}" ]]; then
            SEEN_DOMAINS[$line]=1
            ALLOWED_DOMAINS+=("$line")
        fi
    done < "$domain_file"
done

echo "Initializing firewall with domain whitelist from:"
if [ ${#DOMAIN_FILES[@]} -eq 0 ]; then
    echo "  (no domain files; outbound traffic will be blocked except DNS/loopback/established connections)"
else
    printf '  - %s\n' "${DOMAIN_FILES[@]}"
fi

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
if [ ${#ALLOWED_DOMAINS[@]} -eq 0 ]; then
    echo "  (none)"
else
    printf '  - %s\n' "${ALLOWED_DOMAINS[@]}"
fi
echo ""
echo "Total IPs whitelisted: $(ipset list allowed_ips | grep -c '^[0-9]' || echo 0)"
