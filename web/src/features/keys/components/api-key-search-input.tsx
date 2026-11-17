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
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Input } from '@/components/ui/input'

export function ApiKeySearchInput(props: {
  field: 'name' | 'key'
  value: string
  onCommit: (value: string) => void
}) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState<{ base: string; value: string } | null>(
    null
  )
  const [composing, setComposing] = useState(false)
  const value = draft?.base === props.value ? draft.value : props.value
  const onCommit = useRef(props.onCommit)
  onCommit.current = props.onCommit
  if (draft && draft.base !== props.value) setDraft(null)

  useEffect(() => {
    if (composing || value === props.value) return
    // URL 切换会取消旧草稿的提交，后退时不能再写回前一页的防抖值。
    const timer = window.setTimeout(() => onCommit.current(value), 500)
    return () => window.clearTimeout(timer)
  }, [value, props.value, composing])

  return (
    <Input
      placeholder={
        props.field === 'name'
          ? t('Filter by name...')
          : t('Filter by API key...')
      }
      aria-label={
        props.field === 'name'
          ? t('Filter by name...')
          : t('Filter by API key...')
      }
      value={value}
      onChange={(event) =>
        setDraft({ base: props.value, value: event.target.value })
      }
      onCompositionStart={() => setComposing(true)}
      onCompositionEnd={(event) => {
        setComposing(false)
        setDraft({ base: props.value, value: event.currentTarget.value })
      }}
      className='w-full sm:w-50 lg:w-60'
    />
  )
}
