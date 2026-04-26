#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');
const https = require('https');
const tar = require('tar');

const pkg = require('../package.json');
const VERSION = pkg.version;
const REPO = 'muthuishere/windowctl';

const PLATFORM_MAP = { win32: 'windows', darwin: 'darwin', linux: 'linux' };
const ARCH_MAP = { x64: 'amd64', arm64: 'arm64' };

function assetName() {
  const os = PLATFORM_MAP[process.platform];
  const arch = ARCH_MAP[process.arch];
  if (!os || !arch) {
    throw new Error(`unsupported platform/arch: ${process.platform}/${process.arch}`);
  }
  return `windowctl_${VERSION}_${os}_${arch}.tar.gz`;
}

function get(url, redirects = 5) {
  return new Promise((resolve, reject) => {
    const req = https.get(
      url,
      { headers: { 'user-agent': 'windowctl-npm-install', accept: 'application/octet-stream' } },
      (res) => {
        if ([301, 302, 307, 308].includes(res.statusCode)) {
          if (redirects <= 0) {
            reject(new Error(`too many redirects from ${url}`));
            return;
          }
          res.resume();
          get(res.headers.location, redirects - 1).then(resolve, reject);
          return;
        }
        if (res.statusCode !== 200) {
          reject(new Error(`HTTP ${res.statusCode} for ${url}`));
          return;
        }
        resolve(res);
      }
    );
    req.on('error', reject);
  });
}

async function main() {
  const name = assetName();
  const url = `https://github.com/${REPO}/releases/download/v${VERSION}/${name}`;
  const dest = path.join(__dirname, '..', 'native');
  fs.mkdirSync(dest, { recursive: true });

  console.log(`windowctl: downloading ${url}`);
  const res = await get(url);
  await new Promise((resolve, reject) => {
    res.pipe(tar.x({ cwd: dest })).on('finish', resolve).on('error', reject);
  });
  console.log(`windowctl: installed ${name} -> ${dest}`);
}

main().catch((err) => {
  if (err.message && err.message.includes('HTTP 404')) {
    console.error(
      `windowctl: the binary for v${VERSION} on ${process.platform}/${process.arch} ` +
        'is not yet published.'
    );
    console.error(
      'See https://github.com/muthuishere/windowctl/releases for available versions.'
    );
  } else {
    console.error('windowctl install failed:', err.message);
  }
  process.exit(1);
});
