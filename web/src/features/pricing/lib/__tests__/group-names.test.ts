import { describe, expect, it } from 'vitest'

import { getGroupDisplayName } from '../group-names'

describe('价格分组名称安全解析', () => {
  it.each(['__proto__', 'constructor', 'toString', 'unknown'])(
    '未知或删除的 %s 不读取原型并回退编码',
    (code) => {
      expect(getGroupDisplayName(code, {})).toBe(code)
    }
  )

  it('合法原型属性自有字符串显示当前名称，空白与非字符串回退编码', () => {
    const names = JSON.parse(
      '{"__proto__":" 当前名称 ","constructor":0,"toString":"  ","legacy":"别名名称"}'
    )
    expect(getGroupDisplayName('__proto__', names)).toBe('当前名称')
    expect(getGroupDisplayName('constructor', names)).toBe('constructor')
    expect(getGroupDisplayName('toString', names)).toBe('toString')
    expect(getGroupDisplayName('legacy', names)).toBe('别名名称')
  })
})
