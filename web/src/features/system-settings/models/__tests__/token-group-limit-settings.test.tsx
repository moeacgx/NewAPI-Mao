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
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import type { ComponentProps } from 'react'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { SettingsPageProvider } from '../../components/settings-page-context'
import { RatioSettingsCard } from '../ratio-settings-card'

const settings: ComponentProps<typeof RatioSettingsCard> = {
  modelDefaults: {
    ModelPrice: '{}',
    ModelPriceUnit: '{}',
    ModelPriceVariants: '{}',
    ModelRoutePriceVariants: '{}',
    ModelRatio: '{}',
    CacheRatio: '{}',
    CreateCacheRatio: '{}',
    CompletionRatio: '{}',
    ImageRatio: '{}',
    AudioRatio: '{}',
    AudioCompletionRatio: '{}',
    ExposeRatioEnabled: false,
    BillingMode: '{}',
    BillingExpr: '{}',
  },
  groupDefaults: {
    GroupRatio: '{"standard":1}',
    TopupGroupRatio: '{}',
    UserUsableGroups: '{}',
    GroupGroupRatio: '{}',
    AutoGroups: '[]',
    MaxTokenAutoGroups: 5,
    DefaultUseAutoGroup: false,
    GroupSpecialUsableGroup: '{}',
  },
  toolPricesDefault: '{}',
  visibleTabs: ['groups'],
}

const group = {
  id: 11,
  code: 'standard',
  name: '标准服务',
  description: '',
  ratio: 1,
  user_selectable: true,
  exclusive: false,
  status: 1,
  auto_enabled: false,
  auto_order: 0,
}

let queryClient: QueryClient
let actionsContainer: HTMLDivElement

afterEach(() => {
  cleanup()
  queryClient?.clear()
  actionsContainer?.remove()
})

async function renderGroupSettings() {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: [group] },
  })
  const saveRequest = vi.spyOn(api, 'put').mockResolvedValue({
    data: { success: true, data: [group] },
  })
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  actionsContainer = document.createElement('div')
  document.body.append(actionsContainer)
  render(
    <QueryClientProvider client={queryClient}>
      <SettingsPageProvider actionsContainer={actionsContainer}>
        <RatioSettingsCard {...settings} />
      </SettingsPageProvider>
    </QueryClientProvider>
  )
  await waitFor(() => {
    expect(
      screen.getByRole('button', { name: 'Save group ratios' })
    ).toBeEnabled()
  })
  return saveRequest
}

describe('Default 令牌分组上限设置', () => {
  test.each(['可视化', 'JSON'])(
    '%s 模式修改上限后通过分组接口保存配置',
    async (mode) => {
      const saveRequest = await renderGroupSettings()
      if (mode === 'JSON') {
        fireEvent.click(screen.getByRole('button', { name: 'Switch to JSON' }))
      }
      const input = screen.getByRole('spinbutton', {
        name: 'Maximum groups per token',
      })
      expect(input).toHaveValue(5)
      fireEvent.change(input, { target: { value: '8' } })
      fireEvent.click(screen.getByRole('button', { name: 'Save group ratios' }))

      await waitFor(() => {
        expect(saveRequest).toHaveBeenCalledWith(
          '/api/group/details',
          expect.objectContaining({
            option_updates: { MaxTokenAutoGroups: '8' },
            groups: [
              expect.objectContaining({
                id: 11,
                code: 'standard',
                name: '标准服务',
              }),
            ],
          })
        )
      })
    }
  )

  test.each(['0', '-1', '1.5'])(
    '上限输入 %s 时阻止保存并标明字段错误',
    async (value) => {
      const saveRequest = await renderGroupSettings()
      const input = screen.getByRole('spinbutton', {
        name: 'Maximum groups per token',
      })
      fireEvent.change(input, { target: { value } })
      fireEvent.click(screen.getByRole('button', { name: 'Save group ratios' }))

      await waitFor(() => expect(input).toHaveAttribute('aria-invalid', 'true'))
      expect(saveRequest).not.toHaveBeenCalled()
    }
  )
})
