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
import { zodResolver } from '@hookform/resolvers/zod'
import {
  useMutation,
  useQuery,
  useQueryClient,
  type QueryClient,
} from '@tanstack/react-query'
import { RefreshCw } from 'lucide-react'
import { useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout/components/section-page-layout'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import { hasPermission } from '@/lib/admin-permissions'
import { formatTimestampToDate } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import {
  getAdminSupplierApplication,
  getAdminSupplierApplications,
  reviewSupplierApplication,
} from './api'
import { SupplierConversationDialog } from './conversation'
import { getSupplierErrorKey } from './errors'
import { getSupplierLabels } from './labels'
import { SupplierSiteLink } from './my-submissions'
import { SupplierQuoteList } from './quote-list'
import {
  supplierReviewSchema,
  supplierStatuses,
  type SupplierReviewFormValues,
} from './schema'
import type {
  SupplierApplicationAdmin,
  SupplierApplicationFilters,
  SupplierApplicationList,
  SupplierApplicationStatus,
  SupplierMessage,
} from './types'

const PAGE_SIZE = 20

function ratioSummary(application: SupplierApplicationAdmin): string {
  const ratios = application.groups.map((group) => group.ratio)
  if (ratios.length === 0) return '-'
  const min = Math.min(...ratios)
  const max = Math.max(...ratios)
  return min === max ? `×${min}` : `×${min} – ×${max}`
}

function patchAdminApplication(
  queryClient: QueryClient,
  userId: number | undefined,
  applicationId: number,
  patch: (item: SupplierApplicationAdmin) => SupplierApplicationAdmin,
) {
  queryClient.setQueryData<SupplierApplicationAdmin>(
    ['supplier-applications', userId, 'admin', 'detail', applicationId],
    (current) => (current ? patch(current) : current),
  )
  queryClient.setQueriesData<SupplierApplicationList<SupplierApplicationAdmin>>(
    { queryKey: ['supplier-applications', userId, 'admin', 'list'] },
    (current) => {
      if (!current) return current
      return {
        ...current,
        items: current.items.map((item) =>
          item.id === applicationId ? patch(item) : item,
        ),
      }
    },
  )
}

function withUnread(unreadCount: number) {
  return (item: SupplierApplicationAdmin) => ({
    ...item,
    unread_count: unreadCount,
  })
}

function withSentMessage(message: SupplierMessage) {
  return (item: SupplierApplicationAdmin) => {
    const messages = [...(item.messages ?? [])]
    if (messages.some((entry) => entry.id === message.id)) return item
    messages.push(message)
    return {
      ...item,
      messages,
      message_count: (item.message_count ?? 0) + 1,
    }
  }
}

function ReviewForm(props: {
  application: SupplierApplicationAdmin
  canWrite: boolean
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const labels = getSupplierLabels(t)
  const [serverError, setServerError] = useState('')
  const [editing, setEditing] = useState(false)
  const submitting = useRef(false)
  const form = useForm<SupplierReviewFormValues>({
    resolver: zodResolver(supplierReviewSchema),
    defaultValues: {
      status: props.application.status,
      admin_note: props.application.admin_note,
    },
  })
  const mutation = useMutation({
    mutationFn: (values: SupplierReviewFormValues) =>
      reviewSupplierApplication(props.application.id, {
        ...values,
        version: props.application.version,
      }),
    onMutate: () => useAuthStore.getState().auth.user?.id,
    onSuccess: (_application, _values, submittedUserId) => {
      if (
        submittedUserId === undefined ||
        useAuthStore.getState().auth.user?.id !== submittedUserId
      ) {
        return
      }
      setServerError('')
      void queryClient.invalidateQueries({
        queryKey: ['supplier-applications', submittedUserId],
      })
      toast.success(t('Application updated.'))
    },
    onError: (error, _values, submittedUserId) => {
      if (
        submittedUserId === undefined ||
        useAuthStore.getState().auth.user?.id !== submittedUserId
      ) {
        return
      }
      setServerError(
        getSupplierErrorKey(error, 'Could not update the application.')
      )
      // A version conflict means someone else changed it; reload the latest.
      void queryClient.invalidateQueries({
        queryKey: ['supplier-applications', submittedUserId],
      })
    },
    onSettled: () => {
      submitting.current = false
    },
  })

  return (
    <div className='space-y-3'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <h3 className='text-sm font-semibold'>{t('Processing')}</h3>
        {props.canWrite && !editing && (
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={() => setEditing(true)}
          >
            {t('Edit')}
          </Button>
        )}
      </div>
      <p className='text-sm'>
        <span className='text-muted-foreground'>{t('Status')}: </span>
        <span data-supplier-processing-status>
          {labels.status[props.application.status]}
        </span>
      </p>
      {editing && props.canWrite && (
        <Form {...form}>
          <form
            className='space-y-4'
            noValidate
            onSubmit={form.handleSubmit((values) => {
              if (submitting.current) return
              submitting.current = true
              setServerError('')
              mutation.mutate(values)
            })}
          >
            <FormField
              control={form.control}
              name='status'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Status')}</FormLabel>
                  <FormControl>
                    <NativeSelect
                      {...field}
                      className='w-full'
                      disabled={!props.canWrite || mutation.isPending}
                    >
                      {supplierStatuses.map((value) => (
                        <NativeSelectOption key={value} value={value}>
                          {labels.status[value]}
                        </NativeSelectOption>
                      ))}
                    </NativeSelect>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='admin_note'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Internal note')}</FormLabel>
                  <FormControl>
                    <Textarea
                      {...field}
                      className='min-h-28 resize-y'
                      disabled={!props.canWrite || mutation.isPending}
                    />
                  </FormControl>
                  <p className='text-muted-foreground text-xs'>
                    {t(
                      'Only administrators can see this note. The applicant never does.'
                    )}
                  </p>
                  <FormMessage />
                </FormItem>
              )}
            />
            {serverError && (
              <Alert variant='destructive'>
                <AlertDescription>{t(serverError)}</AlertDescription>
              </Alert>
            )}
            {props.canWrite && (
              <Button type='submit' disabled={mutation.isPending}>
                {mutation.isPending ? t('Saving...') : t('Save')}
              </Button>
            )}
            <p className='text-muted-foreground text-xs'>
              {t(
                'Changing the status never creates channels or changes groups or billing.'
              )}
            </p>
          </form>
        </Form>
      )}
    </div>
  )
}

function SupplierApplicationSheet(props: {
  id: number
  canWrite: boolean
  onClose: () => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const user = useAuthStore((state) => state.auth.user)
  const labels = getSupplierLabels(t)
  const query = useQuery({
    queryKey: ['supplier-applications', user?.id, 'admin', 'detail', props.id],
    queryFn: ({ signal }) => getAdminSupplierApplication(props.id, signal),
  })
  const application = query.data
  const conversationOpen = useRef(false)

  function setUnread(unreadCount: number) {
    patchAdminApplication(
      queryClient,
      user?.id,
      props.id,
      withUnread(unreadCount),
    )
  }

  return (
    <Sheet
      open
      onOpenChange={(open) => {
        if (!open && conversationOpen.current) return
        if (!open) props.onClose()
      }}
    >
      <SheetContent className='w-full overflow-hidden sm:max-w-2xl'>
        <SheetHeader className='shrink-0'>
          <div className='flex min-w-0 items-start gap-2 pr-8'>
            <SheetTitle className='min-w-0 [overflow-wrap:anywhere]'>
              {application
                ? `#${application.id} ${application.site_name}`
                : t('Supplier Application')}
            </SheetTitle>
            {application ? (
              <SupplierConversationDialog
                applicationId={application.id}
                siteName={application.site_name}
                messages={application.messages}
                canSend={props.canWrite}
                mode='admin'
                unreadCount={application.unread_count}
                onOpenChange={(next) => {
                  conversationOpen.current = next
                }}
                onRead={setUnread}
                onSent={(message) =>
                  patchAdminApplication(
                    queryClient,
                    user?.id,
                    application.id,
                    withSentMessage(message),
                  )
                }
              />
            ) : null}
          </div>
          <SheetDescription>
            {t('Full details and processing for this application.')}
          </SheetDescription>
        </SheetHeader>
        <div
          data-supplier-detail
          className='px-4 pb-4'
          style={{
            display: 'flex',
            flexDirection: 'column',
            flex: '1 1 auto',
            minHeight: 0,
            overflow: 'hidden',
            gap: 16,
          }}
        >
          {query.isPending && (
            <p role='status' className='text-muted-foreground text-sm'>
              {t('Loading...')}
            </p>
          )}
          {query.isError && (
            <Alert variant='destructive'>
              <AlertDescription className='flex flex-wrap items-center gap-2'>
                {t(
                  getSupplierErrorKey(
                    query.error,
                    'Could not load the application.'
                  )
                )}
                <Button variant='outline' onClick={() => void query.refetch()}>
                  {t('Retry')}
                </Button>
              </AlertDescription>
            </Alert>
          )}
          {application && (
              <div
                data-supplier-detail-facts
                className='space-y-6'
                style={{
                  flex: '1 1 auto',
                  minHeight: 0,
                  overflowY: 'auto',
                }}
              >
                <section
                  data-supplier-processing
                  className='bg-muted/20 rounded-lg border p-4'
                >
                  <ReviewForm
                    key={`${application.id}-${application.version}`}
                    application={application}
                    canWrite={props.canWrite}
                  />
                </section>
                <dl className='grid gap-x-6 gap-y-3 text-sm sm:grid-cols-2'>
                  <div className='min-w-0'>
                    <dt className='text-muted-foreground'>{t('Applicant')}</dt>
                    <dd className='[overflow-wrap:anywhere]'>
                      {application.username || '-'} (ID {application.user_id})
                    </dd>
                  </div>
                  <div className='min-w-0'>
                    <dt className='text-muted-foreground'>
                      {t('Submitted at')}
                    </dt>
                    <dd>{formatTimestampToDate(application.created_at)}</dd>
                  </div>
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
                  <div className='min-w-0'>
                    <dt className='text-muted-foreground'>{t('Handled by')}</dt>
                    <dd className='[overflow-wrap:anywhere]'>
                      {application.handled_by > 0
                        ? `${application.handled_by_name || application.handled_by} · ${formatTimestampToDate(application.handled_at)}`
                        : '-'}
                    </dd>
                  </div>
                  {application.remark && (
                    <div className='min-w-0 sm:col-span-2'>
                      <dt className='text-muted-foreground'>{t('Remarks')}</dt>
                      <dd className='[overflow-wrap:anywhere] whitespace-pre-wrap'>
                        {application.remark}
                      </dd>
                    </div>
                  )}
                </dl>
                <section className='space-y-2'>
                  <h3 className='text-sm font-semibold'>
                    {t('Group quotes')} ({application.groups.length})
                  </h3>
                  <SupplierQuoteList quotes={application.groups} />
                </section>
              </div>
          )}
        </div>
      </SheetContent>
    </Sheet>
  )
}

export function SupplierApplicationManagement(props: {
  status?: SupplierApplicationStatus | ''
  onStatusChange?: (status: SupplierApplicationStatus | '') => void
}) {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const labels = getSupplierLabels(t)
  const canRead = Boolean(
    user &&
    user.role >= ROLE.ADMIN &&
    hasPermission(user, 'supplier_applications', 'read')
  )
  const canWrite = Boolean(
    user &&
    user.role >= ROLE.ADMIN &&
    hasPermission(user, 'supplier_applications', 'write')
  )
  const [filters, setFilters] = useState<SupplierApplicationFilters>({
    p: 1,
    page_size: PAGE_SIZE,
    status: props.status,
  })
  const [routeStatus, setRouteStatus] = useState(props.status)
  if (routeStatus !== props.status) {
    setRouteStatus(props.status)
    setFilters((current) => ({ ...current, p: 1, status: props.status }))
  }
  const [keyword, setKeyword] = useState('')
  const [userId, setUserId] = useState('')
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const queryClient = useQueryClient()
  const query = useQuery({
    queryKey: ['supplier-applications', user?.id, 'admin', 'list', filters],
    queryFn: ({ signal }) => getAdminSupplierApplications(filters, signal),
    enabled: canRead,
  })
  const totalPages = Math.max(
    1,
    Math.ceil((query.data?.total ?? 0) / filters.page_size)
  )

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Supplier Application Management')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        {canRead && (
          <Button
            variant='outline'
            disabled={query.isFetching}
            onClick={() => void query.refetch()}
          >
            <RefreshCw aria-hidden='true' />
            {t('Refresh')}
          </Button>
        )}
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        {!canRead ? (
          <Alert variant='destructive'>
            <AlertDescription>
              {t('You do not have permission to view supplier applications.')}
            </AlertDescription>
          </Alert>
        ) : (
          <Card>
            <CardHeader>
              <CardTitle>{t('All supplier applications')}</CardTitle>
              <CardDescription>
                {t(
                  'Compare quotes across sites, contact suppliers, and record the outcome.'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-4'>
              <form
                className='bg-muted/20 flex flex-wrap items-end gap-3 rounded-lg border p-3'
                onSubmit={(event) => {
                  event.preventDefault()
                  setFilters((current) => ({
                    ...current,
                    p: 1,
                    keyword: keyword.trim() || undefined,
                    user_id: userId ? Number(userId) : undefined,
                  }))
                }}
              >
                <div className='min-w-40 flex-1 space-y-2'>
                  <Label htmlFor='supplier-search'>
                    {t('Search applications')}
                  </Label>
                  <Input
                    id='supplier-search'
                    value={keyword}
                    placeholder={t('Site, contact, or group name')}
                    onChange={(event) => setKeyword(event.target.value)}
                  />
                </div>
                <div className='w-32 space-y-2'>
                  <Label htmlFor='supplier-user'>{t('User ID')}</Label>
                  <Input
                    id='supplier-user'
                    type='number'
                    min={1}
                    step={1}
                    value={userId}
                    onChange={(event) => setUserId(event.target.value)}
                  />
                </div>
                <div className='space-y-2'>
                  <Label htmlFor='supplier-filter-status'>{t('Status')}</Label>
                  <NativeSelect
                    id='supplier-filter-status'
                    value={filters.status ?? ''}
                    onChange={(event) => {
                      const status = event.target.value as
                        | SupplierApplicationStatus
                        | ''
                      props.onStatusChange?.(status)
                      setFilters((current) => ({ ...current, p: 1, status }))
                    }}
                  >
                    <NativeSelectOption value=''>
                      {t('All statuses')}
                    </NativeSelectOption>
                    {Object.entries(labels.status).map(([value, label]) => (
                      <NativeSelectOption key={value} value={value}>
                        {label}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                </div>
                <Button type='submit'>{t('Search')}</Button>
              </form>
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
                    {t(
                      getSupplierErrorKey(
                        query.error,
                        'Could not load submissions.'
                      )
                    )}
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
                  {t('No applications found.')}
                </p>
              )}
              {query.isSuccess && query.data.items.length > 0 && (
                <div className='overflow-x-auto rounded-lg border'>
                  <Table aria-label={t('All supplier applications')}>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('Site')}</TableHead>
                        <TableHead>{t('Applicant')}</TableHead>
                        <TableHead>{t('Contact')}</TableHead>
                        <TableHead>{t('Groups')}</TableHead>
                        <TableHead>{t('Status')}</TableHead>
                        <TableHead>{t('Submitted at')}</TableHead>
                        <TableHead>{t('Handled by')}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {query.data.items.map((application) => (
                        <TableRow key={application.id}>
                          <TableCell className='max-w-64 min-w-40 whitespace-normal'>
                            <div className='flex items-start gap-2'>
                              <button
                                type='button'
                                className='text-primary block max-w-60 text-left font-medium [overflow-wrap:anywhere] whitespace-normal underline-offset-4 hover:underline focus-visible:underline'
                                onClick={() => setSelectedId(application.id)}
                              >
                                #{application.id} {application.site_name}
                              </button>
                              <SupplierConversationDialog
                                applicationId={application.id}
                                siteName={application.site_name}
                                messages={application.messages}
                                canSend={canWrite}
                                mode='admin'
                                unreadCount={application.unread_count}
                                onRead={(unreadCount) =>
                                  patchAdminApplication(
                                    queryClient,
                                    user?.id,
                                    application.id,
                                    withUnread(unreadCount),
                                  )
                                }
                                onSent={(message) =>
                                  patchAdminApplication(
                                    queryClient,
                                    user?.id,
                                    application.id,
                                    withSentMessage(message),
                                  )
                                }
                              />
                            </div>
                          </TableCell>
                          <TableCell className='max-w-40 [overflow-wrap:anywhere] whitespace-normal'>
                            {application.username || '-'} ({application.user_id}
                            )
                          </TableCell>
                          <TableCell className='max-w-48 [overflow-wrap:anywhere] whitespace-normal'>
                            {labels.contactType[application.contact_type]}:{' '}
                            {application.contact}
                          </TableCell>
                          <TableCell className='whitespace-nowrap tabular-nums'>
                            {t('{{count}} groups', {
                              count: application.groups.length,
                            })}
                            <span className='text-muted-foreground block text-xs'>
                              {ratioSummary(application)}
                            </span>
                          </TableCell>
                          <TableCell>
                            <Badge variant='outline'>
                              {labels.status[application.status]}
                            </Badge>
                          </TableCell>
                          <TableCell className='whitespace-nowrap'>
                            {formatTimestampToDate(application.created_at)}
                          </TableCell>
                          <TableCell className='max-w-40 [overflow-wrap:anywhere] whitespace-normal'>
                            {application.handled_by > 0
                              ? `${application.handled_by_name || application.handled_by}`
                              : '-'}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}
              <div className='flex flex-wrap items-center justify-between gap-3 border-t pt-4'>
                <p aria-live='polite' className='text-muted-foreground text-sm'>
                  {t('Page {{page}} of {{pages}} · {{count}} submissions', {
                    page: filters.p,
                    pages: totalPages,
                    count: query.data?.total ?? 0,
                  })}
                </p>
                <div className='flex gap-2'>
                  <Button
                    variant='outline'
                    disabled={filters.p <= 1 || query.isFetching}
                    onClick={() =>
                      setFilters((current) => ({
                        ...current,
                        p: current.p - 1,
                      }))
                    }
                  >
                    {t('Previous')}
                  </Button>
                  <Button
                    variant='outline'
                    disabled={
                      !query.isSuccess ||
                      filters.p >= totalPages ||
                      query.isFetching
                    }
                    onClick={() =>
                      setFilters((current) => ({
                        ...current,
                        p: current.p + 1,
                      }))
                    }
                  >
                    {t('Next')}
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        )}
        {selectedId !== null && canRead && (
          <SupplierApplicationSheet
            id={selectedId}
            canWrite={canWrite}
            onClose={() => setSelectedId(null)}
          />
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
