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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState, type ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { SupplierApplicationManagement } from '../admin'
import { SupplierApplicationForm } from '../application-form'
import { MySupplierSubmissions } from '../my-submissions'
import type { SupplierApplicationAdmin } from '../types'

vi.mock('@/lib/api', () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn() },
}))
// Only isolate third-party icon JSON assets; the page under test is real.
vi.mock('@lobehub/icons', () => ({}))
vi.mock('@hugeicons/core-free-icons', () => ({
  Cancel01Icon: [],
  UnfoldMoreIcon: [],
  Tick02Icon: [],
}))

function response<T>(data: T) {
  return { data: { success: true, message: '', data } }
}

function Wrapper(props: { children: ReactNode }) {
  const [client] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: { retry: false },
          mutations: { retry: false },
        },
      })
  )
  return (
    <QueryClientProvider client={client}>{props.children}</QueryClientProvider>
  )
}

const adminApplication: SupplierApplicationAdmin = {
  id: 9,
  user_id: 7,
  username: 'supplier-user',
  site_name: 'Long Supplier Site',
  site_url: 'https://supplier.example.com',
  contact_type: 'wechat',
  contact: 'wx-supplier-001',
  remark: 'Monthly settlement',
  status: 'pending',
  admin_note: 'Internal: price is high',
  handled_by: 0,
  handled_by_name: '',
  handled_at: 0,
  version: 3,
  created_at: 1792634400,
  updated_at: 1792634400,
  groups: [
    {
      id: 1,
      group_name: 'Official direct',
      platform: 'openai',
      ratio: 0.35,
      models: ['gpt-5'],
      description: 'concurrency 50',
    },
    {
      id: 2,
      group_name: 'Backup',
      platform: 'anthropic',
      ratio: 1.2,
      models: [],
      description: '',
    },
  ],
}

beforeEach(() => {
  useAuthStore
    .getState()
    .auth.setUser({ id: 7, username: 'supplier-user', role: 1 })
})
afterEach(() => {
  useAuthStore.getState().auth.reset()
  vi.resetAllMocks()
})

async function fillMinimalForm(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText('Site name'), '  My Site  ')
  await user.type(screen.getByLabelText('Contact'), 'wx-001')
  await user.type(screen.getByLabelText('Group name'), 'Direct')
}

describe('supplier application form', { timeout: 30_000 }, () => {
  it('blocks an empty submission with field errors and sends nothing', async () => {
    const user = userEvent.setup()
    render(<SupplierApplicationForm onSubmitted={vi.fn()} />, {
      wrapper: Wrapper,
    })
    await user.click(screen.getByRole('button', { name: 'Submit application' }))
    expect(await screen.findByText('Enter a site name.')).toBeVisible()
    expect(screen.getByText('Enter your contact details.')).toBeVisible()
    expect(screen.getByText('Enter a group name.')).toBeVisible()
    expect(api.post).not.toHaveBeenCalled()
  })

  it('validates URL scheme, URL credentials and ratio range/precision', async () => {
    const user = userEvent.setup()
    render(<SupplierApplicationForm onSubmitted={vi.fn()} />, {
      wrapper: Wrapper,
    })
    await fillMinimalForm(user)
    for (const bad of [
      'ftp://example.com',
      'javascript:alert(1)',
      'https://u:p@example.com',
      'not a url',
    ]) {
      await user.clear(screen.getByLabelText(/Site URL/))
      await user.type(screen.getByLabelText(/Site URL/), bad)
      await user.click(
        screen.getByRole('button', { name: 'Submit application' })
      )
      expect(
        await screen.findByText('Enter a valid http:// or https:// address.')
      ).toBeVisible()
    }
    await user.clear(screen.getByLabelText(/Site URL/))
    for (const bad of ['0', '-1', 'abc', '1000.01', '0.12345', '']) {
      await user.clear(screen.getByLabelText('Ratio'))
      if (bad) await user.type(screen.getByLabelText('Ratio'), bad)
      await user.click(
        screen.getByRole('button', { name: 'Submit application' })
      )
      await waitFor(() =>
        expect(
          screen.getByLabelText('Ratio').getAttribute('aria-invalid')
        ).toBe('true')
      )
    }
    expect(api.post).not.toHaveBeenCalled()
  })

  it('adds and removes groups within the 1-20 limit and shows the count', async () => {
    const user = userEvent.setup()
    render(<SupplierApplicationForm onSubmitted={vi.fn()} />, {
      wrapper: Wrapper,
    })
    expect(screen.getByText('1 / 20 groups')).toBeVisible()
    expect(
      screen.getByRole('button', { name: 'Remove group 1' })
    ).toBeDisabled()
    const add = screen.getByRole('button', { name: 'Add group' })
    for (let index = 1; index < 20; index += 1) await user.click(add)
    expect(screen.getByText('20 / 20 groups')).toBeVisible()
    expect(add).toBeDisabled()
    await user.click(screen.getByRole('button', { name: 'Remove group 20' }))
    expect(screen.getByText('19 / 20 groups')).toBeVisible()
    expect(add).toBeEnabled()
  })

  it('adds model tags with Enter without submitting, rejects duplicates, and removes tags', async () => {
    const user = userEvent.setup()
    render(<SupplierApplicationForm onSubmitted={vi.fn()} />, {
      wrapper: Wrapper,
    })
    const input = screen.getByPlaceholderText(
      'Type a model name and press Enter'
    )
    await user.type(input, 'gpt-5{Enter}')
    await user.type(input, 'claude-opus{Enter}')
    await user.type(input, 'gpt-5{Enter}')
    expect(screen.getByText('This model was already added.')).toBeVisible()
    const tags = screen.getByRole('list', { name: 'Main models' })
    expect(within(tags).getAllByRole('listitem')).toHaveLength(2)
    await user.click(screen.getByRole('button', { name: 'Remove model gpt-5' }))
    expect(within(tags).getAllByRole('listitem')).toHaveLength(1)
    expect(api.post).not.toHaveBeenCalled()
  })

  it('posts a trimmed payload with numeric ratios, resets, and reports success once', async () => {
    const user = userEvent.setup()
    const onSubmitted = vi.fn()
    vi.mocked(api.post).mockResolvedValue(response({ id: 1 }))
    render(<SupplierApplicationForm onSubmitted={onSubmitted} />, {
      wrapper: Wrapper,
    })
    await fillMinimalForm(user)
    await user.clear(screen.getByLabelText('Ratio'))
    await user.type(screen.getByLabelText('Ratio'), '0.35')
    await user.type(
      screen.getByPlaceholderText('Type a model name and press Enter'),
      'gpt-5{Enter}'
    )
    await user.click(screen.getByRole('button', { name: 'Submit application' }))
    await waitFor(() => expect(onSubmitted).toHaveBeenCalledTimes(1))
    expect(api.post).toHaveBeenCalledTimes(1)
    const [path, payload] = vi.mocked(api.post).mock.calls[0]
    expect(path).toBe('/api/supplier-applications')
    expect(payload).toEqual({
      site_name: 'My Site',
      site_url: '',
      contact_type: 'wechat',
      contact: 'wx-001',
      remark: '',
      groups: [
        {
          group_name: 'Direct',
          platform: 'openai',
          ratio: 0.35,
          models: ['gpt-5'],
          description: '',
        },
      ],
    })
    expect(screen.getByLabelText('Site name')).toHaveValue('')
  })

  it('sends one request when the submit button is clicked repeatedly', async () => {
    const user = userEvent.setup()
    let resolvePost: (value: unknown) => void = () => undefined
    vi.mocked(api.post).mockImplementation(
      () => new Promise((resolve) => (resolvePost = resolve))
    )
    render(<SupplierApplicationForm onSubmitted={vi.fn()} />, {
      wrapper: Wrapper,
    })
    await fillMinimalForm(user)
    const submit = screen.getByRole('button', { name: 'Submit application' })
    await user.dblClick(submit)
    await user.click(screen.getByRole('button', { name: 'Submitting...' }))
    expect(api.post).toHaveBeenCalledTimes(1)
    resolvePost(response({ id: 1 }))
    await waitFor(() =>
      expect(screen.getByLabelText('Site name')).toHaveValue('')
    )
  })

  it('keeps the entered values and explains a duplicate or failed submission', async () => {
    const user = userEvent.setup()
    vi.mocked(api.post)
      .mockRejectedValueOnce(new Error('offline'))
      .mockRejectedValueOnce({
        isAxiosError: true,
        response: {
          status: 409,
          data: {
            success: false,
            message: 'The same supplier application was already submitted',
          },
        },
      })
    render(<SupplierApplicationForm onSubmitted={vi.fn()} />, {
      wrapper: Wrapper,
    })
    await fillMinimalForm(user)
    await user.click(screen.getByRole('button', { name: 'Submit application' }))
    expect(
      await screen.findByText('Could not submit the application.')
    ).toBeVisible()
    expect(screen.getByLabelText('Site name')).toHaveValue('  My Site  ')
    await user.click(screen.getByRole('button', { name: 'Submit application' }))
    expect(
      await screen.findByText(
        'The same supplier application was already submitted'
      )
    ).toBeVisible()
    expect(screen.getByLabelText('Contact')).toHaveValue('wx-001')
    expect(api.post).toHaveBeenCalledTimes(2)
  })
})

describe('my submissions', { timeout: 30_000 }, () => {
  it('shows the empty state', async () => {
    vi.mocked(api.get).mockResolvedValue(
      response({ page: 1, page_size: 10, total: 0, items: [] })
    )
    render(<MySupplierSubmissions />, { wrapper: Wrapper })
    expect(await screen.findByText('No submissions yet')).toBeVisible()
    expect(api.get).toHaveBeenCalledWith(
      '/api/supplier-applications',
      expect.objectContaining({ params: { p: 1, page_size: 10 } })
    )
  })

  it('lists quotes and never renders internal fields', async () => {
    const user = userEvent.setup()
    vi.mocked(api.get).mockResolvedValue(
      response({
        page: 1,
        page_size: 10,
        total: 1,
        // Even if a bad server leaked internal fields, the page must not show them.
        items: [adminApplication],
      })
    )
    render(<MySupplierSubmissions />, { wrapper: Wrapper })
    expect(await screen.findByText('#9 Long Supplier Site')).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'View details' }))
    expect(screen.getByText('Official direct')).toBeVisible()
    expect(screen.getByText('gpt-5')).toBeVisible()
    expect(screen.getByText(/×0.35/)).toBeVisible()
    expect(screen.queryByText(/Internal: price is high/)).toBeNull()
    expect(screen.queryByText(/supplier-user/)).toBeNull()
  })

  it('records a negotiation in order and never shows the internal note', async () => {
    const user = userEvent.setup()
    vi.mocked(api.get).mockResolvedValue(
      response({
        page: 1,
        page_size: 10,
        total: 1,
        items: [
          {
            ...adminApplication,
            messages: [
              {
                id: 4,
                author_id: 1,
                author_role: 'admin',
                author_name: 'reviewer',
                type: 'offer',
                body: 'Trial price for this month',
                created_at: 1792634500,
              },
            ],
          },
        ],
      })
    )
    vi.mocked(api.post).mockResolvedValue(
      response({
        id: 5,
        author_id: 7,
        author_role: 'applicant',
        author_name: 'supplier-user',
        type: 'negotiation',
        body: 'Can do 0.8',
        group_name: 'Discount group',
        ratio: 0.8,
        created_at: 1792634600,
      })
    )
    render(<MySupplierSubmissions />, { wrapper: Wrapper })
    await user.click(
      await screen.findByRole('button', { name: 'View details' })
    )
    expect(screen.getByText('Trial price for this month')).toBeVisible()
    expect(screen.queryByText('Internal: price is high')).toBeNull()
    await user.selectOptions(
      screen.getByLabelText('Message type'),
      'negotiation'
    )
    await user.type(screen.getByLabelText('Group name'), 'Discount group')
    await user.type(screen.getByLabelText('Suggested ratio'), '0.8')
    await user.type(screen.getByLabelText('Negotiation notes'), 'Can do 0.8')
    await user.click(screen.getByRole('button', { name: 'Send message' }))
    await waitFor(() => expect(api.post).toHaveBeenCalledTimes(1))
    expect(vi.mocked(api.post).mock.calls[0]?.[0]).toBe(
      '/api/supplier-applications/9/messages'
    )
    expect(vi.mocked(api.post).mock.calls[0]?.[1]).toEqual({
      type: 'negotiation',
      body: 'Can do 0.8',
      group_name: 'Discount group',
      ratio: 0.8,
    })
    expect(
      screen.getByText(/Discount group · Suggested ratio ×0.8/)
    ).toBeVisible()
  })

  it('shows the message count from the list and hides it when there are none', async () => {
    const user = userEvent.setup()
    vi.mocked(api.get).mockResolvedValue(
      response({
        page: 1,
        page_size: 10,
        total: 2,
        items: [
          {
            ...adminApplication,
            id: 9,
            site_name: 'Quiet Site',
            message_count: 0,
            messages: [],
          },
          {
            ...adminApplication,
            id: 10,
            site_name: 'Busy Site',
            message_count: 2,
            messages: [],
          },
        ],
      })
    )
    render(<MySupplierSubmissions />, { wrapper: Wrapper })
    expect(await screen.findByRole('img', { name: '2 messages' })).toBeVisible()
    expect(screen.getByText('#10 Busy Site')).toBeVisible()
    expect(screen.getByText('#9 Quiet Site')).toBeVisible()
    expect(screen.queryByRole('img', { name: '0 messages' })).toBeNull()
    expect(screen.queryByText('Internal: price is high')).toBeNull()
    expect(api.get).toHaveBeenCalledTimes(1)
    expect(String(vi.mocked(api.get).mock.calls[0]?.[0])).not.toContain(
      '/messages'
    )
    const quiet = screen.getByText('#9 Quiet Site').closest('li')
    if (!quiet) throw new Error('quiet submission not found')
    await user.click(
      within(quiet).getByRole('button', { name: 'View details' })
    )
    expect(within(quiet).queryByLabelText('Internal note')).toBeNull()
    expect(
      within(quiet).queryByRole('heading', { name: 'Processing' })
    ).toBeNull()
    expect(api.get).toHaveBeenCalledTimes(1)
  })
})

describe('supplier application management', { timeout: 30_000 }, () => {
  function grant(write: boolean, read = true) {
    useAuthStore.getState().auth.setUser({
      id: 1,
      username: 'admin',
      role: 10,
      permissions: {
        admin_permissions: { supplier_applications: { read, write } },
      },
    } as never)
  }

  it('does not request data without the read permission', () => {
    grant(false, false)
    render(<SupplierApplicationManagement />, { wrapper: Wrapper })
    expect(
      screen.getByText(
        'You do not have permission to view supplier applications.'
      )
    ).toBeVisible()
    expect(api.get).not.toHaveBeenCalled()
  })

  it('searches, opens details and saves status and note with the version', async () => {
    grant(true)
    const user = userEvent.setup()
    vi.mocked(api.get).mockImplementation(async (url) =>
      String(url).endsWith('/9')
        ? response(adminApplication)
        : response({
            page: 1,
            page_size: 20,
            total: 1,
            items: [adminApplication],
          })
    )
    vi.mocked(api.put).mockResolvedValue(
      response({ ...adminApplication, status: 'contacted', version: 4 })
    )
    render(<SupplierApplicationManagement />, { wrapper: Wrapper })
    await user.click(
      await screen.findByRole('button', { name: '#9 Long Supplier Site' })
    )
    expect(await screen.findByText('Official direct')).toBeVisible()
    expect(screen.getByDisplayValue('Internal: price is high')).toBeVisible()
    // The sheet's status select is the last "Status" control (the filter comes first).
    const statusSelect = screen.getAllByLabelText('Status').at(-1)
    if (!statusSelect) throw new Error('status select not found')
    await user.selectOptions(statusSelect, 'contacted')
    const note = screen.getByLabelText('Internal note')
    await user.clear(note)
    await user.type(note, 'Called, waiting for quote')
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(api.put).toHaveBeenCalledTimes(1))
    expect(vi.mocked(api.put).mock.calls[0][0]).toBe(
      '/api/supplier-application-admin/9'
    )
    expect(vi.mocked(api.put).mock.calls[0][1]).toEqual({
      status: 'contacted',
      admin_note: 'Called, waiting for quote',
      version: 3,
    })
  })

  it('hides the save control without the write permission', async () => {
    grant(false)
    const user = userEvent.setup()
    vi.mocked(api.get).mockImplementation(async (url) =>
      String(url).endsWith('/9')
        ? response(adminApplication)
        : response({
            page: 1,
            page_size: 20,
            total: 1,
            items: [adminApplication],
          })
    )
    render(<SupplierApplicationManagement />, { wrapper: Wrapper })
    await user.click(
      await screen.findByRole('button', { name: '#9 Long Supplier Site' })
    )
    expect(await screen.findByText('Official direct')).toBeVisible()
    expect(screen.queryByRole('button', { name: 'Save' })).toBeNull()
    expect(screen.getByLabelText('Internal note')).toBeDisabled()
    expect(screen.queryByRole('button', { name: 'Send message' })).toBeNull()
    expect(screen.queryByLabelText('Message type')).toBeNull()
    expect(screen.getByText('No messages yet')).toBeVisible()
  })

  it('marks only applications with messages and puts processing before the conversation', async () => {
    grant(true)
    const user = userEvent.setup()
    const quiet = {
      ...adminApplication,
      id: 9,
      site_name: 'Quiet Site',
      message_count: 0,
    }
    const busy = {
      ...adminApplication,
      id: 11,
      site_name: 'Busy Site',
      message_count: 3,
    }
    vi.mocked(api.get).mockImplementation(async (url) =>
      String(url).endsWith('/11')
        ? response(busy)
        : response({
            page: 1,
            page_size: 20,
            total: 2,
            items: [quiet, busy],
          })
    )
    render(<SupplierApplicationManagement />, { wrapper: Wrapper })
    expect(await screen.findByRole('img', { name: '3 messages' })).toBeVisible()
    expect(screen.getByRole('button', { name: '#9 Quiet Site' })).toBeVisible()
    expect(screen.queryByRole('img', { name: '0 messages' })).toBeNull()
    expect(api.get).toHaveBeenCalledTimes(1)
    expect(String(vi.mocked(api.get).mock.calls[0]?.[0])).not.toContain(
      '/messages'
    )
    await user.click(screen.getByRole('button', { name: '#11 Busy Site' }))
    const dialog = await screen.findByRole('dialog')
    const processing = within(dialog).getByRole('heading', {
      name: 'Processing',
    })
    const conversation = within(dialog).getByRole('heading', {
      name: 'Conversation',
    })
    const before = (earlier: HTMLElement, later: HTMLElement) => {
      expect(
        earlier.compareDocumentPosition(later) &
          Node.DOCUMENT_POSITION_FOLLOWING
      ).toBeTruthy()
    }
    before(processing, within(dialog).getByText('Applicant'))
    before(processing, within(dialog).getByText('Site URL'))
    before(
      processing,
      within(dialog).getByRole('heading', { name: /Group quotes/ })
    )
    before(processing, conversation)
    expect(within(dialog).getByLabelText('Internal note')).toBeVisible()
    const messageFetches = vi
      .mocked(api.get)
      .mock.calls.filter((call) => String(call[0]).includes('/messages'))
    expect(messageFetches).toEqual([])
  })
})
