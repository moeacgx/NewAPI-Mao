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
import { Power, Save, ShieldCheck, Trash2, Users } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { promptError, promptRequest } from './api';
import PromptCatalog from './PromptCatalog';
import SkillSummary from './SkillSummary';

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
          if (addTemplate && current.length >= 8) {
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
        <p role='status'>{t('Loading prompts')}</p>
      )}
      {ready && (
        <section
          className='prompt-center-section'
          aria-label={t('Installed prompts')}
        >
          <div className='prompt-center-section-heading'>
            <div
              className='prompt-center-section-icon prompt-center-section-icon-blue'
              aria-hidden='true'
            >
              <Power size={18} />
            </div>
            <div>
              <h2>
                {t('Installed prompts')}{' '}
                <span className='prompt-center-count'>{bindings.length}/8</span>
              </h2>
              <p>
                {t(
                  'Prompts load only when the actual request group is allowed. Installation does not change token permissions.',
                )}
              </p>
              <p>
                {t(
                  'Install up to 8 packages. Always-loaded prompts share a 65536-byte enabled limit. On-demand skills add only the files read to model context and may use additional billed model rounds.',
                )}
              </p>
            </div>
            <Button
              className='prompt-center-primary prompt-center-save-button'
              icon={<Save size={15} />}
              disabled={!dirty || saving}
              loading={saving}
              onClick={() => save()}
            >
              {t('Save')}
            </Button>
          </div>
          <div className='prompt-center-installed-list'>
            {!bindings.length && (
              <div className='prompt-center-empty-state' role='status'>
                <Power size={22} aria-hidden='true' />
                <span>{t('No prompts installed')}</span>
              </div>
            )}
            {bindings.map((binding) => (
              <article className='prompt-center-card' key={binding.template.id}>
                <div className='prompt-center-card-topline'>
                  <span
                    className='prompt-center-card-icon prompt-center-card-icon-blue'
                    aria-hidden='true'
                  >
                    <Power size={17} />
                  </span>
                </div>
                <div className='prompt-center-card-copy'>
                  <h3>{binding.template.title}</h3>
                </div>
                <SkillSummary template={binding.template} />
                <div className='prompt-center-card-meta'>
                  <span className='prompt-center-meta-label'>
                    <ShieldCheck size={14} aria-hidden='true' />
                    {t('Allowed groups')}
                  </span>
                  <div className='prompt-center-group-chips'>
                    {binding.template.all_groups && (
                      <span className='prompt-center-group-chip'>
                        <Users size={12} aria-hidden='true' />
                        {t('All groups (no restriction)')}
                      </span>
                    )}
                    {!binding.template.all_groups &&
                      binding.template.group_details?.map((group) => (
                        <span
                          className='prompt-center-group-chip'
                          key={group.id}
                        >
                          <Users size={12} aria-hidden='true' />
                          {group.name}
                        </span>
                      ))}
                    {!binding.template.all_groups &&
                      !binding.template.group_details?.length && (
                        <span className='prompt-center-muted'>
                          {t('No eligible groups')}
                        </span>
                      )}
                  </div>
                </div>
                <div className='prompt-center-card-actions prompt-center-card-actions-management'>
                  <Button
                    className='prompt-center-uninstall-button'
                    type='danger'
                    icon={<Trash2 size={15} />}
                    disabled={saving}
                    onClick={() =>
                      queue(
                        bindings.filter(
                          (item) => item.template.id !== binding.template.id,
                        ),
                      )
                    }
                    aria-label={t('Uninstall {{title}}', {
                      title: binding.template.title,
                    })}
                  >
                    {t('Uninstall')}
                  </Button>
                </div>
              </article>
            ))}
          </div>
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
