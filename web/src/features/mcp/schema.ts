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

export const mcpKeySchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, 'Enter a key name')
    .refine(
      (value) => [...value].length <= 100,
      'Use no more than 100 characters'
    ),
  expires: z.string().refine((value) => {
    const seconds = Math.floor(new Date(value).getTime() / 1000)
    return (
      Number.isFinite(seconds) &&
      seconds > Math.floor(Date.now() / 1000) &&
      seconds <= 253402300799
    )
  }, 'Choose a future expiration time'),
  tools: z.array(z.string()).min(1, 'Select at least one tool'),
})

export type McpKeyFormValues = z.infer<typeof mcpKeySchema>
