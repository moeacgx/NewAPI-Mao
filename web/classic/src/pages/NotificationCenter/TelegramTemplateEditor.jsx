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

import React, { useLayoutEffect, useRef, useState } from 'react';
import { Button, Input } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import TelegramPreview, { isSafeTelegramLink } from './TelegramPreview';
import './telegram-editor.css';

const FORMATS = [
  ['Bold', '<b>', '</b>', 'B'],
  ['Italic', '<i>', '</i>', 'I'],
  ['Underline', '<u>', '</u>', 'U'],
  ['Strikethrough', '<s>', '</s>', 'S'],
  ['Quote', '<blockquote>', '</blockquote>', '❝'],
  ['Expandable quote', '<blockquote expandable>', '</blockquote>', '❝…'],
  ['Inline code', '<code>', '</code>', '</>'],
  ['Code block', '<pre>', '</pre>', '{ }'],
  ['Spoiler', '<tg-spoiler>', '</tg-spoiler>', '▧'],
];
const EMOJIS = ['🎫', '👤', '📂', '💬', '⚠️', '✅', '🕒', '📌'];

export default function TelegramTemplateEditor(props) {
  const { t } = useTranslation();
  const textarea = useRef(null);
  const selection = useRef({ start: 0, end: 0 });
  const pendingSelection = useRef(null);
  const [linkOpen, setLinkOpen] = useState(false);
  const [selectionRevision, setSelectionRevision] = useState(0);
  const [linkUrl, setLinkUrl] = useState('https://');
  const [linkError, setLinkError] = useState(false);

  useLayoutEffect(() => {
    if (!pendingSelection.current || !textarea.current) return;
    const range = pendingSelection.current;
    pendingSelection.current = null;
    textarea.current.focus();
    textarea.current.setSelectionRange(range.start, range.end);
    selection.current = range;
  }, [props.value, selectionRevision]);

  const insert = (prefix, suffix = '') => {
    const value = props.value || '';
    const start = Math.min(selection.current.start, value.length);
    const end = Math.min(selection.current.end, value.length);
    const selected = suffix ? value.slice(start, end) : '';
    pendingSelection.current = {
      start: start + prefix.length,
      end: start + prefix.length + selected.length,
    };
    props.onChange(
      value.slice(0, start) + prefix + selected + suffix + value.slice(end),
    );
    setSelectionRevision((current) => current + 1);
  };

  const applyTicketPreset = () => {
    const template =
      '{{mention}} 🎫 <b>{{action_label}} #{{ticket_id}}：{{title}}</b>\n👤 {{user_id}}\n🏷 {{category}} · {{priority}}\n<blockquote>{{content}}</blockquote>';
    pendingSelection.current = { start: template.length, end: template.length };
    props.onChange(template);
    setSelectionRevision((current) => current + 1);
  };

  return (
    <div className='classic-telegram-editor-layout'>
      <section
        className='classic-telegram-source'
        aria-label={t('Telegram HTML source')}
      >
        <div className='classic-telegram-panel-header'>
          <strong>{t('Telegram HTML source')}</strong>
          <p>{t('Edit all labels, prefixes and emoji in the template.')}</p>
        </div>
        <div
          className='classic-telegram-tools'
          role='toolbar'
          aria-label={t('Telegram formatting')}
        >
          {FORMATS.map(([label, prefix, suffix, icon]) => (
            <Button
              key={label}
              size='small'
              theme='borderless'
              aria-label={t(label)}
              title={t(label)}
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => insert(prefix, suffix)}
            >
              {icon}
            </Button>
          ))}
          <Button
            size='small'
            theme='borderless'
            aria-expanded={linkOpen}
            onMouseDown={(event) => event.preventDefault()}
            onClick={() => setLinkOpen(!linkOpen)}
          >
            {t('Insert link')}
          </Button>
        </div>
        {linkOpen && (
          <div className='classic-telegram-link-form'>
            <Input
              aria-label={t('Link URL')}
              value={linkUrl}
              onChange={(value) => {
                setLinkUrl(value);
                setLinkError(false);
              }}
            />
            <Button
              size='small'
              onClick={() => {
                if (!isSafeTelegramLink(linkUrl)) {
                  setLinkError(true);
                  return;
                }
                const escaped = linkUrl
                  .replaceAll('&', '&amp;')
                  .replaceAll('"', '&quot;')
                  .replaceAll('<', '&lt;')
                  .replaceAll('>', '&gt;');
                insert(`<a href="${escaped}">`, '</a>');
                setLinkOpen(false);
              }}
            >
              {t('Insert link')}
            </Button>
            {linkError && (
              <p role='alert'>{t('Use an http, https, tg or mailto link.')}</p>
            )}
          </div>
        )}
        <textarea
          ref={textarea}
          className='classic-telegram-source-input'
          aria-label={t('消息模板')}
          value={props.value}
          rows={10}
          spellCheck={false}
          onChange={(event) => props.onChange(event.target.value)}
          onSelect={(event) => {
            selection.current = {
              start: event.target.selectionStart,
              end: event.target.selectionEnd,
            };
          }}
          onBlur={(event) => {
            selection.current = {
              start: event.target.selectionStart,
              end: event.target.selectionEnd,
            };
          }}
        />
        <div
          className='classic-telegram-tools'
          role='group'
          aria-label={t('Common emoji')}
        >
          {EMOJIS.map((emoji) => (
            <Button
              key={emoji}
              size='small'
              theme='borderless'
              aria-label={t('Insert emoji {{emoji}}', { emoji })}
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => insert(emoji)}
            >
              {emoji}
            </Button>
          ))}
        </div>
        <div
          className='classic-telegram-variables'
          role='group'
          aria-label={t('Template variables')}
        >
          {(props.event.variables || []).map((variable) => (
            <Button
              key={variable}
              size='small'
              type='tertiary'
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => insert(`{{${variable}}}`)}
            >{`{{${variable}}}`}</Button>
          ))}
        </div>
        {props.eventType === 'ticket_user_message' && (
          <div className='classic-telegram-preset'>
            <Button size='small' type='tertiary' onClick={applyTicketPreset}>
              {t('Use ticket emoji and quote preset')}
            </Button>
            <p>{t('Replaces the current template only when clicked.')}</p>
          </div>
        )}
      </section>
      <TelegramPreview
        eventType={props.eventType}
        template={props.value}
        target={props.target}
      />
    </div>
  );
}
