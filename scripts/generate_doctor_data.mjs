#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';

const repoRoot = path.resolve(path.dirname(new URL(import.meta.url).pathname), '..');
const doctorDataDir = path.join(repoRoot, 'pkg', 'doctor', 'data');
const bridgeDir = path.join(repoRoot, 'pkg', 'bridge');
const devtoolsProtocolPkg = path.join(repoRoot, 'node_modules', 'devtools-protocol', 'package.json');
const browserProtocolPath = path.join(repoRoot, 'node_modules', 'devtools-protocol', 'json', 'browser_protocol.json');
const jsProtocolPath = path.join(repoRoot, 'node_modules', 'devtools-protocol', 'json', 'js_protocol.json');

function loadJSON(file) {
  return JSON.parse(fs.readFileSync(file, 'utf8'));
}

function ensureDir(dir) {
  fs.mkdirSync(dir, { recursive: true });
}

function collectUpstreamMethods() {
  const pkg = loadJSON(devtoolsProtocolPkg);
  const browser = loadJSON(browserProtocolPath);
  const js = loadJSON(jsProtocolPath);
  const domains = new Map();

  for (const protocol of [browser, js]) {
    for (const domain of protocol.domains) {
      if (!domains.has(domain.domain)) {
        domains.set(domain.domain, new Set());
      }
      for (const command of domain.commands || []) {
        domains.get(domain.domain).add(command.name);
      }
    }
  }

  const normalized = {};
  for (const domain of [...domains.keys()].sort()) {
    normalized[domain] = [...domains.get(domain)].sort();
  }

  return {
    source: `devtools-protocol ${pkg.version}`,
    domains: normalized,
  };
}

function collectCurrentBridgeMethods() {
  const files = fs
    .readdirSync(bridgeDir)
    .filter((name) => name.endsWith('.go') && !name.endsWith('_test.go'))
    .sort();

  const methods = {};
  const caseMethod = /case\s+((?:"[A-Za-z][A-Za-z0-9]*\.[A-Za-z][A-Za-z0-9]*"\s*,\s*)*"[\w.]+")\s*:/g;
  const ifMethod = /\bif\s+method\s*==\s*"([A-Za-z][A-Za-z0-9]*\.[A-Za-z][A-Za-z0-9]*)"/g;

  for (const file of files) {
    const fullPath = path.join(bridgeDir, file);
    const text = fs.readFileSync(fullPath, 'utf8');
    for (const match of text.matchAll(caseMethod)) {
      const parts = match[1].split(',').map((value) => value.trim().replace(/^"|"$/g, ''));
      for (const method of parts) {
        if (!method.includes('.')) {
          continue;
        }
        methods[method] = file;
      }
    }
    for (const match of text.matchAll(ifMethod)) {
      methods[match[1]] = file;
    }
  }

  return {
    methods: Object.fromEntries(Object.entries(methods).sort(([a], [b]) => a.localeCompare(b))),
  };
}

ensureDir(doctorDataDir);

fs.writeFileSync(
  path.join(doctorDataDir, 'upstream_methods.json'),
  JSON.stringify(collectUpstreamMethods(), null, 2) + '\n',
);

fs.writeFileSync(
  path.join(doctorDataDir, 'current_bridge_methods.json'),
  JSON.stringify(collectCurrentBridgeMethods(), null, 2) + '\n',
);
