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

import type { SelectHTMLAttributes } from 'react'

type PluginVersionSelectProps = Omit<
  SelectHTMLAttributes<HTMLSelectElement>,
  'value' | 'onChange'
> & {
  options: { value: string; label: string }[]
  value?: string | null
  placeholder?: string
  onValueChange?: (value: string) => void
}
export function PluginVersionSelect(props: PluginVersionSelectProps) {
  const { options, value, placeholder, onValueChange, ...attributes } = props
  return (
    <select
      {...attributes}
      className='border-input bg-background h-9 w-full rounded-md border px-3 text-sm disabled:opacity-50'
      value={value ?? ''}
      onChange={(event) => onValueChange?.(event.target.value)}
    >
      {placeholder && <option value=''>{placeholder}</option>}
      {options.map((option) => (
        <option key={option.value} value={option.value}>
          {option.label}
        </option>
      ))}
    </select>
  )
}
