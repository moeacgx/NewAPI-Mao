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
import { describe, expect, it } from 'vitest'

import { createTicketSchema, ticketReplySchema } from '../schema'

const input = {
  title: 'Billing question',
  category: 'billing',
  priority: 'normal',
  content: 'Please check this order.',
}

describe('工单输入边界', () => {
  it('标题和正文去除首尾空格但保留合法分类优先级', () => {
    expect(
      createTicketSchema.parse({
        ...input,
        title: '  Billing question  ',
        content: '  Please check this order.  ',
      })
    ).toEqual(input)
  })
  it('按 Unicode 字符计算标题上限而不是 UTF-16 长度', () => {
    expect(
      createTicketSchema.safeParse({ ...input, title: '😀'.repeat(120) })
        .success
    ).toBe(true)
    expect(
      createTicketSchema.safeParse({ ...input, title: '😀'.repeat(121) })
        .success
    ).toBe(false)
  })
  it('拒绝空白和超过一万字符的回复', () => {
    expect(ticketReplySchema.safeParse({ content: '   ' }).success).toBe(false)
    expect(
      ticketReplySchema.safeParse({ content: '字'.repeat(10000) }).success
    ).toBe(true)
    expect(
      ticketReplySchema.safeParse({ content: '字'.repeat(10001) }).success
    ).toBe(false)
  })
  it('拒绝未支持的分类和优先级', () => {
    expect(
      createTicketSchema.safeParse({ ...input, category: 'secret' }).success
    ).toBe(false)
    expect(
      createTicketSchema.safeParse({ ...input, priority: 'urgent' }).success
    ).toBe(false)
  })
})
