// Local documentation paths only: no network access and no artifact mutation.
import { readdir, readFile, stat } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const errors = [];
const linked = new Set();
let localLinks = 0;

async function exists(filename) {
  try {
    await stat(filename);
    return true;
  } catch (error) {
    if (error.code !== 'ENOENT' && error.code !== 'ENOTDIR') throw error;
    return false;
  }
}

async function markdownFiles(directory) {
  const files = [];
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const filename = path.join(directory, entry.name);
    if (entry.isDirectory()) files.push(...await markdownFiles(filename));
    else if (entry.isFile() && entry.name.endsWith('.md')) files.push(filename);
  }
  return files;
}

const guides = [
  'README.md', 'AGENTS.md', 'backend/README.md', 'frontend/README.md',
  'backend/node-service/README.md', 'backend/go-service/README.md',
  'backend/go-service/scripts/README.md', 'ai_coach/README.md',
  'outputs/README.md', 'output/README.md',
];
const files = [...await markdownFiles(path.join(root, 'docs')), ...guides.map(p => path.join(root, p))];

for (const filename of files) {
  const relative = path.relative(root, filename);
  const markdown = await readFile(filename, 'utf8');
  // Exclude fenced examples; inspect inline and reference-style destinations.
  const prose = markdown.replace(/^\s*(`{3,}|~{3,})[^\n]*\n[\s\S]*?^\s*\1\s*$/gm, '');
  const targets = [
    ...prose.matchAll(/!?\[[^\]\n]*\]\(\s*(?:<([^>\n]+)>|([^\s)]+))(?:\s+"[^"\n]*")?\s*\)/g),
    ...prose.matchAll(/^\s*\[[^\]\n]+\]:\s*(?:<([^>\n]+)>|([^\s]+))/gm),
  ];
  for (const match of targets) {
    const href = match[1] ?? match[2];
    if (/^(?:[a-z][a-z0-9+.-]*:|#|\/\/)/i.test(href)) continue;
    const target = decodeURIComponent(href.split(/[?#]/, 1)[0]);
    if (!target) continue;
    const destination = path.resolve(path.dirname(filename), target);
    localLinks += 1;
    linked.add(destination);
    if (!await exists(destination)) errors.push(`${relative}: missing ${href}`);
  }
}

const locations = JSON.parse(await readFile(path.join(root, 'docs/references/document-locations.json'), 'utf8'));
for (const [previous, current] of Object.entries(locations.moves)) {
  for (const location of [previous, current]) {
    if (!location.startsWith('docs/') || location.split('/').includes('..')) {
      errors.push(`Invalid relocation path: ${location}`);
    }
  }
  const destination = path.join(root, current);
  if (!await exists(destination)) errors.push(`Relocation missing: ${current}`);
  if (await exists(path.join(root, previous))) errors.push(`Relocation has two copies: ${previous} and ${current}`);
  if (!linked.has(destination)) errors.push(`Relocated document has no incoming Markdown link: ${current}`);
}

const prompts = (await readdir(path.join(root, 'docs/prompts'))).filter(name => name.endsWith('.md'));
if (prompts.length !== 1) errors.push(`Expected one active Markdown prompt; found ${prompts.length}`);

if (errors.length) {
  for (const error of errors) console.error(error);
  process.exitCode = 1;
} else {
  console.log(`Documentation OK: ${files.length} Markdown files, ${localLinks} local links, ${Object.keys(locations.moves).length} relocations, one active prompt.`);
  console.log('Scope: docs/ and maintained entry guides. External URLs, heading anchors and retained historical artifacts are not validated.');
}
