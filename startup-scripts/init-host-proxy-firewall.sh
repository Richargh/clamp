#!/bin/bash
# Permit loopback traffic and outbound TCP to the host proxy only.
# No direct external DNS or internet access.
set -euo pipefail

if [ -z "${CLAMP_HOST_PROXY_HOST:-}" ]; then
    echo "Error: CLAMP_HOST_PROXY_HOST must be set" >&2
    exit 1
fi
if [ -z "${CLAMP_HOST_PROXY_PORT:-}" ]; then
    echo "Error: CLAMP_HOST_PROXY_PORT must be set" >&2
    exit 1
fi
if [[ ! "$CLAMP_HOST_PROXY_PORT" =~ ^[0-9]+$ ]] \
    || (( CLAMP_HOST_PROXY_PORT < 1 || CLAMP_HOST_PROXY_PORT > 65535 )); then
    echo 'Error: invalid host proxy port' >&2
    exit 1
fi
# Resolve the runtime host name before direct DNS is blocked. This script is
# sourced, so startup uses the same address as the firewall.
CLAMP_HOST_PROXY_HOST="$(getent ahostsv4 "$CLAMP_HOST_PROXY_HOST" | awk 'NR == 1 { ip=$1 } END { print ip }')"
if [[ ! "$CLAMP_HOST_PROXY_HOST" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]]; then
    echo 'Error: cannot resolve the host proxy IPv4 address' >&2
    exit 1
fi

IPTABLES=iptables
IP6TABLES=ip6tables
if ! "$IPTABLES" -L OUTPUT -n >/dev/null 2>&1; then
    IPTABLES=iptables-legacy
    IP6TABLES=ip6tables-legacy
fi
# Set the default policy before the flush. Do not retain established-connection
# exceptions: connections opened before this script must not bypass the rules.
"$IPTABLES" -P OUTPUT DROP
"$IPTABLES" -F OUTPUT
"$IPTABLES" -A OUTPUT -o lo -j ACCEPT
if "$IP6TABLES" -L OUTPUT -n >/dev/null 2>&1; then
    "$IP6TABLES" -P OUTPUT DROP
    "$IP6TABLES" -F OUTPUT
    "$IP6TABLES" -A OUTPUT -o lo -j ACCEPT
else
    # Some guest kernels have no IPv6 firewall. Disable IPv6 or stop startup.
    sysctl -w net.ipv6.conf.all.disable_ipv6=1
    sysctl -w net.ipv6.conf.default.disable_ipv6=1
fi
"$IPTABLES" -A OUTPUT -p tcp -d "$CLAMP_HOST_PROXY_HOST" --dport "$CLAMP_HOST_PROXY_PORT" -j ACCEPT
printf '[Clamp Firewall]\n  Loopback: allowed\n  Outbound TCP: %s:%s only\n' "$CLAMP_HOST_PROXY_HOST" "$CLAMP_HOST_PROXY_PORT"
