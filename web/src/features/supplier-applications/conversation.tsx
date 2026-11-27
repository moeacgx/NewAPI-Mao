import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'
import { formatTimestampToDate } from '@/lib/format'

import {
  createAdminSupplierApplicationMessage,
  createSupplierApplicationMessage,
} from './api'
import { getSupplierErrorKey } from './errors'
import { getSupplierLabels } from './labels'
import { parseSupplierQuantity, parseSupplierRatio } from './schema'
import type {
  SupplierMessage,
  SupplierMessageInput,
  SupplierMessageType,
  SupplierQuotaStatus,
} from './types'

const MESSAGE_TYPES: SupplierMessageType[] = [
  'message',
  'negotiation',
  'offer',
  'test_quota',
]

function textLength(value: string): number {
  return [...value].length
}

function buildMessageInput(
  type: SupplierMessageType,
  fields: {
    body: string
    groupName: string
    ratio: string
    quantity: string
    unit: string
    quotaStatus: SupplierQuotaStatus
  }
): SupplierMessageInput | null {
  const body = fields.body.trim()
  if (type === 'message') {
    if (body === '' || textLength(body) > 2000) return null
    return { type, body }
  }
  if (type === 'negotiation') {
    const groupName = fields.groupName.trim()
    const ratio = parseSupplierRatio(fields.ratio)
    if (
      groupName === '' ||
      textLength(groupName) > 64 ||
      ratio === null ||
      body === '' ||
      textLength(body) > 1000
    ) {
      return null
    }
    return { type, body, group_name: groupName, ratio }
  }
  if (type === 'offer') {
    if (body === '' || textLength(body) > 2000) return null
    return { type, body }
  }
  const quantity = parseSupplierQuantity(fields.quantity)
  const unit = fields.unit.trim()
  if (
    quantity === null ||
    unit === '' ||
    textLength(unit) > 32 ||
    body === '' ||
    textLength(body) > 1000
  ) {
    return null
  }
  return {
    type,
    body,
    quantity,
    unit,
    quota_status: fields.quotaStatus,
  }
}

function MessageBody(props: { message: SupplierMessage }) {
  const { t } = useTranslation()
  const labels = getSupplierLabels(t)
  const { message } = props
  return (
    <div className='min-w-0 space-y-1 text-sm'>
      {message.type === 'negotiation' && (
        <p className='[overflow-wrap:anywhere]'>
          <span className='text-muted-foreground'>{t('Group name')}: </span>
          {message.group_name} · {t('Suggested ratio')} ×{message.ratio}
        </p>
      )}
      {message.type === 'test_quota' && (
        <p className='[overflow-wrap:anywhere]'>
          <span className='text-muted-foreground'>{t('Test quota')}: </span>
          {message.quantity} {message.unit}
          {message.quota_status
            ? ` · ${labels.quotaStatus[message.quota_status]}`
            : ''}
        </p>
      )}
      <p className='[overflow-wrap:anywhere] whitespace-pre-wrap'>
        {message.body}
      </p>
    </div>
  )
}

export function SupplierConversation(props: {
  applicationId: number
  messages?: SupplierMessage[]
  canSend: boolean
  mode: 'personal' | 'admin'
  onSent?: (message: SupplierMessage) => void
}) {
  const { t } = useTranslation()
  const labels = getSupplierLabels(t)
  const [type, setType] = useState<SupplierMessageType>('message')
  const [body, setBody] = useState('')
  const [groupName, setGroupName] = useState('')
  const [ratio, setRatio] = useState('')
  const [quantity, setQuantity] = useState('')
  const [unit, setUnit] = useState('')
  const [quotaStatus, setQuotaStatus] = useState<SupplierQuotaStatus>('pending')
  const [sent, setSent] = useState<SupplierMessage[]>([])
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const baseId = `supplier-message-${props.mode}-${props.applicationId}`
  const merged = new Map<number, SupplierMessage>()
  for (const message of [...(props.messages ?? []), ...sent]) {
    merged.set(message.id, message)
  }
  const messages = [...merged.values()].sort(
    (left, right) => left.created_at - right.created_at || left.id - right.id
  )
  let bodyLabel = 'Message text'
  if (type === 'offer') bodyLabel = 'Offer plan'
  else if (type === 'negotiation') bodyLabel = 'Negotiation notes'
  else if (type === 'test_quota') bodyLabel = 'Test quota description'

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!props.canSend || pending) return
    const input = buildMessageInput(type, {
      body,
      groupName,
      ratio,
      quantity,
      unit,
      quotaStatus,
    })
    if (!input) {
      setNotice('')
      setError('Some fields are invalid. Check the form and try again.')
      return
    }
    setPending(true)
    setError('')
    setNotice('')
    try {
      const created =
        props.mode === 'admin'
          ? await createAdminSupplierApplicationMessage(
              props.applicationId,
              input
            )
          : await createSupplierApplicationMessage(props.applicationId, input)
      setSent((current) => [...current, created])
      props.onSent?.(created)
      setBody('')
      setGroupName('')
      setRatio('')
      setQuantity('')
      setUnit('')
      setQuotaStatus('pending')
      setNotice('Message sent')
    } catch (sendError) {
      setError(getSupplierErrorKey(sendError, 'Could not send the message.'))
    } finally {
      setPending(false)
    }
  }

  return (
    <section className='bg-muted/20 min-w-0 space-y-3 overflow-x-hidden rounded-lg border p-4'>
      <div className='min-w-0 space-y-1'>
        <h3 className='text-sm font-semibold'>{t('Conversation')}</h3>
        <p className='text-muted-foreground text-xs [overflow-wrap:anywhere]'>
          {t(
            'Messages only record the discussion. They do not add balance, create redemption codes or channels, or change groups and billing.'
          )}
        </p>
      </div>
      {messages.length === 0 ? (
        <p role='status' className='text-muted-foreground text-sm'>
          {t('No messages yet')}
        </p>
      ) : (
        <ol className='min-w-0 space-y-3' aria-live='polite'>
          {messages.map((message) => (
            <li
              key={message.id}
              className='bg-card min-w-0 space-y-2 rounded-md border p-3'
            >
              <div className='flex min-w-0 flex-wrap items-center gap-2'>
                <span className='text-sm font-medium [overflow-wrap:anywhere]'>
                  {message.author_name ||
                    labels.authorRole[message.author_role]}
                </span>
                <Badge
                  variant='outline'
                  className='max-w-full [overflow-wrap:anywhere] whitespace-normal'
                >
                  {labels.authorRole[message.author_role]}
                </Badge>
                <Badge
                  variant='secondary'
                  className='max-w-full [overflow-wrap:anywhere] whitespace-normal'
                >
                  {labels.messageType[message.type]}
                </Badge>
                <span className='text-muted-foreground text-xs'>
                  {formatTimestampToDate(message.created_at)}
                </span>
              </div>
              <MessageBody message={message} />
            </li>
          ))}
        </ol>
      )}
      {props.canSend && (
        <form className='min-w-0 space-y-3' noValidate onSubmit={submit}>
          <div className='space-y-1'>
            <Label htmlFor={`${baseId}-type`}>{t('Message type')}</Label>
            <NativeSelect
              id={`${baseId}-type`}
              className='w-full max-w-full'
              value={type}
              disabled={pending}
              onChange={(event) =>
                setType(event.target.value as SupplierMessageType)
              }
            >
              {MESSAGE_TYPES.map((value) => (
                <NativeSelectOption key={value} value={value}>
                  {labels.messageType[value]}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </div>
          {type === 'negotiation' && (
            <div className='grid min-w-0 gap-3 sm:grid-cols-2'>
              <div className='min-w-0 space-y-1'>
                <Label htmlFor={`${baseId}-group`}>{t('Group name')}</Label>
                <Input
                  id={`${baseId}-group`}
                  value={groupName}
                  maxLength={64}
                  disabled={pending}
                  onChange={(event) => setGroupName(event.target.value)}
                />
              </div>
              <div className='min-w-0 space-y-1'>
                <Label htmlFor={`${baseId}-ratio`}>
                  {t('Suggested ratio')}
                </Label>
                <Input
                  id={`${baseId}-ratio`}
                  inputMode='decimal'
                  value={ratio}
                  disabled={pending}
                  onChange={(event) => setRatio(event.target.value)}
                />
              </div>
            </div>
          )}
          {type === 'test_quota' && (
            <div className='grid min-w-0 gap-3 sm:grid-cols-3'>
              <div className='min-w-0 space-y-1'>
                <Label htmlFor={`${baseId}-quantity`}>{t('Quantity')}</Label>
                <Input
                  id={`${baseId}-quantity`}
                  inputMode='decimal'
                  value={quantity}
                  disabled={pending}
                  onChange={(event) => setQuantity(event.target.value)}
                />
              </div>
              <div className='min-w-0 space-y-1'>
                <Label htmlFor={`${baseId}-unit`}>{t('Unit')}</Label>
                <Input
                  id={`${baseId}-unit`}
                  value={unit}
                  maxLength={32}
                  disabled={pending}
                  onChange={(event) => setUnit(event.target.value)}
                />
              </div>
              <div className='min-w-0 space-y-1'>
                <Label htmlFor={`${baseId}-status`}>
                  {t('Issuance status')}
                </Label>
                <NativeSelect
                  id={`${baseId}-status`}
                  className='w-full max-w-full'
                  value={quotaStatus}
                  disabled={pending}
                  onChange={(event) =>
                    setQuotaStatus(event.target.value as SupplierQuotaStatus)
                  }
                >
                  <NativeSelectOption value='pending'>
                    {t('Not issued')}
                  </NativeSelectOption>
                  <NativeSelectOption value='issued'>
                    {t('Issued')}
                  </NativeSelectOption>
                </NativeSelect>
              </div>
            </div>
          )}
          <div className='min-w-0 space-y-1'>
            <Label htmlFor={`${baseId}-body`}>{t(bodyLabel)}</Label>
            <Textarea
              id={`${baseId}-body`}
              className='min-h-24 w-full resize-y'
              value={body}
              maxLength={type === 'message' || type === 'offer' ? 2000 : 1000}
              disabled={pending}
              onChange={(event) => setBody(event.target.value)}
            />
          </div>
          <p className='text-muted-foreground text-xs [overflow-wrap:anywhere]'>
            {t('Messages cannot be edited after sending.')}{' '}
            {t('Do not include API keys or access credentials.')}
          </p>
          {error && (
            <Alert variant='destructive'>
              <AlertDescription>{t(error)}</AlertDescription>
            </Alert>
          )}
          {notice && (
            <p role='status' className='text-sm'>
              {t(notice)}
            </p>
          )}
          <Button type='submit' disabled={pending}>
            {pending ? t('Sending...') : t('Send message')}
          </Button>
        </form>
      )}
    </section>
  )
}
