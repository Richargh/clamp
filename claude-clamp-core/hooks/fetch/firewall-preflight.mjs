#!/usr/bin/env node

/**
 * Fetch hook for Claude Code that validates URLs against allowed domains.
 *
 * Allowed domains are read from:
 * 1. ALLOWED_DOMAINS environment variable (comma-separated)
 * 2. /home/dev/.claude/hooks/allowed-domains.txt (one domain per line)
 *
 * The hook receives tool input via stdin and blocks requests to non-allowed domains.
 *
 * @typedef {import('../types.js').WebFetchHookData} WebFetchHookData
 * @typedef {import('../types.js').BlockResponse} BlockResponse
 */

import {readFileSync} from 'fs';

const ALLOWED_DOMAINS_FILE = '/home/dev/.claude/hooks/allowed-domains.txt';

async function main() {
    const input = await readStdin();
    const hookData = parseHookInput(input);
    const hostname = extractHostname(hookData.tool_input?.url);

    const allowedDomains = getAllowedDomains();

    if (!isDomainAllowed(hostname, allowedDomains)) {
        blockRequest(`Domain "${hostname}" is not in the allowed domains list`);
    }
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
 * @returns {Set<string>}
 */
function getAllowedDomains() {
    return new Set([
        ...loadDomainsFromFile(ALLOWED_DOMAINS_FILE),
    ]);
}

/**
 * @param {string} input
 * @returns {WebFetchHookData}
 * @throws Will throw an error if input cannot be parsed.
 */
function parseHookInput(input) {
    return JSON.parse(input);
}

/**
 * @param {string | null} url
 * @returns {string}
 * @throws Will throw an error if URL cannot be parsed.
 */
function extractHostname(url) {
    return new URL(url).hostname.toLowerCase();
}

/**
 * Checks if a hostname matches any allowed domain (exact or subdomain match).
 * @param {string} hostname
 * @param {Set<string>} allowedDomains
 * @returns {boolean}
 */
function isDomainAllowed(hostname, allowedDomains) {
    if (allowedDomains.size === 0) {
        printBlock('No domains allowed.')
        return false;
    }

    return [...allowedDomains].some(
        allowed => hostname === allowed || hostname.endsWith(`.${allowed}`)
    );
}

/**
 * Outputs a block response and exits.
 * @param {string} reason
 * @returns {never}
 */
function blockRequest(reason) {
    printBlock(reason);
    process.exit(0);
}

/**
 * @param {string} reason
 */
function printBlock(reason) {
    /** @type {BlockResponse} */
    const response = {decision: 'block', reason};
    console.log(JSON.stringify(response));
}

/**
 * @param {string} filePath
 * @returns {string[]}
 * @throws Will throw an error if file cannot be found
 */
function loadDomainsFromFile(filePath) {
    const content = readFileSync(filePath, 'utf-8');
    return content
        .split('\n')
        .map(line => line.trim().toLowerCase())
        .filter(line => line && !line.startsWith('#'));
}

main();
