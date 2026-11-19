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
import { Check, X } from 'lucide-react'
import { useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { MultiSelect } from '@/components/multi-select'
import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import { Popover, PopoverContent } from '@/components/ui/popover'
import type { GroupTagOption } from '@/features/group-tags/types'
import { GroupTagCards } from '@/features/keys/components/group-tag-cards'
import { cn } from '@/lib/utils'

export type ChannelGroupOption = {
  id?: number
  value: string
  label: string
}

type ChannelGroupPickerProps = {
  options: ChannelGroupOption[]
  selected: string[]
  onChange: (values: string[]) => void
  tags: GroupTagOption[]
  placeholder?: string
  disabled?: boolean
}

// Channel "Groups" field. With group tags, the tag cards are the entry points
// of a searchable multi-select popover filtered by stable group ID; picks
// accumulate across tags and never reorder existing values. Without usable
// tags the original chip multi-select is kept unchanged.
export function ChannelGroupPicker(props: ChannelGroupPickerProps) {
  const { t } = useTranslation()
  const [activeTag, setActiveTag] = useState('all')
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const cardsRef = useRef<HTMLDivElement | null>(null)
  const returnFocusRef = useRef<HTMLElement | null>(null)

  const usableTags = useMemo(
    () =>
      props.tags.filter((tag) =>
        props.options.some(
          (option) =>
            option.id !== undefined && tag.group_ids.includes(option.id)
        )
      ),
    [props.tags, props.options]
  )
  const currentTag = usableTags.find((tag) => String(tag.id) === activeTag)
  const selectedSet = useMemo(() => new Set(props.selected), [props.selected])
  const labelByValue = useMemo(
    () => new Map(props.options.map((option) => [option.value, option.label])),
    [props.options]
  )

  const visibleOptions = useMemo(() => {
    const keyword = search.trim().toLowerCase()
    return props.options.filter((option) => {
      if (
        currentTag &&
        (option.id === undefined || !currentTag.group_ids.includes(option.id))
      ) {
        return false
      }
      if (!keyword) return true
      return (
        option.label.toLowerCase().includes(keyword) ||
        option.value.toLowerCase().includes(keyword)
      )
    })
  }, [props.options, currentTag, search])

  if (usableTags.length === 0) {
    return (
      <MultiSelect
        options={props.options}
        selected={props.selected}
        onChange={props.onChange}
        placeholder={props.placeholder}
        disabled={props.disabled}
      />
    )
  }

  const toggle = (value: string) => {
    if (selectedSet.has(value)) {
      props.onChange(props.selected.filter((item) => item !== value))
      return
    }
    props.onChange([...props.selected, value])
  }

  return (
    <div className='flex min-w-0 flex-col gap-2'>
      <GroupTagCards
        tags={usableTags}
        options={props.options}
        value={activeTag}
        onChange={(value) => {
          setActiveTag(value)
          setSearch('')
        }}
        expanded={open}
        containerRef={cardsRef}
        onActivate={(element) => {
          returnFocusRef.current = element
          setOpen(true)
        }}
        disabled={props.disabled}
      />
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverContent
          anchor={cardsRef}
          aria-label={currentTag?.name ?? t('All groups')}
          className='max-h-[var(--available-height)] w-[var(--anchor-width)] max-w-[calc(100vw-1rem)] gap-0 overflow-hidden rounded-xl p-0 shadow-lg'
          finalFocus={() => {
            const element = returnFocusRef.current
            returnFocusRef.current = null
            return element?.isConnected ? element : true
          }}
          onWheel={(event) => event.stopPropagation()}
          onTouchMove={(event) => event.stopPropagation()}
          onPointerDown={(event) => event.stopPropagation()}
        >
          <Command shouldFilter={false} className='min-h-0 flex-1'>
            <CommandInput
              placeholder={t('Search...')}
              value={search}
              onValueChange={setSearch}
            />
            <CommandList
              aria-multiselectable='true'
              className='max-h-[min(360px,50vh)] min-h-0 flex-1'
            >
              <CommandEmpty>{t('No group found.')}</CommandEmpty>
              <CommandGroup>
                {visibleOptions.map((option) => {
                  const checked = selectedSet.has(option.value)
                  return (
                    <CommandItem
                      key={option.value}
                      value={option.value}
                      aria-checked={checked}
                      onSelect={() => toggle(option.value)}
                      className='data-[selected=true]:bg-muted min-w-0 gap-3 rounded-lg px-3 py-2.5'
                    >
                      <span
                        aria-hidden='true'
                        className={cn(
                          'flex size-4 shrink-0 items-center justify-center rounded-[4px] border',
                          checked &&
                            'border-primary bg-primary text-primary-foreground'
                        )}
                      >
                        {checked && <Check className='size-3' />}
                      </span>
                      <span
                        title={option.label}
                        className='min-w-0 flex-1 truncate font-medium'
                      >
                        {option.label}
                      </span>
                    </CommandItem>
                  )
                })}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
      <div
        role='group'
        aria-label={t('Selected groups')}
        className='bg-muted/20 flex min-w-0 flex-col gap-2 rounded-lg border p-2'
      >
        <div className='flex min-w-0 items-center justify-between gap-2'>
          <span className='text-muted-foreground min-w-0 truncate text-xs'>
            {t('Selected groups')} ({props.selected.length})
          </span>
          <Button
            type='button'
            variant='ghost'
            size='sm'
            disabled={props.disabled || props.selected.length === 0}
            onClick={() => props.onChange([])}
          >
            {t('Clear all')}
          </Button>
        </div>
        {props.selected.length === 0 ? (
          <span className='text-muted-foreground px-1 text-sm'>
            {props.placeholder}
          </span>
        ) : (
          <ul className='flex max-h-40 min-w-0 flex-wrap gap-1.5 overflow-y-auto'>
            {props.selected.map((value) => {
              const label = labelByValue.get(value) ?? value
              return (
                <li
                  key={value}
                  className='bg-background flex max-w-full min-w-0 items-center gap-1 rounded-md border py-0.5 ps-2 pe-0.5 text-sm'
                >
                  <span title={label} className='min-w-0 truncate'>
                    {label}
                  </span>
                  <Button
                    type='button'
                    variant='ghost'
                    size='icon-xs'
                    disabled={props.disabled}
                    aria-label={t('Remove {{group}}', { group: label })}
                    onClick={() => toggle(value)}
                  >
                    <X aria-hidden='true' />
                  </Button>
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </div>
  )
}
