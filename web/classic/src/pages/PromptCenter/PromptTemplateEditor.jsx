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

import React, { useEffect, useId, useState } from 'react';
import { Button, Modal } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { promptError, promptRequest } from './api';

export default function PromptTemplateEditor(props) {
  const { t } = useTranslation();
  const prefix = useId();
  const [draft, setDraft] = useState({
    title: '',
    description: '',
    content: '',
    group_ids: [],
    enabled: true,
    sort_order: 0,
    ...props.template,
  });
  const [groups, setGroups] = useState([]);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);
  const [groupsLoaded, setGroupsLoaded] = useState(false);
  useEffect(() => {
    const controller = new AbortController();
    promptRequest('get', '/api/group/details', undefined, controller.signal)
      .then((items) => {
        setGroups(items || []);
        setGroupsLoaded(true);
      })
      .catch((reason) => {
        if (!controller.signal.aborted) setError(promptError(reason, t));
      });
    return () => controller.abort();
  }, [t]);
  const update = (key, value) =>
    setDraft((current) => ({ ...current, [key]: value }));
  const save = async (event) => {
    event.preventDefault();
    if (
      !draft.title.trim() ||
      !draft.content.trim() ||
      new TextEncoder().encode(draft.content).length > 32768 ||
      (draft.enabled && !draft.group_ids.length)
    ) {
      setError(
        t(
          'Enter a title, prompt text up to 32768 bytes, and at least one allowed group',
        ),
      );
      return;
    }
    setSaving(true);
    setError('');
    try {
      const {
        title,
        description,
        content,
        group_ids,
        enabled,
        sort_order,
        version,
      } = draft;
      await promptRequest(
        draft.id ? 'put' : 'post',
        `/api/prompt-center/admin/${draft.id || ''}`,
        {
          title: title.trim(),
          description,
          content,
          group_ids,
          enabled,
          sort_order,
          ...(draft.id ? { version } : {}),
        },
      );
      props.onSaved();
    } catch (reason) {
      setError(promptError(reason, t));
    } finally {
      setSaving(false);
    }
  };
  const importText = async (event) => {
    const file = event.target.files?.[0];
    if (!file) return;
    if (file.size > 32768) {
      setError(t('Prompt text must not exceed 32768 bytes'));
      return;
    }
    try {
      update('content', await file.text());
      setError('');
    } catch {
      setError(t('Unable to read prompt file'));
    }
    event.target.value = '';
  };
  return (
    <Modal
      visible
      title={draft.id ? t('Edit template') : t('New template')}
      footer={null}
      onCancel={() => {
        if (!saving) props.onCancel();
      }}
      closable={!saving}
      closeOnEsc={!saving}
      maskClosable={false}
      width={680}
      style={{ maxWidth: 'calc(100vw - 24px)' }}
    >
      <form className='prompt-center-form' onSubmit={save}>
        {error && (
          <p role='alert' className='prompt-center-error'>
            {error}
          </p>
        )}
        <label htmlFor={`${prefix}-title`}>{t('Prompt title')}</label>
        <input
          id={`${prefix}-title`}
          required
          maxLength={80}
          disabled={saving}
          value={draft.title}
          onChange={(event) => update('title', event.target.value)}
        />
        <label htmlFor={`${prefix}-description`}>
          {t('Prompt description')}
        </label>
        <textarea
          id={`${prefix}-description`}
          maxLength={512}
          disabled={saving}
          value={draft.description}
          onChange={(event) => update('description', event.target.value)}
        />
        <label htmlFor={`${prefix}-content`}>{t('Prompt text')}</label>
        <textarea
          id={`${prefix}-content`}
          required
          rows={10}
          disabled={saving}
          value={draft.content}
          onChange={(event) => update('content', event.target.value)}
        />
        <label htmlFor={`${prefix}-file`}>
          {t('Import Markdown or SKILL.md as text')}
        </label>
        <input
          id={`${prefix}-file`}
          type='file'
          accept='.md,.txt,text/plain,text/markdown'
          disabled={saving}
          onChange={importText}
        />
        <p className='prompt-center-muted'>
          {t(
            'Text prompts only. Scripts, tools and attachments in full Skills are not executed.',
          )}
        </p>
        <fieldset disabled={saving || !groupsLoaded}>
          <legend>{t('Allowed groups')}</legend>
          <div className='prompt-center-group-options'>
            {groupsLoaded &&
              draft.group_ids
                .filter((id) => !groups.some((group) => group.id === id))
                .map((id) => (
                  <label key={`deleted-${id}`}>
                    <input
                      type='checkbox'
                      checked
                      onChange={() =>
                        update(
                          'group_ids',
                          draft.group_ids.filter((value) => value !== id),
                        )
                      }
                    />
                    {t('Unavailable group')} · {id}
                  </label>
                ))}
            {groups.map((group) => (
              <label key={group.id}>
                <input
                  type='checkbox'
                  checked={draft.group_ids.includes(group.id)}
                  onChange={(event) =>
                    update(
                      'group_ids',
                      event.target.checked
                        ? [...draft.group_ids, group.id]
                        : draft.group_ids.filter((id) => id !== group.id),
                    )
                  }
                />
                {group.name}
              </label>
            ))}
          </div>
        </fieldset>
        <label htmlFor={`${prefix}-order`}>{t('Sort order')}</label>
        <input
          id={`${prefix}-order`}
          type='number'
          min={-1000000}
          max={1000000}
          disabled={saving}
          value={draft.sort_order}
          onChange={(event) => update('sort_order', Number(event.target.value))}
        />
        <label>
          <input
            type='checkbox'
            checked={draft.enabled}
            disabled={saving}
            onChange={(event) => update('enabled', event.target.checked)}
          />
          {t('Published')}
        </label>
        <div className='prompt-center-row'>
          <Button
            htmlType='submit'
            theme='solid'
            loading={saving}
            disabled={!groupsLoaded}
          >
            {t('Save template')}
          </Button>
          <Button disabled={saving} onClick={props.onCancel}>
            {t('Cancel')}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
