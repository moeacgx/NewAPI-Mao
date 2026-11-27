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

import { SIDEBAR_MODULES_DEFAULT } from '@/features/system-settings/maintenance/config'
import {
  filterSidebarNavGroups,
  parseSidebarConfig,
} from '@/hooks/use-sidebar-config'

import {
  isValidSupplierUrl,
  parseSupplierRatio,
  supplierApplicationSchema,
} from '../schema'

describe('supplier application shared contracts', () => {
  it('older sidebar configs gain the supplier entries enabled by default', () => {
    const config = parseSidebarConfig({
      personal: { invoice: true },
      admin: { user: true },
    })
    expect(config.personal.supplier_application).toBe(true)
    expect(config.admin.supplier_application_admin).toBe(true)
    expect(SIDEBAR_MODULES_DEFAULT.personal).toMatchObject({
      supplier_application: true,
    })
    expect(SIDEBAR_MODULES_DEFAULT.admin).toMatchObject({
      supplier_application_admin: true,
    })
  })

  it('an explicit shared setting hides only the matching sidebar entry', () => {
    const config = parseSidebarConfig({
      personal: { supplier_application: false },
    })
    const groups = filterSidebarNavGroups(
      [
        {
          id: 'personal',
          title: 'Personal',
          items: [
            { title: 'Supplier Application', url: '/supplier-applications' },
            { title: 'Ticket Center', url: '/tickets' },
          ],
        },
      ],
      config,
      null
    )
    expect(groups[0]?.items.map((item) => item.url)).toEqual(['/tickets'])
  })

  it('accepts only finite ratios in 0.0001-1000 with at most 4 decimals', () => {
    expect(parseSupplierRatio('1')).toBe(1)
    expect(parseSupplierRatio('0.0001')).toBe(0.0001)
    expect(parseSupplierRatio('1000')).toBe(1000)
    for (const bad of [
      '0',
      '-1',
      '1e3',
      'abc',
      '',
      '1000.0001',
      '0.00001',
      'Infinity',
      'NaN',
      '1,5',
    ]) {
      expect(parseSupplierRatio(bad)).toBeNull()
    }
  })

  it('accepts only http(s) URLs without credentials', () => {
    expect(isValidSupplierUrl('https://example.com/a?b=1')).toBe(true)
    expect(isValidSupplierUrl('http://example.com:8080')).toBe(true)
    for (const bad of [
      'ftp://example.com',
      'javascript:alert(1)',
      'https://',
      'https://u:p@example.com',
      'https://exa mple.com',
      'example.com',
    ]) {
      expect(isValidSupplierUrl(bad)).toBe(false)
    }
  })

  it('limits groups to 1-20', () => {
    const group = {
      group_name: 'g',
      platform: 'openai' as const,
      ratio: '1',
      models: [],
      description: '',
    }
    const base = {
      site_name: 'site',
      site_url: '',
      contact_type: 'wechat' as const,
      contact: 'c',
      remark: '',
    }
    expect(
      supplierApplicationSchema.safeParse({ ...base, groups: [] }).success
    ).toBe(false)
    expect(
      supplierApplicationSchema.safeParse({
        ...base,
        groups: Array.from({ length: 20 }, () => group),
      }).success
    ).toBe(true)
    expect(
      supplierApplicationSchema.safeParse({
        ...base,
        groups: Array.from({ length: 21 }, () => group),
      }).success
    ).toBe(false)
  })
})
