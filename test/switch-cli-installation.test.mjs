import assert from 'node:assert/strict';
import { test } from 'node:test';
import { execFile as run } from 'node:child_process';
import { promisify } from 'node:util';
import { mkdtemp, mkdir, readFile, rm } from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';
import { repositoryRoot } from '../scripts/skill-registry.mjs';
const execFile = promisify(run);
const installer = path.join(repositoryRoot, 'runtime/puretokens-skill-install.sh');
const cases = [
 ['qwen-code', '.qwen', 'QWEN_HOME'],
 ['codebuddy-code', '.codebuddy', 'CODEBUDDY_CONFIG_DIR'],
 ['copilot-cli', '.copilot', 'COPILOT_HOME'],
 ['factory-droid', '.factory', null],
 ['mimo-coder', '.config/mimocode', 'MIMOCODE_HOME'],
 ['paseo', '.agents', null]
];
test('Switch CLI hosts install ten Skills and one current-platform executor', async t => {
 const root = await mkdtemp(path.join(os.tmpdir(), 'pt-switch-cli-'));
 t.after(() => rm(root, { recursive: true, force: true }));
 const home = path.join(root, 'home'); await mkdir(home);
 const clean = { ...process.env, HOME: home, USERPROFILE: home, PASEO_HOME: '', QWEN_HOME: '', CODEBUDDY_CONFIG_DIR: '', COPILOT_HOME: '', FACTORY_HOME_OVERRIDE: '', MIMOCODE_HOME: '', XDG_CONFIG_HOME: '' };
 for (const [host, folder, override] of cases) {
  const target = path.join(home, folder, 'skills');
  const locate = env => execFile('sh', [installer, 'locate', '--host', host], { env });
  assert.equal((await locate(clean)).stdout.trim(), target);
  if (override) {
   const selected = path.join(root, host + ' custom');
   assert.equal((await locate({ ...clean, [override]: selected })).stdout.trim(), path.join(selected, ...(host === 'mimo-coder' ? ['config'] : []), 'skills'));
   for (const bad of ['relative', selected + '/../other']) await assert.rejects(locate({ ...clean, [override]: bad }));
  }
  await execFile('sh', [installer, 'sync', '--host', host], { env: clean });
  const registry = JSON.parse(await readFile(path.join(repositoryRoot, 'skills/index.json'), 'utf8'));
  for (const skill of registry.skills) assert.ok(JSON.parse(await readFile(path.join(target, skill.name, 'skill.json'), 'utf8')).supportedClients.includes(host));
  await execFile('sh', [installer, 'verify-installed', '--host', host], { env: clean });
 }
 await assert.rejects(execFile('sh', [installer, 'locate', '--host', 'factory-droid', '--target', path.join(root,'explicit')], {env:{...clean,FACTORY_HOME_OVERRIDE:root}}));
});
