#!/bin/bash
# Print a short network summary for the most recent Clamp session.

set -euo pipefail

workspace="${1:-.}"
session_dir="$workspace/.clamp/sessions"

if [ ! -d "$session_dir" ]; then
    echo ""
    echo "[Clamp Shutdown] No session log directory found: $session_dir"
    exit 0
fi

shopt -s nullglob
latest_log=""
for log_file in "$session_dir"/*-network.log; do
    if [ -z "$latest_log" ] || [ "$log_file" -nt "$latest_log" ]; then
        latest_log="$log_file"
    fi
done

if [ -z "$latest_log" ]; then
    echo ""
    echo "[Clamp Shutdown] No network session logs found in: $session_dir"
    exit 0
fi

problem_pattern='BLOCKED|UPSTREAM-FAIL|UPSTREAM-ERROR|CLAMP-BLOCKED'
problem_count="$(grep -E -c "$problem_pattern" "$latest_log" 2>/dev/null || true)"
if [ -z "$problem_count" ]; then
    problem_count=0
fi

echo ""
echo "[Clamp Shutdown] Network summary for most recent session: $latest_log"

if [ "$problem_count" -eq 0 ]; then
    echo "  No network problems detected."
    exit 0
fi

echo "  Network problems detected: $problem_count"
echo "  Recent entries:"
grep -E "$problem_pattern" "$latest_log" | tail -20 | sed 's/^/    /'
