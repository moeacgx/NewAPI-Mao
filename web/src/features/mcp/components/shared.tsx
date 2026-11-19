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
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { toIntlLocale } from '@/i18n/languages'

import { MCP_PAGE_SIZE } from '../api'

export function CopyButton(props: { text: string; label: string }) {
  const { copyToClipboard } = useCopyToClipboard()
  return (
    <Button
      type='button'
      variant='outline'
      onClick={() => void copyToClipboard(props.text)}
    >
      {props.label}
    </Button>
  )
}

export function McpPagination(props: {
  page: number
  total: number
  busy: boolean
  onChange: (page: number) => void
}) {
  const { t } = useTranslation()
  const pages = Math.max(1, Math.ceil(props.total / MCP_PAGE_SIZE))
  return (
    <nav
      aria-label={t('Pagination')}
      className='flex flex-wrap items-center justify-end gap-3 pt-4'
    >
      <span className='text-muted-foreground text-sm'>
        {props.busy
          ? t('Loading...')
          : t('Page {{page}} of {{pages}}', { page: props.page, pages })}
      </span>
      <Button
        type='button'
        variant='outline'
        disabled={props.busy || props.page <= 1}
        onClick={() => props.onChange(props.page - 1)}
      >
        {t('Previous')}
      </Button>
      <Button
        type='button'
        variant='outline'
        disabled={props.busy || props.page >= pages}
        onClick={() => props.onChange(props.page + 1)}
      >
        {t('Next')}
      </Button>
    </nav>
  )
}

export function McpTime(props: { value: number }) {
  const { i18n } = useTranslation()
  if (!Number.isFinite(props.value) || props.value <= 0) return <>—</>
  const date = new Date(props.value * 1000)
  if (!Number.isFinite(date.getTime())) return <>—</>
  return (
    <time dateTime={date.toISOString()}>
      {date.toLocaleString(toIntlLocale(i18n.language))}
    </time>
  )
}
