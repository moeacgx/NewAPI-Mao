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
import { promptError, promptRequest } from './api';
import PromptCatalog from './PromptCatalog';

export default function TokenPromptManager(props) {
  const { t } = useTranslation();
  const [bindings, setBindings] = useState([]);
  const [loadedToken, setLoadedToken] = useState(null);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);
  const [revision, setRevision] = useState(0);
  const tokenId = props.token?.id;
  useEffect(() => {
    const controller = new AbortController();
    setError('');
    setBindings([]);
    setLoadedToken(null);
    if (tokenId) {
      promptRequest(
        'get',
        `/api/token/${tokenId}/prompts`,
        undefined,
        controller.signal,
      )
        .then((items) => {
          setBindings(items || []);
          setLoadedToken(tokenId);
        })
        .catch((reason) => {
          if (!controller.signal.aborted) setError(promptError(reason, t));
        });
    }
    return () => controller.abort();
  }, [tokenId, revision, t]);

  const ready = tokenId && loadedToken === tokenId;
  const save = async (next) => {
    if (!ready || saving) return;
    const bytes = next
      .filter((item) => item.enabled && item.template.enabled)
      .reduce(
        (total, item) =>
          total + new TextEncoder().encode(item.template.content || '').length,
        0,
      );
    if (bytes > 65536) {
      setError(t('Enabled prompts must total no more than 65536 bytes'));
      return;
    }
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
      props.onChange?.(next);
      window.dispatchEvent(new Event('token-prompts-updated'));
    } catch (reason) {
      setError(promptError(reason, t));
    } finally {
      setSaving(false);
    }
  };

  const move = (index, offset) => {
    const next = [...bindings];
    [next[index], next[index + offset]] = [next[index + offset], next[index]];
    save(next);
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
          className='classic-console-panel'
          aria-label={t('Installed prompts')}
        >
          <div className='classic-console-panel-header px-4 py-3'>
            <h2>
              {t('Installed prompts')} · {bindings.length}/8
            </h2>
          </div>
          <div className='classic-console-panel-content prompt-center-stack'>
            <p className='prompt-center-muted'>
              {t(
                'Prompts load only when the actual request group is allowed. Installation does not change token permissions.',
              )}
            </p>
            {!bindings.length && (
              <p role='status'>{t('No prompts installed')}</p>
            )}
            {bindings.map((binding, index) => (
              <article className='prompt-center-card' key={binding.template.id}>
                <div className='prompt-center-row'>
                  <h3>{binding.template.title}</h3>
                  <label>
                    <input
                      type='checkbox'
                      checked={binding.enabled}
                      disabled={
                        saving ||
                        (!binding.template.enabled && !binding.enabled)
                      }
                      onChange={(event) =>
                        save(
                          bindings.map((item) =>
                            item.template.id === binding.template.id
                              ? { ...item, enabled: event.target.checked }
                              : item,
                          ),
                        )
                      }
                      aria-label={t('Enable {{title}}', {
                        title: binding.template.title,
                      })}
                    />{' '}
                    {t('Enabled')}
                  </label>
                </div>
                {!binding.template.enabled && (
                  <p>{t('Unpublished: this prompt will not load')}</p>
                )}
                <p>
                  {t('Allowed groups')}:{' '}
                  {binding.template.group_details
                    ?.map((group) => group.name)
                    .join(', ') || t('No eligible groups')}
                </p>
                <div className='prompt-center-row'>
                  <Button
                    disabled={saving || index === 0}
                    onClick={() => move(index, -1)}
                    aria-label={t('Move {{title}} up', {
                      title: binding.template.title,
                    })}
                  >
                    {t('Move up')}
                  </Button>
                  <Button
                    disabled={saving || index === bindings.length - 1}
                    onClick={() => move(index, 1)}
                    aria-label={t('Move {{title}} down', {
                      title: binding.template.title,
                    })}
                  >
                    {t('Move down')}
                  </Button>
                  <Button
                    type='danger'
                    disabled={saving}
                    onClick={() =>
                      save(
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
          busy={saving}
          onInstall={(template) =>
            save([...bindings, { template, enabled: true }])
          }
          onEdit={props.onEdit}
        />
      )}
    </div>
  );
}
