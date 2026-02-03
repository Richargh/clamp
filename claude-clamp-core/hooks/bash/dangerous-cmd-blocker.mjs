#!/usr/bin/env node

/**
 * PreToolUse hook for Bash that blocks dangerous or destructive commands.
 *
 * This hook validates bash commands against a list of dangerous patterns
 * and blocks execution of potentially harmful operations.
 *
 * @typedef {import('../types.js').BashHookData} BashHookData
 * @typedef {import('../types.js').BlockResponse} BlockResponse
 */

const BLOCKED_PATTERNS = [
  // Destructive file operations
  { pattern: /rm\s+(-[rf]+\s+)*\/(?!tmp|workspace|home\/dev\/workspace)/, reason: 'Removing files outside allowed directories is blocked' },
  { pattern: /rm\s+-[rf]*\s+\*/, reason: 'Recursive wildcard deletion is blocked' },

  // Privilege escalation
  { pattern: /\bsudo\b/, reason: 'sudo commands are blocked' },
  { pattern: /\bsu\s+-?\s*$/, reason: 'su commands are blocked' },
  { pattern: /\bdoas\b/, reason: 'doas commands are blocked' },

  // Dangerous permissions
  { pattern: /chmod\s+777/, reason: 'chmod 777 (world-writable) is blocked' },
  { pattern: /chmod\s+-R\s+777/, reason: 'Recursive chmod 777 is blocked' },
  { pattern: /chown\s+-R\s+root/, reason: 'Recursive chown to root is blocked' },

  // Remote code execution patterns
  { pattern: /curl\s+.*\|\s*(?:ba)?sh/, reason: 'curl-pipe-to-shell is blocked' },
  { pattern: /wget\s+.*\|\s*(?:ba)?sh/, reason: 'wget-pipe-to-shell is blocked' },
  { pattern: /curl\s+.*-o\s*-\s*\|\s*(?:ba)?sh/, reason: 'curl output piped to shell is blocked' },

  // Git dangerous operations
  { pattern: /git\s+push\s+.*--force(?:-with-lease)?(?:\s+origin\s+(?:main|master))?/, reason: 'Force push to main/master is blocked' },
  { pattern: /git\s+reset\s+--hard\s+origin/, reason: 'Hard reset to origin is blocked' },

  // System damage
  { pattern: /:\s*\(\s*\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:/, reason: 'Fork bomb detected and blocked' },
  { pattern: /mkfs\./, reason: 'Filesystem formatting is blocked' },
  { pattern: /dd\s+.*of=\/dev\/(?!null)/, reason: 'Writing to block devices is blocked' },
  { pattern: />\s*\/dev\/sd[a-z]/, reason: 'Writing to block devices is blocked' },

  // Network exfiltration attempts
  { pattern: /nc\s+-[el]/, reason: 'Netcat listener mode is blocked' },
  { pattern: /ncat\s+-[el]/, reason: 'Ncat listener mode is blocked' },

  // Environment manipulation
  { pattern: /export\s+LD_PRELOAD/, reason: 'LD_PRELOAD manipulation is blocked' },
  { pattern: /export\s+LD_LIBRARY_PATH=\//, reason: 'LD_LIBRARY_PATH root manipulation is blocked' },

  // History tampering
  { pattern: /history\s+-[cd]/, reason: 'History clearing/deletion is blocked' },
  { pattern: />\s*~?\/?\.bash_history/, reason: 'Bash history tampering is blocked' },
  { pattern: /unset\s+HISTFILE/, reason: 'History file unsetting is blocked' },
];

async function main() {
  const input = await readStdin();
  /** @type {BashHookData} */
  const hookData = JSON.parse(input);
  const command = hookData.tool_input?.command || '';

  for (const { pattern, reason } of BLOCKED_PATTERNS) {
    if (pattern.test(command)) {
      blockRequest(reason);
    }
  }

  // Allow the command
  process.exit(0);
}

async function readStdin() {
  let input = '';
  for await (const chunk of process.stdin) {
    input += chunk;
  }
  return input;
}

/**
 * @param {string} reason
 */
function blockRequest(reason) {
  /** @type {BlockResponse} */
  const response = {
    decision: 'block',
    reason: `[Security] ${reason}`
  };
  console.log(JSON.stringify(response));
  process.exit(0);
}

main();
