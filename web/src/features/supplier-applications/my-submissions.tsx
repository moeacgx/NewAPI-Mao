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
import { ChevronDown, ChevronUp, RefreshCw } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { formatTimestampToDate } from '@/lib/format'
import { useAuthStore } from '@/stores/auth-store'

import { getSupplierApplications } from './api'
import { getSupplierErrorKey } from './errors'
import { getSupplierLabels } from './labels'
import { SupplierQuoteList } from './quote-list'
import type { SupplierApplication } from './types'

const PAGE_SIZE = 10

export function SupplierSiteLink(props: { url: string }) {
  // Only http(s) is ever rendered as a link; the address is never requested by us.
  if (!/^https?:\/\//i.test(props.url)) {
    return <span className='[overflow-wrap:anywhere]'>{props.url}</span>
  }
  return (
    <a
      href={props.url}
      target='_blank'
      rel='noopener noreferrer nofollow'
      className='text-primary [overflow-wrap:anywhere] underline-offset-4 hover:underline'
    >
      {props.url}
    </a>
  )
}

function SubmissionCard(props: { application: SupplierApplication }) {
  const { t } = useTranslation()
  const labels = getSupplierLabels(t)
  const [expanded, setExpanded] = useState(false)
  const { application } = props
  const panelId = `supplier-submission-${application.id}`

  return (
    <li className='bg-card min-w-0 space-y-3 rounded-lg border p-4'>
      <div className='flex flex-wrap items-start justify-between gap-2'>
        <div className='min-w-0 space-y-1'>
          <h3 className='font-medium [overflow-wrap:anywhere]'>
            #{application.id} {application.site_name}
          </h3>
          <p className='text-muted-foreground text-sm'>
            {t('Submitted at')} {formatTimestampToDate(application.created_at)}{' '}
            · {t('{{count}} groups', { count: application.groups.length })}
          </p>
        </div>
        <Badge variant='outline'>{labels.status[application.status]}</Badge>
      </div>
      <Button
        type='button'
        variant='ghost'
        size='sm'
        aria-expanded={expanded}
        aria-controls={panelId}
        onClick={() => setExpanded((current) => !current)}
      >
        {expanded ? (
          <ChevronUp aria-hidden='true' />
        ) : (
          <ChevronDown aria-hidden='true' />
        )}
        {expanded ? t('Hide details') : t('View details')}
      </Button>
      {expanded && (
        <div id={panelId} className='space-y-4 border-t pt-3'>
          <dl className='grid gap-x-6 gap-y-2 text-sm md:grid-cols-2'>
            <div className='min-w-0'>
              <dt className='text-muted-foreground'>{t('Site URL')}</dt>
              <dd>
                {application.site_url ? (
                  <SupplierSiteLink url={application.site_url} />
                ) : (
                  '-'
                )}
              </dd>
            </div>
            <div className='min-w-0'>
              <dt className='text-muted-foreground'>
                {labels.contactType[application.contact_type]}
              </dt>
              <dd className='[overflow-wrap:anywhere]'>
                {application.contact}
              </dd>
            </div>
            {application.remark && (
              <div className='min-w-0 md:col-span-2'>
                <dt className='text-muted-foreground'>{t('Remarks')}</dt>
                <dd className='[overflow-wrap:anywhere] whitespace-pre-wrap'>
                  {application.remark}
                </dd>
              </div>
            )}
          </dl>
          <SupplierQuoteList quotes={application.groups} />
        </div>
      )}
    </li>
  )
}

export function MySupplierSubmissions() {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const [page, setPage] = useState(1)
  const query = useQuery({
    queryKey: ['supplier-applications', user?.id, 'list', page],
    queryFn: ({ signal }) =>
      getSupplierApplications({ p: page, page_size: PAGE_SIZE }, signal),
    enabled: Boolean(user),
  })
  const totalPages = Math.max(
    1,
    Math.ceil((query.data?.total ?? 0) / PAGE_SIZE)
  )

  return (
    <div className='space-y-4'>
      <div className='flex items-center justify-between gap-2'>
        <p className='text-muted-foreground text-sm'>
          {t(
            'Every submission is kept as its own record. To correct one, submit a new version.'
          )}
        </p>
        <Button
          type='button'
          variant='outline'
          size='sm'
          disabled={query.isFetching}
          onClick={() => void query.refetch()}
        >
          <RefreshCw aria-hidden='true' />
          {t('Refresh')}
        </Button>
      </div>
      {query.isPending && (
        <p
          role='status'
          className='bg-muted/20 rounded-lg border p-6 text-center'
        >
          {t('Loading submissions...')}
        </p>
      )}
      {query.isError && (
        <Alert variant='destructive'>
          <AlertDescription className='flex flex-wrap items-center gap-2'>
            {t(getSupplierErrorKey(query.error, 'Could not load submissions.'))}
            <Button
              variant='outline'
              disabled={query.isFetching}
              onClick={() => void query.refetch()}
            >
              {t('Retry')}
            </Button>
          </AlertDescription>
        </Alert>
      )}
      {query.isSuccess && query.data.items.length === 0 && (
        <p
          role='status'
          className='bg-muted/20 rounded-lg border p-6 text-center'
        >
          {t('No submissions yet')}
        </p>
      )}
      {query.isSuccess && query.data.items.length > 0 && (
        <ul className='space-y-3'>
          {query.data.items.map((application) => (
            <SubmissionCard key={application.id} application={application} />
          ))}
        </ul>
      )}
      <div className='flex flex-wrap items-center justify-between gap-3 border-t pt-4'>
        <p aria-live='polite' className='text-muted-foreground text-sm'>
          {t('Page {{page}} of {{pages}} · {{count}} submissions', {
            page,
            pages: totalPages,
            count: query.data?.total ?? 0,
          })}
        </p>
        <div className='flex gap-2'>
          <Button
            variant='outline'
            disabled={page <= 1 || query.isFetching}
            onClick={() => setPage((current) => current - 1)}
          >
            {t('Previous')}
          </Button>
          <Button
            variant='outline'
            disabled={
              !query.isSuccess || page >= totalPages || query.isFetching
            }
            onClick={() => setPage((current) => current + 1)}
          >
            {t('Next')}
          </Button>
        </div>
      </div>
    </div>
  )
}
