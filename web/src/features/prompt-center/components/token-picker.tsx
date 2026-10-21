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
import { useQuery } from '@tanstack/react-query'
import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { searchApiKeys } from '@/features/keys/api'

import type { PromptToken } from '../types'

export function TokenPicker(props: {
  onClose: () => void
  onSelect: (token: PromptToken) => void
}) {
  const { t } = useTranslation()
  const fieldId = useId()
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [selectedId, setSelectedId] = useState('')
  const query = useQuery({
    queryKey: ['prompt-center-tokens', search, page],
    queryFn: async () => {
      const result = await searchApiKeys({ keyword: search, p: page, size: 20 })
      if (!result.success || !result.data) {
        throw new Error(result.message || 'Failed to load API keys')
      }
      return result.data
    },
  })
  const selected = query.data?.items.find(
    (token) => String(token.id) === selectedId
  )

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) props.onClose()
      }}
    >
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>{t('Choose an API key')}</DialogTitle>
          <DialogDescription>
            {t(
              'Install prompts without changing the key, quota or group permissions.'
            )}
          </DialogDescription>
        </DialogHeader>
        <div className='bg-card space-y-3 rounded-lg border p-3'>
          <Label htmlFor={`${fieldId}-search`}>{t('Search by name')}</Label>
          <Input
            id={`${fieldId}-search`}
            value={search}
            onChange={(event) => {
              setSearch(event.target.value)
              setPage(1)
              setSelectedId('')
            }}
          />
          {query.isPending && <p role='status'>{t('Loading...')}</p>}
          {query.isError && (
            <div role='alert'>
              <p>{t('Failed to load API keys')}</p>
              <Button variant='outline' onClick={() => query.refetch()}>
                {t('Retry')}
              </Button>
            </div>
          )}
          {query.isSuccess && (
            <>
              <Label htmlFor={`${fieldId}-token`}>{t('API Key')}</Label>
              <select
                id={`${fieldId}-token`}
                className='bg-background w-full min-w-0 rounded-md border p-2 text-sm'
                value={selectedId}
                onChange={(event) => setSelectedId(event.target.value)}
              >
                <option value=''>{t('Select API key')}</option>
                {query.data.items.map((token) => (
                  <option value={token.id} key={token.id}>
                    {token.name}
                  </option>
                ))}
              </select>
              {query.data.total === 0 && (
                <p className='text-muted-foreground text-sm'>
                  {t('No API keys found')}
                </p>
              )}
              <div className='flex items-center justify-between gap-2'>
                <Button
                  variant='outline'
                  size='sm'
                  disabled={page === 1 || query.isFetching}
                  onClick={() => {
                    setPage((value) => value - 1)
                    setSelectedId('')
                  }}
                >
                  {t('Previous')}
                </Button>
                <span className='text-muted-foreground text-xs'>
                  {t('Page {{page}}', { page })}
                </span>
                <Button
                  variant='outline'
                  size='sm'
                  disabled={page * 20 >= query.data.total || query.isFetching}
                  onClick={() => {
                    setPage((value) => value + 1)
                    setSelectedId('')
                  }}
                >
                  {t('Next')}
                </Button>
              </div>
            </>
          )}
        </div>
        <Button
          disabled={!selected || query.isFetching}
          onClick={() => {
            if (selected) props.onSelect(selected)
          }}
        >
          {t('Continue')}
        </Button>
      </DialogContent>
    </Dialog>
  )
}
