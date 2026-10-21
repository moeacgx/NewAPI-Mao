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

export const promptTemplateSchema = z
  .object({
    title: z.string().trim().min(1).max(80),
    description: z.string().max(512),
    content: z
      .string()
      .refine((value) => value.trim().length > 0)
      .refine(
        (value) => new TextEncoder().encode(value).length <= 32768,
        'Prompt content must not exceed 32768 bytes'
      ),
    enabled: z.boolean(),
    sort_order: z.number().int().min(-1000000).max(1000000),
    version: z.number().int().min(0),
    group_ids: z.array(z.number().int().positive()).max(1000),
  })
  .refine((value) => !value.enabled || value.group_ids.length > 0, {
    path: ['group_ids'],
    message: 'Select at least one allowed group',
  })

export type PromptTemplateForm = z.infer<typeof promptTemplateSchema>
