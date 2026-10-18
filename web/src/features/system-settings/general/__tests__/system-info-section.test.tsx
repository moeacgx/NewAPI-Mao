import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { toast } from 'sonner'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { SystemBrand } from '@/components/layout/components/system-brand'
import { useStatus } from '@/hooks/use-status'
import { api } from '@/lib/api'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { SettingsPageProvider } from '../../components/settings-page-context'
import { SystemInfoSection } from '../system-info-section'

afterEach(() => {
  cleanup()
  document.head.innerHTML = ''
  localStorage.clear()
})

const defaultValues = {
  theme: { frontend: 'default' as const },
  SystemName: 'Configured site',
  SystemDescription: 'Initial description',
  ServerAddress: '',
  Logo: '',
  Footer: '',
  About: '',
  HomePageContent: '',
  legal: { user_agreement: '', privacy_policy: '' },
}

const initialStatus = {
  system_name: 'Configured site',
  system_description: 'Initial description',
  logo: '/logo.png',
  quota_per_unit: 500000,
}

function StatusConsumer() {
  const { status } = useStatus()
  const systemName = useSystemConfigStore((state) => state.config.systemName)
  return (
    <>
      <SystemBrand variant='inline' />
      <output aria-label='Status description'>
        {String(status?.system_description ?? '')}
      </output>
      <output aria-label='Config system name'>{systemName}</output>
    </>
  )
}

function SystemInfoFixture() {
  const [actions, setActions] = useState<HTMLDivElement | null>(null)
  const [titleStatus, setTitleStatus] = useState<HTMLSpanElement | null>(null)

  return (
    <>
      <StatusConsumer />
      <div ref={setActions} />
      <span ref={setTitleStatus} />
      <SettingsPageProvider
        actionsContainer={actions}
        titleStatusContainer={titleStatus}
      >
        <SystemInfoSection defaultValues={defaultValues} />
      </SettingsPageProvider>
    </>
  )
}

async function renderSection() {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: initialStatus },
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  client.setQueryData(['status'], initialStatus)
  localStorage.setItem('status', JSON.stringify(initialStatus))
  useSystemConfigStore
    .getState()
    .setConfig({ systemName: initialStatus.system_name })
  const rootRoute = createRootRoute({
    component: SystemInfoFixture,
  })
  const router = createRouter({
    routeTree: rootRoute,
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  await router.load()
  const view = render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  await screen.findByRole('textbox', { name: 'Website Description' })
  return view
}

function submitForm(container: HTMLElement) {
  const form = container.querySelector('form')
  if (!form) throw new Error('Settings form was not rendered')
  fireEvent.submit(form)
}

describe('Default site description setting', () => {
  it('saves the description to the shared SystemDescription option and updates metadata', async () => {
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true } })
    document.head.innerHTML = `
      <title>Configured site</title>
      <meta name="description" content="Initial description">
      <meta property="og:description" content="Initial description">
    `
    const view = await renderSection()

    const description = screen.getByRole('textbox', {
      name: 'Website Description',
    })
    await userEvent.clear(description)
    await userEvent.type(description, 'Service API for daily use.')
    submitForm(view.container)

    await waitFor(() =>
      expect(put).toHaveBeenCalledWith('/api/option/', {
        key: 'SystemDescription',
        value: 'Service API for daily use.',
      })
    )
    expect(
      document.head
        .querySelector('meta[name="description"]')
        ?.getAttribute('content')
    ).toBe('Service API for daily use.')
    expect(
      document.head
        .querySelector('meta[property="og:description"]')
        ?.getAttribute('content')
    ).toBe('Service API for daily use.')
  })

  it('allows clearing the description and removes description metadata', async () => {
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true } })
    document.head.innerHTML = `
      <title>Configured site</title>
      <meta name="description" content="Initial description">
      <meta property="og:description" content="Initial description">
    `
    const view = await renderSection()

    await userEvent.clear(
      screen.getByRole('textbox', { name: 'Website Description' })
    )
    submitForm(view.container)

    await waitFor(() =>
      expect(put).toHaveBeenCalledWith('/api/option/', {
        key: 'SystemDescription',
        value: '',
      })
    )
    expect(document.head.querySelector('meta[name="description"]')).toBeNull()
    expect(
      document.head.querySelector('meta[property="og:description"]')
    ).toBeNull()
    expect(document.title).toBe('Configured site')
  })

  it.each([
    {
      label: 'System Name',
      key: 'SystemName',
      value: 'Updated site',
      expectedTitle: 'Updated site',
      expectedDescription: 'Initial description',
    },
    {
      label: 'Website Description',
      key: 'SystemDescription',
      value: 'Updated description',
      expectedTitle: 'Configured site',
      expectedDescription: 'Updated description',
    },
    {
      label: 'Website Description',
      key: 'SystemDescription',
      value: '',
      expectedTitle: 'Configured site',
      expectedDescription: null,
    },
  ])('业务拒绝 $key=$value 后保留未保存状态，原值可重试', async (scenario) => {
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValueOnce({ data: { success: false, message: 'Rejected' } })
      .mockResolvedValueOnce({ data: { success: true } })
    const successToast = vi.spyOn(toast, 'success')
    const errorToast = vi.spyOn(toast, 'error')
    document.head.innerHTML = `
      <title>Configured site</title>
      <meta name="description" content="Initial description">
      <meta property="og:description" content="Initial description">
    `
    await renderSection()

    const input = screen.getByRole('textbox', { name: scenario.label })
    await userEvent.clear(input)
    if (scenario.value) await userEvent.type(input, scenario.value)
    await userEvent.click(screen.getByRole('button', { name: 'Save Changes' }))

    await waitFor(() => expect(errorToast).toHaveBeenCalledWith('Rejected'))
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Save Changes' })).toBeEnabled()
    )
    expect(errorToast).toHaveBeenCalledTimes(1)
    expect(successToast).not.toHaveBeenCalled()
    expect(screen.getByText('Unsaved changes')).toBeInTheDocument()
    expect(input).toHaveValue(scenario.value)
    expect(document.title).toBe('Configured site')
    expect(screen.getByRole('link', { name: 'Go to home' })).toHaveTextContent(
      'Configured site'
    )
    expect(
      screen.getByRole('status', { name: 'Status description' })
    ).toHaveTextContent('Initial description')
    expect(
      screen.getByRole('status', { name: 'Config system name' })
    ).toHaveTextContent('Configured site')
    expect(JSON.parse(localStorage.getItem('status') ?? '{}')).toEqual(
      initialStatus
    )
    expect(
      document.head
        .querySelector('meta[name="description"]')
        ?.getAttribute('content')
    ).toBe('Initial description')
    expect(
      document.head
        .querySelector('meta[property="og:description"]')
        ?.getAttribute('content')
    ).toBe('Initial description')

    await userEvent.click(screen.getByRole('button', { name: 'Save Changes' }))

    await waitFor(() => expect(successToast).toHaveBeenCalledTimes(1))
    expect(put).toHaveBeenCalledTimes(2)
    expect(put).toHaveBeenNthCalledWith(2, '/api/option/', {
      key: scenario.key,
      value: scenario.value,
    })
    await waitFor(() =>
      expect(screen.queryByText('Unsaved changes')).not.toBeInTheDocument()
    )
    expect(document.title).toBe(scenario.expectedTitle)
    await waitFor(() => {
      expect(
        screen.getByRole('link', { name: 'Go to home' })
      ).toHaveTextContent(scenario.expectedTitle)
      expect(
        screen.getByRole('status', { name: 'Status description' }).textContent
      ).toBe(scenario.expectedDescription ?? '')
      expect(
        screen.getByRole('status', { name: 'Config system name' })
      ).toHaveTextContent(scenario.expectedTitle)
    })
    expect(JSON.parse(localStorage.getItem('status') ?? '{}')).toEqual({
      ...initialStatus,
      system_name: scenario.expectedTitle,
      system_description: scenario.expectedDescription ?? '',
    })
    expect(api.get).not.toHaveBeenCalled()
    for (const selector of [
      'meta[name="description"]',
      'meta[property="og:description"]',
    ]) {
      const meta = document.head.querySelector(selector)
      if (scenario.expectedDescription === null) {
        expect(meta).toBeNull()
      } else {
        expect(meta).toHaveAttribute('content', scenario.expectedDescription)
      }
    }
  })

  it('rejects descriptions over 200 Unicode code points before saving', async () => {
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true } })
    const view = await renderSection()

    fireEvent.change(
      screen.getByRole('textbox', { name: 'Website Description' }),
      {
        target: { value: '界'.repeat(201) },
      }
    )
    submitForm(view.container)

    expect(
      await screen.findByText(
        'Website description must be 200 characters or fewer.'
      )
    ).toBeTruthy()
    expect(put).not.toHaveBeenCalled()
  })
})
