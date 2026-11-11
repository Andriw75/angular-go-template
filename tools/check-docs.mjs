#!/usr/bin/env node
/**
 * Verificación de documentación del template.
 *
 * Comprueba:
 *  - rutas del router Go presentes en docs/flujos/*.md;
 *  - archivos citados en Markdown;
 *  - JSON de Archify: sintaxis, paths y rangos de línea;
 *  - HTML generado al día respecto al JSON (receipt de finalize);
 *  - seed de permisos vs tabla de usuarios-permisos.md;
 *  - frases/endpoints obsoletos que no deben volver.
 *
 * Uso: node tools/check-docs.mjs
 */
import { createHash } from 'node:crypto';
import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const DOCS = join(ROOT, 'docs');
const FLUJOS = join(DOCS, 'flujos');
const FUENTES = join(DOCS, 'diagramas', 'fuentes');
const PUBLIC_FLUJOS = join(ROOT, 'front', 'public', 'flujos');

const errors = [];
const warnings = [];
const ok = [];

function fail(message) { errors.push(message); }
function warn(message) { warnings.push(message); }
function pass(message) { ok.push(message); }
function rel(path) { return relative(ROOT, path).replaceAll('\\', '/'); }
function read(path) { return readFileSync(path, 'utf8'); }
function lines(path) { return read(path).split(/\r?\n/).length; }
function sha256(text) { return createHash('sha256').update(text.replace(/\r\n/g, '\n')).digest('hex'); }
function listFiles(dir, predicate = () => true) {
  if (!existsSync(dir)) return [];
  return readdirSync(dir)
    .map((name) => join(dir, name))
    .filter((path) => statSync(path).isFile())
    .filter(predicate);
}

function walkFiles(dir, predicate = () => true) {
  const result = [];
  if (!existsSync(dir)) return result;
  for (const name of readdirSync(dir)) {
    const path = join(dir, name);
    const stat = statSync(path);
    if (stat.isDirectory()) result.push(...walkFiles(path, predicate));
    else if (predicate(path)) result.push(path);
  }
  return result;
}

// ---------------------------------------------------------------------------
// 1. Rutas del router
// ---------------------------------------------------------------------------
function extractRoutes() {
  const source = read(join(ROOT, 'back', 'api', 'router.go'));
  const routes = new Set();
  const prefixes = [{ prefix: '', depth: 0 }];
  let depth = 0;

  const routeRe = /r\.Route\("([^"]+)"/;
  const methodRe = /r\.(?:With\([^\n]*?\)\.)?(Get|Post|Put|Delete|Patch)\("([^"]*)"/;

  for (const line of source.split(/\r?\n/)) {
    const routeMatch = line.match(routeRe);
    if (routeMatch && !line.trim().startsWith('//')) {
      prefixes.push({ prefix: routeMatch[1], depth });
    }

    const methodMatch = line.match(methodRe);
    if (methodMatch && !line.trim().startsWith('//')) {
      const prefix = prefixes.at(-1)?.prefix ?? '';
      const path = methodMatch[2];
      const full = path === '/' ? (prefix || '/') : `${prefix}${path}`;
      routes.add(full);
    }

    for (const ch of line) {
      if (ch === '{') depth += 1;
      if (ch === '}') depth -= 1;
    }
    while (prefixes.length > 1 && depth <= prefixes.at(-1).depth) prefixes.pop();
  }

  return [...routes].sort();
}

function checkRoutesInDocs() {
  const docs = listFiles(FLUJOS, (p) => p.endsWith('.md'))
    .map((p) => read(p))
    .join('\n');
  const routes = extractRoutes();
  const allowedMissing = new Set();
  let missing = 0;
  for (const route of routes) {
    if (allowedMissing.has(route)) continue;
    if (!docs.includes(route)) {
      fail(`ruta sin documentar en docs/flujos: ${route}`);
      missing += 1;
    }
  }
  if (missing === 0) pass(`${routes.length} rutas del router cubiertas por docs/flujos`);
}

// ---------------------------------------------------------------------------
// 2. Archivos citados en Markdown
// ---------------------------------------------------------------------------
function checkMarkdownFiles() {
  const markdown = walkFiles(DOCS, (p) => p.endsWith('.md'));
  const fileRe = /\b((?:back|front|docs|deploy)\/[A-Za-z0-9_./<>-]+\.(?:go|ts|html|css|json|md|conf|ya?ml))\b/g;
  const missing = new Set();

  for (const doc of markdown) {
    const text = read(doc);
    for (const match of text.matchAll(fileRe)) {
      const candidate = match[1];
      if (candidate.includes('<') || candidate.includes('>')) continue;
      if (!existsSync(join(ROOT, candidate))) missing.add(`${rel(doc)} -> ${candidate}`);
    }

    // Enlaces Markdown locales: [texto](./ruta.md), [texto](../ruta/)
    for (const match of text.matchAll(/\[[^\]]+\]\(([^)]+)\)/g)) {
      const target = match[1].split('#')[0].split('?')[0].trim();
      if (!target || target.startsWith('http') || target.startsWith('mailto:') || target.startsWith('/')) continue;
      const resolved = resolve(dirname(doc), target);
      if (!existsSync(resolved)) missing.add(`${rel(doc)} -> ${target}`);
    }
  }

  for (const item of missing) fail(`archivo/enlace citado inexistente: ${item}`);
  if (missing.size === 0) pass('todos los archivos y enlaces locales de Markdown existen');
}

// ---------------------------------------------------------------------------
// 3. JSON de diagramas
// ---------------------------------------------------------------------------
function collectSourceRefs(value) {
  const refs = [];
  function visit(node) {
    if (!node || typeof node !== 'object') return;
    if (typeof node.path === 'string' && (typeof node.line === 'number' || typeof node.end_line === 'number')) {
      refs.push(node);
    }
    for (const child of Object.values(node)) visit(child);
  }
  visit(value);
  return refs;
}

function checkDiagramSources() {
  const jsonFiles = listFiles(FUENTES, (p) => p.endsWith('.json'));
  let checked = 0;
  for (const file of jsonFiles) {
    let spec;
    try {
      spec = JSON.parse(read(file));
    } catch (error) {
      fail(`JSON inválido ${rel(file)}: ${error.message}`);
      continue;
    }

    const output = spec.meta?.output;
    if (output && !existsSync(join(ROOT, output))) {
      fail(`${rel(file)} declara una salida inexistente: ${output}`);
    }

    for (const ref of collectSourceRefs(spec)) {
      const path = join(ROOT, ref.path);
      if (!existsSync(path)) {
        fail(`${rel(file)} cita un archivo inexistente: ${ref.path}`);
        continue;
      }
      const count = lines(path);
      if (typeof ref.line === 'number' && ref.line > count) {
        fail(`${rel(file)}: ${ref.path}:${ref.line} excede ${count} líneas`);
      }
      if (typeof ref.end_line === 'number' && ref.end_line > count) {
        fail(`${rel(file)}: ${ref.path}:${ref.end_line} excede ${count} líneas`);
      }
      checked += 1;
    }
  }
  if (checked > 0) pass(`${checked} referencias de código dentro de rango`);
}

function checkGeneratedDiagrams() {
  const manifestPath = join(DOCS, 'diagramas', 'manifest.json');
  if (!existsSync(manifestPath)) {
    fail('falta docs/diagramas/manifest.json; ejecuta node tools/update-diagram-manifest.mjs');
    return;
  }
  let manifest;
  try {
    manifest = JSON.parse(read(manifestPath));
  } catch (error) {
    fail(`manifest inválido: ${error.message}`);
    return;
  }

  const jsonFiles = listFiles(FUENTES, (p) => p.endsWith('.json'));
  for (const file of jsonFiles) {
    const slug = file.split(/[\\/]/).at(-1).replace(/\.json$/, '');
    const html = join(PUBLIC_FLUJOS, `${slug}.html`);
    const entry = manifest.diagrams?.[slug];
    if (!existsSync(html)) {
      fail(`diagrama sin HTML generado: ${slug}.html`);
      continue;
    }
    if (!entry) {
      fail(`${slug} no está en docs/diagramas/manifest.json`);
      continue;
    }
    if (entry.spec_sha256 !== sha256(read(file))) {
      fail(`${slug}.html está desactualizado respecto a ${slug}.json`);
    }
    if (entry.artifact_sha256 !== sha256(read(html))) {
      fail(`${slug}.html cambió sin actualizar el manifest`);
    }
  }
  pass('HTML de diagramas comprobado contra el manifest');
}

// ---------------------------------------------------------------------------
// 4. Permisos
// ---------------------------------------------------------------------------
function checkPermissions() {
  const db = read(join(ROOT, 'back', 'infrastructure', 'database', 'db.go'));
  const start = db.indexOf('perms := []struct');
  const end = db.indexOf('for _, p := range perms', start);
  const seedBlock = start >= 0 && end > start ? db.slice(start, end) : '';
  const names = [...seedBlock.matchAll(/\{"([a-z_]+)",/g)].map((m) => m[1]);
  const doc = read(join(FLUJOS, 'usuarios-permisos.md'));
  if (names.length === 0) {
    fail('no se pudieron extraer los permisos del seed');
    return;
  }
  for (const name of names) {
    if (!doc.includes(`\`${name}\``)) fail(`permiso del seed sin documentar en usuarios-permisos.md: ${name}`);
  }
  pass(`${names.length} permisos del seed documentados`);
}

// ---------------------------------------------------------------------------
// 5. Invariantes de contenido obsoleto
// ---------------------------------------------------------------------------
function checkForbiddenContent() {
  const forbidden = [
    '/mensajes_pendientes/events',
    'Eliminar un usuario no revoca',
    'ruta directa puede instanciarse',
    'no consulta JWTStore ni activo',
    'Token nuevo',
  ];
  const files = [
    ...listFiles(FUENTES, (p) => p.endsWith('.json')),
    ...listFiles(FLUJOS, (p) => p.endsWith('.md')),
    ...listFiles(PUBLIC_FLUJOS, (p) => p.endsWith('.html')),
  ];
  let hits = 0;
  for (const file of files) {
    const text = read(file);
    for (const phrase of forbidden) {
      if (text.includes(phrase)) {
        fail(`contenido obsoleto en ${rel(file)}: "${phrase}"`);
        hits += 1;
      }
    }
  }
  if (hits === 0) pass('sin endpoints ni frases obsoletas conocidas');
}

// ---------------------------------------------------------------------------
// Ejecución
// ---------------------------------------------------------------------------
console.log('Comprobando documentación del template...\n');
checkRoutesInDocs();
checkMarkdownFiles();
checkDiagramSources();
checkGeneratedDiagrams();
checkPermissions();
checkForbiddenContent();

for (const message of ok) console.log(`  OK   ${message}`);
for (const message of warnings) console.log(`  WARN ${message}`);
for (const message of errors) console.log(`  FAIL ${message}`);

console.log(`\n${ok.length} comprobaciones OK, ${warnings.length} avisos, ${errors.length} errores.`);
if (errors.length > 0) process.exit(1);
