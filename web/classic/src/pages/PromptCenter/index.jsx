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
import { Button } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { isAdmin } from '../../helpers/utils';
import { promptError, promptRequest } from './api';
import TokenPromptManager from './TokenPromptManager';
import PromptTemplateEditor from './PromptTemplateEditor';
import './style.css';

export default function PromptCenter() {
  const { t } = useTranslation();
  const admin = isAdmin();
  const [templates, setTemplates] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [revision, setRevision] = useState(0);
  const [editing, setEditing] = useState(null);
  const [tokens, setTokens] = useState([]);
  const [token, setToken] = useState(null);
  const [keyword, setKeyword] = useState('');
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [tokenError, setTokenError] = useState('');
  const [tokenLoading, setTokenLoading] = useState(false);
  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError('');
    promptRequest(
      'get',
      admin ? '/api/prompt-center/admin/' : '/api/prompt-center/',
      undefined,
      controller.signal,
    )
      .then((items) => setTemplates(items || []))
      .catch((reason) => {
        if (!controller.signal.aborted) setError(promptError(reason, t));
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [admin, revision, t]);
  useEffect(() => {
    const controller = new AbortController();
    setTokenLoading(true);
    setTokenError('');
    const url = search
      ? `/api/token/search?keyword=${encodeURIComponent(search)}&p=${page}&size=20`
      : `/api/token/?p=${page}&size=20`;
    promptRequest('get', url, undefined, controller.signal)
      .then((data) => {
        setTokens(data.items || []);
        setTotal(data.total || 0);
      })
      .catch((reason) => {
        if (!controller.signal.aborted) setTokenError(promptError(reason, t));
      })
      .finally(() => {
        if (!controller.signal.aborted) setTokenLoading(false);
      });
    return () => controller.abort();
  }, [search, page, revision, t]);
  return (
    <main className='classic-console-page'>
      <div className='classic-console-page-container prompt-center-stack'>
        <header className='prompt-center-row'>
          <h1>{t('Prompt center')}</h1>
          {admin && (
            <Button onClick={() => setEditing({})}>{t('New template')}</Button>
          )}
        </header>
        <section className='classic-console-panel'>
          <div className='classic-console-panel-content prompt-center-stack'>
            <p>
              {t(
                'Install prompts on your API keys. Existing client instructions are preserved; loaded text counts toward input tokens.',
              )}
            </p>
            <p className='prompt-center-muted'>
              {t(
                'Supports Chat Completions, Responses, Claude Messages and Gemini conversations. Other endpoints do not load prompts.',
              )}
            </p>
            <p className='prompt-center-muted'>
              {t(
                'Text prompts only. Scripts, tools and attachments in full Skills are not executed.',
              )}
            </p>
            <form
              className='prompt-center-row'
              onSubmit={(event) => {
                event.preventDefault();
                setSearch(keyword);
                setPage(1);
              }}
            >
              <label htmlFor='prompt-token-search'>
                {t('Search token names')}
              </label>
              <input
                id='prompt-token-search'
                value={keyword}
                onChange={(event) => setKeyword(event.target.value)}
              />
              <Button htmlType='submit' disabled={tokenLoading}>
                {t('Search')}
              </Button>
            </form>
            <label htmlFor='prompt-token'>{t('Choose a token')}</label>
            <select
              id='prompt-token'
              disabled={tokenLoading}
              value={token?.id || ''}
              onChange={(event) =>
                setToken(
                  tokens.find(
                    (item) => String(item.id) === event.target.value,
                  ) || null,
                )
              }
            >
              <option value=''>{t('Choose a token')}</option>
              {token && !tokens.some((item) => item.id === token.id) && (
                <option value={token.id}>{token.name}</option>
              )}
              {tokens.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.name}
                </option>
              ))}
            </select>
            {!tokens.length && !tokenLoading && !tokenError && (
              <p>{t('No tokens found. Create an API key first.')}</p>
            )}
            {tokenError && (
              <p role='alert' className='prompt-center-error'>
                {tokenError}
                <Button onClick={() => setRevision((value) => value + 1)}>
                  {t('Retry')}
                </Button>
              </p>
            )}
            {total > 20 && (
              <div className='prompt-center-row'>
                <Button
                  disabled={tokenLoading || page === 1}
                  onClick={() => setPage((value) => value - 1)}
                >
                  {t('Previous')}
                </Button>
                <span>{page}</span>
                <Button
                  disabled={tokenLoading || page * 20 >= total}
                  onClick={() => setPage((value) => value + 1)}
                >
                  {t('Next')}
                </Button>
              </div>
            )}
          </div>
        </section>
        {error && (
          <div role='alert' className='prompt-center-error'>
            {error}
            <Button onClick={() => setRevision((value) => value + 1)}>
              {t('Retry')}
            </Button>
          </div>
        )}
        {loading ? (
          <p role='status'>{t('Loading prompts')}</p>
        ) : (
          !error && (
            <TokenPromptManager
              key={`${token?.id || 'none'}-${revision}`}
              token={token}
              templates={templates}
              onEdit={admin ? setEditing : undefined}
            />
          )
        )}
        {editing && (
          <PromptTemplateEditor
            template={editing}
            onCancel={() => setEditing(null)}
            onSaved={() => {
              setEditing(null);
              setRevision((value) => value + 1);
            }}
          />
        )}
      </div>
    </main>
  );
}
