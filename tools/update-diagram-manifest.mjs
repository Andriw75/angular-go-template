#!/usr/bin/env node
/**
 * Regenera docs/diagramas/manifest.json con el hash del JSON y del HTML de cada
 * diagrama. Sirve para que check-docs detecte diagramas desactualizados sin
 * guardar receipts con rutas absolutas.
 *
 * Uso: node tools/update-diagram-manifest.mjs
 */
import { createHash } from 'node:crypto';
import { existsSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const FUENTES = join(ROOT, 'docs', 'diagramas', 'fuentes');
const PUBLIC = join(ROOT, 'front', 'public', 'flujos');
const OUT = join(ROOT, 'docs', 'diagramas', 'manifest.json');

const normalize = (text) => text.replace(/\r\n/g, '\n');
const sha256 = (text) => createHash('sha256').update(normalize(text)).digest('hex');

const diagrams = {};
let revision = null;
const missing = [];

for (const file of readdirSync(FUENTES).filter((name) => name.endsWith('.json')).sort()) {
  const slug = file.replace(/\.json$/, '');
  const specPath = join(FUENTES, file);
  const htmlPath = join(PUBLIC, `${slug}.html`);
  if (!existsSync(htmlPath)) {
    missing.push(`${slug}.html`);
    continue;
  }
  const spec = JSON.parse(readFileSync(specPath, 'utf8'));
  revision ??= spec.meta?.repository?.revision ?? null;
  diagrams[slug] = {
    spec_sha256: sha256(readFileSync(specPath, 'utf8')),
    artifact_sha256: sha256(readFileSync(htmlPath, 'utf8')),
  };
}

if (missing.length > 0) {
  console.error(`Faltan HTML generados: ${missing.join(', ')}`);
  process.exit(1);
}

const manifest = {
  generated_with: 'archify',
  revision,
  diagrams,
};

writeFileSync(OUT, `${JSON.stringify(manifest, null, 2)}\n`, 'utf8');
console.log(`Manifest actualizado: ${Object.keys(diagrams).length} diagramas`);
