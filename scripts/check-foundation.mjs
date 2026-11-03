import { readFileSync, existsSync, readdirSync } from 'node:fs';
import { createHash } from 'node:crypto';
import { join, resolve } from 'node:path';
import process from 'node:process';

const root = resolve(import.meta.dirname, '..');
const read = (path) => readFileSync(join(root, path), 'utf8');
const checks = [];
const verifyOriginal = process.argv.includes('--verify-original');
const check = (label, ok) => { checks.push([label, ok]); if (!ok) process.exitCode = 1; };
const source = JSON.parse(read('docs/design/original-source-sha256.json'));
for (const [file, digest] of Object.entries(source)) {
  const path = join(root, file);
  check(`${verifyOriginal ? 'original bytes' : 'source exists'}: ${file}`, existsSync(path) && (!verifyOriginal || createHash('sha256').update(readFileSync(path)).digest('hex') === digest));
}
check('25-step onboarding source', (read('src/data.js').match(/  '\w[\w-]*',/g) || []).length >= 25);
check('RTL App', read('src/App.jsx').includes('dir="rtl"'));
check('designer chat component present', read('src/screens/Main.jsx').includes('function Chat('));
check('backend API snapshot', read('contracts/openapi/backend-openapi.snapshot.yaml').includes('/api/v1/conversations/{id}/turns:'));
check('loopback proxy', read('vite.config.js').includes('127.0.0.1:8080') && !read('vite.config.js').includes('allowedHosts: true'));
for (const file of ['AGENTS.md','MVP_SCOPE.md','PRODUCT_CONSTITUTION.md','TECHNICAL_CONSTITUTION.md','FRONTEND_DESIGN_SPEC.md','REPO_MAP.md','NEXT_AGENT_TASKS.md','README_FA.md','VALIDATION_REPORT.md','docs/integration/UI_BACKEND_GAP_MATRIX.md','docs/roadmap/BACKEND_FRONTEND_INTEGRATION_PLAN.md']) {
  check(`agent doc ${file}`, existsSync(join(root,file)));
}
const skills = readdirSync(join(root, '.agents/skills'), { withFileTypes: true }).filter(d=>d.isDirectory());
check('exactly 7 specialist skills',skills.length===7);
for (const d of skills) {
  check(`skill ${d.name}`, existsSync(join(root,'.agents/skills',d.name,'SKILL.md')) && existsSync(join(root,'.agents/skills',d.name,'agents/openai.yaml')));
}
const failed=checks.filter(([,ok])=>!ok);
console.log(`Hamrah foundation checks: ${checks.length-failed.length}/${checks.length} passed`);
for (const [label] of failed) console.error('FAIL:',label);
