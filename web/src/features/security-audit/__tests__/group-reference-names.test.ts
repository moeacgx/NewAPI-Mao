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

import {
  formatAuditGroupReference,
  getAuditRouteGroupReference,
  getAuditChannelGroupReferences,
  getAuditTokenGroupReference,
} from '../event-routing-display'

describe('审计分组标签与原始标识分离', () => {
  it.each([
    ['当前名称', 'legacy-alias', 7, '当前名称'],
    ['改名后的名称', 'stable-code', 7, '改名后的名称'],
    ['', 'deleted', 7, 'deleted'],
    ['', 'unknown', 0, 'unknown'],
    ['', '', 7, '#7'],
    ['', '', 0, ''],
  ])('名称 %s、code %s、ID %s 显示 %s', (name, code, id, expected) => {
    const reference = Object.freeze({
      name,
      code,
      id,
      source: 'route' as const,
    })
    expect(formatAuditGroupReference(reference)).toBe(expected)
    expect(reference).toEqual({ name, code, id, source: 'route' })
  })

  it('优先只读当前投影，路由、渠道与令牌历史证据保持不变', () => {
    const group = Object.freeze({
      id: 7,
      code: 'legacy-alias',
      name: '历史名称',
      display_name: '当前名称',
    })
    const event = Object.freeze({
      stage: 'response',
      channel_id: 1,
      channel_name: '渠道',
      channel_groups: [group],
      token_group_mode: 'explicit',
      token_groups: [group],
      group_id: 7,
      group_code: 'legacy-alias',
      group_name: '历史名称',
      group_display_name: '当前名称',
    })
    expect(getAuditRouteGroupReference(event)?.name).toBe('当前名称')
    expect(getAuditChannelGroupReferences(event)[0].name).toBe('当前名称')
    expect(getAuditTokenGroupReference(event)).toMatchObject({
      groups: [{ name: '当前名称', code: 'legacy-alias', id: 7 }],
    })
    expect(event.group_name).toBe('历史名称')
    expect(group.name).toBe('历史名称')
  })

  it.each([
    ['数字', 42],
    ['对象', {}],
    ['数组', ['当前名称']],
    ['null', null],
    ['空白字符串', '   '],
  ])('路由、渠道与令牌的显示字段为%s时回退历史名称', (_label, value) => {
    const group = {
      id: 7,
      code: 'stable-code',
      name: '历史名称',
      display_name: value,
    }
    const event = {
      stage: 'response',
      channel_id: 1,
      channel_name: '渠道',
      channel_groups: [group],
      token_group_mode: 'explicit',
      token_groups: [group],
      group_id: 7,
      group_code: 'stable-code',
      group_name: '历史名称',
      group_display_name: value,
    } as never
    expect(getAuditRouteGroupReference(event)?.name).toBe('历史名称')
    expect(getAuditChannelGroupReferences(event)[0].name).toBe('历史名称')
    expect(getAuditTokenGroupReference(event)).toMatchObject({
      groups: [{ name: '历史名称', code: 'stable-code', id: 7 }],
    })
  })

  it('仅存在于原型链上的显示字段被忽略，不覆盖历史名称', () => {
    const group = Object.assign(Object.create({ display_name: '继承名称' }), {
      id: 7,
      code: 'stable-code',
      name: '历史名称',
    })
    const event = Object.assign(
      Object.create({ group_display_name: '继承名称' }),
      {
        stage: 'response',
        channel_id: 1,
        channel_name: '渠道',
        channel_groups: [group],
        token_group_mode: 'explicit',
        token_groups: [group],
        group_id: 7,
        group_code: 'stable-code',
        group_name: '历史名称',
      }
    ) as never
    expect(getAuditRouteGroupReference(event)?.name).toBe('历史名称')
    expect(getAuditChannelGroupReferences(event)[0].name).toBe('历史名称')
    expect(getAuditTokenGroupReference(event)).toMatchObject({
      groups: [{ name: '历史名称', code: 'stable-code', id: 7 }],
    })
  })
})
