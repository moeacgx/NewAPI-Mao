import { useEffect, useId, useRef, useState, type FormEvent } from 'react'
import { createPortal } from 'react-dom'
import { useTranslation } from 'react-i18next'
import { X } from 'lucide-react'

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
  markAdminSupplierApplicationRead,
  markSupplierApplicationRead,
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
  /** fill: occupy the detail column so the composer sits on the sheet bottom. */
  dock?: 'fill' | 'cap'
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
  const dock = props.dock ?? 'cap'
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
    <section
      data-supplier-conversation
      className='bg-muted/20 min-w-0 rounded-lg border p-4'
      style={{
        display: 'flex',
        flexDirection: 'column',
        flex: dock === 'fill' ? '1 1 auto' : undefined,
        minWidth: 0,
        width: '100%',
        maxHeight: dock === 'fill' ? undefined : '32rem',
        height: dock === 'fill' ? '100%' : undefined,
        overflow: 'hidden',
        gap: 12,
      }}
    >
      <div className='min-w-0 space-y-1' style={{ flexShrink: 0 }}>
        <h3 className='text-sm font-semibold'>{t('Conversation')}</h3>
        <p className='text-muted-foreground text-xs [overflow-wrap:anywhere]'>
          {t(
            'Messages only record the discussion. They do not add balance, create redemption codes or channels, or change groups and billing.'
          )}
        </p>
      </div>
      <div
        data-supplier-message-scroll
        style={{
          flex: '1 1 auto',
          minHeight: '8rem',
          overflowY: 'auto',
          overflowX: 'hidden',
          paddingBottom: '12px',
        }}
      >
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
      </div>
      {props.canSend && (
        <form
          data-supplier-composer
          className='min-w-0 space-y-3 border-t pt-3'
          style={{ flexShrink: 0 }}
          noValidate
          onSubmit={submit}
        >
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

// 详情保持挂载时，弹窗只隐藏不卸载，未发送的草稿留在输入框里。
export function SupplierConversationDialog(props: {
  applicationId: number
  siteName: string
  messages?: SupplierMessage[]
  canSend: boolean
  mode: 'personal' | 'admin'
  unreadCount?: number
  onOpenChange?: (open: boolean) => void
  onSent?: (message: SupplierMessage) => void
  onRead?: (unreadCount: number) => void
}) {
  const { t } = useTranslation()
  const titleId = useId()
  const [open, setOpen] = useState(false)
  const [mounted, setMounted] = useState(false)
  const onReadRef = useRef(props.onRead)
  const onOpenChangeRef = useRef(props.onOpenChange)
  onReadRef.current = props.onRead
  onOpenChangeRef.current = props.onOpenChange
  const unread = props.unreadCount ?? 0
  const title = t('Conversation with {{site}}', { site: props.siteName })
  const buttonLabel =
    unread > 0
      ? `${t('Conversation')}, ${t('{{count}} unread', { count: unread })}`
      : t('Conversation')

  function changeOpen(next: boolean) {
    setOpen(next)
    onOpenChangeRef.current?.(next)
  }

  useEffect(() => {
    if (!open) return
    let cancelled = false
    const mark =
      props.mode === 'admin'
        ? markAdminSupplierApplicationRead
        : markSupplierApplicationRead
    void mark(props.applicationId)
      .then((result) => {
        if (!cancelled) onReadRef.current?.(result.unread_count)
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [open, props.applicationId, props.mode])

  useEffect(() => {
    if (!open) return
    function onKey(event: KeyboardEvent) {
      if (event.key !== 'Escape') return
      event.preventDefault()
      event.stopPropagation()
      event.stopImmediatePropagation()
      setOpen(false)
      onOpenChangeRef.current?.(false)
    }
    document.addEventListener('keydown', onKey, true)
    return () => document.removeEventListener('keydown', onKey, true)
  }, [open])

  return (
    <>
      <Button
        type='button'
        variant='outline'
        aria-label={buttonLabel}
        onClick={() => {
          setMounted(true)
          changeOpen(true)
        }}
      >
        {t('Conversation')}
        {unread > 0 ? <span className='tabular-nums'>{unread}</span> : null}
      </Button>
      {mounted &&
        createPortal(
          <div hidden={!open} style={{ position: 'fixed', inset: 0, zIndex: 80 }}>
            <div
              data-supplier-conversation-backdrop
              onClick={() => changeOpen(false)}
              style={{
                position: 'absolute',
                inset: 0,
                background: 'rgba(0, 0, 0, 0.45)',
              }}
            />
            <div
              role='dialog'
              aria-modal='true'
              aria-labelledby={titleId}
              data-supplier-conversation-dialog
              style={{
                position: 'absolute',
                top: '50%',
                left: '50%',
                transform: 'translate(-50%, -50%)',
                display: 'flex',
                flexDirection: 'column',
                width: '56rem',
                maxWidth: '100vw',
                height: '48rem',
                maxHeight: '100dvh',
                minHeight: 0,
                overflow: 'hidden',
                background: 'var(--background, Canvas)',
                borderRadius: 12,
              }}
            >
              <div
                className='flex items-center justify-between gap-3 p-4'
                style={{ flexShrink: 0 }}
              >
                <h2
                  id={titleId}
                  className='min-w-0 text-base font-semibold [overflow-wrap:anywhere]'
                >
                  {title}
                </h2>
                <Button
                  type='button'
                  variant='ghost'
                  size='icon'
                  aria-label={t('Close')}
                  onClick={() => changeOpen(false)}
                >
                  <X aria-hidden='true' />
                </Button>
              </div>
              <div
                data-supplier-conversation-slot
                style={{
                  display: 'flex',
                  flex: '1 1 auto',
                  minHeight: 0,
                  overflow: 'hidden',
                  padding: 16,
                  paddingTop: 0,
                }}
              >
                <SupplierConversation
                  applicationId={props.applicationId}
                  messages={props.messages}
                  canSend={props.canSend}
                  mode={props.mode}
                  dock='fill'
                  onSent={props.onSent}
                />
              </div>
            </div>
          </div>,
          document.body
        )}
    </>
  )
}
