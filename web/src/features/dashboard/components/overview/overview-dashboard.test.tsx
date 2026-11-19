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
import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { getApiKeys } from '@/features/keys/api'
import { useAuthStore, type AuthUser } from '@/stores/auth-store'

import { OverviewDashboard } from './overview-dashboard'

vi.mock('@tanstack/react-router', () => ({
  Link: (props: { to: string; children?: React.ReactNode }) => (
    <a href={props.to}>{props.children}</a>
  ),
}))
vi.mock('@/features/keys/api', () => ({
  getApiKeys: vi.fn(),
  fetchTokenKey: vi.fn(),
}))
vi.mock('@/lib/api', () => ({
  getUserModels: vi.fn(async () => ({ success: true, data: ['gpt-test'] })),
}))
vi.mock('../../hooks/use-status-data', () => ({
  useApiInfo: () => ({ items: [] }),
  useDashboardContentVisibility: () => ({
    apiInfo: false,
    announcements: false,
    faq: false,
    uptimeKuma: false,
  }),
}))
vi.mock('./summary-cards', () => ({ SummaryCards: () => null }))
vi.mock('./performance-health-panel', () => ({
  PerformanceHealthPanel: () => null,
}))
vi.mock('./revenue-panel', () => ({ RevenuePanel: () => null }))
vi.mock('./uptime-panel', () => ({ UptimePanel: () => null }))
vi.mock('./api-info-panel', () => ({ ApiInfoPanel: () => null }))
vi.mock('./announcements-panel', () => ({ AnnouncementsPanel: () => null }))
vi.mock('./faq-panel', () => ({ FAQPanel: () => null }))
vi.mock('@/features/tickets/pending-ticket-card', () => ({
  PendingTicketCard: () => null,
}))

// Node >= 25 exposes its own (disabled) localStorage that shadows jsdom's,
// so give the browser storage contract an explicit in-memory backing.
const storageData = new Map<string, string>()
Object.defineProperty(window, 'localStorage', {
  configurable: true,
  value: {
    getItem: (k: string) => storageData.get(k) ?? null,
    setItem: (k: string, v: string) => void storageData.set(k, String(v)),
    removeItem: (k: string) => void storageData.delete(k),
    clear: () => storageData.clear(),
    key: (i: number) => [...storageData.keys()][i] ?? null,
    get length() {
      return storageData.size
    },
  } satisfies Storage,
})

const STORAGE_KEY = 'dashboard_overview_setup_guide_expanded'
const key = { id: 1, name: 'default', key: 'abcdefghijklmnop', status: 1 }

const newUser: AuthUser = {
  id: 11,
  username: 'new-user',
  role: 1,
  quota: 0,
  used_quota: 0,
  request_count: 0,
}
const doneUser: AuthUser = {
  id: 12,
  username: 'done-user',
  role: 1,
  quota: 500,
  used_quota: 20,
  request_count: 3,
}

type KeysResult = Awaited<ReturnType<typeof getApiKeys>>

function keysOf(items: unknown[]): KeysResult {
  return {
    success: true,
    data: { items, total: items.length, page: 1, page_size: 10 },
  } as KeysResult
}

function mockKeys(byUser: Record<number, unknown[]>) {
  vi.mocked(getApiKeys).mockImplementation(async () => {
    const id = useAuthStore.getState().auth.user?.id ?? 0
    return keysOf(byUser[id] ?? [])
  })
}

function renderOverview(client = new QueryClient()) {
  const view = render(
    <QueryClientProvider client={client}>
      <OverviewDashboard />
    </QueryClientProvider>
  )
  return { client, ...view }
}

function guideVisible() {
  return (
    screen.queryByText('Build on your API gateway in minutes') !== null ||
    screen.queryByText('Show setup guide') !== null
  )
}

describe('OverviewDashboard setup guide', () => {
  beforeEach(() => {
    window.localStorage.clear()
    useAuthStore.getState().auth.setUser(null)
  })
  afterEach(() => {
    window.localStorage.clear()
  })

  it('shows the expanded guide on first visit and keeps collapse across reloads', async () => {
    mockKeys({})
    useAuthStore.getState().auth.setUser(newUser)
    const first = renderOverview()

    await screen.findByText('Build on your API gateway in minutes')
    expect(screen.getByText('Hide setup guide')).toBeInTheDocument()

    await userEvent.click(screen.getByText('Hide setup guide'))
    expect(screen.getByText('Show setup guide')).toBeInTheDocument()
    expect(screen.getByText('Setup progress: 0/3')).toBeInTheDocument()
    expect(window.localStorage.getItem(STORAGE_KEY)).toBe('collapsed')
    for (const target of ['/keys', '/usage-logs', '/pricing']) {
      expect(document.querySelector(`a[href="${target}"]`)).not.toBeNull()
    }
    // Channels stays admin-only.
    expect(document.querySelector('a[href="/channels"]')).toBeNull()
    first.unmount()

    renderOverview()
    await screen.findByText('Show setup guide')
    expect(
      screen.queryByText('Build on your API gateway in minutes')
    ).not.toBeInTheDocument()
  })

  it('removes the guide entirely once setup is complete, ignoring a saved expanded state', async () => {
    window.localStorage.setItem(STORAGE_KEY, 'expanded')
    mockKeys({ [doneUser.id]: [key] })
    useAuthStore.getState().auth.setUser(doneUser)
    renderOverview()

    await waitFor(() => expect(getApiKeys).toHaveBeenCalled())
    await act(async () => {})
    expect(guideVisible()).toBe(false)
    expect(screen.queryByText(/Setup progress/)).not.toBeInTheDocument()
    expect(screen.queryByText('Setup guide complete')).not.toBeInTheDocument()
  })

  it('never renders the guide during a cold start before key data arrives', async () => {
    let resolveKeys: (value: KeysResult) => void = () => undefined
    vi.mocked(getApiKeys).mockImplementation(
      () => new Promise((resolve) => (resolveKeys = resolve))
    )
    useAuthStore.getState().auth.setUser(doneUser)
    renderOverview()

    await waitFor(() => expect(getApiKeys).toHaveBeenCalled())
    expect(guideVisible()).toBe(false)

    await act(async () => resolveKeys(keysOf([key])))
    expect(guideVisible()).toBe(false)
  })

  it('does not flash the guide back on refetch failure or account switch', async () => {
    mockKeys({ [doneUser.id]: [key] })
    useAuthStore.getState().auth.setUser(doneUser)
    const { client } = renderOverview(
      new QueryClient({ defaultOptions: { queries: { retry: false } } })
    )
    await waitFor(() => expect(getApiKeys).toHaveBeenCalledTimes(1))
    await act(async () => {})
    expect(guideVisible()).toBe(false)

    vi.mocked(getApiKeys).mockResolvedValueOnce({
      success: false,
      message: 'boom',
    } as KeysResult)
    await act(async () => {
      await client.refetchQueries({ queryKey: ['dashboard', 'overview'] })
    })
    expect(guideVisible()).toBe(false)

    // Switching to an incomplete account must wait for that account's data,
    // and switching back must not reuse the incomplete account's keys.
    let resolveNew: (value: KeysResult) => void = () => undefined
    vi.mocked(getApiKeys).mockImplementationOnce(
      () => new Promise((resolve) => (resolveNew = resolve))
    )
    await act(async () => useAuthStore.getState().auth.setUser(newUser))
    expect(guideVisible()).toBe(false)
    await act(async () => resolveNew(keysOf([])))
    await screen.findByText('Build on your API gateway in minutes')

    mockKeys({ [doneUser.id]: [key] })
    await act(async () => useAuthStore.getState().auth.setUser(doneUser))
    expect(guideVisible()).toBe(false)
  })
})
