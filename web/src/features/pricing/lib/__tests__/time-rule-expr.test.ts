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
import { describe, expect, test } from 'vitest'

import {
  buildRequestRuleExpr,
  MATCH_EQ,
  MATCH_RANGE,
  requestRuleGroupsFromTrace,
  tryParseRequestRuleExpr,
  type TimeCondition,
} from '../billing-expr'

const timeRange: TimeCondition = {
  source: 'time',
  timeFunc: 'hour',
  timezone: 'Asia/Shanghai',
  mode: MATCH_RANGE,
  value: '',
  rangeStart: '9',
  rangeEnd: '12',
}

describe('时间范围生成与解析', () => {
  test.each([
    ['同日范围只在两个边界之间生效', '9', '12', '&&'],
    ['跨午夜范围保留两端匹配', '21', '6', '||'],
    ['相同边界为空范围', '9', '9', '&&'],
  ])('%s', (_name, rangeStart, rangeEnd, operator) => {
    expect(
      buildRequestRuleExpr([
        {
          conditions: [{ ...timeRange, rangeStart, rangeEnd }],
          multiplier: '2',
        },
      ])
    ).toBe(
      `(hour("Asia/Shanghai") >= ${rangeStart} ${operator} hour("Asia/Shanghai") < ${rangeEnd} ? 2 : 1)`
    )
  })

  test.each([
    ['hour', '9', '24'],
    ['minute', '0', '60'],
    ['weekday', '0', '7'],
    ['month', '1', '13'],
    ['day', '1', '32'],
  ] as const)(
    '%s 的排他上界 %s 至 %s 不得丢失时间条件',
    (timeFunc, rangeStart, rangeEnd) => {
      const expression = `(param("service_tier") == "fast" && ${timeFunc}("Asia/Shanghai") >= ${rangeStart} && ${timeFunc}("Asia/Shanghai") < ${rangeEnd} ? 2 : 1)`
      const parsed = tryParseRequestRuleExpr(expression)
      expect(parsed).not.toBeNull()
      expect(buildRequestRuleExpr(parsed ?? [])).toBe(expression)
    }
  )

  test('已有永不匹配的交集不能被重建为跨午夜并集', () => {
    const expression =
      '(hour("Asia/Shanghai") >= 21 && hour("Asia/Shanghai") < 6 ? 2 : 1)'
    const parsed = tryParseRequestRuleExpr(expression)
    expect(parsed).not.toBeNull()
    expect(buildRequestRuleExpr(parsed ?? [])).toBe(expression)
  })

  test('已有全天匹配的并集保留原文模式而非静默修正价格', () => {
    expect(
      tryParseRequestRuleExpr(
        '(hour("Asia/Shanghai") >= 9 || hour("Asia/Shanghai") < 12 ? 2 : 1)'
      )
    ).toBeNull()
    const traces = requestRuleGroupsFromTrace([
      {
        cond: 'hour("Asia/Shanghai") >= 9 || hour("Asia/Shanghai") < 12',
        multiplier: 2,
        matched: true,
      },
    ])
    expect(traces[0].conditions).toEqual([])
    expect(traces[0].conditionText).toBe(
      'hour("Asia/Shanghai") >= 9 || hour("Asia/Shanghai") < 12'
    )
  })

  test('请求条件与同日范围组合时保持单行解析与往返稳定', () => {
    const expression =
      '(param("service_tier") == "fast" && hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12 ? 2 : 1)'
    const parsed = tryParseRequestRuleExpr(expression)
    expect(parsed?.[0].conditions.map((condition) => condition.mode)).toEqual([
      MATCH_EQ,
      MATCH_RANGE,
    ])
    expect(buildRequestRuleExpr(parsed ?? [])).toBe(expression)
  })

  test('跨午夜范围与请求条件组合后仍保留范围语义', () => {
    const expression =
      '((hour("Asia/Shanghai") >= 21 || hour("Asia/Shanghai") < 6) && param("service_tier") == "fast" ? 3 : 1)'
    const parsed = tryParseRequestRuleExpr(expression)
    expect(parsed?.[0].conditions.map((condition) => condition.mode)).toEqual([
      MATCH_RANGE,
      MATCH_EQ,
    ])
    expect(buildRequestRuleExpr(parsed ?? [])).toBe(expression)
  })

  test('日志规则保留服务端倍率命中状态及未知条件原文', () => {
    const traces = requestRuleGroupsFromTrace([
      {
        cond: 'hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12',
        multiplier: 3,
        matched: false,
      },
      { cond: 'custom_probe() == true', multiplier: 5, matched: true },
    ])
    expect(traces[0]).toMatchObject({
      multiplier: '3',
      matched: false,
      conditions: [{ mode: MATCH_RANGE }],
    })
    expect(traces[1]).toMatchObject({
      multiplier: '5',
      matched: true,
      conditionText: 'custom_probe() == true',
      conditions: [],
    })
  })
})
