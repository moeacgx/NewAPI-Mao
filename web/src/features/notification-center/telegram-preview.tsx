/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { Fragment, useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { isTelegramLink } from './template'

function Spoiler(props: { children: ReactNode }) {
  const { t } = useTranslation()
  const [revealed, setRevealed] = useState(false)
  return (
    <button
      type='button'
      aria-label={t('Reveal spoiler')}
      aria-pressed={revealed}
      className='bg-muted inline rounded px-0.5 text-start'
      onClick={() => setRevealed((current) => !current)}
    >
      <span className={revealed ? '' : 'blur-sm'}>{props.children}</span>
    </button>
  )
}

// 只生成固定的 React 元素，不把服务器返回的属性或 HTML 注入页面。
function renderTelegramNode(
  node: Node,
  key: string,
  quoteLabel: string
): ReactNode {
  if (node.nodeType === Node.TEXT_NODE) return node.textContent
  if (!(node instanceof Element)) return null
  const tag = node.tagName.toLowerCase()
  if (['script', 'style', 'iframe', 'object', 'svg', 'math'].includes(tag)) {
    return null
  }
  const children = [...node.childNodes].map((child, index) =>
    renderTelegramNode(child, `${key}-${index}`, quoteLabel)
  )
  switch (tag) {
    case 'b':
    case 'strong':
      return <strong key={key}>{children}</strong>
    case 'i':
    case 'em':
      return <em key={key}>{children}</em>
    case 'u':
    case 'ins':
      return <u key={key}>{children}</u>
    case 's':
    case 'strike':
    case 'del':
      return <s key={key}>{children}</s>
    case 'code':
      return (
        <code key={key} className='bg-muted rounded px-1 font-mono'>
          {children}
        </code>
      )
    case 'pre':
      return (
        <pre
          key={key}
          className='bg-muted my-2 overflow-x-auto rounded-md p-3 font-mono'
        >
          {children}
        </pre>
      )
    case 'a': {
      const href = node.getAttribute('href') ?? ''
      if (!isTelegramLink(href)) {
        return <Fragment key={key}>{children}</Fragment>
      }
      return (
        <a
          key={key}
          href={href}
          target='_blank'
          rel='noopener noreferrer'
          className='text-primary underline'
        >
          {children}
        </a>
      )
    }
    case 'blockquote':
      if (node.hasAttribute('expandable')) {
        return (
          <details
            key={key}
            className='border-primary/50 bg-muted/40 my-2 rounded-md border-l-4 p-2'
          >
            <summary className='cursor-pointer text-xs'>{quoteLabel}</summary>
            <div className='pt-2'>{children}</div>
          </details>
        )
      }
      return (
        <blockquote
          key={key}
          className='border-primary/50 bg-muted/40 my-2 rounded-md border-l-4 p-2'
        >
          {children}
        </blockquote>
      )
    case 'tg-spoiler':
      return <Spoiler key={key}>{children}</Spoiler>
    case 'span':
      if (node.getAttribute('class') === 'tg-spoiler') {
        return <Spoiler key={key}>{children}</Spoiler>
      }
      return <Fragment key={key}>{children}</Fragment>
    default:
      return <Fragment key={key}>{children}</Fragment>
  }
}

export function TelegramPreview(props: { html: string }) {
  const { t } = useTranslation()
  const quoteLabel = t('Expandable quote')
  const content = useMemo(() => {
    const document = new DOMParser().parseFromString(props.html, 'text/html')
    return [...document.body.childNodes].map((node, index) =>
      renderTelegramNode(node, String(index), quoteLabel)
    )
  }, [props.html, quoteLabel])
  return (
    <div className='min-w-0 [overflow-wrap:anywhere] break-words whitespace-pre-wrap'>
      {content}
    </div>
  )
}
