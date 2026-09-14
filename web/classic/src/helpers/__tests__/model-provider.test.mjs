import test from 'node:test';
import assert from 'node:assert/strict';
import { resolveModelProvider } from '../modelProvider.js';
for (const [name, expected] of [
  ['360gpt-pro', '360 AI'],
  ['wan2.2-t2v', 'Wan'],
  ['text-embedding-v4', 'Qwen'],
  ['qwen3-embedding', 'Qwen'],
  ['step-tts-mini', 'StepFun'],
  ['gpt-4o', 'OpenAI'],
  ['vendor/unknown', null],
  ['@cf/meta/llama-3', 'Cloudflare'],
]) {
  test(`${name}供应商推断为${expected}`, () =>
    assert.equal(resolveModelProvider(name)?.name ?? null, expected));
}
