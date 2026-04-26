#!/usr/bin/env node
'use strict';

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

const isWin = process.platform === 'win32';
const binary = isWin ? 'windowctl.exe' : 'windowctl';
const binPath = path.join(__dirname, '..', 'native', binary);

if (!fs.existsSync(binPath)) {
  console.error(`windowctl: native binary not found at ${binPath}.`);
  console.error('The postinstall download likely failed. Try reinstalling:');
  console.error('  npm install -g windowctl');
  console.error('Or download a binary directly from');
  console.error('  https://github.com/muthuishere/windowctl/releases');
  process.exit(1);
}

const child = spawn(binPath, process.argv.slice(2), { stdio: 'inherit' });
child.on('exit', (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
    return;
  }
  process.exit(code ?? 1);
});
child.on('error', (err) => {
  console.error('windowctl:', err.message);
  process.exit(1);
});
