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
import * as z from 'zod'

export const groupTagFormSchema = z.object({
  name: z.string().trim().min(1).max(64),
  description: z.string().trim().max(512),
  sort_order: z.number().int().min(-1000000).max(1000000),
  icons: z
    .string()
    .transform((value) =>
      value
        .split('\n')
        .map((item) => item.trim())
        .filter(Boolean)
    )
    .pipe(
      z
        .array(
          z
            .string()
            .max(2048)
            .refine((value) => {
              if (
                /^[A-Za-z][A-Za-z0-9]*(\.[A-Za-z][A-Za-z0-9]*)*$/.test(value)
              ) {
                return true
              }
              try {
                const url = new URL(value)
                return (
                  (url.protocol === 'http:' || url.protocol === 'https:') &&
                  !!url.hostname &&
                  !url.username &&
                  !url.password &&
                  !/[\r\n\t ]/.test(value)
                )
              } catch {
                return false
              }
            })
        )
        .max(6)
    ),
  match_keywords: z
    .string()
    .transform((value) =>
      value
        .split('\n')
        .map((item) => item.trim())
        .filter(Boolean)
    )
    .pipe(z.array(z.string().max(128)).max(32)),
  group_ids: z.array(z.number().int().positive()).max(10000),
})
