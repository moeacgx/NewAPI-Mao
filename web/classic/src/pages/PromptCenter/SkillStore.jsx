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
import { Button, Modal } from '@douyinfe/semi-ui';
import { BookOpen, RefreshCw, Search } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { promptError, promptRequest } from './api';
import ArchiveImport from './ArchiveImport';
import RepositoryImport from './RepositoryImport';

export default function SkillStore(props) {
  const { t } = useTranslation();
  const [items, setItems] = useState([]);
  const [search, setSearch] = useState('');
  const [busy, setBusy] = useState(true);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState('');
  useEffect(() => {
    const controller = new AbortController();
    promptRequest(
      'get',
      '/api/prompt-center/admin/store/',
      undefined,
      controller.signal,
    )
      .then((data) => {
        if (!controller.signal.aborted) setItems(data || []);
      })
      .catch((reason) => {
        if (!controller.signal.aborted) setError(promptError(reason, t));
      })
      .finally(() => {
        if (!controller.signal.aborted) setBusy(false);
      });
    return () => controller.abort();
  }, [t]);
  const refresh = async () => {
    setBusy(true);
    setError('');
    try {
      setItems(
        (await promptRequest(
          'get',
          '/api/prompt-center/admin/store/?source=online',
        )) || [],
      );
    } catch (reason) {
      setError(promptError(reason, t));
    } finally {
      setBusy(false);
    }
  };
  const importItem = async (item) => {
    setBusy(true);
    setError('');
    try {
      props.onImport(
        await promptRequest(
          'post',
          `/api/prompt-center/admin/store/${encodeURIComponent(item.id)}/import`,
        ),
      );
    } catch (reason) {
      setError(promptError(reason, t));
    } finally {
      setBusy(false);
    }
  };
  const filtered = items.filter((item) =>
    `${item.title} ${item.description || ''} ${item.category || ''}`
      .toLocaleLowerCase()
      .includes(search.toLocaleLowerCase()),
  );
  return (
    <Modal
      visible
      title={t('Skill store')}
      footer={null}
      onCancel={() => {
        if (!busy && !uploading) props.onCancel();
      }}
      closable={!busy && !uploading}
      closeOnEsc={!busy && !uploading}
      maskClosable={false}
      width={840}
      style={{ maxWidth: 'calc(100vw - 24px)' }}
    >
      <div className='prompt-center-form prompt-skill-store'>
        <p className='prompt-center-muted'>
          {t(
            'Import a copy from the built-in or online store, then edit and publish it yourself.',
          )}
        </p>
        <div className='prompt-center-row'>
          <label className='prompt-store-search'>
            <Search size={15} aria-hidden='true' />
            <input
              aria-label={t('Search store skills')}
              placeholder={t('Search store skills')}
              value={search}
              onChange={(event) => setSearch(event.target.value)}
            />
          </label>
          <Button
            icon={<RefreshCw size={14} />}
            disabled={busy || uploading}
            onClick={refresh}
          >
            {t('Refresh online store')}
          </Button>
        </div>
        <ArchiveImport
          disabled={busy || uploading}
          onBusyChange={setUploading}
          onImport={props.onImport}
        />
        <RepositoryImport
          disabled={busy || uploading}
          onBusyChange={setUploading}
          onImport={props.onImport}
        />
        {error && (
          <p className='prompt-center-error' role='alert'>
            {error}
          </p>
        )}
        {busy && <p role='status'>{t('Loading prompts')}</p>}
        <div className='prompt-store-grid' aria-busy={busy}>
          {!busy && !filtered.length && (
            <div className='prompt-center-empty-state'>
              {t('No prompts available')}
            </div>
          )}
          {filtered.map((item) => (
            <article key={item.id} className='prompt-center-card'>
              <div className='prompt-center-card-topline'>
                <BookOpen size={16} aria-hidden='true' />
                <span className='prompt-center-group-chip'>
                  {item.source === 'online'
                    ? t('Online store')
                    : t('Built-in store')}
                </span>
              </div>
              <strong>{item.title}</strong>
              <p className='prompt-center-muted'>{item.description}</p>
              <div className='prompt-center-group-chips'>
                <span className='prompt-center-group-chip'>
                  {item.kind === 'bundle'
                    ? t('Skill bundle')
                    : t('Single skill')}
                </span>
                <span className='prompt-center-group-chip'>
                  {t('{{count}} skills', { count: item.skill_count || 1 })}
                </span>
                {item.category && (
                  <span className='prompt-center-group-chip'>
                    {item.category}
                  </span>
                )}
              </div>
              <Button
                disabled={busy || uploading}
                onClick={() => importItem(item)}
                aria-label={t('Import {{title}} and edit', {
                  title: item.title,
                })}
              >
                {t('Import and edit')}
              </Button>
            </article>
          ))}
        </div>
      </div>
    </Modal>
  );
}
