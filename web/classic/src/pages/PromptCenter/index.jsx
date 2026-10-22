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
import { Button, Input, Modal } from '@douyinfe/semi-ui';
import {
  KeyRound,
  BookOpen,
  Plus,
  Search,
  ShieldCheck,
  SlidersHorizontal,
  Sparkles,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { isAdmin } from '../../helpers/utils';
import { promptError, promptRequest } from './api';
import TokenPromptManager from './TokenPromptManager';
import PromptTemplateEditor from './PromptTemplateEditor';
import PromptCatalog from './PromptCatalog';
import TokenPicker from './TokenPicker';
import './style.css';

export default function PromptCenter() {
  const { t } = useTranslation();
  const admin = isAdmin();
  const [templates, setTemplates] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [revision, setRevision] = useState(0);
  const [editing, setEditing] = useState(null);
  const [manage, setManage] = useState(false);
  const [search, setSearch] = useState('');
  const [picker, setPicker] = useState(null);
  const [installation, setInstallation] = useState(null);
  const [selectedToken, setSelectedToken] = useState(null);
  const publishedCount = templates.filter((item) => item.enabled).length;
  const groupCount = new Set(
    templates.flatMap((item) =>
      (item.group_details || []).map((group) => group.id),
    ),
  ).size;
  const filteredTemplates = templates.filter((item) =>
    `${item.title} ${item.description || ''}`
      .toLocaleLowerCase()
      .includes(search.toLocaleLowerCase()),
  );
  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError('');
    promptRequest(
      'get',
      admin && manage ? '/api/prompt-center/admin/' : '/api/prompt-center/',
      undefined,
      controller.signal,
    )
      .then((items) => {
        if (!controller.signal.aborted) setTemplates(items || []);
      })
      .catch((reason) => {
        if (!controller.signal.aborted) setError(promptError(reason, t));
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [admin, manage, revision, t]);
  return (
    <main className='classic-console-page'>
      <div className='classic-console-page-container prompt-center-page'>
        <div className='prompt-center-surface'>
          <header className='prompt-center-hero'>
            <div className='prompt-center-hero-main'>
              <div className='prompt-center-hero-heading'>
                <div className='prompt-center-hero-mark' aria-hidden='true'>
                  <Sparkles size={26} />
                </div>
                <div className='prompt-center-hero-copy'>
                  <div className='prompt-center-eyebrow'>
                    <span>
                      <BookOpen size={13} aria-hidden='true' />
                      {t('Prompt center')}
                    </span>
                    <span>{t('Prompt')}</span>
                  </div>
                  <h1>{t('Reusable prompts for your API keys')}</h1>
                  <p>
                    {t(
                      'Install text prompts to a token. The gateway loads them only when the actual request group is allowed.',
                    )}
                  </p>
                </div>
              </div>
              <div className='prompt-center-hero-side'>
                <div className='prompt-center-hero-actions'>
                  <Button
                    className='prompt-center-secondary'
                    icon={<KeyRound size={15} />}
                    onClick={() => setPicker({})}
                  >
                    {t('Manage token prompts')}
                  </Button>
                  {admin && (
                    <>
                      <Button
                        className='prompt-center-secondary'
                        icon={<SlidersHorizontal size={15} />}
                        onClick={() => setManage((value) => !value)}
                      >
                        {manage ? t('Browse prompts') : t('Manage templates')}
                      </Button>
                      <Button
                        className='prompt-center-primary'
                        icon={<Plus size={15} />}
                        onClick={() => setEditing({})}
                      >
                        {t('New template')}
                      </Button>
                    </>
                  )}
                </div>
                <div
                  className='prompt-center-stats'
                  aria-label={t('Prompt center')}
                >
                  <div className='prompt-center-stat'>
                    <small>{t('Templates')}</small>
                    <strong>{templates.length}</strong>
                  </div>
                  <div className='prompt-center-stat'>
                    <small>{t('Published')}</small>
                    <strong>{publishedCount}</strong>
                  </div>
                  <div className='prompt-center-stat'>
                    <small>{t('Allowed groups')}</small>
                    <strong>{groupCount}</strong>
                  </div>
                </div>
              </div>
            </div>
            <div className='prompt-center-hero-bottom'>
              <p>
                <ShieldCheck size={16} aria-hidden='true' />
                <span>
                  {t(
                    'Supports Chat Completions, Responses, Claude Messages and Gemini conversations. Markdown and SKILL.md are text only; scripts, attachments and tools do not run.',
                  )}
                </span>
              </p>
              <Input
                prefix={<Search size={16} aria-hidden='true' />}
                aria-label={t('Search prompts')}
                placeholder={t('Search prompts')}
                value={search}
                onChange={setSearch}
                showClear
              />
            </div>
          </header>
          <div className='prompt-center-content-area'>
            {error && (
              <div role='alert' className='prompt-center-error'>
                {error}
                <Button onClick={() => setRevision((value) => value + 1)}>
                  {t('Retry')}
                </Button>
              </div>
            )}
            {loading && (
              <div role='status' className='prompt-center-empty-state'>
                <Sparkles size={18} aria-hidden='true' />
                {t('Loading prompts')}
              </div>
            )}
            {!loading && !error && (
              <PromptCatalog
                templates={filteredTemplates}
                bindings={[]}
                canInstall
                onInstall={(template) => setPicker({ template })}
                onEdit={admin && manage ? setEditing : undefined}
              />
            )}
          </div>
        </div>
        {picker && (
          <TokenPicker
            open
            hideTrigger
            token={selectedToken}
            onClose={() => setPicker(null)}
            revision={revision}
            onSelect={(token) => {
              setSelectedToken(token);
              setInstallation({ token, template: picker.template });
              setPicker(null);
            }}
          />
        )}
        {installation && (
          <Modal
            visible
            title={
              <span className='prompt-installation-title'>
                <KeyRound size={18} aria-hidden='true' />
                <span title={installation.token.name}>
                  {installation.token.name}
                </span>
              </span>
            }
            footer={null}
            width={760}
            style={{ maxWidth: 'calc(100vw - 24px)' }}
            onCancel={() => setInstallation(null)}
            className='prompt-installation-modal'
          >
            <div className='prompt-center-page prompt-installation-content'>
              <TokenPromptManager
                token={installation.token}
                templateToInstall={installation.template}
                onChange={(items) =>
                  setSelectedToken((current) => ({
                    ...current,
                    prompt_count: items.length,
                  }))
                }
              />
            </div>
          </Modal>
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
