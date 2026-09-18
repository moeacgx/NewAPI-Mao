import { readFile } from 'node:fs/promises'
import path from 'node:path'

import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import i18next from 'i18next'
import type { ComponentType } from 'react'
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { registerDefaultNativeExtensionSdk } from '../native-sdk'

const base = '/api/extensions/upstream-model-guard'
let Page: ComponentType
const configuration = {
  config_version: 7,
  enabled: false,
  rules: [],
}
const ok = (data: unknown) => ({ data: { success: true, data } })

beforeAll(async () => {
  registerDefaultNativeExtensionSdk()
  const source = await readFile(
    path.resolve(
      '../extensions/upstream-model-guard/public/native/default.mjs'
    ),
    'utf8'
  )
  const entry = await import(
    /* @vite-ignore */ `data:text/javascript;base64,${Buffer.from(source).toString('base64')}`
  )
  Page = entry.default as ComponentType
})

beforeEach(async () => {
  await i18next.changeLanguage('en')
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === `${base}/config`) return ok(configuration)
    if (url === `${base}/groups`) {
      return ok([
        { id: 1, code: 'default', name: 'Default group' },
        { id: 2, code: 'vip', name: 'VIP group' },
      ])
    }
    return ok({ items: [], total: 0, page: 1, page_size: 20 })
  })
  vi.spyOn(api, 'put').mockResolvedValue(
    ok({ config_version: 8, enabled: true, rules: [] })
  )
})

describe('upstream model guard native page', () => {
  it('saves multiple checked groups and exact upstream names with the loaded version', async () => {
    render(<Page />)
    await screen.findByRole('button', { name: 'Add rule' })
    fireEvent.click(screen.getByRole('switch', { name: 'Detection enabled' }))
    fireEvent.click(screen.getByRole('button', { name: 'Add rule' }))
    expect(screen.getByLabelText('Consecutive mismatch threshold')).toHaveValue(
      2
    )
    const rule = screen.getByRole('group', { name: 'Rule 1' })
    fireEvent.click(
      within(rule).getByRole('checkbox', { name: 'Default group' })
    )
    fireEvent.click(within(rule).getByRole('checkbox', { name: 'VIP group' }))
    fireEvent.change(within(rule).getByLabelText('Request model'), {
      target: { value: ' gpt-test ' },
    })
    fireEvent.change(within(rule).getByLabelText('Allowed upstream models'), {
      target: { value: ' provider-A\r\nprovider-B\nprovider-A\n' },
    })
    const save = screen.getByRole('button', { name: 'Save settings' })
    await waitFor(() => expect(save).toBeEnabled())
    fireEvent.click(save)
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith(
        `${base}/config`,
        {
          expected_version: 7,
          enabled: true,
          failure_threshold: 2,
          excluded_channel_ids: [],
          rules: [
            {
              enabled: true,
              group_codes: ['default', 'vip'],
              model: 'gpt-test',
              upstream_models: ['provider-A', 'provider-B'],
            },
          ],
        },
        { skipErrorHandler: true }
      )
    )
    expect(await screen.findByRole('status')).toHaveTextContent(
      'Model guard settings saved'
    )
  })

  it('removes only the selected rule and keeps the remaining draft', async () => {
    render(<Page />)
    const add = await screen.findByRole('button', { name: 'Add rule' })
    fireEvent.click(add)
    fireEvent.click(add)
    const rules = screen.getAllByRole('group', { name: /Rule \d/ })
    fireEvent.change(within(rules[1]).getByLabelText('Request model'), {
      target: { value: 'keep-this-model' },
    })
    fireEvent.click(
      within(rules[0]).getByRole('button', { name: 'Remove rule' })
    )
    expect(screen.getAllByRole('group', { name: /Rule \d/ })).toHaveLength(1)
    expect(screen.getByLabelText('Request model')).toHaveValue(
      'keep-this-model'
    )
  })

  it('shows the stable code only when a selected group has no display name', async () => {
    vi.mocked(api.get).mockImplementation(async (url) => {
      if (url === `${base}/config`) {
        return ok({
          ...configuration,
          rules: [
            {
              enabled: true,
              group_codes: ['archived-code'],
              model: 'gpt-test',
              upstream_models: ['provider-A'],
            },
          ],
        })
      }
      if (url === `${base}/groups`) {
        return ok([{ id: 1, code: 'unnamed-code', name: '' }])
      }
      return ok({ items: [], total: 0, page: 1, page_size: 20 })
    })
    render(<Page />)
    expect(
      await screen.findByRole('checkbox', { name: 'archived-code' })
    ).toBeChecked()
    expect(
      screen.getByRole('checkbox', { name: 'unnamed-code' })
    ).not.toBeChecked()
  })

  it('shows record group names before cached names and uses codes only as a last fallback', async () => {
    vi.mocked(api.get).mockImplementation(async (url) => {
      if (url === `${base}/config`) return ok(configuration)
      if (url === `${base}/groups`) {
        return ok([
          { id: 1, code: 'vip', name: 'Premium group' },
          { id: 2, code: 'official', name: 'Official group' },
        ])
      }
      return ok({
        items: [
          {
            id: 1,
            channel_id: 906,
            channel_name: 'Channel A',
            group: 'vip',
            group_name: 'Current premium group',
          },
          {
            id: 2,
            channel_id: 907,
            channel_name: 'Channel B',
            group: 'official',
          },
          {
            id: 3,
            channel_id: 908,
            channel_name: 'Channel C',
            group: 'deleted-code',
          },
        ],
        total: 3,
        page: 1,
        page_size: 20,
      })
    })
    render(<Page />)
    const table = await screen.findByRole('table')
    expect(
      await within(table).findByRole('cell', { name: 'Current premium group' })
    ).toBeInTheDocument()
    expect(
      await within(table).findByRole('cell', { name: 'Official group' })
    ).toBeInTheDocument()
    expect(
      within(table).getByRole('cell', { name: 'deleted-code' })
    ).toBeInTheDocument()
    expect(
      within(table).queryByRole('cell', { name: 'vip' })
    ).not.toBeInTheDocument()
    expect(
      within(table).queryByRole('cell', { name: 'Premium group' })
    ).not.toBeInTheDocument()
  })

  it('keeps an edited draft when the server rejects a stale configuration', async () => {
    vi.mocked(api.put).mockRejectedValue({
      response: { status: 409, data: { message: 'Changed elsewhere' } },
    })
    render(<Page />)
    const save = await screen.findByRole('button', { name: 'Save settings' })
    await waitFor(() => expect(save).toBeEnabled())
    fireEvent.click(screen.getByRole('switch', { name: 'Detection enabled' }))
    fireEvent.click(save)
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Settings changed elsewhere. Reload before saving again.'
    )
    expect(
      screen.getByRole('switch', { name: 'Detection enabled' })
    ).toBeChecked()
    expect(screen.getByRole('button', { name: 'Save settings' })).toBeDisabled()
    expect(
      screen.getByRole('button', { name: 'Reload settings' })
    ).toBeEnabled()
  })

  it('blocks configuration edits if groups fail while keeping records visible', async () => {
    vi.mocked(api.get).mockImplementation(async (url) => {
      if (url === `${base}/config`) return ok(configuration)
      if (url === `${base}/groups`) throw new Error('Groups unavailable')
      return ok({
        items: [
          {
            id: 1,
            channel_id: 906,
            channel_name: 'Channel A',
            group: 'vip',
            requested_model: 'gpt-test',
            expected_upstream_models: ['provider-A'],
            actual_upstream_model: 'provider-B',
            created_at: 1758150000,
          },
        ],
        total: 1,
        page: 1,
        page_size: 20,
      })
    })
    render(<Page />)
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Groups unavailable'
    )
    expect(
      screen.queryByRole('button', { name: 'Save settings' })
    ).not.toBeInTheDocument()
    expect(await screen.findByText('Channel A')).toBeInTheDocument()
    expect(screen.getByText('provider-B')).toBeInTheDocument()
  })

  it('rejects an incomplete rule before sending a configuration update', async () => {
    render(<Page />)
    fireEvent.click(await screen.findByRole('button', { name: 'Add rule' }))
    const save = screen.getByRole('button', { name: 'Save settings' })
    await waitFor(() => expect(save).toBeEnabled())
    fireEvent.click(save)
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Complete groups and model names for rule 1.'
    )
    expect(api.put).not.toHaveBeenCalled()
  })

  it('registers bundled translations without adding keys to the host namespace', async () => {
    await i18next.changeLanguage('zhCN')
    render(<Page />)
    expect(
      await screen.findByRole('heading', { name: '上游模型校验' })
    ).toBeInTheDocument()
    expect(
      await screen.findByRole('button', { name: '添加规则' })
    ).toBeInTheDocument()
    expect(
      i18next.getResource('zhCN', 'translation', 'Upstream model guard')
    ).toBeUndefined()
    expect(
      i18next.getResource(
        'zhCN',
        'upstream-model-guard',
        'Upstream model guard'
      )
    ).toBe('上游模型校验')
  })

  it('loads the next records page without replacing an unsaved rule draft', async () => {
    vi.mocked(api.get).mockImplementation(async (url) => {
      if (url === `${base}/config`) return ok(configuration)
      if (url === `${base}/groups`) {
        return ok([{ id: 1, code: 'default', name: 'Default group' }])
      }
      return ok({
        items: [],
        total: 21,
        page: 1,
        page_size: 20,
      })
    })
    render(<Page />)
    fireEvent.click(await screen.findByRole('button', { name: 'Add rule' }))
    fireEvent.change(screen.getByLabelText('Request model'), {
      target: { value: 'unsaved-model' },
    })
    fireEvent.click(
      within(
        screen.getByRole('region', { name: 'Model mismatch records' })
      ).getByRole('button', { name: 'Next page' })
    )
    await waitFor(() =>
      expect(api.get).toHaveBeenCalledWith(`${base}/records`, {
        params: { page: 2, page_size: 20 },
        skipErrorHandler: true,
      })
    )
    expect(screen.getByLabelText('Request model')).toHaveValue('unsaved-model')
    expect(await screen.findByText('2 / 2')).toBeInTheDocument()
    expect(
      screen.getByRole('link', { name: 'Notification Center' })
    ).toHaveAttribute('href', '/notification-center')
  })

  it.each(['0', '101', '1.5', ''])(
    'rejects invalid mismatch threshold %s without saving',
    async (value) => {
      render(<Page />)
      await screen.findByRole('button', { name: 'Save settings' })
      fireEvent.change(
        screen.getByLabelText('Consecutive mismatch threshold'),
        { target: { value } }
      )
      await waitFor(() =>
        expect(
          screen.getByRole('button', { name: 'Save settings' })
        ).toBeEnabled()
      )
      fireEvent.click(screen.getByRole('button', { name: 'Save settings' }))
      expect(screen.getByRole('alert')).toHaveTextContent(
        'Enter an integer from 1 to 100.'
      )
      expect(api.put).not.toHaveBeenCalled()
    }
  )

  it.each([1, 100])(
    'saves the allowed threshold boundary %s',
    async (threshold) => {
      render(<Page />)
      await screen.findByRole('button', { name: 'Save settings' })
      fireEvent.change(
        screen.getByLabelText('Consecutive mismatch threshold'),
        { target: { value: String(threshold) } }
      )
      await waitFor(() =>
        expect(
          screen.getByRole('button', { name: 'Save settings' })
        ).toBeEnabled()
      )
      fireEvent.click(screen.getByRole('button', { name: 'Save settings' }))
      await waitFor(() =>
        expect(api.put).toHaveBeenCalledWith(
          `${base}/config`,
          expect.objectContaining({
            failure_threshold: threshold,
            excluded_channel_ids: [],
          }),
          { skipErrorHandler: true }
        )
      )
    }
  )

  it('keeps selected channel identities and edited rules across allowlist search and pagination', async () => {
    vi.mocked(api.get).mockImplementation(async (url, options) => {
      if (url === `${base}/config`) {
        return ok({
          ...configuration,
          failure_threshold: 3,
          excluded_channel_ids: [900, 901],
          excluded_channels: [
            { id: 900, name: 'Existing provider', status: 1 },
            { id: 901, name: '', status: 0 },
          ],
        })
      }
      if (url === `${base}/groups`) {
        return ok([{ id: 2, code: 'vip', name: 'VIP group' }])
      }
      if (url === `${base}/channels`) {
        const params = options?.params as { keyword: string; page: number }
        const item =
          params.page === 2
            ? { id: 903, name: 'Provider second page', status: 1 }
            : { id: 902, name: 'Provider first page', status: 1 }
        return ok({
          items: [item],
          total: 51,
          page: params.page,
          page_size: 50,
        })
      }
      return ok({ items: [], total: 0, page: 1, page_size: 20 })
    })
    render(<Page />)
    await screen.findByText('Existing provider (#900)')
    expect(screen.getByText('Unavailable channel (#901)')).toBeInTheDocument()
    fireEvent.click(
      screen.getByRole('button', {
        name: 'Remove Unavailable channel (#901) from allowlist',
      })
    )
    fireEvent.change(screen.getByLabelText('Consecutive mismatch threshold'), {
      target: { value: '4' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Add rule' }))
    const rule = screen.getByRole('group', { name: 'Rule 1' })
    fireEvent.click(within(rule).getByRole('checkbox', { name: 'VIP group' }))
    fireEvent.change(within(rule).getByLabelText('Request model'), {
      target: { value: 'unsaved-model' },
    })
    fireEvent.change(within(rule).getByLabelText('Allowed upstream models'), {
      target: { value: 'provider-A' },
    })
    fireEvent.change(screen.getByLabelText('Search channels'), {
      target: { value: 'Provider' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Search' }))
    const choices = screen.getByRole('group', { name: 'Available channels' })
    fireEvent.click(
      await within(choices).findByRole('checkbox', {
        name: 'Provider first page (#902)',
      })
    )
    fireEvent.click(within(choices).getByRole('button', { name: 'Next page' }))
    fireEvent.click(
      await within(choices).findByRole('checkbox', {
        name: 'Provider second page (#903)',
      })
    )
    expect(screen.getByLabelText('Request model')).toHaveValue('unsaved-model')
    expect(screen.getByLabelText('Consecutive mismatch threshold')).toHaveValue(
      4
    )
    const save = screen.getByRole('button', { name: 'Save settings' })
    await waitFor(() => expect(save).toBeEnabled())
    fireEvent.click(save)
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith(
        `${base}/config`,
        {
          expected_version: 7,
          enabled: false,
          failure_threshold: 4,
          excluded_channel_ids: [900, 902, 903],
          rules: [
            {
              enabled: true,
              group_codes: ['vip'],
              model: 'unsaved-model',
              upstream_models: ['provider-A'],
            },
          ],
        },
        { skipErrorHandler: true }
      )
    )
    expect(api.get).toHaveBeenCalledWith(`${base}/channels`, {
      params: { keyword: 'Provider', page: 2, page_size: 50 },
      skipErrorHandler: true,
    })
  })

  it('blocks saving when allowlist options fail and preserves selected channels on retry', async () => {
    let failed = true
    vi.mocked(api.get).mockImplementation(async (url) => {
      if (url === `${base}/config`) {
        return ok({
          ...configuration,
          excluded_channel_ids: [900],
          excluded_channels: [
            { id: 900, name: 'Existing provider', status: 1 },
          ],
        })
      }
      if (url === `${base}/groups`) return ok([])
      if (url === `${base}/channels` && failed) {
        throw new Error('Channels unavailable')
      }
      return ok({ items: [], total: 0, page: 1, page_size: 50 })
    })
    render(<Page />)
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Channels unavailable'
    )
    expect(screen.getByRole('button', { name: 'Save settings' })).toBeDisabled()
    expect(screen.getByText('Existing provider (#900)')).toBeInTheDocument()
    failed = false
    fireEvent.click(
      screen.getByRole('button', { name: 'Retry channel search' })
    )
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Save settings' })
      ).toBeEnabled()
    )
    fireEvent.click(screen.getByRole('button', { name: 'Save settings' }))
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith(
        `${base}/config`,
        expect.objectContaining({
          failure_threshold: 2,
          excluded_channel_ids: [900],
        }),
        { skipErrorHandler: true }
      )
    )
  })

  it('distinguishes below-threshold records from disabled channels and legacy records', async () => {
    vi.mocked(api.get).mockImplementation(async (url) => {
      if (url === `${base}/config`) return ok(configuration)
      if (url === `${base}/groups`) return ok([])
      if (url === `${base}/channels`) {
        return ok({ items: [], total: 0, page: 1, page_size: 50 })
      }
      return ok({
        items: [
          {
            id: 1,
            channel_id: 901,
            channel_name: 'Pending channel',
            consecutive_mismatches: 1,
            failure_threshold: 2,
            channel_disabled: false,
          },
          {
            id: 2,
            channel_id: 902,
            channel_name: 'Disabled channel',
            consecutive_mismatches: 2,
            failure_threshold: 2,
            channel_disabled: true,
          },
          { id: 3, channel_id: 903, channel_name: 'Legacy channel' },
        ],
        total: 3,
        page: 1,
        page_size: 20,
      })
    })
    render(<Page />)
    const row = await screen.findByRole('row', { name: /Pending channel/ })
    expect(within(row).getByRole('cell', { name: '1 / 2' })).toBeInTheDocument()
    expect(
      within(row).getByRole('cell', { name: 'Not disabled' })
    ).toBeInTheDocument()
    const disabled = screen.getByRole('row', { name: /Disabled channel/ })
    expect(
      within(disabled).getByRole('cell', { name: '2 / 2' })
    ).toBeInTheDocument()
    const legacy = screen.getByRole('row', { name: /Legacy channel/ })
    expect(
      within(legacy).getByRole('cell', { name: '1 / 1' })
    ).toBeInTheDocument()
    expect(
      within(legacy).getByRole('cell', { name: 'Channel disabled' })
    ).toBeInTheDocument()
  })
})
