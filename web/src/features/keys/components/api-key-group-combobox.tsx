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
import { Check, ChevronsUpDown } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { GroupTagFilter } from '@/features/group-tags/components/group-tag-filter'
import type { GroupTagOption } from '@/features/group-tags/types'
import { useMediaQuery } from '@/hooks'
import { cn } from '@/lib/utils'

import {
  AUTO_GROUP_FRAME_CLASS_NAME,
  AutoGroupFlowBorder,
  GroupRatioBadge,
} from './auto-group-visuals'

export type ApiKeyGroupOption = {
  id?: number
  value: string
  label: string
  desc?: string
  ratio?: number | string
  exclusive?: boolean
}

type ApiKeyGroupComboboxProps = Pick<
  React.ComponentProps<'button'>,
  'id' | 'aria-describedby' | 'aria-invalid'
> & {
  options: ApiKeyGroupOption[]
  value?: string
  selectedValues?: string[]
  onValueChange: (value: string) => void
  placeholder?: string
  disabled?: boolean
  disabledValues?: readonly string[]
  tags?: GroupTagOption[]
  activeTag?: string
  onActiveTagChange?: (value: string) => void
  hideTagFilter?: boolean
  open?: boolean
  onOpenChange?: (open: boolean) => void
  // Element that opened the picker from outside (for example a tag card);
  // focus returns to it once on close, then the ref is cleared.
  returnFocusRef?: React.RefObject<HTMLElement | null>
  // Render only the popup (no trigger button) anchored to `anchorRef`; used
  // when tag cards are the sole entry points of the picker.
  anchorRef?: React.RefObject<HTMLElement | null>
}

export function ApiKeyGroupCombobox({
  options,
  value,
  selectedValues,
  onValueChange,
  placeholder,
  disabled,
  disabledValues,
  tags,
  activeTag: controlledTag,
  onActiveTagChange,
  hideTagFilter,
  open: controlledOpen,
  onOpenChange,
  returnFocusRef,
  anchorRef,
  ...triggerProps
}: ApiKeyGroupComboboxProps) {
  const { t } = useTranslation()
  const [innerOpen, setInnerOpen] = useState(false)
  const open = controlledOpen ?? innerOpen
  const setOpen = onOpenChange ?? setInnerOpen
  const [searchValue, setSearchValue] = useState('')
  const [innerTag, setInnerTag] = useState('all')
  const activeTag = controlledTag ?? innerTag
  const setActiveTag = onActiveTagChange ?? setInnerTag
  const availableTags = useMemo(
    () =>
      (tags ?? []).filter((tag) =>
        options.some(
          (option) =>
            option.id !== undefined &&
            option.value !== 'auto' &&
            tag.group_ids.includes(option.id)
        )
      ),
    [tags, options]
  )
  const selectedTag = availableTags.find((tag) => String(tag.id) === activeTag)
  useEffect(() => {
    if (activeTag !== 'all' && !selectedTag) setActiveTag('all')
  }, [activeTag, selectedTag, setActiveTag])
  const shouldReduceMotion = useMediaQuery('(prefers-reduced-motion: reduce)')
  const { selectedOptions, selectedValueSet } = useMemo(() => {
    const activeValues = selectedValues ?? (value ? [value] : [])
    const optionsByValue = new Map(
      options.map((option) => [option.value, option])
    )
    return {
      selectedOptions: activeValues.flatMap((activeValue) => {
        const option = optionsByValue.get(activeValue)
        return option ? [option] : []
      }),
      selectedValueSet: new Set(activeValues),
    }
  }, [options, selectedValues, value])
  const selectedOption =
    selectedOptions.length === 1 ? selectedOptions[0] : undefined
  const isAutoSelected = selectedValueSet.has('auto')

  const disabledValueSet = useMemo(
    () => new Set(disabledValues ?? []),
    [disabledValues]
  )

  const filteredOptions = useMemo(() => {
    const search = searchValue.trim().toLowerCase()
    // Auto has no group ID, so it is only offered first under "All groups".
    const ordered = selectedTag
      ? options
      : [
          ...options.filter((option) => option.value === 'auto'),
          ...options.filter((option) => option.value !== 'auto'),
        ]
    return ordered.filter((option) => {
      if (
        selectedTag &&
        (option.id === undefined || !selectedTag.group_ids.includes(option.id))
      ) {
        return false
      }
      if (!search) return true
      const ratioText = String(option.ratio ?? '').toLowerCase()
      return (
        option.value.toLowerCase().includes(search) ||
        option.label.toLowerCase().includes(search) ||
        option.desc?.toLowerCase().includes(search) ||
        ratioText.includes(search)
      )
    })
  }, [options, searchValue, selectedTag])

  const handleSelect = (selectedValue: string) => {
    onValueChange(selectedValue)
    setOpen(false)
    setSearchValue('')
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      {!anchorRef && (
        <PopoverTrigger
          render={
            <Button
              {...triggerProps}
              type='button'
              variant='outline'
              role='combobox'
              aria-expanded={open}
              data-auto-group-effect={isAutoSelected ? 'trigger' : undefined}
              disabled={disabled}
              className={cn(
                'border-input bg-muted/40 hover:bg-muted/55 hover:text-foreground active:bg-background data-popup-open:border-ring data-popup-open:bg-background data-popup-open:ring-ring/20 relative h-auto min-h-14 w-full justify-between gap-2 rounded-lg px-3 py-2 text-start shadow-none transition-[background-color,border-color,box-shadow] duration-150 data-popup-open:ring-[3px] sm:min-h-20 sm:gap-3 sm:px-4 sm:py-3',
                isAutoSelected &&
                  cn(
                    AUTO_GROUP_FRAME_CLASS_NAME,
                    'hover:border-primary/55 data-popup-open:border-primary/55 data-popup-open:ring-primary/20'
                  )
              )}
            />
          }
        >
          {isAutoSelected && (
            <AutoGroupFlowBorder shouldReduceMotion={shouldReduceMotion} />
          )}
          <span className='flex min-w-0 flex-1 items-center justify-between gap-2 sm:gap-3'>
            <span className='min-w-0'>
              <span className='block truncate font-medium'>
                {selectedOptions.length > 1
                  ? t('Add group')
                  : selectedOption?.label || placeholder || t('Select a group')}
              </span>
              {selectedOption?.desc && (
                <span className='text-muted-foreground block truncate text-[11px] sm:text-xs'>
                  {selectedOption.desc}
                </span>
              )}
            </span>
            <span className='hidden sm:block'>
              <GroupRatioBadge
                ratio={selectedOption?.ratio}
                isAuto={isAutoSelected}
                shouldReduceMotion={shouldReduceMotion}
              />
            </span>
          </span>
          <ChevronsUpDown
            aria-hidden='true'
            className='size-4 shrink-0 opacity-50'
          />
        </PopoverTrigger>
      )}
      <PopoverContent
        anchor={anchorRef}
        className='data-closed:zoom-out-100 data-open:zoom-in-100 data-[side=bottom]:slide-in-from-top-0 data-[side=left]:slide-in-from-right-0 data-[side=right]:slide-in-from-left-0 data-[side=top]:slide-in-from-bottom-0 max-h-[var(--available-height)] w-[var(--anchor-width)] max-w-[calc(100vw-1rem)] overflow-hidden rounded-xl p-0 shadow-lg data-closed:duration-75 data-open:duration-100'
        finalFocus={() => {
          const element = returnFocusRef?.current
          if (returnFocusRef) returnFocusRef.current = null
          return element?.isConnected ? element : true
        }}
        onWheel={(event) => event.stopPropagation()}
        onTouchMove={(event) => event.stopPropagation()}
        onPointerDown={(event) => event.stopPropagation()}
      >
        {!hideTagFilter && (tags?.length ?? 0) > 0 && (
          <GroupTagFilter
            tags={availableTags}
            value={activeTag}
            onChange={(value) => {
              setActiveTag(value)
              setSearchValue('')
            }}
            disabled={disabled}
          />
        )}
        <Command shouldFilter={false} className='min-h-0 flex-1'>
          <CommandInput
            placeholder={t('Search...')}
            value={searchValue}
            onValueChange={setSearchValue}
          />
          <CommandList className='max-h-[360px] min-h-0 flex-1'>
            <CommandEmpty>{t('No group found.')}</CommandEmpty>
            <CommandGroup>
              {filteredOptions.map((option) => {
                const isAutoOption = option.value === 'auto'
                const isOptionDisabled = disabledValueSet.has(option.value)

                return (
                  <CommandItem
                    key={option.value}
                    value={option.value}
                    disabled={isOptionDisabled}
                    data-auto-group-effect={isAutoOption ? 'option' : undefined}
                    onSelect={() => {
                      if (isOptionDisabled) return
                      handleSelect(option.value)
                    }}
                    className={cn(
                      'data-[selected=true]:bg-muted items-start gap-3 rounded-lg px-3 py-3 transition-colors',
                      isAutoOption &&
                        cn(
                          AUTO_GROUP_FRAME_CLASS_NAME,
                          'border-primary/35 data-[selected=true]:border-primary/55'
                        )
                    )}
                  >
                    {isAutoOption && (
                      <AutoGroupFlowBorder
                        shouldReduceMotion={shouldReduceMotion}
                      />
                    )}
                    <Check
                      aria-hidden='true'
                      className={cn(
                        'mt-0.5 size-4',
                        selectedValueSet.has(option.value)
                          ? 'opacity-100'
                          : 'opacity-0'
                      )}
                    />
                    <span className='min-w-0 flex-1'>
                      <span className='block truncate font-medium'>
                        {option.label}
                      </span>
                      {option.desc && (
                        <span className='text-muted-foreground block truncate text-xs'>
                          {option.desc}
                        </span>
                      )}
                    </span>
                    <GroupRatioBadge
                      ratio={option.ratio}
                      isAuto={isAutoOption}
                      shouldReduceMotion={shouldReduceMotion}
                    />
                  </CommandItem>
                )
              })}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}
