#!/bin/bash
# Firewall initialization script for clamp containers.
#
# The coding agent is not allowed to connect directly to the internet. It may
# only talk to loopback services started here:
#   - DNS firewall/logger on 127.0.0.1:53
#   - HTTP(S) forward proxy on 127.0.0.1:8888
# The proxy runs as clamp-proxy and is the only non-root user allowed to open
# outbound TCP connections to dynamically allowlisted IPs for approved domains.

set -e

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
        line="${line%%#*}"
        line="${line//[[:space:]]/}"
        line="${line,,}"
        line="${line%.}"
        [[ -z "$line" ]] && continue
        if [[ -z "${SEEN_DOMAINS[$line]:-}" ]]; then
            SEEN_DOMAINS[$line]=1
            ALLOWED_DOMAINS+=("$line")
        fi
    done < "$domain_file"
done

BLOCKED_LOG_FILE="${CLAMP_BLOCKED_LOG_FILE:-/workspace/.clamp/sessions/$(date +%Y%m%d-%H%M%S)-network.log}"
BLOCKED_LOG_PID_FILE="/tmp/clamp-blocked-network-log.pid"
DNS_PID_FILE="/tmp/clamp-dns-firewall.pid"
PROXY_PID_FILE="/tmp/clamp-http-proxy.pid"
PROXY_UID="$(id -u clamp-proxy)"
UPSTREAM_DNS="${CLAMP_UPSTREAM_DNS:-$(awk '/^nameserver[[:space:]]+/ { print $2 }' /etc/resolv.conf | paste -sd, -)}"
if [ -z "$UPSTREAM_DNS" ]; then
    UPSTREAM_DNS="1.1.1.1,8.8.8.8"
fi

mkdir -p "$(dirname "$BLOCKED_LOG_FILE")"
touch "$BLOCKED_LOG_FILE"
chown root:root "$BLOCKED_LOG_FILE" 2>/dev/null || true
chmod 644 "$BLOCKED_LOG_FILE" 2>/dev/null || true

stop_pid_file() {
    local pid_file="$1" old_pid
    if [ -f "$pid_file" ]; then
        old_pid="$(cat "$pid_file" 2>/dev/null || true)"
        if [ -n "$old_pid" ]; then
            kill "$old_pid" 2>/dev/null || true
        fi
    fi
}

stop_pid_file "$BLOCKED_LOG_PID_FILE"
stop_pid_file "$DNS_PID_FILE"
stop_pid_file "$PROXY_PID_FILE"

cat >> "$BLOCKED_LOG_FILE" <<EOF

=== Firewall initialized at $(date -Is) ===
Allowed domain files:
EOF
if [ ${#DOMAIN_FILES[@]} -eq 0 ]; then
    echo "  (none)" >> "$BLOCKED_LOG_FILE"
else
    printf '  - %s\n' "${DOMAIN_FILES[@]}" >> "$BLOCKED_LOG_FILE"
fi
echo "Upstream DNS: $UPSTREAM_DNS" >> "$BLOCKED_LOG_FILE"

# Mirror kernel iptables LOG entries into a normal file. Domain-level denials are
# logged directly by the local DNS firewall and HTTP(S) proxy.
(
    dmesg --follow --ctime 2>/dev/null \
        | grep --line-buffered 'CLAMP-BLOCKED:' \
        >> "$BLOCKED_LOG_FILE"
) &
echo $! > "$BLOCKED_LOG_PID_FILE"

# Force normal resolver users through the local DNS firewall/logger.
printf 'nameserver 127.0.0.1\noptions timeout:1 attempts:1\n' > /etc/resolv.conf

# Flush existing firewall state.
iptables -F OUTPUT 2>/dev/null || true
ip6tables -F OUTPUT 2>/dev/null || true
ipset destroy allowed_ips 2>/dev/null || true

# Dynamic IPv4 set. DNS answers for approved domains are added by the DNS
# firewall with TTL-bounded timeouts.
ipset create allowed_ips hash:ip family inet hashsize 4096 maxelem 65536 timeout 86400 2>/dev/null || true

# IPv4 policy:
# - everyone may use loopback (agent -> local DNS/proxy)
# - root may query upstream DNS for the local DNS firewall
# - clamp-proxy may connect to HTTP(S) ports on IPs produced by allowed DNS
# - everything else is logged and dropped
iptables -A OUTPUT -o lo -j ACCEPT
iptables -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -A OUTPUT -m owner --uid-owner 0 -p udp --dport 53 -j ACCEPT
iptables -A OUTPUT -m owner --uid-owner 0 -p tcp --dport 53 -j ACCEPT
iptables -A OUTPUT -m owner --uid-owner "$PROXY_UID" -p tcp -m multiport --dports 80,443 -m set --match-set allowed_ips dst -j ACCEPT
iptables -A OUTPUT -m limit --limit 12/min --limit-burst 50 -j LOG --log-prefix "CLAMP-BLOCKED: " --log-level 4
iptables -A OUTPUT -j DROP

# IPv6 is closed by default to avoid bypassing IPv4-only domain/IP enforcement.
ip6tables -A OUTPUT -o lo -j ACCEPT 2>/dev/null || true
ip6tables -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT 2>/dev/null || true
ip6tables -A OUTPUT -m limit --limit 12/min --limit-burst 50 -j LOG --log-prefix "CLAMP-BLOCKED6: " --log-level 4 2>/dev/null || true
ip6tables -A OUTPUT -j DROP 2>/dev/null || true

# Start local DNS firewall as root because it binds :53 and updates ipset.
CLAMP_BLOCKED_LOG_FILE="$BLOCKED_LOG_FILE" CLAMP_UPSTREAM_DNS="$UPSTREAM_DNS" \
    /opt/node/bin/node /usr/local/bin/clamp-dns-firewall.mjs "${DOMAIN_FILES[@]}" &
echo $! > "$DNS_PID_FILE"

# Start the HTTP(S) proxy as an unprivileged user. This is the only user allowed
# to reach whitelisted HTTP(S) IPs externally.
# shellcheck disable=SC2024,SC2094 # Redirection is intentionally opened by root; proxy writes logs to stdout.
CLAMP_BLOCKED_LOG_FILE="$BLOCKED_LOG_FILE" CLAMP_LOG_STDOUT=1 \
    sudo -u clamp-proxy -E /opt/node/bin/node /usr/local/bin/clamp-http-proxy.mjs "${DOMAIN_FILES[@]}" \
        >> "$BLOCKED_LOG_FILE" 2>&1 &
echo $! > "$PROXY_PID_FILE"

sleep 0.2

echo "Firewall initialized. Allowed domains:"
if [ ${#ALLOWED_DOMAINS[@]} -eq 0 ]; then
    echo "  (none)"
else
    printf '  - %s\n' "${ALLOWED_DOMAINS[@]}"
fi
echo ""
echo "Local DNS firewall: 127.0.0.1:53"
echo "Local HTTP(S) proxy: http://127.0.0.1:8888"
echo "Blocked domain/network log: $BLOCKED_LOG_FILE"
echo "Upstream DNS: $UPSTREAM_DNS"
echo "Total IPs currently whitelisted: $(ipset list allowed_ips | grep -c '^[0-9]' || echo 0)"
