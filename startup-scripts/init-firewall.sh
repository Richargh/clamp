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

FIREWALL_ALLOW_ALL=false
if [ "${1:-}" = "--allow-all" ]; then
    FIREWALL_ALLOW_ALL=true
    shift
fi

DOMAIN_SOURCES=("$@")

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

if [ "$FIREWALL_ALLOW_ALL" = true ]; then
    echo "Warning: --no-firewall is enabled; domain allowlist enforcement is disabled." >&2
elif [ ${#DOMAIN_FILES[@]} -eq 0 ]; then
    echo "Warning: no allowed domain files found in: ${DOMAIN_SOURCES[*]:-(none)}" >&2
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

NETWORK_LOG_FILE="${CLAMP_NETWORK_LOG_FILE:-/workspace/.clamp/sessions/$(date +%Y%m%d-%H%M%S)-network.log}"
BLOCKED_LOG_PID_FILE="/tmp/clamp-blocked-network-log.pid"
DNS_PID_FILE="/tmp/clamp-dns-firewall.pid"
PROXY_PID_FILE="/tmp/clamp-http-proxy.pid"
PROXY_UID="$(id -u clamp-proxy)"
UPSTREAM_DNS="${CLAMP_UPSTREAM_DNS:-$(awk '/^nameserver[[:space:]]+/ { print $2 }' /etc/resolv.conf | paste -sd, -)}"
if [ -z "$UPSTREAM_DNS" ]; then
    UPSTREAM_DNS="1.1.1.1,8.8.8.8"
fi

# Debian defaults to the nftables-backed iptables frontend. Some lightweight
# guest kernels (including versions used by Apple container) only support the
# legacy xtables interface. Select a backend by testing it against this kernel;
# never continue without a working firewall backend.
IPTABLES="iptables"
IP6TABLES="ip6tables"
if ! "$IPTABLES" -L OUTPUT -n >/dev/null 2>&1; then
    if command -v iptables-legacy >/dev/null 2>&1 \
        && iptables-legacy -L OUTPUT -n >/dev/null 2>&1; then
        IPTABLES="iptables-legacy"
        IP6TABLES="ip6tables-legacy"
    else
        echo "Error: the guest kernel supports neither iptables-nft nor iptables-legacy." >&2
        echo "Clamp cannot enable its network firewall; use --no-firewall only if unrestricted networking is acceptable." >&2
        exit 1
    fi
fi
echo "Firewall backend: $IPTABLES"

mkdir -p "$(dirname "$NETWORK_LOG_FILE")"
touch "$NETWORK_LOG_FILE"
chown root:root "$NETWORK_LOG_FILE" 2>/dev/null || true
chmod 644 "$NETWORK_LOG_FILE" 2>/dev/null || true

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

cat >> "$NETWORK_LOG_FILE" <<EOF

=== Firewall initialized at $(date -Is) ===
Mode: $([ "$FIREWALL_ALLOW_ALL" = true ] && echo "allow-all logging" || echo "allowlist enforcement")
Allowed domain files:
EOF
if [ ${#DOMAIN_FILES[@]} -eq 0 ]; then
    echo "  (none)" >> "$NETWORK_LOG_FILE"
else
    printf '  - %s\n' "${DOMAIN_FILES[@]}" >> "$NETWORK_LOG_FILE"
fi
echo "Upstream DNS: $UPSTREAM_DNS" >> "$NETWORK_LOG_FILE"

# Mirror kernel iptables LOG entries into a normal file. Domain-level decisions
# are logged directly by the local DNS firewall and HTTP(S) proxy.
(
    dmesg --follow --ctime 2>/dev/null \
        | grep --line-buffered 'CLAMP-KERNEL-' \
        >> "$NETWORK_LOG_FILE"
) &
echo $! > "$BLOCKED_LOG_PID_FILE"

# Force normal resolver users through the local DNS firewall/logger.
printf 'nameserver 127.0.0.1\noptions timeout:1 attempts:1\n' > /etc/resolv.conf

# Flush existing firewall state.
"$IPTABLES" -F OUTPUT 2>/dev/null || true
"$IP6TABLES" -F OUTPUT 2>/dev/null || true
ipset destroy allowed_ips 2>/dev/null || true

# Dynamic IPv4 set. DNS answers for approved domains are added by the DNS
# firewall with TTL-bounded timeouts.
ipset create allowed_ips hash:ip family inet hashsize 4096 maxelem 65536 timeout 86400 2>/dev/null || true

# IPv4 policy:
# - everyone may use loopback (agent -> local DNS/proxy)
# - root may query upstream DNS for the local DNS firewall
# - strict mode: clamp-proxy may connect to HTTP(S) ports on IPs produced by allowed DNS; everything else is logged and dropped
# - allow-all mode (--no-firewall): proxy traffic is allowed, direct outbound traffic is logged and allowed
"$IPTABLES" -A OUTPUT -o lo -j ACCEPT
"$IPTABLES" -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
"$IPTABLES" -A OUTPUT -m owner --uid-owner 0 -p udp --dport 53 -j ACCEPT
"$IPTABLES" -A OUTPUT -m owner --uid-owner 0 -p tcp --dport 53 -j ACCEPT
if [ "$FIREWALL_ALLOW_ALL" = true ]; then
    "$IPTABLES" -A OUTPUT -m owner --uid-owner "$PROXY_UID" -j ACCEPT
    "$IPTABLES" -A OUTPUT -m limit --limit 12/min --limit-burst 50 -j LOG --log-prefix "CLAMP-KERNEL-ALLOWED: " --log-level 4
    "$IPTABLES" -A OUTPUT -j ACCEPT
else
    "$IPTABLES" -A OUTPUT -m owner --uid-owner "$PROXY_UID" -p tcp -m multiport --dports 80,443 -m set --match-set allowed_ips dst -j ACCEPT
    "$IPTABLES" -A OUTPUT -m limit --limit 12/min --limit-burst 50 -j LOG --log-prefix "CLAMP-KERNEL-BLOCKED: " --log-level 4
    "$IPTABLES" -A OUTPUT -j DROP
fi

# IPv6 is closed by default in strict mode to avoid bypassing IPv4-only domain/IP enforcement.
"$IP6TABLES" -A OUTPUT -o lo -j ACCEPT 2>/dev/null || true
"$IP6TABLES" -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT 2>/dev/null || true
if [ "$FIREWALL_ALLOW_ALL" = true ]; then
    "$IP6TABLES" -A OUTPUT -m owner --uid-owner "$PROXY_UID" -j ACCEPT 2>/dev/null || true
    "$IP6TABLES" -A OUTPUT -m limit --limit 12/min --limit-burst 50 -j LOG --log-prefix "CLAMP-KERNEL-ALLOWED6: " --log-level 4 2>/dev/null || true
    "$IP6TABLES" -A OUTPUT -j ACCEPT 2>/dev/null || true
else
    "$IP6TABLES" -A OUTPUT -m limit --limit 12/min --limit-burst 50 -j LOG --log-prefix "CLAMP-KERNEL-BLOCKED6: " --log-level 4 2>/dev/null || true
    "$IP6TABLES" -A OUTPUT -j DROP 2>/dev/null || true
fi

# Start local DNS firewall as root because it binds :53 and updates ipset.
CLAMP_NETWORK_LOG_FILE="$NETWORK_LOG_FILE" CLAMP_UPSTREAM_DNS="$UPSTREAM_DNS" \
    CLAMP_DNS_ALLOW_ALL=$([ "$FIREWALL_ALLOW_ALL" = true ] && echo 1 || echo 0) \
    /opt/node/bin/node /usr/local/bin/clamp-dns-firewall.mjs "${DOMAIN_FILES[@]}" &
echo $! > "$DNS_PID_FILE"

# Start the HTTP(S) proxy as an unprivileged user. In strict mode, this is the
# only user allowed to reach whitelisted HTTP(S) IPs externally.
# shellcheck disable=SC2094 # Proxy writes logs to stdout; root-owned shell redirects to the log file.
CLAMP_NETWORK_LOG_FILE="$NETWORK_LOG_FILE" CLAMP_LOG_STDOUT=1 \
    CLAMP_PROXY_ALLOW_ALL=$([ "$FIREWALL_ALLOW_ALL" = true ] && echo 1 || echo 0) \
    runuser -u clamp-proxy --preserve-environment -- /opt/node/bin/node /usr/local/bin/clamp-http-proxy.mjs "${DOMAIN_FILES[@]}" \
        >> "$NETWORK_LOG_FILE" 2>&1 &
echo $! > "$PROXY_PID_FILE"

sleep 0.2

if [ "$FIREWALL_ALLOW_ALL" = true ]; then
    echo "Firewall initialized in allow-all logging mode (--no-firewall)."
else
    echo "Firewall initialized. Allowed domains:"
    if [ ${#ALLOWED_DOMAINS[@]} -eq 0 ]; then
        echo "  (none)"
    else
        printf '  - %s\n' "${ALLOWED_DOMAINS[@]}"
    fi
fi
echo ""
echo "Local DNS firewall: 127.0.0.1:53"
echo "Local HTTP(S) proxy: http://127.0.0.1:8888"
echo "Network log: $NETWORK_LOG_FILE"
echo "Upstream DNS: $UPSTREAM_DNS"
WHITELISTED_IP_COUNT="$(ipset list allowed_ips 2>/dev/null | grep -c '^[0-9]' || true)"
echo "Total IPs currently whitelisted: ${WHITELISTED_IP_COUNT:-0}"
