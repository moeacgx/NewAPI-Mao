import { createInstance } from 'i18next'
import { expect, test } from 'vitest'

import en from '@/i18n/locales/en.json'
import fr from '@/i18n/locales/fr.json'
import ja from '@/i18n/locales/ja.json'
import ru from '@/i18n/locales/ru.json'
import vi from '@/i18n/locales/vi.json'
import zhTW from '@/i18n/locales/zh-TW.json'
import zh from '@/i18n/locales/zh.json'

const resources = { en, fr, ja, ru, vi, zh, 'zh-TW': zhTW }
const keys = [
  'Built-in plugins only. Marketplace, upload, deletion and sandbox are unavailable in this version.',
  'Disabling task plugins stops new submissions. Existing tasks continue with their pinned version.',
  'Not activated',
  'Select a task plugin',
] as const

test.each(Object.entries(resources))(
  '%s 实际语言包显示可读的插件限制与状态',
  async (locale, resource) => {
    const i18n = createInstance()
    await i18n.init({
      lng: locale,
      fallbackLng: false,
      resources: { [locale]: resource },
    })
    for (const key of keys) {
      expect(i18n.exists(key)).toBe(true)
      expect(i18n.t(key)).not.toMatch(/[?\uFFFD]/)
      if (locale !== 'en') expect(i18n.t(key)).not.toBe(key)
    }
    if (locale === 'zh') {
      expect(i18n.t('Not activated')).toBe('未激活')
      expect(i18n.t(keys[1])).toContain('已有任务继续使用固定版本处理')
    }
  }
)
