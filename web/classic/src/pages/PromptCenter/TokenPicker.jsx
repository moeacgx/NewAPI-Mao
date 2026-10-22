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

import React, { useEffect, useState } from 'react';
import { Button, Input, Modal, Spin } from '@douyinfe/semi-ui';
import {
  AlertCircle,
  ArrowLeft,
  ArrowRight,
  Check,
  ChevronsUpDown,
  KeyRound,
  Search,
  SearchX,
  Sparkles,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { promptError, promptRequest } from './api';

export default function TokenPicker(props) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(Boolean(props.open));
  const [keyword, setKeyword] = useState('');
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [items, setItems] = useState([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [retry, setRetry] = useState(0);
  useEffect(() => {
    const timer = setTimeout(() => {
      setSearch(keyword.trim());
      setPage(1);
    }, 250);
    return () => clearTimeout(timer);
  }, [keyword]);
  useEffect(() => {
    if (!open) return;
    const controller = new AbortController();
    setLoading(true);
    setError('');
    const url = search
      ? `/api/token/search?keyword=${encodeURIComponent(search)}&p=${page}&size=20`
      : `/api/token/?p=${page}&size=20`;
    promptRequest('get', url, undefined, controller.signal)
      .then((data) => {
        if (controller.signal.aborted) return;
        setItems(data.items || []);
        setTotal(data.total || 0);
      })
      .catch((reason) => {
        if (!controller.signal.aborted) setError(promptError(reason, t));
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [open, search, page, retry, props.revision, t]);
  const pending = loading || keyword.trim() !== search;
  return (
    <>
      {!props.hideTrigger && (
        <button
          type='button'
          className='prompt-token-trigger'
          aria-label={t('Choose a token')}
          aria-haspopup='dialog'
          aria-expanded={open}
          disabled={props.disabled}
          onClick={() => setOpen(true)}
        >
          <span className='prompt-token-avatar' aria-hidden='true'>
            <KeyRound size={19} />
          </span>
          <span className='prompt-token-trigger-copy'>
            <small>{t('Current token')}</small>
            <strong title={props.token?.name}>
              {props.token?.name || t('Choose a token')}
            </strong>
          </span>
          {props.token && (
            <span className='prompt-token-count'>
              <Sparkles size={13} aria-hidden='true' />
              {props.token.prompt_count || 0}
            </span>
          )}
          <ChevronsUpDown
            size={16}
            className='prompt-token-chevron'
            aria-hidden='true'
          />
        </button>
      )}
      <Modal
        visible={open}
        title={t('Choose a token')}
        footer={null}
        onCancel={() => {
          setOpen(false);
          props.onClose?.();
        }}
        width={560}
        style={{ maxWidth: 'calc(100vw - 24px)' }}
        className='prompt-token-modal'
      >
        <div className='prompt-token-picker'>
          <p className='prompt-token-intro'>
            {props.introduction ||
              t('Find a token and install prompts in one place')}
          </p>
          <Input
            aria-label={t('Search token names')}
            placeholder={t('Search token names')}
            prefix={<Search size={17} aria-hidden='true' />}
            value={keyword}
            onChange={setKeyword}
            showClear
            autoFocus
            size='large'
          />
          <div className='prompt-token-results' aria-busy={pending}>
            {props.allowAccount && (
              <button
                type='button'
                className='prompt-token-option'
                aria-label={t('Account mode')}
                aria-pressed={!props.token}
                disabled={props.disabled}
                onClick={() => {
                  props.onSelect(null);
                  setOpen(false);
                }}
              >
                <span className='prompt-token-avatar' aria-hidden='true'>
                  <KeyRound size={18} />
                </span>
                <span className='prompt-token-option-copy'>
                  <strong>{t('Account mode')}</strong>
                  <small>{t('Use account balance and selected group')}</small>
                </span>
                {!props.token && (
                  <span className='prompt-token-selected'>
                    <Check size={16} aria-hidden='true' />
                  </span>
                )}
              </button>
            )}
            {pending && (
              <div className='prompt-token-state' role='status'>
                <Spin size='middle' />
                <span>{t('Loading tokens')}</span>
              </div>
            )}
            {!pending && error && (
              <div className='prompt-token-state' role='alert'>
                <AlertCircle size={30} aria-hidden='true' />
                <p>{error}</p>
                <Button onClick={() => setRetry((value) => value + 1)}>
                  {t('Retry')}
                </Button>
              </div>
            )}
            {!pending && !error && !items.length && (
              <div className='prompt-token-state'>
                <SearchX size={32} aria-hidden='true' />
                <strong>{t('No tokens found')}</strong>
                <p>
                  {search
                    ? t('Try another token name')
                    : t('No tokens found. Create an API key first.')}
                </p>
              </div>
            )}
            {!pending &&
              !error &&
              items.map((item) => {
                const selected = props.token?.id === item.id;
                const count = selected
                  ? props.token.prompt_count || 0
                  : item.prompt_count || 0;
                const groupName =
                  item.group_details
                    ?.map((group) => group.name)
                    .filter(Boolean)
                    .join('、') ||
                  item.group_name ||
                  (item.group === 'auto' ? t('自动分组') : t('继承账号分组'));
                return (
                  <button
                    type='button'
                    key={item.id}
                    className='prompt-token-option'
                    aria-label={item.name}
                    aria-pressed={selected}
                    disabled={props.disabled}
                    onClick={() => {
                      const selection = props.summaryOnly
                        ? {
                            id: item.id,
                            name: item.name,
                            group_name: groupName,
                            prompt_count: count,
                          }
                        : { ...item, prompt_count: count };
                      props.onSelect(selection);
                      setOpen(false);
                    }}
                  >
                    <span className='prompt-token-avatar' aria-hidden='true'>
                      <KeyRound size={18} />
                    </span>
                    <span className='prompt-token-option-copy'>
                      <strong title={item.name}>{item.name}</strong>
                      {props.showGroup && <small>{groupName}</small>}
                      <small>
                        <Sparkles size={12} aria-hidden='true' />
                        {t('{{count}} installed prompts', { count })}
                      </small>
                    </span>
                    {selected && (
                      <span className='prompt-token-selected'>
                        <Check size={16} aria-hidden='true' />
                      </span>
                    )}
                  </button>
                );
              })}
          </div>
          <div className='prompt-token-footer'>
            <span>{t('{{count}} tokens', { count: total })}</span>
            {total > 20 && (
              <div className='prompt-token-pagination'>
                <Button
                  aria-label={t('Previous')}
                  icon={<ArrowLeft size={15} />}
                  disabled={pending || page === 1}
                  onClick={() => setPage((value) => value - 1)}
                />
                <span>
                  {page} / {Math.ceil(total / 20)}
                </span>
                <Button
                  aria-label={t('Next')}
                  icon={<ArrowRight size={15} />}
                  disabled={pending || page * 20 >= total}
                  onClick={() => setPage((value) => value + 1)}
                />
              </div>
            )}
          </div>
        </div>
      </Modal>
    </>
  );
}
