import { readFileSync, existsSync, readdirSync } from 'node:fs';
import { createHash } from 'node:crypto';
import { join, resolve } from 'node:path';
import process from 'node:process';
import { FLOW } from '../src/data.js';

const root = resolve(import.meta.dirname, '..');
const read = (path) => readFileSync(join(root, path), 'utf8');
const checks = [];
const verifyOriginal = process.argv.includes('--verify-original');
const check = (label, ok) => { checks.push([label, ok]); if (!ok) process.exitCode = 1; };
const source = JSON.parse(read('docs/design/original-source-sha256.json'));
// The manifest describes the full designer ZIP. This checkout intentionally
// ships only the tracked source files plus one retired SVG, so the default
// audit checks the tracked originals. `--verify-original` still demands the
// complete byte-for-byte baseline and is expected to fail after a redesign.
const trackedOriginals = Object.entries(source).filter(([file]) => existsSync(join(root, file)));
check('original source manifest present', Object.keys(source).length > 0);
check('tracked original source files remain present', trackedOriginals.length >= 8);
if (verifyOriginal) {
  for (const [file, digest] of Object.entries(source)) {
    const path = join(root, file);
    check(`original bytes: ${file}`, existsSync(path) && createHash('sha256').update(readFileSync(path)).digest('hex') === digest);
  }
}
check('retired design artifact preserved', existsSync(join(root, 'public/assets/checkins-handdrawn.svg')));
check('complete onboarding source', FLOW.length >= 25 && new Set(FLOW).size === FLOW.length && FLOW.at(-1) === 'home');
check('RTL App', read('src/App.jsx').includes('dir="rtl"'));
check('designer chat component present', read('src/screens/Main.jsx').includes('function Chat('));
check('backend API snapshot', read('contracts/openapi/backend-openapi.snapshot.yaml').includes('/api/v1/conversations/{id}/turns:'));
check('loopback proxy', read('vite.config.js').includes('127.0.0.1:8080') && !read('vite.config.js').includes('allowedHosts: true'));
for (const file of ['AGENTS.md','MVP_SCOPE.md','PRODUCT_CONSTITUTION.md','TECHNICAL_CONSTITUTION.md','FRONTEND_DESIGN_SPEC.md','REPO_MAP.md','NEXT_AGENT_TASKS.md','README_FA.md','VALIDATION_REPORT.md','docs/integration/UI_BACKEND_GAP_MATRIX.md','docs/roadmap/BACKEND_FRONTEND_INTEGRATION_PLAN.md']) {
  check(`agent doc ${file}`, existsSync(join(root,file)));
}
// The local-only skill catalog is not tracked in this checkout. When it is
// present it must still contain the complete seven-workflow set; when it is
// absent, the foundation audit should not crash on a missing optional folder.
const skillsRoot = join(root, '.agents/skills');
const skills = existsSync(skillsRoot)
  ? readdirSync(skillsRoot, { withFileTypes: true }).filter(d => d.isDirectory())
  : [];
check('specialist skill catalog is complete or absent', skills.length === 0 || skills.length === 7);
for (const d of skills) {
  check(`skill ${d.name}`, existsSync(join(skillsRoot,d.name,'SKILL.md')) && existsSync(join(skillsRoot,d.name,'agents/openai.yaml')));
}
const failed=checks.filter(([,ok])=>!ok);
console.log(`Hamrah foundation checks: ${checks.length-failed.length}/${checks.length} passed`);
for (const [label] of failed) console.error('FAIL:',label);
