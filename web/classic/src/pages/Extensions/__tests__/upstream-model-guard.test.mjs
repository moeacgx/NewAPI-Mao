import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import path from 'node:path'
import { after, test } from 'node:test'
import { fileURLToPath, pathToFileURL } from 'node:url'

const directory = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(directory, '../../../../../..')
const require = createRequire(path.join(root, 'web/classic/package.json'))
require.extensions['.css'] = () => {}
const { JSDOM } = require('jsdom')
const dom = new JSDOM('<!doctype html><html><body></body></html>', {
  url: 'http://localhost/',
})
for (const key of [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'Element',
  'Node',
  'DocumentFragment',
  'Event',
  'MouseEvent',
  'HTMLInputElement',
  'getComputedStyle',
]) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: dom.window[key],
  })
}
globalThis.IS_REACT_ACT_ENVIRONMENT = true
globalThis.requestAnimationFrame = (callback) => setTimeout(callback, 0)
globalThis.cancelAnimationFrame = clearTimeout
globalThis.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
}
window.matchMedia = () => ({
  matches: false,
  addListener() {},
  removeListener() {},
  addEventListener() {},
  removeEventListener() {},
})
const React = require('react')
const runtime = require('react/jsx-runtime')
const I18n = require('react-i18next')
const i18next = require('i18next')
const Semi = Object.fromEntries(
  ['Button', 'Card', 'Input', 'Table'].map((name) => [
    name,
    require(`@douyinfe/semi-ui/lib/cjs/${name.toLowerCase()}`).default,
  ])
)
const { fireEvent, screen, within } = require('@testing-library/dom')
const { createRoot } = require('react-dom/client')
const { act } = React
await i18next.use(I18n.initReactI18next).init({
  lng: 'en',
  fallbackLng: 'en',
  resources: {
    en: { translation: {} },
    'zh-CN': { translation: {} },
  },
})

const base = '/api/extensions/upstream-model-guard'
const writes = []
let records = []
after(() => dom.window.close())
const ok = (data) => ({ data: { success: true, data } })
const api = {
  async get(url, options) {
    if (url === `${base}/config`) {
      return ok({ config_version: 4, enabled: true, rules: [] })
    }
    if (url === `${base}/groups`) {
      return ok([
        { id: 2, code: 'vip', name: 'Premium group' },
        { id: 3, code: 'official', name: 'Official group' },
      ])
    }
    assert.deepEqual(options.params, { page: 1, page_size: 20 })
    return ok({ items: records, total: records.length, page: 1, page_size: 20 })
  },
  async put(url, data) {
    writes.push({ url, data })
    return ok({ ...data, config_version: 5 })
  },
}
const helpers = { getAPI: () => api }
globalThis.__NEW_API_EXTENSION_NATIVE_SDK__ = {
  sdk: 'v1',
  platform: 'classic',
  modules: {
    react: React,
    'react/jsx-runtime': runtime,
    'react-i18next': I18n,
    '@douyinfe/semi-ui': Semi,
    '../../helpers': helpers,
  },
}
const { default: Page } = await import(
  pathToFileURL(
    path.join(root, 'extensions/upstream-model-guard/public/native/classic.mjs')
  )
)

test(
  'Classic SDK controls save checked groups and preserve multiple upstream models',
  { timeout: 15000 },
  async () => {
    const container = document.createElement('div')
    document.body.appendChild(container)
    const renderer = createRoot(container)
    try {
      await act(async () => renderer.render(React.createElement(Page)))
      await act(async () =>
        fireEvent.click(screen.getByRole('button', { name: 'Add rule' }))
      )
      const rule = screen.getByRole('group', { name: 'Rule 1' })
      await act(async () =>
        fireEvent.click(
          within(rule).getByRole('checkbox', { name: 'Premium group' })
        )
      )
      await act(async () =>
        fireEvent.change(within(rule).getByLabelText('Request model'), {
          target: { value: 'gpt-test' },
        })
      )
      await act(async () =>
        fireEvent.change(
          within(rule).getByLabelText('Allowed upstream models'),
          {
            target: { value: 'provider-A\nprovider-B' },
          }
        )
      )
      await act(async () =>
        fireEvent.click(screen.getByRole('button', { name: 'Save settings' }))
      )
      assert.equal(writes.length, 1)
      assert.deepEqual(writes[0], {
        url: `${base}/config`,
        data: {
          expected_version: 4,
          enabled: true,
          rules: [
            {
              enabled: true,
              group_codes: ['vip'],
              model: 'gpt-test',
              upstream_models: ['provider-A', 'provider-B'],
            },
          ],
        },
      })
      assert.equal(
        screen
          .getByRole('link', { name: 'Notification Center' })
          .getAttribute('href'),
        '/notification-center'
      )
      assert.ok(screen.getByText('No channel disable records'))
      await act(async () => i18next.changeLanguage('zh-CN'))
      assert.ok(screen.getByRole('button', { name: '添加规则' }))
      assert.ok(screen.getByRole('link', { name: '通知中心' }))
      assert.ok(screen.getByText('暂无渠道禁用记录'))
      assert.equal(
        i18next.getResource('zh-CN', 'translation', 'Upstream model guard'),
        undefined
      )
      assert.equal(
        i18next.getResource(
          'zh-CN',
          'upstream-model-guard',
          'Upstream model guard'
        ),
        '上游模型校验'
      )
    } finally {
      await act(async () => renderer.unmount())
      container.remove()
    }
  }
)

test(
  'Classic records prefer explicit group names and fall back to available names before codes',
  { timeout: 15000 },
  async () => {
    await i18next.changeLanguage('en')
    records = [
      {
        id: 1,
        channel_id: 906,
        channel_name: 'Channel A',
        group: 'vip',
        group_name: 'Current premium group',
      },
      { id: 2, channel_id: 907, channel_name: 'Channel B', group: 'official' },
      {
        id: 3,
        channel_id: 908,
        channel_name: 'Channel C',
        group: 'deleted-code',
      },
    ]
    const container = document.createElement('div')
    document.body.appendChild(container)
    const renderer = createRoot(container)
    try {
      await act(async () => renderer.render(React.createElement(Page)))
      const table = screen.getByRole('grid')
      assert.ok(
        within(table).getByRole('gridcell', { name: 'Current premium group' })
      )
      assert.ok(within(table).getByRole('gridcell', { name: 'Official group' }))
      assert.ok(within(table).getByRole('gridcell', { name: 'deleted-code' }))
      assert.equal(within(table).queryByRole('gridcell', { name: 'vip' }), null)
      assert.equal(
        within(table).queryByRole('gridcell', { name: 'Premium group' }),
        null
      )
    } finally {
      await act(async () => renderer.unmount())
      container.remove()
      records = []
    }
  }
)
