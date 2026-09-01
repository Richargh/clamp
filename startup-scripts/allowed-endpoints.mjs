import fs from 'node:fs';

const ipLiteral = /^(?:\d{1,3}\.){3}\d{1,3}$|^\[[0-9a-f:]+\]$|^[0-9a-f:]*:[0-9a-f:]+$/i;

export function normalizeHost(host) {
  return String(host || '').trim().toLowerCase().replace(/^\[/, '').replace(/\]$/, '').replace(/\.$/, '');
}

export function loadAllowList(files, options = {}) {
  return AllowedEndpoints.load(files, options);
}

class AllowedEndpoints {
  static load(sources, options = {}) {
    const allowed = new AllowedEndpoints(options);
    for (const file of allowListFiles(sources)) {
      for (const endpoint of allowedEndpointsFromFile(file)) {
        allowed.#add(endpoint);
      }
    }
    return allowed;
  }

  #allowAll;
  #allowLoopback;
  #portsByHost = new Map();

  constructor({ allowAll = false, allowLoopback = false } = {}) {
    this.#allowAll = allowAll;
    this.#allowLoopback = allowLoopback;
  }

  get size() {
    return this.#portsByHost.size;
  }

  #add({ host, port }) {
    if (!this.#portsByHost.has(host)) this.#portsByHost.set(host, new Set());
    this.#portsByHost.get(host).add(port);
  }

  isEndpointAllowed(host, port) {
    const normalized = normalizeHost(host);
    if (!normalized || !validPort(port)) return false;
    if (this.#allowAll || (this.#allowLoopback && loopbackHost(normalized))) return true;
    if (isIpAddress(normalized)) return false;

    return this.#portsByHost.get(normalized)?.has(port) || false;
  }

  isEndpointBlocked(host, port) {
    return !this.isEndpointAllowed(host, port);
  }

  isHostAllowed(host) {
    const normalized = normalizeHost(host);
    if (!normalized) return false;
    if (this.#allowAll) return true;
    if (isIpAddress(normalized)) return false;

    return this.#portsByHost.has(normalized);
  }

  isHostBlocked(host) {
    return !this.isHostAllowed(host);
  }

  allowedPortsForHost(host) {
    const normalized = normalizeHost(host);
    if (!normalized || isIpAddress(normalized)) return [];
    return [...(this.#portsByHost.get(normalized) || [])];
  }
}

function allowListFiles(sources) {
  return sources.flatMap(allowListFilesFromSource);
}

function allowListFilesFromSource(source) {
  if (!fs.existsSync(source)) return [];
  const stat = fs.statSync(source);
  if (stat.isFile()) return [source];
  if (!stat.isDirectory()) return [];

  return fs.readdirSync(source)
    .filter(name => name.endsWith('.txt'))
    .sort()
    .map(name => `${source}/${name}`)
    .filter(file => fs.statSync(file).isFile());
}

function allowedEndpointsFromFile(file) {
  return fs.readFileSync(file, 'utf8')
    .split(/\r?\n/)
    .map(stripCommentAndWhitespace)
    .filter(Boolean)
    .map(line => parseAllowedEndpoint(line, file));
}

function stripCommentAndWhitespace(line) {
  return line.replace(/#.*/, '').replace(/\s+/g, '').toLowerCase();
}

function parseAllowedEndpoint(line, file) {
  const match = line.match(/^([^:]+):(\d+)$/);
  if (!match) throwInvalidEndpoint(line, file);

  const host = normalizeHost(match[1]);
  const port = Number(match[2]);
  if (!host || !validPort(port)) throwInvalidEndpoint(line, file);

  return { host, port };
}

function validPort(port) {
  return Number.isInteger(port) && port >= 1 && port <= 65535;
}

function throwInvalidEndpoint(line, file) {
  throw new Error(`Invalid allowed endpoint in ${file}: '${line}' (expected domain:port)`);
}

function isIpAddress(host) {
  return ipLiteral.test(host);
}

function loopbackHost(host) {
  const normalized = normalizeHost(host);
  return normalized === 'localhost' || normalized === '::1' || normalized.startsWith('127.');
}
