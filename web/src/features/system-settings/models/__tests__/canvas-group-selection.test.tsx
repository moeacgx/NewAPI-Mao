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
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { SettingsPageProvider } from '../../components/settings-page-context'
import { GlobalSettingsCard } from '../global-settings-card'

test('画布默认组按名称选择保存 code，刷新当前名称不改变选择', async () => {
  const user = userEvent.setup()
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const actions = document.createElement('div')
  document.body.appendChild(actions)
  const group = { id: 7, code: 'stable-code', name: '当前名称', status: 1 }
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: [group] },
  })
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { success: true } })
  const view = render(
    <QueryClientProvider client={client}>
      <SettingsPageProvider actionsContainer={actions}>
        <GlobalSettingsCard
          defaultValues={{
            global: {
              pass_through_request_enabled: false,
              thinking_model_blacklist: '[]',
              chat_completions_to_responses_policy: '{}',
              canvas_default_group: '',
            },
            general_setting: {
              ping_interval_enabled: false,
              ping_interval_seconds: 30,
            },
          }}
        />
      </SettingsPageProvider>
    </QueryClientProvider>
  )
  try {
    await user.click(
      screen.getByRole('combobox', { name: 'Infinite Canvas Default Group' })
    )
    await user.click(await screen.findByRole('option', { name: '当前名称' }))
    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }))
    await waitFor(() =>
      expect(put).toHaveBeenCalledWith('/api/option/', {
        key: 'global.canvas_default_group',
        value: 'stable-code',
      })
    )
    act(() =>
      client.setQueryData(['canvas-default-group-options'], {
        groups: [{ ...group, name: '改名后的名称' }],
      })
    )
    await waitFor(() =>
      expect(
        screen.getByRole('combobox', { name: 'Infinite Canvas Default Group' })
      ).toHaveTextContent('改名后的名称')
    )
    expect(put).toHaveBeenCalledTimes(1)
  } finally {
    view.unmount()
    actions.remove()
    client.clear()
  }
})

test('停用的当前画布默认组显示当前名称，且不能被新选择', async () => {
  const user = userEvent.setup()
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const actions = document.createElement('div')
  document.body.appendChild(actions)
  const groups = [
    { id: 7, code: 'stable-code', name: '当前名称', status: 1 },
    { id: 8, code: 'disabled-code', name: '停用当前名', status: 0 },
  ]
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: groups },
  })
  const view = render(
    <QueryClientProvider client={client}>
      <SettingsPageProvider actionsContainer={actions}>
        <GlobalSettingsCard
          defaultValues={{
            global: {
              pass_through_request_enabled: false,
              thinking_model_blacklist: '[]',
              chat_completions_to_responses_policy: '{}',
              canvas_default_group: 'disabled-code',
            },
            general_setting: {
              ping_interval_enabled: false,
              ping_interval_seconds: 30,
            },
          }}
        />
      </SettingsPageProvider>
    </QueryClientProvider>
  )
  try {
    const combobox = screen.getByRole('combobox', {
      name: 'Infinite Canvas Default Group',
    })
    await waitFor(() => expect(combobox).toHaveTextContent('停用当前名'))
    expect(combobox).not.toHaveTextContent('disabled-code')
    await user.click(combobox)
    expect(await screen.findByRole('option', { name: '当前名称' })).toBeTruthy()
    expect(screen.queryByRole('option', { name: '停用当前名' })).toBeNull()
  } finally {
    view.unmount()
    actions.remove()
    client.clear()
  }
})
