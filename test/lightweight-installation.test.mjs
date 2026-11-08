import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import path from 'node:path';
import {test} from 'node:test';
import {repositoryRoot} from '../scripts/skill-registry.mjs';

test('installation entry keeps one short normal path with no duplicate bootstrap lookup', async () => {
  const text = await readFile(path.join(repositoryRoot, 'skills/puretokens-update/SKILL.md'), 'utf8');
  assert.ok(Buffer.byteLength(text) <= 9500, 'move exceptional detail to existing conditional references');
  assert.match(text, /--release-metadata/);
  assert.match(text, /-ReleaseMetadata/);
  assert.match(text, /已有结果不另跑一次 init/);
  assert.match(text, /一个.*下一步/);
  assert.match(text, /不下载 ZIP、不写入、不 init/);
});

test('copyable install prompts are concise without dropping bootstrap size and digest checks', async () => {
  for (const file of ['README.md', 'README.zh-CN.md']) {
    const text = await readFile(path.join(repositoryRoot, file), 'utf8');
    const zh = text.match(/### (?:Chinese installation prompt|中文安装提示词)\n\n```text\n([\s\S]*?)\n```/)?.[1];
    const en = text.match(/### (?:English installation prompt|英文安装提示词)\n\n```text\n([\s\S]*?)\n```/)?.[1];
    assert.ok(zh && en);
    assert.ok(zh.length <= 260 && en.length <= 440);
    for (const prompt of [zh, en]) {
      assert.match(prompt, /size/);
      assert.match(prompt, /digest/);
      assert.match(prompt, /Accept: application\/octet-stream/);
      assert.match(prompt, /puretokens-skill-fetch.sh/);
      assert.match(prompt, /puretokens-skill-fetch.ps1/);
      assert.match(prompt, /releases\/latest/);
    }
  }
});
