#!/usr/bin/env node
import dgram from 'node:dgram';
import fs from 'node:fs';
import { spawnSync } from 'node:child_process';
import { localNow } from './local-time.mjs';

const listenHost = process.env.CLAMP_DNS_LISTEN_HOST || '127.0.0.1';
const listenPort = Number(process.env.CLAMP_DNS_LISTEN_PORT || '53');
const upstreams = (process.env.CLAMP_UPSTREAM_DNS || '1.1.1.1,8.8.8.8')
  .split(',').map(s => s.trim()).filter(Boolean);
const logFile = process.env.CLAMP_BLOCKED_DOMAIN_LOG_FILE || process.env.CLAMP_BLOCKED_LOG_FILE || '/workspace/.clamp/sessions/network.log';
const domainFiles = process.argv.slice(2);

function log(line) {
  fs.appendFileSync(logFile, `${localNow()} ${line}\n`);
}

function loadAllowed(files) {
  const allowed = new Set();
  for (const file of files) {
    if (!fs.existsSync(file) || !fs.statSync(file).isFile()) continue;
    for (const raw of fs.readFileSync(file, 'utf8').split(/\r?\n/)) {
      const line = raw.replace(/#.*/, '').replace(/\s+/g, '').toLowerCase();
      if (line) allowed.add(line.endsWith('.') ? line.slice(0, -1) : line);
    }
  }
  return allowed;
}

const allowed = loadAllowed(domainFiles);

function parseQuestion(msg) {
  let off = 12;
  const labels = [];
  while (off < msg.length) {
    const len = msg[off++];
    if (len === 0) break;
    if ((len & 0xc0) !== 0 || off + len > msg.length) return null;
    labels.push(msg.subarray(off, off + len).toString('ascii'));
    off += len;
  }
  if (off + 4 > msg.length) return null;
  return { name: labels.join('.').toLowerCase(), type: msg.readUInt16BE(off), qEnd: off + 4 };
}

function skipName(msg, off) {
  while (off < msg.length) {
    const len = msg[off++];
    if ((len & 0xc0) === 0xc0) return off + 1;
    if (len === 0) return off;
    off += len;
  }
  return off;
}

function addIp(ip, ttl) {
  const timeout = Math.max(60, Math.min(Number(ttl) || 300, 86400));
  spawnSync('ipset', ['add', 'allowed_ips', ip, 'timeout', String(timeout), '-exist'], { stdio: 'ignore' });
}

function parseAndAllowAnswers(msg) {
  if (msg.length < 12) return;
  const qd = msg.readUInt16BE(4);
  const an = msg.readUInt16BE(6);
  let off = 12;
  for (let i = 0; i < qd; i++) {
    off = skipName(msg, off) + 4;
  }
  for (let i = 0; i < an && off + 12 <= msg.length; i++) {
    off = skipName(msg, off);
    if (off + 10 > msg.length) return;
    const type = msg.readUInt16BE(off); off += 2;
    const klass = msg.readUInt16BE(off); off += 2;
    const ttl = msg.readUInt32BE(off); off += 4;
    const rdlen = msg.readUInt16BE(off); off += 2;
    if (off + rdlen > msg.length) return;
    if (type === 1 && klass === 1 && rdlen === 4) {
      addIp([...msg.subarray(off, off + 4)].join('.'), ttl);
    }
    off += rdlen;
  }
}

function nxdomainResponse(query) {
  const response = Buffer.from(query);
  response[2] = 0x81; // response + recursion desired
  response[3] = 0x83; // recursion available + NXDOMAIN
  response.writeUInt16BE(0, 6); // ANCOUNT
  response.writeUInt16BE(0, 8); // NSCOUNT
  response.writeUInt16BE(0, 10); // ARCOUNT
  return response;
}

let upstreamIndex = 0;
function forward(query, cb) {
  const upstream = upstreams[upstreamIndex++ % upstreams.length];
  const sock = dgram.createSocket('udp4');
  const timer = setTimeout(() => { sock.close(); cb(null); }, 5000);
  sock.on('message', (response) => { clearTimeout(timer); sock.close(); cb(response); });
  sock.on('error', () => { clearTimeout(timer); sock.close(); cb(null); });
  sock.send(query, 53, upstream);
}

const server = dgram.createSocket('udp4');
server.on('message', (msg, rinfo) => {
  const q = parseQuestion(msg);
  if (!q || !q.name || !allowed.has(q.name)) {
    const name = q?.name || '(malformed)';
    log(`BLOCKED-DNS domain=${name} client=${rinfo.address}:${rinfo.port}`);
    server.send(nxdomainResponse(msg), rinfo.port, rinfo.address);
    return;
  }

  forward(msg, (response) => {
    if (!response) {
      log(`DNS-UPSTREAM-FAIL domain=${q.name}`);
      server.send(nxdomainResponse(msg), rinfo.port, rinfo.address);
      return;
    }
    parseAndAllowAnswers(response);
    server.send(response, rinfo.port, rinfo.address);
  });
});

server.bind(listenPort, listenHost, () => {
  log(`DNS firewall listening on ${listenHost}:${listenPort}; allowed_domains=${allowed.size}; upstreams=${upstreams.join(',')}`);
});
