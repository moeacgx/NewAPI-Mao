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
import BundleSkillsEditor from './BundleSkillsEditor';
import ArchiveImport from './ArchiveImport';
import RepositoryImport from './RepositoryImport';
import SkillResourcesEditor from './SkillResourcesEditor';
import { validatePromptTemplateDraft } from './validation';

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
    kind: 'prompt',
    category_id: 0,
    ...props.template,
    skills: props.template?.skills || [],
    load_mode:
      props.template?.load_mode ||
      (props.template?.id ? 'eager' : 'progressive'),
  });
  const [groups, setGroups] = useState([]);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);
  const [importing, setImporting] = useState(false);
  const busy = saving || importing;
  const progressive = draft.load_mode === 'progressive';
  const [groupsLoaded, setGroupsLoaded] = useState(false);
  const [categories, setCategories] = useState(props.categories || []);
  const [categoriesLoaded, setCategoriesLoaded] = useState(
    Boolean(props.categories),
  );
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
  useEffect(() => {
    if (props.categories) {
      setCategories(props.categories);
      setCategoriesLoaded(true);
      return;
    }
    const controller = new AbortController();
    promptRequest(
      'get',
      '/api/prompt-center/categories/',
      undefined,
      controller.signal,
    )
      .then((items) => {
        if (!controller.signal.aborted) {
          setCategories(items || []);
          setCategoriesLoaded(true);
        }
      })
      .catch((reason) => {
        if (!controller.signal.aborted) setError(promptError(reason, t));
      });
    return () => controller.abort();
  }, [props.categories, t]);
  const update = (key, value) =>
    setDraft((current) => ({ ...current, [key]: value }));
  const save = async (event) => {
    event.preventDefault();
    if (busy) return;
    const validationError = validatePromptTemplateDraft(draft);
    if (validationError) {
      setError(t(validationError));
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
          kind: draft.kind,
          load_mode: draft.load_mode,
          category_id: draft.category_id,
          ...(draft.kind === 'bundle'
            ? {
                skills: draft.skills.map((skill) => ({
                  ...skill,
                  title: skill.title.trim(),
                  content: skill.content,
                })),
              }
            : {
                skills: [
                  {
                    ...draft.skills[0],
                    title: title.trim(),
                    description,
                    content,
                    path: draft.skills[0]?.path || 'SKILL.md',
                    resources: draft.skills[0]?.resources || [],
                  },
                ],
              }),
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
  const importDraft = (imported) => {
    setDraft({
      title: '',
      description: '',
      content: '',
      kind: 'prompt',
      category_id: 0,
      group_ids: [],
      enabled: false,
      sort_order: 0,
      ...imported,
      skills: imported.skills || [],
      load_mode: imported.load_mode || 'progressive',
    });
    setError('');
  };
  const importText = async (event) => {
    const file = event.target.files?.[0];
    if (!file) return;
    event.target.value = '';
    if (file.size > 8 * 1024 * 1024) {
      setError(t('Each skill file must not exceed 8 MiB'));
      return;
    }
    setImporting(true);
    try {
      const content = await file.text();
      setDraft((current) => ({
        ...current,
        content,
        load_mode: 'progressive',
      }));
      setError('');
    } catch {
      setError(t('Unable to read prompt file'));
    } finally {
      setImporting(false);
    }
  };
  return (
    <Modal
      visible
      className='prompt-template-modal'
      title={draft.id ? t('Edit template') : t('New template')}
      footer={null}
      onCancel={() => {
        if (!busy) props.onCancel();
      }}
      closable={!busy}
      closeOnEsc={!busy}
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
        <ArchiveImport
          disabled={busy}
          onBusyChange={setImporting}
          onImport={importDraft}
        />
        <RepositoryImport
          disabled={busy}
          onBusyChange={setImporting}
          onImport={importDraft}
        />
        <label htmlFor={`${prefix}-load-mode`}>{t('Loading mode')}</label>
        <select
          id={`${prefix}-load-mode`}
          disabled={busy}
          value={draft.load_mode}
          onChange={(event) => update('load_mode', event.target.value)}
        >
          <option value='progressive'>{t('On-demand skill')}</option>
          <option value='eager'>{t('Always-loaded prompt')}</option>
        </select>
        <p className='prompt-center-muted'>
          {progressive
            ? t(
                'The model sees the skill directory first. The gateway reads selected files as needed and bills each model round.',
              )
            : t(
                'The full prompt is added to every request. A template is limited to 32768 bytes.',
              )}
        </p>
        <label htmlFor={`${prefix}-kind`}>{t('Skill type')}</label>
        <select
          id={`${prefix}-kind`}
          disabled={busy}
          value={draft.kind || 'prompt'}
          onChange={(event) =>
            setDraft((current) => ({
              ...current,
              kind: event.target.value,
              skills:
                current.skills.length >= 2
                  ? current.skills
                  : [
                      {
                        ...current.skills[0],
                        title: current.title,
                        description: current.description,
                        content: current.content,
                        path: current.skills[0]?.path || 'SKILL.md',
                      },
                      { title: '', content: '' },
                    ],
            }))
          }
        >
          <option value='prompt'>{t('Single skill')}</option>
          <option value='bundle'>{t('Skill bundle')}</option>
        </select>
        <label htmlFor={`${prefix}-category`}>{t('Skill category')}</label>
        <select
          id={`${prefix}-category`}
          disabled={busy || !categoriesLoaded}
          value={draft.category_id}
          onChange={(event) =>
            update('category_id', Number(event.target.value))
          }
        >
          <option value={0}>{t('Uncategorized')}</option>
          {categories.map((category) => (
            <option key={category.id} value={category.id}>
              {category.name}
            </option>
          ))}
        </select>
        <label htmlFor={`${prefix}-title`}>{t('Prompt title')}</label>
        <input
          id={`${prefix}-title`}
          required
          maxLength={80}
          disabled={busy}
          value={draft.title}
          onChange={(event) => update('title', event.target.value)}
        />
        <label htmlFor={`${prefix}-description`}>
          {t('Prompt description')}
        </label>
        <textarea
          id={`${prefix}-description`}
          maxLength={512}
          disabled={busy}
          value={draft.description}
          onChange={(event) => update('description', event.target.value)}
        />
        {draft.kind === 'bundle' ? (
          <BundleSkillsEditor
            prefix={prefix}
            skills={draft.skills}
            progressive={progressive}
            disabled={busy}
            onChange={(items) => update('skills', items)}
          />
        ) : (
          <>
            <label htmlFor={`${prefix}-content`}>{t('Prompt text')}</label>
            <textarea
              id={`${prefix}-content`}
              required
              rows={10}
              disabled={busy}
              value={draft.content}
              onChange={(event) => update('content', event.target.value)}
            />
            {progressive && (
              <>
                <label htmlFor={`${prefix}-skill-path`}>
                  {t('Skill file path')}
                </label>
                <input
                  id={`${prefix}-skill-path`}
                  disabled={busy}
                  value={draft.skills[0]?.path || 'SKILL.md'}
                  onChange={(event) =>
                    update('skills', [
                      { ...draft.skills[0], path: event.target.value },
                      ...draft.skills.slice(1),
                    ])
                  }
                />
                <SkillResourcesEditor
                  prefix={prefix}
                  resources={draft.skills[0]?.resources}
                  disabled={busy}
                  onChange={(resources) =>
                    update('skills', [
                      { ...draft.skills[0], resources },
                      ...draft.skills.slice(1),
                    ])
                  }
                />
              </>
            )}
            <label htmlFor={`${prefix}-file`}>
              {t('Import Markdown or SKILL.md as text')}
            </label>
            <input
              id={`${prefix}-file`}
              type='file'
              accept='.md,.txt,text/plain,text/markdown'
              disabled={busy}
              onChange={importText}
            />
          </>
        )}
        <p className='prompt-center-muted'>
          {t(
            'Skill files and reference resources are text only. The gateway does not execute scripts.',
          )}
        </p>
        <fieldset disabled={busy || !groupsLoaded}>
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
          disabled={busy}
          value={draft.sort_order}
          onChange={(event) => update('sort_order', Number(event.target.value))}
        />
        <label>
          <input
            type='checkbox'
            checked={draft.enabled}
            disabled={busy}
            onChange={(event) => update('enabled', event.target.checked)}
          />
          {t('Published')}
        </label>
        <div className='prompt-center-row'>
          <Button
            htmlType='submit'
            theme='solid'
            loading={saving}
            disabled={busy || !groupsLoaded || !categoriesLoaded}
          >
            {t('Save template')}
          </Button>
          <Button disabled={busy} onClick={props.onCancel}>
            {t('Cancel')}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
