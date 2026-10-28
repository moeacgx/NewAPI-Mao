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
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { NotificationCenter } from '..'

vi.mock(
  '@/components/layout',
  async () => import('@/components/layout/components/section-page-layout')
)

const ticketEvent = {
  value: 'ticket_user_message',
  label: 'Ticket created or user replied',
  variables: ['ticket_id', 'content'],
  default_template: '{{ticket_id}}\n正文：{{content}}',
}

beforeEach(() => {
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    let data: unknown = []
    if (url.endsWith('/event-types')) data = [ticketEvent]
    if (url.endsWith('/bots')) {
      data = [{ id: 1, name: 'Support bot', enabled: true }]
    }
    if (url.endsWith('/tasks')) {
      data = [
        {
          id: 2,
          name: 'Support notifications',
          event_type: ticketEvent.value,
          bot_id: 1,
          template: 'Custom {{ticket_id}}',
          enabled: true,
          targets: [{ chat_id: '-1001', enabled: true }],
        },
      ]
    }
    return { data: { success: true, data } }
  })
  vi.spyOn(api, 'put').mockResolvedValue({ data: { success: true } })
})

test('工单模板展示正文摘要说明及示例，编辑自定义模板可插入正文并保留接收目标', async () => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <NotificationCenter />
    </QueryClientProvider>
  )
  await screen.findByText('Support notifications')
  fireEvent.click(screen.getByRole('button', { name: 'Edit' }))
  expect(await screen.findByText('Current user message excerpt')).toBeVisible()
  expect(
    screen.getByText('Example: I need help with this request.')
  ).toBeVisible()
  expect(
    screen.getByText(
      'Ticket notifications send an excerpt of the current user message to the recipients configured by Root. The escaped excerpt is limited to 2000 characters; longer messages include a truncation notice. View the full message in ticket details.'
    )
  ).toBeVisible()
  const template = screen.getByRole<HTMLTextAreaElement>('textbox', {
    name: 'Message template',
  })
  expect(template).toHaveValue('Custom {{ticket_id}}')
  template.focus()
  const end = template.value.length
  template.setSelectionRange(end, end)
  fireEvent.click(
    screen.getByRole('button', { name: 'Insert template variable {{content}}' })
  )
  expect(template).toHaveValue('Custom {{ticket_id}}{{content}}')
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() =>
    expect(api.put).toHaveBeenCalledWith(
      '/api/notification/tasks/2',
      expect.objectContaining({
        template: 'Custom {{ticket_id}}{{content}}',
        targets: [{ chat_id: '-1001', enabled: true }],
      })
    )
  )
})
