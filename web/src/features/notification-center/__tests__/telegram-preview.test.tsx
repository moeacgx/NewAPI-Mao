/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { fireEvent, render, screen } from '@testing-library/react'
import { expect, test } from 'vitest'

import { TelegramPreview } from '../telegram-preview'

test('Telegram 格式转换为安全语义元素，引用可展开且剧透可揭示', () => {
  const view = render(
    <TelegramPreview html='<b>Title</b><i>Italic text</i><u>Underline text</u><s>Old text</s><blockquote>Body</blockquote><blockquote expandable>More</blockquote><tg-spoiler>Secret</tg-spoiler><pre><code>const n = 1</code></pre><a href="tg://user?id=123">Ops</a>' />
  )
  expect(view.container.querySelector('strong')).toHaveTextContent('Title')
  expect(view.container.querySelector('blockquote')).toHaveTextContent('Body')
  expect(view.container.querySelector('details')).not.toHaveAttribute('open')
  fireEvent.click(screen.getByText('Expandable quote'))
  expect(view.container.querySelector('details')).toHaveAttribute('open')
  const spoiler = screen.getByRole('button', { name: 'Reveal spoiler' })
  expect(spoiler).toHaveAttribute('aria-pressed', 'false')
  fireEvent.click(spoiler)
  expect(spoiler).toHaveAttribute('aria-pressed', 'true')
  expect(screen.getByRole('link', { name: 'Ops' })).toHaveAttribute(
    'href',
    'tg://user?id=123'
  )
})

test('预览拒绝脚本、任意标签属性和危险链接协议，保留转义的用户正文', () => {
  const view = render(
    <TelegramPreview
      html={
        '<script>alert(1)</script><svg onload="alert(1)"></svg><img src="x" onerror="alert(1)"><b onclick="alert(1)" style="color:red">Safe title</b><a href="javascript:alert(1)">Unsafe link</a><a href="data:text/html,test">Data link</a><a href="https://example.com" onclick="alert(1)">Safe link</a><blockquote>&lt;img src=x onerror=alert(1)&gt;</blockquote>'
      }
    />
  )
  expect(
    view.container.querySelector('script,svg,img,[onclick],[onerror],[style]')
  ).toBeNull()
  expect(screen.getAllByRole('link')).toHaveLength(1)
  expect(screen.getByRole('link', { name: 'Safe link' })).toHaveAttribute(
    'rel',
    'noopener noreferrer'
  )
  expect(view.container).toHaveTextContent('Unsafe link')
  expect(view.container.querySelector('blockquote')).toHaveTextContent(
    '<img src=x onerror=alert(1)>'
  )
})
