#!/usr/bin/env node
import http from 'node:http';
import net from 'node:net';
import fs from 'node:fs';
import { loadAllowList } from './allowed-endpoints.mjs';
import { localNow } from './local-time.mjs';

const listenHost = process.env.CLAMP_PROXY_LISTEN_HOST || '127.0.0.1';
const listenPort = Number(process.env.CLAMP_PROXY_LISTEN_PORT || '8888');
const logFile = process.env.CLAMP_NETWORK_LOG_FILE || '/workspace/.clamp/sessions/network.log';
const domainFiles = process.argv.slice(2);
const allowAll = process.env.CLAMP_PROXY_ALLOW_ALL === '1';

function log(line) {
  const entry = `${localNow()} ${line}\n`;
  if (process.env.CLAMP_LOG_STDOUT === '1') process.stdout.write(entry);
  else fs.appendFileSync(logFile, entry);
}

const allowList = loadAllowList(domainFiles, { allowAll, allowLoopback: true });

function deny(resOrSocket, status, msg) {
  if (typeof resOrSocket.writeHead === 'function') {
    resOrSocket.writeHead(status, { 'content-type': 'text/plain' });
    resOrSocket.end(`${msg}\n`);
  } else {
    resOrSocket.write(`HTTP/1.1 ${status} ${msg}\r\ncontent-length: 0\r\n\r\n`);
    resOrSocket.destroy();
  }
}

const server = http.createServer((clientReq, clientRes) => {
  let target;
  try {
    target = new URL(clientReq.url);
  } catch {
    const host = clientReq.headers.host;
    if (!host) return deny(clientRes, 400, 'Bad Request');
    target = new URL(`http://${host}${clientReq.url}`);
  }

  const host = target.hostname;
  const port = Number(target.port || (target.protocol === 'https:' ? 443 : 80));
  if (allowList.isEndpointBlocked(host, port)) {
    log(`CLAMP-PROXY-HTTP-BLOCKED method=${clientReq.method} host=${host} port=${port} url=${clientReq.url}`);
    return deny(clientRes, 403, 'Forbidden');
  }

  log(`CLAMP-PROXY-HTTP-ALLOWED method=${clientReq.method} host=${host} port=${port}`);
  const headers = { ...clientReq.headers };
  delete headers['proxy-connection'];
  const upstreamReq = http.request({
    hostname: host,
    port,
    method: clientReq.method,
    path: `${target.pathname}${target.search}`,
    headers,
  }, (upstreamRes) => {
    clientRes.writeHead(upstreamRes.statusCode || 502, upstreamRes.headers);
    upstreamRes.pipe(clientRes);
  });
  upstreamReq.on('error', (err) => {
    log(`CLAMP-PROXY-UPSTREAM-ERROR host=${host} port=${port} error=${err.code || err.message}`);
    if (!clientRes.headersSent) deny(clientRes, 502, 'Bad Gateway');
    else clientRes.destroy();
  });
  clientReq.pipe(upstreamReq);
});

server.on('connect', (req, clientSocket, head) => {
  const [rawHost, rawPort] = String(req.url || '').split(':');
  const host = rawHost;
  const port = Number(rawPort || 443);
  if (allowList.isEndpointBlocked(host, port)) {
    log(`CLAMP-PROXY-CONNECT-BLOCKED host=${host} port=${port}`);
    return deny(clientSocket, 403, 'Forbidden');
  }

  const upstream = net.connect(port, host, () => {
    log(`CLAMP-PROXY-CONNECT-ALLOWED host=${host} port=${port}`);
    clientSocket.write('HTTP/1.1 200 Connection Established\r\n\r\n');
    if (head?.length) upstream.write(head);
    upstream.pipe(clientSocket);
    clientSocket.pipe(upstream);
  });
  upstream.on('error', (err) => {
    log(`CLAMP-PROXY-CONNECT-UPSTREAM-ERROR host=${host} port=${port} error=${err.code || err.message}`);
    deny(clientSocket, 502, 'Bad Gateway');
  });
});

server.listen(listenPort, listenHost, () => {
  log(`CLAMP-PROXY-LISTENING host=${listenHost} port=${listenPort} allowed_domains=${allowList.size} allow_all=${allowAll}`);
});
