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
import { useId, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Field, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'

import type { McpTool } from '../types'
import { useMcpCategoryLabels } from './tool-categories'

type CatalogTool = Pick<McpTool, 'name' | 'description' | 'category'> &
  Partial<Pick<McpTool, 'write' | 'inputSchema'>>

interface ToolCatalogProps {
  tools: CatalogTool[]
  selected?: string[]
  onChange?: (names: string[]) => void
  disabled?: boolean
  permissions?: boolean
}

export function ToolCatalog(props: ToolCatalogProps) {
  const { t } = useTranslation()
  const categoryLabels = useMcpCategoryLabels()
  const id = useId()
  const [search, setSearch] = useState('')
  const [category, setCategory] = useState('')
  const [access, setAccess] = useState('all')
  const selected = new Set(props.selected ?? [])
  const categories = useMemo(
    () => [...new Set(props.tools.map((tool) => tool.category))].sort(),
    [props.tools]
  )
  const effectiveCategory = categories.includes(category) ? category : ''
  const filtered = useMemo(() => {
    const needle = search.trim().toLowerCase()
    return props.tools.filter(
      (tool) =>
        (!effectiveCategory || tool.category === effectiveCategory) &&
        (access === 'all' ||
          (access === 'unknown'
            ? tool.write === undefined
            : tool.write === (access === 'write'))) &&
        `${tool.name} ${tool.description} ${tool.category} ${categoryLabels[tool.category] ?? ''}`
          .toLowerCase()
          .includes(needle)
    )
  }, [props.tools, search, effectiveCategory, access, categoryLabels])
  const groups = useMemo(() => {
    const result = new Map<string, CatalogTool[]>()
    for (const tool of filtered) {
      const items = result.get(tool.category) ?? []
      items.push(tool)
      result.set(tool.category, items)
    }
    return [...result].sort(([a], [b]) => a.localeCompare(b))
  }, [filtered])
  const reads = filtered.filter((tool) => tool.write === false)
  const writes = filtered.filter((tool) => tool.write === true)
  const unknown = filtered.filter((tool) => tool.write === undefined).length
  const selectedWrites = props.tools.filter(
    (tool) => tool.write && selected.has(tool.name)
  ).length

  function selectTools(tools: CatalogTool[]) {
    props.onChange?.([
      ...new Set([...selected, ...tools.map((tool) => tool.name)]),
    ])
  }

  return (
    <div className='min-w-0 space-y-4'>
      <div className='grid gap-3 sm:grid-cols-[minmax(0,2fr)_minmax(0,1fr)_minmax(0,1fr)]'>
        <Field>
          <FieldLabel htmlFor={`${id}-search`}>{t('Search tools')}</FieldLabel>
          <Input
            id={`${id}-search`}
            value={search}
            placeholder={t('Search by tool name, description or category')}
            aria-describedby={`${id}-search-help`}
            disabled={props.disabled}
            onChange={(event) => setSearch(event.target.value)}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor={`${id}-category`}>{t('Category')}</FieldLabel>
          <NativeSelect
            id={`${id}-category`}
            value={effectiveCategory}
            disabled={props.disabled}
            onChange={(event) => setCategory(event.target.value)}
          >
            <NativeSelectOption value=''>
              {t('All categories')}
            </NativeSelectOption>
            {categories.map((value) => (
              <NativeSelectOption key={value} value={value}>
                {categoryLabels[value] ?? value}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </Field>
        <Field>
          <FieldLabel htmlFor={`${id}-access`}>{t('Access')}</FieldLabel>
          <NativeSelect
            id={`${id}-access`}
            value={access}
            disabled={props.disabled}
            onChange={(event) => setAccess(event.target.value)}
          >
            <NativeSelectOption value='all'>{t('All')}</NativeSelectOption>
            <NativeSelectOption value='read'>
              {t('Read only')}
            </NativeSelectOption>
            <NativeSelectOption value='write'>
              {t('Write access')}
            </NativeSelectOption>
            {props.permissions && (
              <NativeSelectOption value='unknown'>
                {t('Unknown access')}
              </NativeSelectOption>
            )}
          </NativeSelect>
        </Field>
      </div>
      <p id={`${id}-search-help`} className='text-muted-foreground text-xs'>
        {props.permissions
          ? t(
              'Search only the tools authorized for this key. This view does not change permissions.'
            )
          : t(
              'Filter the tool list below. Searching does not change selected permissions.'
            )}
      </p>
      <p role='status' className='text-sm'>
        {t('{{total}} tools · {{read}} read-only · {{write}} write', {
          total: filtered.length,
          read: reads.length,
          write: writes.length,
        })}
        {unknown > 0 && (
          <span> · {t('{{total}} unknown access', { total: unknown })}</span>
        )}
      </p>
      {props.onChange && (
        <>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Write tools can change data or trigger external actions. No tools are authorized by default.'
            )}
          </p>
          <div className='flex flex-wrap gap-2'>
            <Button
              className='h-auto text-left whitespace-normal'
              type='button'
              variant='outline'
              disabled={props.disabled || !reads.length}
              onClick={() => selectTools(reads)}
            >
              {t('Select visible read-only tools')}
            </Button>
            <Button
              className='h-auto text-left whitespace-normal'
              type='button'
              variant='destructive'
              disabled={props.disabled || !writes.length}
              onClick={() => selectTools(writes)}
            >
              {t('Explicitly select visible write tools')}
            </Button>
            <Button
              className='h-auto text-left whitespace-normal'
              type='button'
              variant='outline'
              disabled={
                props.disabled || !effectiveCategory || !filtered.length
              }
              onClick={() => selectTools(filtered)}
            >
              {t('Select all visible tools in this category (including write)')}
            </Button>
            <Button
              className='h-auto text-left whitespace-normal'
              type='button'
              variant='outline'
              disabled={
                props.disabled ||
                !filtered.some((tool) => selected.has(tool.name))
              }
              onClick={() => {
                const visible = new Set(filtered.map((tool) => tool.name))
                props.onChange?.(
                  [...selected].filter((name) => !visible.has(name))
                )
              }}
            >
              {t('Clear visible selection')}
            </Button>
          </div>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Bulk actions apply only to filtered tools. Hidden selections are kept. Choose a category to select all its visible tools.'
            )}
          </p>
          <p role='status' className='text-sm font-medium'>
            {t('{{selected}} selected in all categories · {{write}} write', {
              selected: selected.size,
              write: selectedWrites,
            })}
          </p>
        </>
      )}
      {!filtered.length && (
        <p className='bg-muted rounded-lg border p-4'>
          {t('No matching tools')}
        </p>
      )}
      <div
        role='region'
        aria-label={t('Allowed tools')}
        tabIndex={0}
        className='bg-card focus-visible:ring-ring h-96 min-w-0 space-y-4 overflow-y-auto rounded-lg border p-3 focus-visible:ring-2'
      >
        {groups.map(([name, tools]) => {
          const write = tools.filter((tool) => tool.write).length
          const read = tools.filter((tool) => tool.write === false).length
          return (
            <section
              key={name}
              aria-label={categoryLabels[name] ?? name}
              className='min-w-0 space-y-2'
            >
              <div className='bg-muted/40 flex flex-wrap items-center justify-between gap-2 rounded-md border-b px-2 py-2'>
                <h3 className='text-sm font-semibold break-all'>
                  {categoryLabels[name] ?? name}
                </h3>
                <p className='text-muted-foreground text-sm'>
                  {t('{{total}} tools · {{read}} read-only · {{write}} write', {
                    total: tools.length,
                    read,
                    write,
                  })}
                </p>
              </div>
              {tools.map((tool) => {
                let label = t('Unknown access')
                if (tool.write === true) label = t('Write access')
                if (tool.write === false) label = t('Read only')
                if (props.onChange) {
                  return (
                    <Field
                      key={tool.name}
                      orientation='horizontal'
                      className='items-start border-b px-2 py-2 last:border-b-0'
                    >
                      <Checkbox
                        id={`${id}-${tool.name}`}
                        checked={selected.has(tool.name)}
                        disabled={props.disabled}
                        onCheckedChange={(checked) => {
                          if (checked) selectTools([tool])
                          else {
                            props.onChange?.(
                              [...selected].filter((name) => name !== tool.name)
                            )
                          }
                        }}
                      />
                      <div className='min-w-0'>
                        <FieldLabel
                          htmlFor={`${id}-${tool.name}`}
                          className='flex-wrap break-all'
                        >
                          {tool.name}
                          <Badge
                            variant={tool.write ? 'destructive' : 'secondary'}
                          >
                            {label}
                          </Badge>
                        </FieldLabel>
                        <p className='text-muted-foreground text-sm break-words'>
                          {tool.description}
                        </p>
                      </div>
                    </Field>
                  )
                }
                if (props.permissions) {
                  return (
                    <div
                      key={tool.name}
                      className='flex min-w-0 flex-wrap items-start justify-between gap-2 border-b px-2 py-2 last:border-b-0'
                    >
                      <div className='min-w-0 flex-1 basis-48'>
                        <p className='text-sm font-medium break-all'>
                          {tool.name}
                        </p>
                        <p className='text-muted-foreground mt-1 text-xs break-words'>
                          {tool.description}
                        </p>
                      </div>
                      <Badge
                        variant={
                          tool.write === true ? 'destructive' : 'outline'
                        }
                      >
                        {label}
                      </Badge>
                    </div>
                  )
                }
                return (
                  <details
                    key={tool.name}
                    className='min-w-0 border-b px-2 py-3 last:border-b-0'
                  >
                    <summary className='cursor-pointer font-medium break-words'>
                      <span className='mr-2 font-mono'>{tool.name}</span>
                      <Badge variant={tool.write ? 'destructive' : 'secondary'}>
                        {label}
                      </Badge>
                    </summary>
                    <p className='mt-3 text-sm break-words'>
                      {tool.description}
                    </p>
                    <p className='text-muted-foreground my-2 text-xs'>
                      {t('Input schema')}
                    </p>
                    <pre className='bg-muted overflow-x-auto rounded-md border p-3 text-xs'>
                      <code>{JSON.stringify(tool.inputSchema, null, 2)}</code>
                    </pre>
                  </details>
                )
              })}
            </section>
          )
        })}
      </div>
    </div>
  )
}
