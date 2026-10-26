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

import React, { useEffect, useRef, useState } from 'react';
import { Button } from '@douyinfe/semi-ui';
import {
  FileText,
  Info,
  Layers3,
  PackageCheck,
  Save,
  ShieldCheck,
  Trash2,
  Undo2,
  Users,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { promptError, promptRequest } from './api';
import PromptCatalog from './PromptCatalog';
import SkillSummary from './SkillSummary';

const PROMPT_LIMIT = 8;

export default function TokenPromptManager(props) {
  const { t } = useTranslation();
  const [bindings, setBindings] = useState([]);
  const [loadedToken, setLoadedToken] = useState(null);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [revision, setRevision] = useState(0);
  const savedBindings = useRef([]);
  const tokenId = props.token?.id;
  const templateToInstall = props.templateToInstall;
  useEffect(() => {
    const controller = new AbortController();
    setError('');
    setBindings([]);
    setLoadedToken(null);
    setDirty(false);
    if (tokenId) {
      promptRequest(
        'get',
        `/api/token/${tokenId}/prompts`,
        undefined,
        controller.signal,
      )
        .then((items) => {
          if (controller.signal.aborted) return;
          const current = items || [];
          const addTemplate =
            templateToInstall &&
            !current.some((item) => item.template.id === templateToInstall.id);
          savedBindings.current = current;
          setLoadedToken(tokenId);
          if (addTemplate && current.length >= PROMPT_LIMIT) {
            setBindings(current);
            setError(t('A token can have at most 8 prompts'));
            return;
          }
          setBindings(
            addTemplate
              ? [
                  ...current,
                  {
                    template: templateToInstall,
                    enabled: true,
                    sort_order: current.length,
                  },
                ]
              : current,
          );
          setDirty(Boolean(addTemplate));
        })
        .catch((reason) => {
          if (!controller.signal.aborted) setError(promptError(reason, t));
        });
    }
    return () => controller.abort();
  }, [tokenId, revision, templateToInstall, t]);

  const ready = tokenId && loadedToken === tokenId;
  const save = async (next = bindings) => {
    if (!ready || saving) return;
    setSaving(true);
    setError('');
    try {
      await promptRequest('put', `/api/token/${tokenId}/prompts`, {
        bindings: next.map((item, index) => ({
          template_id: item.template.id,
          enabled: item.enabled,
          sort_order: index,
        })),
      });
      setBindings(next);
      savedBindings.current = next;
      setDirty(false);
      props.onChange?.(next);
      window.dispatchEvent(new Event('token-prompts-updated'));
    } catch (reason) {
      setBindings(savedBindings.current);
      setDirty(false);
      setError(promptError(reason, t));
    } finally {
      setSaving(false);
    }
  };

  const queue = (next) => {
    setBindings(next);
    setDirty(true);
    setError('');
  };

  // 本地草稿中尚未保存到令牌的模板，用于提示「待保存」。
  const savedIds = new Set(
    savedBindings.current.map((item) => item.template.id),
  );

  return (
    <div className='prompt-center-stack' aria-busy={saving}>
      {error && (
        <div role='alert' className='prompt-center-error'>
          {error}
          {!ready && (
            <Button onClick={() => setRevision((value) => value + 1)}>
              {t('Retry')}
            </Button>
          )}
        </div>
      )}
      {tokenId && !ready && !error && (
        <p role='status' className='prompt-center-loading'>
          {t('Loading prompts')}
        </p>
      )}
      {ready && (
        <section
          className='prompt-center-section prompt-installed'
          aria-label={t('Installed prompts')}
        >
          <header className='prompt-installed-header'>
            <div className='prompt-installed-heading'>
              <div
                className='prompt-center-section-icon prompt-center-section-icon-blue'
                aria-hidden='true'
              >
                <PackageCheck size={18} />
              </div>
              <div className='prompt-installed-heading-copy'>
                <h2>{t('Installed prompts')}</h2>
                <p>
                  {t(
                    'Prompts load only when the actual request group is allowed. Installation does not change token permissions.',
                  )}
                </p>
              </div>
            </div>
            <div
              className='prompt-installed-capacity'
              role='img'
              aria-label={t('{{count}} of {{limit}} prompt slots used', {
                count: bindings.length,
                limit: PROMPT_LIMIT,
              })}
            >
              <span className='prompt-installed-slots' aria-hidden='true'>
                {Array.from({ length: PROMPT_LIMIT }, (_, index) => (
                  <i key={index} data-filled={index < bindings.length} />
                ))}
              </span>
              <strong aria-hidden='true'>
                {bindings.length}/{PROMPT_LIMIT}
              </strong>
            </div>
          </header>
          <p className='prompt-installed-note'>
            <Info size={14} aria-hidden='true' />
            <span>
              {t(
                'Install up to 8 packages. Always-loaded prompts share a 65536-byte enabled limit. On-demand skills add only the files read to model context and may use additional billed model rounds.',
              )}
            </span>
          </p>
          {!bindings.length && (
            <div className='prompt-center-empty-state'>
              <PackageCheck size={22} aria-hidden='true' />
              <span>{t('No prompts installed')}</span>
            </div>
          )}
          {bindings.length > 0 && (
            <ul className='prompt-installed-list'>
              {bindings.map((binding) => {
                const template = binding.template;
                return (
                  <li
                    className='prompt-installed-item'
                    data-active={binding.enabled && template.enabled}
                    key={template.id}
                  >
                    <span
                      className='prompt-installed-item-icon'
                      aria-hidden='true'
                    >
                      {template.kind === 'bundle' ? (
                        <Layers3 size={18} />
                      ) : (
                        <FileText size={18} />
                      )}
                    </span>
                    <div className='prompt-installed-item-body'>
                      <div className='prompt-installed-item-title'>
                        <h3 title={template.title}>{template.title}</h3>
                        {!savedIds.has(template.id) && (
                          <span className='prompt-installed-flag is-pending'>
                            {t('Not saved yet')}
                          </span>
                        )}
                        {!template.enabled && (
                          <span className='prompt-installed-flag'>
                            {t('Unpublished')}
                          </span>
                        )}
                        {template.enabled && !binding.enabled && (
                          <span className='prompt-installed-flag'>
                            {t('Disabled')}
                          </span>
                        )}
                      </div>
                      <SkillSummary template={template} />
                      <div className='prompt-installed-groups'>
                        <span className='prompt-center-meta-label'>
                          <ShieldCheck size={13} aria-hidden='true' />
                          {t('Allowed groups')}
                        </span>
                        <div className='prompt-center-group-chips'>
                          {template.all_groups && (
                            <span className='prompt-center-group-chip is-group'>
                              <Users size={12} aria-hidden='true' />
                              {t('All groups (no restriction)')}
                            </span>
                          )}
                          {!template.all_groups &&
                            template.group_details?.map((group) => (
                              <span
                                className='prompt-center-group-chip is-group'
                                key={group.id}
                                title={group.name}
                              >
                                <Users size={12} aria-hidden='true' />
                                <span>{group.name}</span>
                              </span>
                            ))}
                          {!template.all_groups &&
                            !template.group_details?.length && (
                              <span className='prompt-center-muted'>
                                {t('No eligible groups')}
                              </span>
                            )}
                        </div>
                      </div>
                    </div>
                    <Button
                      className='prompt-center-uninstall-button'
                      type='danger'
                      theme='borderless'
                      icon={<Trash2 size={15} />}
                      disabled={saving}
                      onClick={() =>
                        queue(
                          bindings.filter(
                            (item) => item.template.id !== template.id,
                          ),
                        )
                      }
                      aria-label={t('Uninstall {{title}}', {
                        title: template.title,
                      })}
                    >
                      {t('Uninstall')}
                    </Button>
                  </li>
                );
              })}
            </ul>
          )}
          <footer className='prompt-installed-actions' data-dirty={dirty}>
            <span
              className='prompt-installed-save-state'
              role='status'
              aria-live='polite'
            >
              {dirty ? t('Unsaved changes') : t('No unsaved changes')}
            </span>
            {dirty && (
              <Button
                className='prompt-center-secondary'
                icon={<Undo2 size={15} />}
                disabled={saving}
                onClick={() => {
                  setBindings(savedBindings.current);
                  setDirty(false);
                  setError('');
                }}
              >
                {t('Discard changes')}
              </Button>
            )}
            <Button
              className='prompt-center-primary prompt-center-save-button'
              icon={<Save size={15} />}
              disabled={!dirty || saving}
              loading={saving}
              onClick={() => save()}
            >
              {t('Save')}
            </Button>
          </footer>
        </section>
      )}
      {props.templates && (
        <PromptCatalog
          templates={props.templates}
          bindings={ready ? bindings : []}
          canInstall={ready}
          busy={saving || dirty}
          onInstall={(template) =>
            queue([
              ...bindings,
              { template, enabled: true, sort_order: bindings.length },
            ])
          }
          onEdit={props.onEdit}
        />
      )}
    </div>
  );
}
