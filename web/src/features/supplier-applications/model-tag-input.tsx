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
import { X } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

import { SUPPLIER_MAX_MODELS, SUPPLIER_MODEL_MAX_LENGTH } from './schema'

interface ModelTagInputProps {
  id: string
  value: string[]
  onChange: (value: string[]) => void
  disabled?: boolean
}

// Type a model name and press Enter to add it as a removable tag.
export function ModelTagInput(props: ModelTagInputProps) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState('')
  const [error, setError] = useState('')

  const addDraft = () => {
    const name = draft.trim()
    if (!name) return
    if ([...name].length > SUPPLIER_MODEL_MAX_LENGTH) {
      setError(t('Model name must be 100 characters or fewer.'))
      return
    }
    if (props.value.includes(name)) {
      setError(t('This model was already added.'))
      return
    }
    if (props.value.length >= SUPPLIER_MAX_MODELS) {
      setError(t('Add at most 20 models per group.'))
      return
    }
    props.onChange([...props.value, name])
    setDraft('')
    setError('')
  }

  return (
    <div className='space-y-2'>
      <Input
        id={props.id}
        value={draft}
        disabled={props.disabled}
        placeholder={t('Type a model name and press Enter')}
        aria-describedby={error ? `${props.id}-error` : undefined}
        onChange={(event) => {
          setDraft(event.target.value)
          setError('')
        }}
        onKeyDown={(event) => {
          // Enter confirms a tag, never the surrounding form; ignore IME
          // composition so confirming a candidate does not add a tag.
          if (event.key !== 'Enter' || event.nativeEvent.isComposing) return
          event.preventDefault()
          addDraft()
        }}
      />
      {error && (
        <p
          id={`${props.id}-error`}
          role='alert'
          className='text-destructive text-sm'
        >
          {error}
        </p>
      )}
      {props.value.length > 0 && (
        <ul className='flex flex-wrap gap-1.5' aria-label={t('Main models')}>
          {props.value.map((name) => (
            <li key={name} className='max-w-full'>
              <Badge
                variant='secondary'
                className='h-auto max-w-full gap-1 py-0.5 whitespace-normal'
              >
                <span className='[overflow-wrap:anywhere]'>{name}</span>
                <Button
                  type='button'
                  variant='ghost'
                  size='icon-xs'
                  disabled={props.disabled}
                  aria-label={t('Remove model {{name}}', { name })}
                  onClick={() =>
                    props.onChange(props.value.filter((item) => item !== name))
                  }
                >
                  <X aria-hidden='true' />
                </Button>
              </Badge>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
