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
    fireEvent.click(screen.getByRole('button', { name: 'Save settings' }))
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith(
        `${base}/config`,
        {
          expected_version: 7,
          enabled: true,
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
    await screen.findByRole('button', { name: 'Save settings' })
    fireEvent.click(screen.getByRole('switch', { name: 'Detection enabled' }))
    fireEvent.click(screen.getByRole('button', { name: 'Save settings' }))
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
    fireEvent.click(screen.getByRole('button', { name: 'Save settings' }))
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
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }))
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
})
