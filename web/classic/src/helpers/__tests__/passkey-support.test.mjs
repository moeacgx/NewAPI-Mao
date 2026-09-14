import test from 'node:test';
import assert from 'node:assert/strict';
import { isPasskeySupported } from '../passkey.js';
test('无平台认证器仍允许外置密钥，SSR和不支持WebAuthn返回false', async () => {
  const old = globalThis.window;
  try {
    delete globalThis.window;
    assert.equal(await isPasskeySupported(), false);
    globalThis.window = {};
    assert.equal(await isPasskeySupported(), false);
    globalThis.window.PublicKeyCredential = {
      isUserVerifyingPlatformAuthenticatorAvailable: async () => false,
    };
    assert.equal(await isPasskeySupported(), true);
    globalThis.window.PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable =
      async () => {
        throw new Error('不可用');
      };
    assert.equal(await isPasskeySupported(), true);
  } finally {
    if (old === undefined) delete globalThis.window;
    else globalThis.window = old;
  }
});
