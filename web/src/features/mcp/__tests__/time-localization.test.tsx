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
import type { QueryClient } from '@tanstack/react-query'
import { act, cleanup, render, screen, within } from '@testing-library/react'
import { createInstance, type i18n } from 'i18next'
import { I18nextProvider, initReactI18next } from 'react-i18next'
import { afterEach, beforeEach, expect, test } from 'vitest'

import en from '@/i18n/locales/en.json'
import zhTW from '@/i18n/locales/zh-TW.json'
import zhCN from '@/i18n/locales/zh.json'
import { api } from '@/lib/api'

import { KeysPanel } from '../components/keys-panel'
import { LogsPanel } from '../components/logs-panel'
import { McpTime } from '../components/shared'
import { key, renderMcp, response, tools } from './fixtures'

const originalAdapter = api.defaults.adapter
const timestamp = 1700000000
let language: i18n
let client: QueryClient | undefined

beforeEach(async () => {
  language = createInstance()
  await language.use(initReactI18next).init({
    lng: 'en',
    fallbackLng: 'en',
    resources: { en, zhCN, zhTW },
    interpolation: { escapeValue: false },
  })
})

afterEach(() => {
  cleanup()
  client?.clear()
  client = undefined
  api.defaults.adapter = originalAdapter
})

test.each([8640000000001, Number.MAX_VALUE, Number.NaN, Infinity, -1, 0])(
  '时间值为 %s 时显示占位符而不抛出异常',
  (value) => {
    render(
      <I18nextProvider i18n={language}>
        <McpTime value={value} />
      </I18nextProvider>
    )
    expect(screen.getByText('—')).toBeVisible()
    expect(screen.queryByRole('time')).not.toBeInTheDocument()
  }
)

test('日期范围内的最大时间仍可正常显示', () => {
  render(
    <I18nextProvider i18n={language}>
      <McpTime value={8640000000000} />
    </I18nextProvider>
  )
  expect(screen.getByRole('time')).toHaveAttribute(
    'datetime',
    '+275760-09-13T00:00:00.000Z'
  )
})

test('服务端返回超大 expires_at 时密钥行仍可见且到期时间显示占位符', async () => {
  await language.changeLanguage('zhCN')
  ;({ client } = renderMcp(
    <I18nextProvider i18n={language}>
      <KeysPanel />
    </I18nextProvider>,
    async (config) =>
      response(
        config,
        config.url?.endsWith('/tools')
          ? tools
          : { items: [{ ...key, expires_at: 8640000000001 }], total: 1 }
      )
  ))
  const row = await screen.findByRole('row', { name: new RegExp(key.name) })
  expect(within(row).getByRole('cell', { name: '—' })).toBeVisible()
  expect(
    within(row).getByRole('button', { name: language.t('Revoke') })
  ).toBeEnabled()
})

test.each([
  ['zhCN', 'zh-CN'],
  ['zhTW', 'zh-TW'],
  ['invalid_locale!', undefined],
])(
  '真实语言切换为 %s 后日期正确显示且不抛出 RangeError',
  async (code, locale) => {
    render(
      <I18nextProvider i18n={language}>
        <McpTime value={timestamp} />
      </I18nextProvider>
    )
    await act(async () => {
      await language.changeLanguage(code)
    })
    expect(language.language).toBe(code)
    const date = new Date(timestamp * 1000)
    expect(screen.getByRole('time')).toHaveTextContent(
      date.toLocaleString(locale).replaceAll(/\s/g, ' ')
    )
    expect(screen.getByRole('time')).toHaveAttribute(
      'datetime',
      date.toISOString()
    )
  }
)

test('真实 zhCN 语言下密钥列表和调用记录完整渲染创建、过期、最近使用及撤销时间', async () => {
  await language.changeLanguage('zhCN')
  ;({ client } = renderMcp(
    <I18nextProvider i18n={language}>
      <KeysPanel />
      <LogsPanel />
    </I18nextProvider>,
    async (config) => {
      if (config.url?.endsWith('/tools')) return response(config, tools)
      if (config.url?.endsWith('/keys')) {
        return response(config, {
          items: [
            { ...key, last_used_at: timestamp + 1, revoked_at: timestamp + 2 },
          ],
          total: 1,
        })
      }
      return response(config, {
        items: [
          {
            id: 1,
            key_id: key.id,
            tool_name: 'newapi_context',
            request_id: 'req-zhCN',
            success: true,
            duration_ms: 25,
            created_at: timestamp + 3,
          },
        ],
        total: 1,
      })
    }
  ))
  expect(await screen.findByText(key.name)).toBeVisible()
  expect(await screen.findByText('req-zhCN')).toBeVisible()
  expect(screen.getByText('已撤销')).toBeVisible()
  expect(screen.getByText('调用记录')).toBeVisible()
  for (const value of [
    key.created_at,
    key.expires_at,
    timestamp + 1,
    timestamp + 2,
    timestamp + 3,
  ]) {
    expect(
      screen.getByText(new Date(value * 1000).toLocaleString('zh-CN'))
    ).toBeVisible()
  }
})
