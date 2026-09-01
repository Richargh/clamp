#!/bin/bash
# Print a short network summary for this Clamp session.

set -euo pipefail

workspace="${1:?Usage: $0 <workspace> <session-log-file>}"
session_log_file="${2:?Usage: $0 <workspace> <session-log-file>}"
log_file="$workspace/.clamp/sessions/$session_log_file"

if [ ! -f "$log_file" ]; then
    echo ""
    echo "[Clamp Shutdown] Session network log not found: $log_file"
    exit 0
fi

problem_pattern='CLAMP-.*(BLOCKED|UPSTREAM-FAIL|UPSTREAM-ERROR)'
problem_count="$(grep -E -c "$problem_pattern" "$log_file" 2>/dev/null || true)"
if [ -z "$problem_count" ]; then
    problem_count=0
fi

echo ""
echo "[Clamp Shutdown] Network summary for session: $log_file"

if [ "$problem_count" -eq 0 ]; then
    echo "  No network problems detected."
    exit 0
fi

echo "  Network problems detected: $problem_count"
echo "  Recent entries:"
grep -E "$problem_pattern" "$log_file" | tail -20 | sed 's/^/    /'
