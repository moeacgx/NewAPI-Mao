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
import { Check, Layers3, X } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import type { GroupDetail } from '@/features/system-settings/types'
import { getLobeIcon } from '@/lib/lobe-icon'

import type { PromptGroupTag } from '../types'

type PromptGroupSelectorProps = {
  groups: GroupDetail[]
  tags: PromptGroupTag[]
  selectedIds: number[]
  disabled?: boolean
  onChange: (ids: number[]) => void
}

export function PromptGroupSelector(props: PromptGroupSelectorProps) {
  const { t } = useTranslation()
  const [activeTag, setActiveTag] = useState('all')
  const [openTag, setOpenTag] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const groupById = useMemo(
    () => new Map(props.groups.map((group) => [group.id, group])),
    [props.groups]
  )
  const taggedIds = useMemo(
    () => new Set(props.tags.flatMap((tag) => tag.group_ids)),
    [props.tags]
  )
  const choices = useMemo(() => {
    const availableGroups = props.groups.filter((group) => group.id > 0)
    const untaggedCount = availableGroups.filter(
      (group) => !taggedIds.has(group.id)
    ).length
    return [
      {
        id: 'all',
        name: t('All groups'),
        icons: [],
        count: availableGroups.length,
      },
      ...(untaggedCount > 0
        ? [
            {
              id: 'untagged',
              name: t('Untagged'),
              icons: [],
              count: untaggedCount,
            },
          ]
        : []),
      ...props.tags.map((tag) => ({
        ...tag,
        id: String(tag.id),
        count: availableGroups.filter((group) =>
          tag.group_ids.includes(group.id)
        ).length,
      })),
    ].filter((choice) => choice.id === 'all' || choice.count > 0)
  }, [props.groups, props.tags, taggedIds, t])
  const selectedTag = choices.some((choice) => choice.id === activeTag)
    ? activeTag
    : 'all'
  const visibleGroups = useMemo(() => {
    const query = search.trim().toLocaleLowerCase()
    return props.groups.filter((group) => {
      if (group.id <= 0) return false
      if (selectedTag === 'untagged' && taggedIds.has(group.id)) return false
      if (selectedTag !== 'all' && selectedTag !== 'untagged') {
        const tag = props.tags.find((item) => String(item.id) === selectedTag)
        if (!tag?.group_ids.includes(group.id)) return false
      }
      return (
        !query ||
        (group.name || group.code || '').toLocaleLowerCase().includes(query) ||
        (group.code || '').toLocaleLowerCase().includes(query)
      )
    })
  }, [selectedTag, props.groups, props.tags, search, taggedIds])
  const selectedGroups = props.selectedIds.map((id) => ({
    id,
    name:
      groupById.get(id)?.name ||
      groupById.get(id)?.code ||
      t('Deleted ({{id}})', { id }),
  }))

  const toggleGroup = (id: number, checked: boolean) => {
    props.onChange(
      checked
        ? [...props.selectedIds, id]
        : props.selectedIds.filter((selectedId) => selectedId !== id)
    )
  }

  return (
    <div className='space-y-3'>
      {props.tags.length > 0 && (
        <fieldset className='space-y-2'>
          <legend className='text-sm font-medium'>{t('Group tags')}</legend>
          <div className='grid grid-cols-2 gap-2 sm:grid-cols-3'>
            {choices.map((choice) => (
              <Popover
                key={choice.id}
                open={openTag === choice.id}
                onOpenChange={(open) =>
                  setOpenTag((current) => {
                    if (open) return choice.id
                    return current === choice.id ? null : current
                  })
                }
              >
                <PopoverTrigger
                  render={
                    <button
                      type='button'
                      aria-pressed={selectedTag === choice.id}
                      disabled={props.disabled}
                      onClick={() => {
                        setActiveTag(choice.id)
                        setSearch('')
                      }}
                      className={`bg-background hover:border-primary/60 flex min-w-0 flex-col items-center gap-1.5 rounded-xl border px-2 py-3 text-center transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${selectedTag === choice.id ? 'border-primary bg-primary/5 ring-primary ring-1' : 'border-border'}`}
                    />
                  }
                >
                  <span
                    className='flex min-h-8 max-w-24 flex-wrap items-center justify-center gap-1'
                    aria-hidden='true'
                  >
                    {choice.icons.length > 0 ? (
                      [...new Set(choice.icons)].map((icon) => (
                        <span
                          key={icon}
                          className='bg-muted flex size-7 items-center justify-center rounded-lg'
                        >
                          {/^https?:\/\//i.test(icon) ? (
                            <img
                              src={icon}
                              alt=''
                              loading='lazy'
                              className='size-5 object-contain'
                              onError={(event) => {
                                event.currentTarget.hidden = true
                              }}
                            />
                          ) : (
                            getLobeIcon(icon, 20)
                          )}
                        </span>
                      ))
                    ) : (
                      <span className='bg-muted text-muted-foreground flex size-7 items-center justify-center rounded-lg'>
                        {choice.id === 'all' ? (
                          <Layers3 className='size-4' />
                        ) : (
                          <Check className='size-4' />
                        )}
                      </span>
                    )}
                  </span>
                  <span
                    className='w-full truncate text-sm font-medium'
                    title={choice.name}
                  >
                    {choice.name}
                  </span>
                  <span className='text-muted-foreground text-xs'>
                    {t('Groups: {{count}}', { count: choice.count })}
                  </span>
                </PopoverTrigger>
                <PopoverContent
                  align='start'
                  aria-label={choice.name}
                  aria-hidden={openTag !== choice.id || undefined}
                  className='w-[min(360px,calc(100vw-32px))]'
                >
                  {openTag === choice.id && (
                    <GroupWhitelistOptions
                      groups={visibleGroups}
                      selectedIds={props.selectedIds}
                      search={search}
                      onSearchChange={setSearch}
                      onToggle={toggleGroup}
                      disabled={props.disabled}
                    />
                  )}
                </PopoverContent>
              </Popover>
            ))}
          </div>
        </fieldset>
      )}
      {props.tags.length === 0 && (
        <GroupWhitelistOptions
          groups={visibleGroups}
          selectedIds={props.selectedIds}
          search={search}
          onSearchChange={setSearch}
          onToggle={toggleGroup}
          disabled={props.disabled}
        />
      )}

      {selectedGroups.length > 0 && (
        <div className='space-y-1.5'>
          <div className='text-muted-foreground flex items-center gap-1.5 text-xs font-medium'>
            <Layers3 className='size-3.5' aria-hidden='true' />
            {t('Selected groups')} ({selectedGroups.length})
          </div>
          <div className='flex flex-wrap gap-1.5'>
            {selectedGroups.map((group) => (
              <span
                key={group.id}
                className='bg-muted/40 inline-flex max-w-full items-center gap-1 rounded-md border px-2 py-1 text-xs'
              >
                <span className='max-w-48 truncate' title={group.name}>
                  {group.name}
                </span>
                <Button
                  type='button'
                  variant='ghost'
                  size='icon-xs'
                  aria-label={t('Remove group {{name}}', { name: group.name })}
                  disabled={props.disabled}
                  onClick={() => toggleGroup(group.id, false)}
                  className='size-5'
                >
                  <X aria-hidden='true' />
                </Button>
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function GroupWhitelistOptions(props: {
  groups: GroupDetail[]
  selectedIds: number[]
  search: string
  onSearchChange: (search: string) => void
  onToggle: (id: number, checked: boolean) => void
  disabled?: boolean
}) {
  const { t } = useTranslation()
  return (
    <div className='space-y-2'>
      <Input
        value={props.search}
        onChange={(event) => props.onSearchChange(event.target.value)}
        placeholder={t('Search groups...')}
        aria-label={t('Search groups...')}
        disabled={props.disabled}
      />
      <div className='bg-background grid max-h-48 gap-1 overflow-y-auto rounded-lg border p-2'>
        {props.groups.map((group) => (
          <label
            key={group.id}
            className='hover:bg-muted/50 flex min-w-0 items-center gap-2 rounded-md px-2 py-1.5 text-sm'
          >
            <input
              type='checkbox'
              checked={props.selectedIds.includes(group.id)}
              disabled={props.disabled}
              onChange={(event) =>
                props.onToggle(group.id, event.target.checked)
              }
            />
            <span className='min-w-0 truncate' title={group.name || group.code}>
              {group.name || group.code}
            </span>
          </label>
        ))}
        {props.groups.length === 0 && (
          <p className='text-muted-foreground px-2 py-3 text-center text-sm'>
            {t('No available groups')}
          </p>
        )}
      </div>
    </div>
  )
}
