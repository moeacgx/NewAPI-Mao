import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import { LogGroupSelect } from '@/features/usage-logs/components/log-group-select'
import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { SettingsPageProvider } from '../../components/settings-page-context'
import type { GroupDetail, UpdateGroupDetailsRequest } from '../../types'
import { GroupRatioForm } from '../group-ratio-form'

type FormValues = {
  GroupRatio: string
  TopupGroupRatio: string
  UserUsableGroups: string
  GroupGroupRatio: string
  AutoGroups: string
  MaxTokenAutoGroups: number
  DefaultUseAutoGroup: boolean
  GroupSpecialUsableGroup: string
}

const originalGroup: GroupDetail = {
  id: 42,
  code: 'public',
  name: '改名前当前名称',
  description: '',
  ratio: 1,
  user_selectable: true,
  exclusive: false,
  single_user_concurrency_limit: 0,
  status: 1,
  auto_enabled: false,
  auto_order: 0,
}
const privateGroup: GroupDetail = {
  ...originalGroup,
  id: 90,
  code: 'private',
  name: '仅管理员可见名称',
  user_selectable: false,
}
const autoGroup = { user_selectable: false, description: '' }
let client: QueryClient
let actions: HTMLDivElement
let groups: GroupDetail[]
let requests: string[]
let saved: UpdateGroupDetailsRequest | undefined

function GroupSettings() {
  const form = useForm<
    Omit<FormValues, 'MaxTokenAutoGroups'> & { MaxTokenAutoGroups: unknown },
    unknown,
    FormValues
  >({
    defaultValues: {
      GroupRatio: '{"public":1}',
      TopupGroupRatio: '{}',
      UserUsableGroups: '{}',
      GroupGroupRatio: '{}',
      AutoGroups: '[]',
      MaxTokenAutoGroups: 0,
      DefaultUseAutoGroup: false,
      GroupSpecialUsableGroup: '{}',
    },
  })
  return (
    <SettingsPageProvider actionsContainer={actions}>
      <GroupRatioForm
        form={form}
        onSave={() => ({ MaxTokenAutoGroups: '0' })}
        isSaving={false}
      />
    </SettingsPageProvider>
  )
}

function Logs() {
  const [value, setValue] = useState('public')
  const [submitted, setSubmitted] = useState('')
  const isAdmin = useAuthStore((state) => (state.auth.user?.role ?? 0) >= 10)
  return (
    <>
      <LogGroupSelect
        value={value}
        onValueChange={setValue}
        isAdmin={isAdmin}
      />
      <button type='button' onClick={() => setSubmitted(value)}>
        提交日志筛选
      </button>
      <output aria-label='已提交分组'>{submitted}</output>
    </>
  )
}

function Fixture(props: { mounted?: boolean }) {
  const [page, setPage] = useState<'logs' | 'settings'>('logs')
  return (
    <>
      <button type='button' onClick={() => setPage('logs')}>
        打开日志
      </button>
      <button type='button' onClick={() => setPage('settings')}>
        打开分组设置
      </button>
      <button
        type='button'
        onClick={() => {
          useAuthStore.getState().auth.setUser({
            id: 7,
            username: 'reader',
            role: 1,
            group: 'public',
          })
          setPage('logs')
        }}
      >
        切换普通用户
      </button>
      {page === 'logs' || props.mounted ? <Logs /> : null}
      {page === 'settings' ? <GroupSettings /> : null}
    </>
  )
}

beforeEach(() => {
  client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, staleTime: 10_000 },
      mutations: { retry: false },
    },
  })
  actions = document.createElement('div')
  document.body.append(actions)
  groups = [{ ...originalGroup }, { ...privateGroup }]
  requests = []
  saved = undefined
  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'root',
    role: 100,
    group: 'public',
  })
  vi.spyOn(toast, 'success')
  // 仅模拟网络返回；表单、保存 API、失效逻辑、查询客户端和选组组件均执行真实代码。
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    requests.push(String(url))
    if (url === '/api/group/tags') {
      return { data: { success: true, data: [] } }
    }
    const allowed =
      url === '/api/user/self/groups'
        ? groups.filter((group) => group.code === 'public')
        : groups
    return {
      data: {
        success: true,
        data:
          url === '/api/user/self/groups'
            ? Object.fromEntries(allowed.map((group) => [group.code, group]))
            : allowed,
        auto_group: autoGroup,
        group_names: Object.fromEntries(
          allowed.map((group) => [group.code, group.name])
        ),
      },
    }
  })
  vi.spyOn(api, 'put').mockImplementation(async (_url, payload) => {
    saved = payload as UpdateGroupDetailsRequest
    groups = saved.groups.map((group) => ({
      ...group,
      id: group.id ?? 0,
    }))
    return { data: { success: true, data: groups, auto_group: autoGroup } }
  })
})

afterEach(() => {
  cleanup()
  client.clear()
  actions.remove()
  localStorage.clear()
  useAuthStore.getState().auth.reset()
})

async function renameThroughSettings() {
  fireEvent.click(screen.getByRole('button', { name: '打开分组设置' }))
  await waitFor(() =>
    expect(
      screen.getByRole('button', { name: 'Save group ratios' })
    ).toBeEnabled()
  )
  fireEvent.change(screen.getByDisplayValue('改名前当前名称'), {
    target: { value: '保存后最新名称' },
  })
  fireEvent.click(screen.getByRole('button', { name: 'Save group ratios' }))
  await waitFor(() =>
    expect(toast.success).toHaveBeenCalledWith(
      'Group settings saved successfully'
    )
  )
  expect(api.put).toHaveBeenCalledWith('/api/group/details', saved)
  expect(saved).toEqual(
    expect.objectContaining({
      option_updates: { MaxTokenAutoGroups: '0' },
      groups: expect.arrayContaining([
        expect.objectContaining({
          id: 42,
          code: 'public',
          name: '保存后最新名称',
        }),
      ]),
    })
  )
}

test('管理员改名保存后立即返回日志，仍在新鲜期的缓存重取当前名称并提交原code', async () => {
  render(
    <QueryClientProvider client={client}>
      <Fixture />
    </QueryClientProvider>
  )
  await waitFor(() =>
    expect(screen.getByRole('combobox', { name: 'Group' })).toHaveTextContent(
      '改名前当前名称'
    )
  )
  await renameThroughSettings()
  fireEvent.click(screen.getByRole('button', { name: '打开日志' }))
  await waitFor(() =>
    expect(screen.getByRole('combobox', { name: 'Group' })).toHaveTextContent(
      '保存后最新名称'
    )
  )
  fireEvent.click(screen.getByRole('button', { name: '提交日志筛选' }))
  expect(screen.getByLabelText('已提交分组')).toHaveTextContent('public')
})

test('管理员日志选组已挂载时，真实保存成功立即刷新观察者且不改变稳定code', async () => {
  render(
    <QueryClientProvider client={client}>
      <Fixture mounted />
    </QueryClientProvider>
  )
  const trigger = screen.getByRole('combobox', { name: 'Group' })
  await waitFor(() => expect(trigger).toHaveTextContent('改名前当前名称'))
  await renameThroughSettings()
  await waitFor(() => expect(trigger).toHaveTextContent('保存后最新名称'))
  fireEvent.click(screen.getByRole('button', { name: '提交日志筛选' }))
  expect(screen.getByLabelText('已提交分组')).toHaveTextContent('public')
})

test('分组保存后切换普通用户只读取self合法组，不沿用管理员名称或组选项', async () => {
  render(
    <QueryClientProvider client={client}>
      <Fixture />
    </QueryClientProvider>
  )
  await waitFor(() =>
    expect(screen.getByRole('combobox', { name: 'Group' })).toHaveTextContent(
      '改名前当前名称'
    )
  )
  await renameThroughSettings()
  const beforeSwitch = requests.length
  fireEvent.click(screen.getByRole('button', { name: '切换普通用户' }))
  const trigger = screen.getByRole('combobox', { name: 'Group' })
  await waitFor(() => expect(trigger).toHaveTextContent('保存后最新名称'))
  expect(requests.slice(beforeSwitch)).toEqual(['/api/user/self/groups'])
  fireEvent.click(trigger)
  expect(
    await screen.findByRole('option', { name: '保存后最新名称' })
  ).toBeInTheDocument()
  expect(
    screen.queryByRole('option', { name: '仅管理员可见名称' })
  ).not.toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: '提交日志筛选' }))
  expect(screen.getByLabelText('已提交分组')).toHaveTextContent('public')
})
