import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  RouterProvider,
} from '@tanstack/react-router'
import { act, render, screen } from '@testing-library/react'
import { expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { ModelsProvider } from '../models-provider'
import { ModelsTable } from '../models-table'

vi.mock('@lobehub/icons', () => ({}))

it('真实模型列表查询刷新保留原型自有名称并在删除后回退编码', async () => {
  vi.spyOn(window, 'scrollTo').mockImplementation(() => {})
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  let names = '{"__proto__":"原名称"}'
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: {
      success: true,
      data: {
        items: url.startsWith('/api/models')
          ? [
              {
                id: 1,
                model_name: 'fixture',
                icon: 'https://example.com/fixture.svg',
                status: 1,
                name_rule: 0,
                sync_official: 1,
                created_time: 0,
                updated_time: 0,
                enable_groups: ['__proto__'],
                enable_group_names: JSON.parse(names),
              },
            ]
          : [],
        total: url.startsWith('/api/models') ? 1 : 0,
      },
    },
  }))
  const rootRoute = createRootRoute({ component: Outlet })
  const authenticated = createRoute({
    getParentRoute: () => rootRoute,
    id: '_authenticated',
    component: Outlet,
  })
  const route = createRoute({
    getParentRoute: () => authenticated,
    path: 'models/$section',
    component: () => (
      <ModelsProvider>
        <ModelsTable />
      </ModelsProvider>
    ),
  })
  const router = createRouter({
    routeTree: rootRoute.addChildren([authenticated.addChildren([route])]),
    history: createMemoryHistory({ initialEntries: ['/models/metadata'] }),
  })
  await router.load()
  const view = render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  try {
    expect(await screen.findByText('原名称')).toBeInTheDocument()
    names = '{"__proto__":"改名后"}'
    await act(async () => {
      await client.invalidateQueries({ queryKey: ['models'] })
    })
    expect(await screen.findByText('改名后')).toBeInTheDocument()
    expect(screen.queryByText('原名称')).not.toBeInTheDocument()
    names = '{}'
    await act(async () => {
      await client.invalidateQueries({ queryKey: ['models'] })
    })
    expect(await screen.findByText('__proto__')).toBeInTheDocument()
  } finally {
    view.unmount()
    client.clear()
  }
})
