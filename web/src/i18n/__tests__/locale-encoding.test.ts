import { expect, test } from 'vitest'

import fr from '@/i18n/locales/fr.json'
import ja from '@/i18n/locales/ja.json'
import ru from '@/i18n/locales/ru.json'
import vi from '@/i18n/locales/vi.json'
import zhTW from '@/i18n/locales/zh-TW.json'
import zh from '@/i18n/locales/zh.json'

const resources = { fr, ja, ru, vi, zh, 'zh-TW': zhTW }

// 经非 UTF-8 代码页写入的译文会把每个非 ASCII 字符替换成 "?"，界面显示成一串问号。
// 特征：整条译文没有任何非 ASCII 字符，却比原文多出紧贴文字的 "?"；
// 法语问句在 "?" 前有空格，不属于此类。
function isEncodingDamaged(key: string, value: string) {
  const hasNonAscii = [...value].some(
    (char) => (char.codePointAt(0) ?? 0) > 0x7f
  )
  if (!value.includes('?') || hasNonAscii) return false
  const keyMarks = (key.match(/[?？]/g) ?? []).length
  const valueMarks = (value.match(/\?/g) ?? []).length
  return valueMarks > keyMarks && /(^|\S)\?/.test(value)
}

test.each(Object.entries(resources))(
  '%s 译文没有被非 UTF-8 写入替换成问号',
  (_locale, resource) => {
    const damaged = Object.entries(
      resource.translation as Record<string, unknown>
    )
      .filter(
        ([key, value]) =>
          typeof value === 'string' && isEncodingDamaged(key, value)
      )
      .map(([key, value]) => `${key} => ${value}`)
    expect(damaged).toEqual([])
  }
)
