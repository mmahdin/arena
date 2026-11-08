import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, readdirSync, existsSync } from 'node:fs';
import { join } from 'node:path';
const root = new URL('../',import.meta.url);
const read = (file) => readFileSync(new URL(file,root),'utf8');

test('designer routes and visual assets remain present', () => {
  const data=read('src/data.js');
  const app=read('src/App.jsx');
  for (const r of ['welcome','privacy','age','therapist','voice','approach','ready','home']) assert.match(data,new RegExp(`'${r}'`));
  assert.match(app,/routeFromHash/);
  assert.match(app,/dir="rtl"/);
  // Kenar identity: the floral Ash artwork is replaced by the stone/shore set.
  for (const asset of ['brand/kenar-mark.svg', 'brand/kenar-icon.svg', 'kenar/welcome-figures.svg',
    'kenar/companion-small.svg', 'kenar/companion-medium.svg', 'kenar/companion-large.svg',
    'kenar/companion-tall.svg', 'kenar/privacy-ring.svg', 'kenar/ready-companions.svg',
    'kenar/checkins-rhythm.svg']) {
    assert.ok(existsSync(new URL(`public/assets/${asset}`, root)), asset);
  }
  assert.ok(!existsSync(new URL('public/assets/welcome-figures.png', root)), 'Ash floral art must not return');
});

test('handoff defines synthetic-only release scope and future API integration', () => {
  assert.match(read('AGENTS.md'),/synthetic|ساختگی/i);
  assert.match(read('MVP_SCOPE.md'),/synthetic|ساختگی/i);
  assert.match(read('docs/roadmap/BACKEND_FRONTEND_INTEGRATION_PLAN.md'),/Vite|MSSQL/);
});

test('Task07 snapshot provides ten core API methods', () => {
  const yaml=read('contracts/openapi/backend-openapi.snapshot.yaml');
  for (const route of ['/api/v1/dev/users:','/api/v1/dev/session:','/api/v1/me:','/api/v1/me/profile:','/api/v1/conversations:','/api/v1/conversations/{id}/messages:','/api/v1/conversations/{id}/turns:']) assert.ok(yaml.includes(route),route);
  assert.match(yaml,/clientTurnId/);
});

test('network is local and no direct provider integration', () => {
  const vite=read('vite.config.js');
  assert.match(vite,/127\.0\.0\.1:8080/);
  assert.doesNotMatch(vite,/allowedHosts:\s*true/);
  assert.doesNotMatch(read('src/App.jsx')+read('src/screens/Main.jsx'),/api\.openai\.com|mssql:|fetch\('/);
});
