/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useQuery } from '@tanstack/react-query'
import {
  Bold,
  Code,
  FileCode,
  Italic,
  Link,
  Quote,
  Strikethrough,
  Underline,
  VenetianMask,
  WrapText,
} from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

import { previewNotificationTemplate } from './api'
import { TelegramPreview } from './telegram-preview'
import { isTelegramLink } from './template'
import type { NotificationPreviewInput, NotificationTarget } from './types'

const EMOJI = ['🎫', '👤', '🏷️', '💬', '✅', '⚠️', '🔔', '📌']

export const TICKET_EMOJI_TEMPLATE =
  '{{mention}} 🎫 <b>{{action_label}} #{{ticket_id}}：{{title}}</b>\n👤 {{user_id}}\n🏷 {{category}} · {{priority}}\n<blockquote>{{content}}</blockquote>'

export function TelegramTemplateEditor(props: {
  value: string
  eventType: string
  target?: NotificationTarget
  onChange: (value: string) => void
}) {
  const { t } = useTranslation()
  const textarea = useRef<HTMLTextAreaElement>(null)
  const selection = useRef({ start: 0, end: 0 })
  const pendingSelection = useRef<{ start: number; end: number } | null>(null)
  const [linkOpen, setLinkOpen] = useState(false)
  const [linkUrl, setLinkUrl] = useState('https://')
  const request: NotificationPreviewInput = {
    event_type: props.eventType,
    template: props.value,
    target: {
      mention_user_id: props.target?.mention_user_id ?? '',
      mention_name: props.target?.mention_name ?? '',
    },
  }
  const requestKey = JSON.stringify(request)
  const [debouncedKey, setDebouncedKey] = useState('')
  useEffect(() => {
    const timer = window.setTimeout(() => setDebouncedKey(requestKey), 300)
    return () => window.clearTimeout(timer)
  }, [requestKey])
  const preview = useQuery({
    queryKey: ['notification-template-preview', debouncedKey],
    queryFn: ({ signal }) =>
      previewNotificationTemplate(
        JSON.parse(debouncedKey) as NotificationPreviewInput,
        signal
      ),
    enabled: debouncedKey !== '' && debouncedKey === requestKey,
    retry: false,
    gcTime: 0,
  })
  // 表单值变化后只恢复由工具栏发起的选区，普通输入保持浏览器原生行为。
  useEffect(() => {
    if (!pendingSelection.current || !textarea.current) return
    textarea.current.focus()
    textarea.current.setSelectionRange(
      pendingSelection.current.start,
      pendingSelection.current.end
    )
    selection.current = pendingSelection.current
    pendingSelection.current = null
  }, [props.value])

  const insert = (prefix: string, suffix = '') => {
    const start = Math.min(selection.current.start, props.value.length)
    const end = Math.min(
      Math.max(selection.current.end, start),
      props.value.length
    )
    const content = suffix ? props.value.slice(start, end) : ''
    pendingSelection.current = {
      start: start + prefix.length,
      end: end + prefix.length,
    }
    if (!suffix) {
      pendingSelection.current = {
        start: start + prefix.length,
        end: start + prefix.length,
      }
    }
    props.onChange(
      props.value.slice(0, start) +
        prefix +
        content +
        suffix +
        props.value.slice(end)
    )
  }

  const formats = [
    { label: t('Bold'), icon: Bold, open: '<b>', close: '</b>' },
    { label: t('Italic'), icon: Italic, open: '<i>', close: '</i>' },
    { label: t('Underline'), icon: Underline, open: '<u>', close: '</u>' },
    {
      label: t('Strikethrough'),
      icon: Strikethrough,
      open: '<s>',
      close: '</s>',
    },
    {
      label: t('Quote'),
      icon: Quote,
      open: '<blockquote>',
      close: '</blockquote>',
    },
    {
      label: t('Expandable quote'),
      icon: WrapText,
      open: '<blockquote expandable>',
      close: '</blockquote>',
    },
    { label: t('Inline code'), icon: Code, open: '<code>', close: '</code>' },
    { label: t('Code block'), icon: FileCode, open: '<pre>', close: '</pre>' },
    {
      label: t('Spoiler'),
      icon: VenetianMask,
      open: '<tg-spoiler>',
      close: '</tg-spoiler>',
    },
  ]
  const waiting =
    requestKey !== debouncedKey || preview.isPending || preview.isFetching

  return (
    <div className='grid min-w-0 gap-3 md:grid-cols-2'>
      <div className='bg-card min-w-0 rounded-lg border'>
        <div className='space-y-2 border-b p-3'>
          <Label htmlFor='notification-template'>{t('Message template')}</Label>
          <div
            role='group'
            aria-label={t('Telegram formatting')}
            className='flex flex-wrap gap-1'
          >
            {formats.map((format) => (
              <Button
                key={format.open}
                type='button'
                size='icon-xs'
                variant='ghost'
                title={format.label}
                aria-label={format.label}
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => insert(format.open, format.close)}
              >
                <format.icon className='size-4' />
              </Button>
            ))}
            <Button
              type='button'
              size='icon-xs'
              variant='ghost'
              title={t('Insert link')}
              aria-label={t('Insert link')}
              aria-expanded={linkOpen}
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => setLinkOpen((current) => !current)}
            >
              <Link className='size-4' />
            </Button>
          </div>
          <div
            role='group'
            aria-label={t('Common emoji')}
            className='flex flex-wrap gap-1'
          >
            {EMOJI.map((emoji) => (
              <Button
                key={emoji}
                type='button'
                size='icon-xs'
                variant='ghost'
                aria-label={t('Insert emoji {{emoji}}', { emoji })}
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => insert(emoji)}
              >
                {emoji}
              </Button>
            ))}
          </div>
          {linkOpen && (
            <div className='flex min-w-0 flex-wrap items-center gap-2'>
              <Input
                aria-label={t('Link URL')}
                value={linkUrl}
                className='min-w-0 flex-1'
                onChange={(event) => setLinkUrl(event.target.value)}
              />
              <Button
                type='button'
                size='sm'
                variant='outline'
                disabled={!isTelegramLink(linkUrl)}
                onClick={() => {
                  const href = linkUrl
                    .replaceAll('&', '&amp;')
                    .replaceAll('"', '&quot;')
                    .replaceAll('<', '&lt;')
                    .replaceAll('>', '&gt;')
                  insert(`<a href="${href}">`, '</a>')
                  setLinkOpen(false)
                }}
              >
                {t('Insert link')}
              </Button>
            </div>
          )}
          {props.eventType === 'ticket_user_message' && (
            <Button
              type='button'
              size='sm'
              variant='outline'
              onClick={() => props.onChange(TICKET_EMOJI_TEMPLATE)}
            >
              {t('Use ticket emoji preset')}
            </Button>
          )}
        </div>
        <div className='space-y-2 p-3'>
          <Textarea
            ref={textarea}
            id='notification-template'
            className='min-h-64 font-mono text-xs'
            value={props.value}
            onChange={(event) => props.onChange(event.target.value)}
            onSelect={(event) => {
              selection.current = {
                start: event.currentTarget.selectionStart,
                end: event.currentTarget.selectionEnd,
              }
            }}
          />
          <p className='text-muted-foreground text-xs'>
            {t(
              'Customize all text, emoji and prefixes. Formatting buttons insert Telegram HTML.'
            )}
          </p>
        </div>
      </div>
      <section
        aria-label={t('Live Telegram preview')}
        className='bg-card min-w-0 rounded-lg border'
      >
        <div className='space-y-1 border-b p-3'>
          <p className='text-sm font-medium'>{t('Live Telegram preview')}</p>
          <p className='text-muted-foreground text-xs'>
            {t(
              'Sample preview only. Uses the first recipient mention and does not send a message.'
            )}
          </p>
        </div>
        <div className='min-h-64 p-3' aria-busy={waiting}>
          {waiting && (
            <p role='status' className='text-muted-foreground text-sm'>
              {t('Updating preview…')}
            </p>
          )}
          {!waiting && preview.isError && (
            <p
              role='alert'
              className='text-destructive bg-destructive/5 rounded-md border p-3 text-sm'
            >
              {t('Preview failed')}: {preview.error.message}
            </p>
          )}
          {!waiting && !preview.isError && preview.data && (
            <div className='space-y-3'>
              <div className='bg-muted/20 rounded-lg border p-3 text-sm'>
                <TelegramPreview html={preview.data.html} />
              </div>
              <p className='text-muted-foreground text-xs'>
                {t('{{count}} characters', { count: preview.data.text_length })}
              </p>
            </div>
          )}
        </div>
      </section>
    </div>
  )
}
