/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { afterEach, expect, test, vi } from 'vitest'

import { isPasskeySupported } from '../passkey'

afterEach(() => vi.unstubAllGlobals())

test('浏览器支持 WebAuthn 但没有平台认证器时仍允许外部密钥与跨设备验证', async () => {
  vi.stubGlobal('PublicKeyCredential', {
    isConditionalMediationAvailable: async () => false,
    isUserVerifyingPlatformAuthenticatorAvailable: async () => false,
  })
  expect(await isPasskeySupported()).toBe(true)
})

test('浏览器缺少 WebAuthn 时不提供 Passkey', async () => {
  vi.stubGlobal('PublicKeyCredential', undefined)
  expect(await isPasskeySupported()).toBe(false)
})

test('服务端没有 window 时不提供 Passkey', async () => {
  vi.stubGlobal('window', undefined)
  expect(await isPasskeySupported()).toBe(false)
})
