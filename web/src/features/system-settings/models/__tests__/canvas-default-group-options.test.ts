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
  buildCanvasDefaultGroupDisplayItems,
  buildCanvasDefaultGroupOptions,
} from '../canvas-default-group-options'

describe('buildCanvasDefaultGroupOptions', () => {
  const groups = [
    {
      id: 1,
      code: 'default',
      name: '默认分组',
      description: '',
      ratio: 1,
      user_selectable: true,
      exclusive: false,
      status: 1,
      auto_enabled: true,
      auto_order: 1,
    },
    {
      id: 2,
      code: 'disabled',
      name: '停用分组',
      description: '',
      ratio: 1,
      user_selectable: true,
      exclusive: false,
      status: 0,
      auto_enabled: false,
      auto_order: 0,
    },
  ]

  it('places selectable auto before active physical groups', () => {
    expect(
      buildCanvasDefaultGroupOptions(groups, {
        user_selectable: true,
        description: '自动选择可用分组',
      })
    ).toEqual([
      { value: 'auto', label: 'Automatic selection' },
      { value: 'default', label: '默认分组' },
    ])
  })

  it('omits auto when users cannot select it', () => {
    expect(
      buildCanvasDefaultGroupOptions(groups, {
        user_selectable: false,
        description: '',
      })
    ).toEqual([{ value: 'default', label: '默认分组' }])
  })

  it('名称刷新只改变标签，缺名仍保留原 code', () => {
    expect(
      buildCanvasDefaultGroupOptions([
        { ...groups[0], name: '改名后的分组' },
        { ...groups[0], id: 3, code: 'unknown', name: '' },
      ])
    ).toEqual([
      { value: 'default', label: '改名后的分组' },
      { value: 'unknown', label: 'unknown' },
    ])
  })
})

describe('buildCanvasDefaultGroupDisplayItems', () => {
  const base = {
    id: 1,
    description: '',
    ratio: 1,
    user_selectable: true,
    exclusive: false,
    auto_enabled: false,
    auto_order: 0,
  }
  const groups = [
    { ...base, id: 1, code: 'default', name: '默认分组', status: 1 },
    { ...base, id: 2, code: 'disabled-code', name: '停用当前名', status: 0 },
  ]
  const options = buildCanvasDefaultGroupOptions(groups)

  it('停用的当前值显示当前名称，且不进入可选项', () => {
    expect(options).toEqual([{ value: 'default', label: '默认分组' }])
    expect(
      buildCanvasDefaultGroupDisplayItems(options, groups, 'disabled-code')
    ).toEqual([
      { value: 'default', label: '默认分组' },
      { value: 'disabled-code', label: '停用当前名' },
    ])
  })

  it('停用分组改名后显示新名称，保存值仍是 code', () => {
    const renamed = [groups[0], { ...groups[1], name: '改名后的停用组' }]
    const items = buildCanvasDefaultGroupDisplayItems(
      options,
      renamed,
      'disabled-code'
    )
    expect(items.at(-1)).toEqual({
      value: 'disabled-code',
      label: '改名后的停用组',
    })
  })

  it('当前值已在可选项中时不重复追加', () => {
    expect(
      buildCanvasDefaultGroupDisplayItems(options, groups, 'default')
    ).toBe(options)
  })

  it('未知、已删除或缺名的当前值不追加显示项，回退原 code', () => {
    expect(
      buildCanvasDefaultGroupDisplayItems(options, groups, 'deleted-code')
    ).toBe(options)
    expect(
      buildCanvasDefaultGroupDisplayItems(
        options,
        [{ ...groups[1], name: '' }],
        'disabled-code'
      )
    ).toBe(options)
  })

  it('空值与 auto 行为不变', () => {
    const withAuto = buildCanvasDefaultGroupOptions(groups, {
      user_selectable: true,
      description: '',
    })
    expect(buildCanvasDefaultGroupDisplayItems(withAuto, groups, '')).toBe(
      withAuto
    )
    expect(buildCanvasDefaultGroupDisplayItems(withAuto, groups, 'auto')).toBe(
      withAuto
    )
  })

  it('原型键与非字符串名称不会被当作显示名称', () => {
    expect(
      buildCanvasDefaultGroupDisplayItems(options, groups, 'constructor')
    ).toBe(options)
    expect(
      buildCanvasDefaultGroupDisplayItems(
        options,
        [{ ...groups[1], name: { toString: () => 'x' } as unknown as string }],
        'disabled-code'
      )
    ).toBe(options)
  })
})
