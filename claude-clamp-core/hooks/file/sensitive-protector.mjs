#!/usr/bin/env node

/**
 * PreToolUse hook for Read|Edit|Write that protects sensitive files.
 *
 * This hook blocks access to files that may contain secrets, credentials,
 * or other sensitive information that should not be accessed by the agent.
 *
 * @typedef {import('../types.js').FileHookData} FileHookData
 * @typedef {import('../types.js').BlockResponse} BlockResponse
 */

const PROTECTED_PATTERNS = [
  // Environment and secrets files
  { pattern: /\.env($|\.)/, reason: 'Environment files may contain secrets' },
  { pattern: /\.env\.local$/, reason: 'Local environment files may contain secrets' },
  { pattern: /\.env\.production$/, reason: 'Production environment files contain secrets' },
  { pattern: /secrets?\.(json|ya?ml|txt|conf)$/i, reason: 'Secrets files are protected' },
  { pattern: /credentials?\.(json|ya?ml|txt|conf)$/i, reason: 'Credentials files are protected' },

  // SSH and crypto keys
  { pattern: /\.ssh\//, reason: 'SSH directory is protected' },
  { pattern: /id_rsa/, reason: 'SSH private keys are protected' },
  { pattern: /id_ed25519/, reason: 'SSH private keys are protected' },
  { pattern: /id_ecdsa/, reason: 'SSH private keys are protected' },
  { pattern: /id_dsa/, reason: 'SSH private keys are protected' },
  { pattern: /\.pem$/, reason: 'PEM certificate files are protected' },
  { pattern: /\.key$/, reason: 'Key files are protected' },
  { pattern: /\.p12$/, reason: 'PKCS12 files are protected' },
  { pattern: /\.pfx$/, reason: 'PFX certificate files are protected' },

  // Git internals (protect credentials)
  { pattern: /\.git\/config$/, reason: 'Git config may contain credentials' },
  { pattern: /\.git-credentials$/, reason: 'Git credentials file is protected' },
  { pattern: /\.gitconfig$/, reason: 'Global git config may contain credentials' },
  { pattern: /\.netrc$/, reason: 'Netrc file contains credentials' },

  // Cloud provider credentials
  { pattern: /\.aws\/credentials/, reason: 'AWS credentials are protected' },
  { pattern: /\.aws\/config/, reason: 'AWS config may contain sensitive info' },
  { pattern: /gcloud.*credentials/, reason: 'GCloud credentials are protected' },
  { pattern: /\.azure\//, reason: 'Azure credentials are protected' },
  { pattern: /\.kube\/config/, reason: 'Kubernetes config is protected' },

  // Database and service configs
  { pattern: /database\.(ya?ml|json)$/i, reason: 'Database configs may contain credentials' },
  { pattern: /\.pgpass$/, reason: 'PostgreSQL password file is protected' },
  { pattern: /\.my\.cnf$/, reason: 'MySQL config file is protected' },
  { pattern: /\.mongorc\.js$/, reason: 'MongoDB config file is protected' },

  // Token and auth files
  { pattern: /token(s)?\.(json|txt|ya?ml)$/i, reason: 'Token files are protected' },
  { pattern: /auth\.(json|ya?ml)$/i, reason: 'Auth files are protected' },
  { pattern: /\.npmrc$/, reason: 'NPM config may contain auth tokens' },
  { pattern: /\.pypirc$/, reason: 'PyPI config may contain auth tokens' },

  // Password files
  { pattern: /password(s)?\.(txt|json|ya?ml)$/i, reason: 'Password files are protected' },
  { pattern: /\/etc\/shadow$/, reason: 'System shadow file is protected' },
  { pattern: /\/etc\/passwd$/, reason: 'System passwd file is protected' },

  // History files (may contain secrets typed accidentally)
  { pattern: /\.bash_history$/, reason: 'Bash history may contain secrets' },
  { pattern: /\.zsh_history$/, reason: 'Zsh history may contain secrets' },
  { pattern: /\.node_repl_history$/, reason: 'Node REPL history may contain secrets' },
  { pattern: /\.python_history$/, reason: 'Python history may contain secrets' },
];

// Paths that are always allowed (overrides protected patterns)
const ALLOWED_PATHS = [
  /\/home\/dev\/workspace\//,  // Workspace is always accessible
];

async function main() {
  const input = await readStdin();
  /** @type {FileHookData} */
  const hookData = JSON.parse(input);
  const filePath = hookData.tool_input?.file_path || '';

  // Check if path is explicitly allowed
  for (const allowed of ALLOWED_PATHS) {
    if (allowed.test(filePath)) {
      process.exit(0);
    }
  }

  // Check against protected patterns
  for (const { pattern, reason } of PROTECTED_PATTERNS) {
    if (pattern.test(filePath)) {
      blockRequest(reason);
    }
  }

  // Allow access
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
