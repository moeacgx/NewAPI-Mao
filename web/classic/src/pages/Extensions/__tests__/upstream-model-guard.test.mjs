import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import path from 'node:path'
import { after, beforeEach, test } from 'node:test'
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
  ]),
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
let configuration = {}
let channelOptions = []
let channelError = false
beforeEach(async () => {
  await i18next.changeLanguage('en')
  writes.length = 0
  records = []
  configuration = { config_version: 4, enabled: true, rules: [] }
  channelOptions = []
  channelError = false
})
after(() => dom.window.close())
const ok = (data) => ({ data: { success: true, data } })
const api = {
  async get(url, options) {
    if (url === `${base}/config`) {
      return ok(configuration)
    }
    if (url === `${base}/channels`) {
      if (channelError) throw new Error('Channels unavailable')
      return ok({
        items: channelOptions,
        total: channelOptions.length,
        page: 1,
        page_size: 50,
      })
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
    path.join(
      root,
      'extensions/upstream-model-guard/public/native/classic.mjs',
    ),
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
      assert.equal(
        screen.getByLabelText('Consecutive mismatch threshold').value,
        '2',
      )
      await act(async () =>
        fireEvent.click(screen.getByRole('button', { name: 'Add rule' })),
      )
      const rule = screen.getByRole('group', { name: 'Rule 1' })
      await act(async () =>
        fireEvent.click(
          within(rule).getByRole('checkbox', { name: 'Premium group' }),
        ),
      )
      await act(async () =>
        fireEvent.change(within(rule).getByLabelText('Request model'), {
          target: { value: 'gpt-test' },
        }),
      )
      await act(async () =>
        fireEvent.change(
          within(rule).getByLabelText('Allowed upstream models'),
          {
            target: { value: 'provider-A\nprovider-B' },
          },
        ),
      )
      await act(async () =>
        fireEvent.click(screen.getByRole('button', { name: 'Save settings' })),
      )
      assert.equal(writes.length, 1)
      assert.deepEqual(writes[0], {
        url: `${base}/config`,
        data: {
          expected_version: 4,
          enabled: true,
          failure_threshold: 2,
          excluded_channel_ids: [],
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
        '/notification-center',
      )
      assert.ok(screen.getByText('No model mismatch records'))
      await act(async () => i18next.changeLanguage('zh-CN'))
      assert.ok(screen.getByRole('button', { name: '添加规则' }))
      assert.ok(screen.getByRole('link', { name: '通知中心' }))
      assert.ok(screen.getByText('暂无模型异常记录'))
      assert.equal(
        i18next.getResource('zh-CN', 'translation', 'Upstream model guard'),
        undefined,
      )
      assert.equal(
        i18next.getResource(
          'zh-CN',
          'upstream-model-guard',
          'Upstream model guard',
        ),
        '上游模型校验',
      )
    } finally {
      await act(async () => renderer.unmount())
      container.remove()
    }
  },
)

test(
  'Classic validates threshold and saves allowlist additions and removals using channel IDs',
  { timeout: 15000 },
  async () => {
    configuration = {
      ...configuration,
      failure_threshold: 3,
      excluded_channel_ids: [900, 901],
      excluded_channels: [
        { id: 900, name: 'Known provider', status: 1 },
        { id: 901, name: '', status: 0 },
      ],
    }
    channelOptions = [{ id: 902, name: 'New provider', status: 1 }]
    const container = document.createElement('div')
    document.body.appendChild(container)
    const renderer = createRoot(container)
    try {
      await act(async () => renderer.render(React.createElement(Page)))
      assert.ok(screen.getByText('Known provider (#900)'))
      await act(async () =>
        fireEvent.change(
          screen.getByLabelText('Consecutive mismatch threshold'),
          { target: { value: '1.5' } },
        ),
      )
      await act(async () =>
        fireEvent.click(screen.getByRole('button', { name: 'Save settings' })),
      )
      assert.match(
        screen.getByRole('alert').textContent,
        /Enter an integer from 1 to 100/,
      )
      assert.equal(writes.length, 0)
      await act(async () =>
        fireEvent.change(
          screen.getByLabelText('Consecutive mismatch threshold'),
          { target: { value: '4' } },
        ),
      )
      await act(async () =>
        fireEvent.click(
          screen.getByRole('button', {
            name: 'Remove Unavailable channel (#901) from allowlist',
          }),
        ),
      )
      await act(async () =>
        fireEvent.click(
          screen.getByRole('checkbox', { name: 'New provider (#902)' }),
        ),
      )
      await act(async () =>
        fireEvent.change(screen.getByLabelText('Search channels'), {
          target: { value: 'New' },
        }),
      )
      await act(async () =>
        fireEvent.click(screen.getByRole('button', { name: 'Search' })),
      )
      assert.equal(
        screen.getByLabelText('Consecutive mismatch threshold').value,
        '4',
      )
      await act(async () =>
        fireEvent.click(screen.getByRole('button', { name: 'Save settings' })),
      )
      assert.deepEqual(writes[0].data, {
        expected_version: 4,
        enabled: true,
        rules: [],
        failure_threshold: 4,
        excluded_channel_ids: [900, 902],
      })
    } finally {
      await act(async () => renderer.unmount())
      container.remove()
    }
  },
)

test(
  'Classic blocks saving when channel options fail',
  { timeout: 15000 },
  async () => {
    channelError = true
    const container = document.createElement('div')
    document.body.appendChild(container)
    const renderer = createRoot(container)
    try {
      await act(async () => renderer.render(React.createElement(Page)))
      assert.match(
        screen.getByRole('alert').textContent,
        /Channels unavailable/,
      )
      assert.equal(
        screen.getByRole('button', { name: 'Save settings' }).disabled,
        true,
      )
    } finally {
      await act(async () => renderer.unmount())
      container.remove()
    }
  },
)

test(
  'Classic 直接填写渠道 ID，非法值禁止保存并可清空',
  { timeout: 15000 },
  async () => {
    configuration = {
      ...configuration,
      excluded_channel_ids: [900],
      excluded_channels: [{ id: 900, name: 'Known', status: 1 }],
    }
    channelOptions = [{ id: 902, name: 'Provider', status: 1 }]
    const container = document.createElement('div')
    document.body.appendChild(container)
    const renderer = createRoot(container)
    try {
      await act(async () => renderer.render(React.createElement(Page)))
      const ids = screen.getByLabelText('Allowlisted channel IDs')
      const save = screen.getByRole('button', { name: 'Save settings' })
      assert.equal(ids.value, '900')
      await act(async () =>
        fireEvent.change(ids, { target: { value: '900，901\n900;902；903' } }),
      )
      assert.equal(
        screen.getByRole('checkbox', { name: 'Provider (#902)' }).checked,
        true,
      )
      await act(async () =>
        fireEvent.click(
          screen.getByRole('checkbox', { name: 'Provider (#902)' }),
        ),
      )
      assert.equal(ids.value, '900\n901\n903')
      await act(async () => fireEvent.click(save))
      assert.deepEqual(writes[0].data.excluded_channel_ids, [900, 901, 903])
      for (const value of ['0', '-1', '1.5', '1e3', '9007199254740992']) {
        await act(async () => fireEvent.change(ids, { target: { value } }))
        assert.equal(save.disabled, true)
        assert.equal(ids.getAttribute('aria-invalid'), 'true')
      }
      await act(async () =>
        fireEvent.change(ids, {
          target: {
            value: Array.from({ length: 1001 }, (_, i) => i + 1).join(','),
          },
        }),
      )
      assert.match(
        screen.getByRole('alert').textContent,
        /Select no more than 1000/,
      )
      await act(async () => fireEvent.change(ids, { target: { value: '' } }))
      assert.equal(save.disabled, false)
      await act(async () => fireEvent.click(save))
      assert.deepEqual(writes.at(-1).data.excluded_channel_ids, [])
    } finally {
      await act(async () => renderer.unmount())
      container.remove()
    }
  },
)

test(
  'Classic displays mismatch progress and legacy records as disabled',
  { timeout: 15000 },
  async () => {
    records = [
      {
        id: 1,
        channel_id: 901,
        channel_name: 'Pending channel',
        consecutive_mismatches: 1,
        failure_threshold: 2,
        channel_disabled: false,
      },
      { id: 2, channel_id: 902, channel_name: 'Legacy channel' },
    ]
    const container = document.createElement('div')
    document.body.appendChild(container)
    const renderer = createRoot(container)
    try {
      await act(async () => renderer.render(React.createElement(Page)))
      const pending = screen.getByRole('row', { name: /Pending channel/ })
      assert.ok(within(pending).getByRole('gridcell', { name: '1 / 2' }))
      assert.ok(within(pending).getByRole('gridcell', { name: 'Not disabled' }))
      const legacy = screen.getByRole('row', { name: /Legacy channel/ })
      assert.ok(within(legacy).getByRole('gridcell', { name: '1 / 1' }))
      assert.ok(
        within(legacy).getByRole('gridcell', { name: 'Channel disabled' }),
      )
    } finally {
      await act(async () => renderer.unmount())
      container.remove()
    }
  },
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
        within(table).getByRole('gridcell', { name: 'Current premium group' }),
      )
      assert.ok(within(table).getByRole('gridcell', { name: 'Official group' }))
      assert.ok(within(table).getByRole('gridcell', { name: 'deleted-code' }))
      assert.equal(within(table).queryByRole('gridcell', { name: 'vip' }), null)
      assert.equal(
        within(table).queryByRole('gridcell', { name: 'Premium group' }),
        null,
      )
    } finally {
      await act(async () => renderer.unmount())
      container.remove()
      records = []
    }
  },
)
