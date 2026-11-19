import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { LogGroupSelect } from '../log-group-select'

vi.mock('@/lib/api', () => ({ api: { get: vi.fn() } }))
const get = vi.mocked(api.get)
const clients: QueryClient[] = []

function Fixture(props: { initial?: string; admin?: boolean }) {
  const [value, setValue] = useState(props.initial ?? '')
  return (
    <>
      <LogGroupSelect
        value={value}
        onValueChange={setValue}
        isAdmin={props.admin ?? false}
      />
      <output aria-label='提交标识'>{value}</output>
    </>
  )
}

function renderSelect(initial = '', admin = false) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  clients.push(client)
  const view = render(
    <QueryClientProvider client={client}>
      <Fixture initial={initial} admin={admin} />
    </QueryClientProvider>
  )
  return { client, ...view }
}

beforeEach(() => {
  useAuthStore
    .getState()
    .auth.setUser({ id: 7, username: 'reader', role: 1, group: 'public' })
  get.mockReset()
  get.mockResolvedValue({
    data: {
      success: true,
      data: { public: { id: 42, code: 'public', name: '当前显示名称' } },
      group_names: { public: '当前显示名称', 'old-public': '当前显示名称' },
    },
  })
})
afterEach(() => {
  clients.splice(0).forEach((client) => client.clear())
  useAuthStore.getState().auth.reset()
})

describe('普通日志分组选择', () => {
  test.each(['__proto__', 'constructor', 'toString'])(
    '未知原型属性名%s回退原筛选值，不读取继承名称或导致渲染错误',
    async (value) => {
      get.mockResolvedValue({
        data: { success: true, data: {}, group_names: {} },
      })
      renderSelect(value)
      const trigger = screen.getByRole('combobox', { name: 'Group' })
      await waitFor(() =>
        expect(trigger).not.toHaveAttribute('aria-busy', 'true')
      )
      expect(trigger).toHaveTextContent(value)
      expect(screen.getByLabelText('提交标识')).toHaveTextContent(value)
    }
  )

  test.each(['__proto__', 'constructor', 'toString'])(
    '名称映射真实自有属性%s为字符串时正常展示，提交仍保留原值',
    async (value) => {
      get.mockResolvedValue({
        data: {
          success: true,
          data: {},
          group_names: Object.fromEntries([[value, '真实当前分组名称']]),
        },
      })
      renderSelect(value)
      await waitFor(() =>
        expect(
          screen.getByRole('combobox', { name: 'Group' })
        ).toHaveTextContent('真实当前分组名称')
      )
      expect(screen.getByLabelText('提交标识')).toHaveTextContent(value)
    }
  )

  test.each([
    ['空字符串', ''],
    ['空值', null],
    ['数字', 42],
    ['对象', { name: '无效名称' }],
    ['数组', ['无效名称']],
  ])('名称映射为%s时回退原标识而不崩溃', async (_description, name) => {
    get.mockResolvedValue({
      data: {
        success: true,
        data: {},
        group_names: { 'legacy-invalid': name },
      },
    })
    renderSelect('legacy-invalid')
    const trigger = screen.getByRole('combobox', { name: 'Group' })
    await waitFor(() =>
      expect(trigger).not.toHaveAttribute('aria-busy', 'true')
    )
    expect(trigger).toHaveTextContent('legacy-invalid')
    expect(screen.getByLabelText('提交标识')).toHaveTextContent(
      'legacy-invalid'
    )
  })

  test('普通用户只加载合法自用组选项，显示名称并提交稳定code，支持清空', async () => {
    renderSelect()
    const user = userEvent.setup()
    const trigger = screen.getByRole('combobox', { name: 'Group' })
    await waitFor(() =>
      expect(trigger).not.toHaveAttribute('aria-busy', 'true')
    )
    await user.click(trigger)
    await user.click(screen.getByRole('option', { name: '当前显示名称' }))
    expect(screen.getByLabelText('提交标识')).toHaveTextContent('public')
    expect(trigger).toHaveTextContent('当前显示名称')
    expect(
      get.mock.calls.every(([url]) => url === '/api/user/self/groups')
    ).toBe(true)
    await user.click(screen.getByRole('button', { name: 'Clear group filter' }))
    expect(screen.getByLabelText('提交标识')).toBeEmptyDOMElement()
  })

  test('旧alias链接展示当前名称且不会重写提交标识，刷新后跟随改名', async () => {
    const { client } = renderSelect('old-public')
    const trigger = screen.getByRole('combobox', { name: 'Group' })
    await waitFor(() => expect(trigger).toHaveTextContent('当前显示名称'))
    expect(screen.getByLabelText('提交标识')).toHaveTextContent('old-public')
    get.mockResolvedValue({
      data: {
        success: true,
        data: { public: { id: 42, code: 'public', name: '改名后名称' } },
        group_names: { public: '改名后名称', 'old-public': '改名后名称' },
      },
    })
    await client.invalidateQueries({ queryKey: ['user-groups'] })
    await waitFor(() => expect(trigger).toHaveTextContent('改名后名称'))
    expect(screen.getByLabelText('提交标识')).toHaveTextContent('old-public')
  })

  test.each(['unknown', 'deleted'])(
    '未知或删除组%s保留原值且不能成为普通用户可选项',
    async (value) => {
      renderSelect(value)
      const user = userEvent.setup()
      const trigger = screen.getByRole('combobox', { name: 'Group' })
      await waitFor(() => expect(trigger).toHaveTextContent(value))
      await user.click(trigger)
      expect(screen.getByRole('option', { name: value })).toHaveAttribute(
        'aria-disabled',
        'true'
      )
      expect(screen.getByLabelText('提交标识')).toHaveTextContent(value)
    }
  )

  test('管理员全部视图保留停用组选项并使用管理接口，切回本人后立即隔离选项', async () => {
    get.mockImplementation(async (url) => ({
      data:
        url === '/api/group/details'
          ? {
              success: true,
              data: [
                { id: 8, code: 'disabled', name: '停用组名称', status: 0 },
              ],
              group_names: { disabled: '停用组名称' },
            }
          : { success: true, data: {}, group_names: {} },
    }))
    const { rerender } = renderSelect('', true)
    const user = userEvent.setup()
    await waitFor(() =>
      expect(get).toHaveBeenCalledWith('/api/group/details', expect.anything())
    )
    await user.click(screen.getByRole('combobox', { name: 'Group' }))
    expect(
      await screen.findByRole('option', { name: '停用组名称' })
    ).toBeInTheDocument()
    await user.keyboard('{Escape}')
    rerender(
      <QueryClientProvider client={clients[0]}>
        <Fixture admin={false} />
      </QueryClientProvider>
    )
    await waitFor(() =>
      expect(get).toHaveBeenCalledWith(
        '/api/user/self/groups',
        expect.anything()
      )
    )
    await user.click(screen.getByRole('combobox', { name: 'Group' }))
    expect(screen.queryByRole('option', { name: '停用组名称' })).toBeNull()
    expect(
      screen.getByRole('option', { name: 'All Groups' })
    ).toBeInTheDocument()
  })

  test('名称映射缺失时保留选项和旧筛选并提供重试', async () => {
    get.mockResolvedValueOnce({
      data: {
        success: true,
        data: { public: { code: 'public', name: '当前显示名称' } },
      },
    })
    renderSelect('old-public')
    const user = userEvent.setup()
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Failed to load group names'
    )
    expect(screen.getByLabelText('提交标识')).toHaveTextContent('old-public')
    await user.click(screen.getByRole('button', { name: 'Retry' }))
    await waitFor(() =>
      expect(screen.getByRole('combobox', { name: 'Group' })).toHaveTextContent(
        '当前显示名称'
      )
    )
    expect(screen.queryByRole('alert')).toBeNull()
  })

  test('请求失败后能够重试，空结果只允许全部分组', async () => {
    get.mockRejectedValueOnce(new Error('网络失败'))
    get.mockResolvedValue({
      data: { success: true, data: {}, group_names: {} },
    })
    renderSelect()
    const user = userEvent.setup()
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Failed to load groups'
    )
    await user.click(screen.getByRole('button', { name: 'Retry' }))
    await waitFor(() => expect(screen.queryByRole('alert')).toBeNull())
    await user.click(screen.getByRole('combobox', { name: 'Group' }))
    expect(screen.getAllByRole('option')).toHaveLength(1)
    expect(
      screen.getByRole('option', { name: 'All Groups' })
    ).toBeInTheDocument()
  })

  test('切换账户不保留上个账户缓存，加载中仍能清空现有筛选', async () => {
    renderSelect('old-public')
    const trigger = screen.getByRole('combobox', { name: 'Group' })
    await waitFor(() => expect(trigger).toHaveTextContent('当前显示名称'))
    get.mockImplementation(() => new Promise(() => {}))
    useAuthStore
      .getState()
      .auth.setUser({ id: 9, username: 'next', role: 1, group: 'other' })
    await waitFor(() => expect(trigger).toHaveAttribute('aria-busy', 'true'))
    expect(trigger).not.toHaveTextContent('当前显示名称')
    await userEvent
      .setup()
      .click(screen.getByRole('button', { name: 'Clear group filter' }))
    expect(screen.getByLabelText('提交标识')).toBeEmptyDOMElement()
  })
})
