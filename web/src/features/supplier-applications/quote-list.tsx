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

import { Badge } from '@/components/ui/badge'

import { getSupplierLabels } from './labels'
import type { SupplierQuote } from './types'

// Shared by the applicant and admin views so quotes read identically for both.
// Long names/models wrap instead of stretching the page.
export function SupplierQuoteList(props: { quotes: SupplierQuote[] }) {
  const { t } = useTranslation()
  const labels = getSupplierLabels(t)

  return (
    <ul
      aria-label={t('Group quotes')}
      className='grid gap-3 md:grid-cols-2'
      data-testid='supplier-quote-list'
    >
      {props.quotes.map((quote) => (
        <li
          key={quote.id}
          className='bg-muted/20 min-w-0 space-y-2 rounded-lg border p-3'
        >
          <div className='flex flex-wrap items-center gap-2'>
            <span className='min-w-0 font-medium [overflow-wrap:anywhere]'>
              {quote.group_name}
            </span>
            <Badge variant='outline'>{labels.platform[quote.platform]}</Badge>
            <span className='text-muted-foreground ml-auto text-sm tabular-nums'>
              {t('Ratio')} ×{quote.ratio}
            </span>
          </div>
          {quote.models.length > 0 && (
            <div className='flex flex-wrap gap-1.5'>
              {quote.models.map((model) => (
                <Badge
                  key={model}
                  variant='secondary'
                  className='h-auto max-w-full py-0.5 [overflow-wrap:anywhere] whitespace-normal'
                >
                  {model}
                </Badge>
              ))}
            </div>
          )}
          {quote.description && (
            <p className='text-muted-foreground text-sm [overflow-wrap:anywhere] whitespace-pre-wrap'>
              {quote.description}
            </p>
          )}
        </li>
      ))}
    </ul>
  )
}
