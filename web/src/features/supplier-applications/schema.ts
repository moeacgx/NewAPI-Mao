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
import { z } from 'zod'

// Mirrors the backend limits in model/supplier_application.go.
export const SUPPLIER_MAX_GROUPS = 20
export const SUPPLIER_MAX_MODELS = 20
export const SUPPLIER_MODEL_MAX_LENGTH = 100
export const SUPPLIER_RATIO_MIN = 0.0001
export const SUPPLIER_RATIO_MAX = 1000
export const SUPPLIER_ADMIN_NOTE_MAX_LENGTH = 2000
export const SUPPLIER_QUANTITY_MIN = 0.0001
export const SUPPLIER_QUANTITY_MAX = 1000000000

export const supplierContactTypes = [
  'wechat',
  'qq',
  'telegram',
  'email',
  'phone',
  'other',
] as const
export const supplierPlatforms = [
  'openai',
  'anthropic',
  'gemini',
  'grok',
  'deepseek',
  'other',
] as const
export const supplierStatuses = [
  'pending',
  'contacted',
  'cooperating',
  'rejected',
] as const

export const supplierManagementSearchSchema = z.object({
  status: z
    .enum(['pending', 'contacted', 'cooperating', 'rejected', ''])
    .catch(''),
})

// Single-line text must not contain control characters or line breaks.
const singleLine = (value: string) =>
  // eslint-disable-next-line no-control-regex
  !/[\u0000-\u001f\u007f]/.test(value)

export function isValidSupplierUrl(value: string): boolean {
  if (/\s/.test(value)) return false
  try {
    const url = new URL(value)
    return (
      (url.protocol === 'http:' || url.protocol === 'https:') &&
      url.hostname !== '' &&
      url.username === '' &&
      url.password === ''
    )
  } catch {
    return false
  }
}

export function parseSupplierRatio(value: string): number | null {
  const text = value.trim()
  if (!/^\d+(\.\d{1,4})?$/.test(text)) return null
  const ratio = Number(text)
  if (!Number.isFinite(ratio)) return null
  if (ratio < SUPPLIER_RATIO_MIN || ratio > SUPPLIER_RATIO_MAX) return null
  return ratio
}

export function parseSupplierQuantity(value: string): number | null {
  const text = value.trim()
  if (!/^\d+(\.\d{1,4})?$/.test(text)) return null
  const quantity = Number(text)
  if (!Number.isFinite(quantity)) return null
  if (quantity < SUPPLIER_QUANTITY_MIN || quantity > SUPPLIER_QUANTITY_MAX) {
    return null
  }
  return quantity
}

export const supplierQuoteSchema = z.object({
  group_name: z
    .string()
    .trim()
    .min(1, 'Enter a group name.')
    .refine(singleLine, 'Enter a group name.')
    .refine(
      (value) => [...value].length <= 64,
      'Group name must be 64 characters or fewer.'
    ),
  platform: z.enum(supplierPlatforms),
  ratio: z
    .string()
    .trim()
    .min(1, 'Enter a ratio.')
    .refine(
      (value) => parseSupplierRatio(value) !== null,
      'Ratio must be between 0.0001 and 1000 with at most 4 decimals.'
    ),
  models: z
    .array(z.string().trim().min(1).max(SUPPLIER_MODEL_MAX_LENGTH))
    .max(SUPPLIER_MAX_MODELS, 'Add at most 20 models per group.'),
  description: z
    .string()
    .trim()
    .refine(
      (value) => [...value].length <= 500,
      'Notes must be 500 characters or fewer.'
    ),
})

export const supplierApplicationSchema = z.object({
  site_name: z
    .string()
    .trim()
    .min(1, 'Enter a site name.')
    .refine(singleLine, 'Enter a site name.')
    .refine(
      (value) => [...value].length <= 100,
      'Site name must be 100 characters or fewer.'
    ),
  site_url: z
    .string()
    .trim()
    .refine(
      (value) => [...value].length <= 500,
      'Site URL must be 500 characters or fewer.'
    )
    .refine(
      (value) => value === '' || isValidSupplierUrl(value),
      'Enter a valid http:// or https:// address.'
    ),
  contact_type: z.enum(supplierContactTypes),
  contact: z
    .string()
    .trim()
    .min(1, 'Enter your contact details.')
    .refine(singleLine, 'Enter your contact details.')
    .refine(
      (value) => [...value].length <= 200,
      'Contact must be 200 characters or fewer.'
    ),
  remark: z
    .string()
    .trim()
    .refine(
      (value) => [...value].length <= 1000,
      'Remarks must be 1000 characters or fewer.'
    ),
  groups: z
    .array(supplierQuoteSchema)
    .min(1, 'Add at least one group.')
    .max(SUPPLIER_MAX_GROUPS, 'Add at most 20 groups.'),
})

export type SupplierApplicationFormValues = z.infer<
  typeof supplierApplicationSchema
>

export const supplierReviewSchema = z.object({
  status: z.enum(supplierStatuses),
  admin_note: z
    .string()
    .trim()
    .refine(
      (value) => [...value].length <= SUPPLIER_ADMIN_NOTE_MAX_LENGTH,
      'Internal note must be 2000 characters or fewer.'
    ),
})

export type SupplierReviewFormValues = z.infer<typeof supplierReviewSchema>
