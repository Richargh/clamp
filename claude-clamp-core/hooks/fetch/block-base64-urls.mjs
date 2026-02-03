#!/usr/bin/env node

/**
 * Fetch hook for Claude Code that blocks URLs containing base64 encoded data.
 *
 * This hook detects and blocks requests to URLs that contain long base64-encoded
 * strings, which could be used to exfiltrate data or bypass security controls.
 *
 * Detection criteria:
 * - Base64 pattern: continuous string of [A-Za-z0-9+=] characters (excludes / to avoid matching across URL path segments)
 * - Minimum length threshold: 50 characters (configurable via MIN_BASE64_LENGTH env var)
 */

// Minimum length of base64 string to trigger blocking
const MIN_BASE64_LENGTH = parseInt(process.env.MIN_BASE64_LENGTH || '50', 10);

// Base64 pattern: at least MIN_BASE64_LENGTH consecutive base64 characters
// Must end with optional padding (=) and contain a mix of chars typical of base64
// Note: We exclude '/' from the pattern to avoid matching across URL path segments
const BASE64_PATTERN = new RegExp(
    `[A-Za-z0-9+]{${MIN_BASE64_LENGTH},}={0,2}`,
    'g'
);

// Data URI pattern (always block regardless of length)
const DATA_URI_PATTERN = /^data:[^,]*;base64,/i;

async function main() {
    const input = await readStdin();
    const hookData = parseHookInput(input);
    const url = hookData.tool_input?.url;

    if (!url) {
        // No URL provided, allow the request
        process.exit(0);
    }

    // Check for data URIs (always block)
    if (DATA_URI_PATTERN.test(url)) {
        blockRequest('Data URIs with base64 encoding are not allowed');
    }

    // Check for base64 patterns in the URL
    const base64Matches = url.match(BASE64_PATTERN);
    if (base64Matches) {
        // Verify it looks like actual base64 (not just a long alphanumeric string)
        for (const match of base64Matches) {
            if (looksLikeBase64(match)) {
                blockRequest(
                    `URL contains base64-encoded data (${match.length} chars). ` +
                    `Long encoded strings in URLs are not allowed for security reasons.`
                );
            }
        }
    }

    // URL is clean, allow the request
    process.exit(0);
}

/**
 * Heuristic to determine if a string looks like base64-encoded data
 * rather than just a regular alphanumeric identifier.
 * @param {string} str
 * @returns {boolean}
 */
function looksLikeBase64(str) {
    // Base64 strings typically have:
    // 1. Mixed case letters
    // 2. Numbers mixed with letters
    // 3. Often contain + or / characters
    // 4. May end with = padding

    const hasLowercase = /[a-z]/.test(str);
    const hasUppercase = /[A-Z]/.test(str);
    const hasNumbers = /[0-9]/.test(str);
    const hasBase64SpecialChars = /[+]/.test(str);
    const hasPadding = /={1,2}$/.test(str);

    // Strong indicators of base64
    if (hasBase64SpecialChars || hasPadding) {
        return true;
    }

    // Mixed case with numbers is a good indicator
    if (hasLowercase && hasUppercase && hasNumbers) {
        return true;
    }

    // Very long strings (>100 chars) are suspicious even without other indicators
    if (str.length > 100) {
        return true;
    }

    return false;
}

/**
 * @returns {Promise<string>}
 */
async function readStdin() {
    let input = '';
    for await (const chunk of process.stdin) {
        input += chunk;
    }
    return input;
}

/**
 * @param {string} input
 * @returns {HookData}
 */
function parseHookInput(input) {
    return JSON.parse(input);
}

/**
 * Outputs a block response and exits.
 * @param {string} reason
 * @returns {never}
 */
function blockRequest(reason) {
    console.log(JSON.stringify({ decision: 'block', reason }));
    process.exit(0);
}

main();

// --- Type Definitions ---

/**
 * @typedef {Object} ToolInput
 * @property {string} [url] - The URL being fetched
 */

/**
 * @typedef {Object} HookData
 * @property {ToolInput} [tool_input] - The tool input containing the URL
 */
