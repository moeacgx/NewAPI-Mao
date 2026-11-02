/*
Copyright (C) 2025 QuantumNous

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

import React, { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { API } from '../../helpers/api';

export function isSafeTelegramLink(value) {
  if (!value || /[\u0000-\u0020\u007f]/.test(value)) return false;
  try {
    return ['http:', 'https:', 'tg:', 'mailto:'].includes(
      new URL(value).protocol,
    );
  } catch {
    return false;
  }
}

function renderTelegramNode(node, key, t, depth = 0) {
  if (node.nodeType === 3) return node.textContent;
  if (node.nodeType !== 1 || depth > 64) return null;
  const tag = node.tagName.toLowerCase();
  const allowed = [
    'b',
    'strong',
    'i',
    'em',
    'u',
    'ins',
    's',
    'strike',
    'del',
    'span',
    'tg-spoiler',
    'a',
    'code',
    'pre',
    'blockquote',
  ];
  // 仅重建 Telegram 支持的标签，不复制来源属性或未知标签的子树。
  if (!allowed.includes(tag)) return null;
  const children = Array.from(node.childNodes).map((child, index) =>
    renderTelegramNode(child, `${key}-${index}`, t, depth + 1),
  );
  if (tag === 'a') {
    const href = node.getAttribute('href') || '';
    if (!isSafeTelegramLink(href)) return <span key={key}>{children}</span>;
    return (
      <a key={key} href={href} target='_blank' rel='noopener noreferrer'>
        {children}
      </a>
    );
  }
  if (tag === 'span' || tag === 'tg-spoiler') {
    if (tag === 'span' && node.getAttribute('class') !== 'tg-spoiler')
      return <span key={key}>{children}</span>;
    return (
      <details key={key} className='classic-telegram-spoiler'>
        <summary>{t('Spoiler')}</summary>
        {children}
      </details>
    );
  }
  if (tag === 'blockquote' && node.hasAttribute('expandable')) {
    return (
      <details key={key} className='classic-telegram-quote'>
        <summary>{t('Expandable quote')}</summary>
        <blockquote>{children}</blockquote>
      </details>
    );
  }
  return React.createElement(tag, { key }, children);
}

export function TelegramMessage(props) {
  const { t } = useTranslation();
  const content = useMemo(() => {
    const doc = new DOMParser().parseFromString(props.html, 'text/html');
    return Array.from(doc.body.childNodes).map((node, index) =>
      renderTelegramNode(node, String(index), t),
    );
  }, [props.html, t]);
  return <div className='classic-telegram-message'>{content}</div>;
}

export default function TelegramPreview(props) {
  const { t } = useTranslation();
  const requestKey = JSON.stringify({
    event_type: props.eventType,
    template: props.template,
    target: {
      mention_user_id: String(props.target?.mention_user_id || ''),
      mention_name: String(props.target?.mention_name || ''),
    },
  });
  const [result, setResult] = useState(null);

  useEffect(() => {
    const controller = new AbortController();
    let active = true;
    setResult(null);
    const timer = setTimeout(async () => {
      try {
        const response = await API.post(
          '/api/notification/preview',
          JSON.parse(requestKey),
          {
            signal: controller.signal,
            skipErrorHandler: true,
          },
        );
        if (!active) return;
        if (
          !response.data?.success ||
          typeof response.data.data?.html !== 'string'
        )
          throw new Error(
            response.data?.message || t('Unable to preview template'),
          );
        setResult({ key: requestKey, data: response.data.data });
      } catch (error) {
        if (!active) return;
        setResult({
          key: requestKey,
          error:
            error?.response?.data?.message ||
            error.message ||
            t('Unable to preview template'),
        });
      }
    }, 300);
    return () => {
      active = false;
      clearTimeout(timer);
      controller.abort();
    };
  }, [requestKey, t]);

  // 当前输入与结果身份一起比较，避免 effect 执行前短暂展示旧预览。
  const current = result?.key === requestKey ? result : null;
  return (
    <section
      className='classic-telegram-preview'
      aria-label={t('Live message preview')}
    >
      <div className='classic-telegram-panel-header'>
        <strong>{t('Live message preview')}</strong>
        <p>
          {t(
            'Sample data only. No message is sent. Uses the first recipient mention.',
          )}
        </p>
      </div>
      <div
        className='classic-telegram-preview-body'
        aria-live='polite'
        aria-busy={!current}
      >
        {!current && <p role='status'>{t('Updating preview…')}</p>}
        {current?.error && (
          <p role='alert' className='classic-telegram-preview-error'>
            {current.error}
          </p>
        )}
        {current?.data && (
          <>
            <TelegramMessage html={current.data.html} />
            <p className='classic-telegram-length'>
              {t('Message length: {{count}} / 4096', {
                count: current.data.text_length,
              })}
            </p>
          </>
        )}
      </div>
    </section>
  );
}
