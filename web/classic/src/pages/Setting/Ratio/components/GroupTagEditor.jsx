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

import React, { useId, useMemo, useState } from 'react';
import {
  AutoComplete,
  Button,
  Input,
  TextArea,
  InputNumber,
  Modal,
  Select,
  TagInput,
  Typography,
} from '@douyinfe/semi-ui';
import { IconDelete, IconPlus } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../../helpers';
import GroupTagLogos from '../../../../components/common/group-tags/GroupTagLogos';

const ICONS = [
  'OpenAI',
  'Claude.Color',
  'Gemini.Color',
  'DeepSeek.Color',
  'Moonshot',
  'Qwen.Color',
  'Zhipu.Color',
  'XAI',
  'Minimax.Color',
  'Doubao.Color',
  'OpenRouter',
];

export default function GroupTagEditor(props) {
  const { t } = useTranslation();
  const prefix = useId();
  const [draft, setDraft] = useState(
    () =>
      props.tag || {
        name: '',
        description: '',
        sort_order: 0,
        icons: [],
        group_ids: [],
        match_keywords: [],
      },
  );
  const [saving, setSaving] = useState(false);
  const matchingGroups = useMemo(() => {
    const keywords = (draft.match_keywords || [])
      .map((keyword) => keyword.trim().toLowerCase())
      .filter(Boolean);
    if (keywords.length === 0) return [];
    return props.groups.filter((group) => {
      const name = String(group.name || group.code || '').toLowerCase();
      return keywords.some((keyword) => name.includes(keyword));
    });
  }, [draft.match_keywords, props.groups]);
  const update = (field, value) =>
    setDraft((previous) => ({ ...previous, [field]: value }));

  const save = async () => {
    if (!draft.name.trim()) {
      showError(t('Enter a tag name'));
      return;
    }
    if (draft.icons.some((icon) => !icon.trim())) {
      showError(t('Enter an icon name or image URL for each logo'));
      return;
    }
    setSaving(true);
    try {
      const matchedIds = matchingGroups.map((group) => group.id);
      const payload = {
        ...draft,
        name: draft.name.trim(),
        group_ids: Array.from(new Set([...draft.group_ids, ...matchedIds])),
      };
      const response = draft.id
        ? await API.put(`/api/group/tags/${draft.id}`, payload)
        : await API.post('/api/group/tags', payload);
      if (!response.data.success) throw new Error(response.data.message);
      showSuccess(t('Group tag saved'));
      props.onSaved();
    } catch (error) {
      showError(error.message || t('操作失败'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      visible
      title={draft.id ? t('Edit group tag') : t('New group tag')}
      onOk={save}
      onCancel={() => {
        if (!saving) props.onCancel();
      }}
      confirmLoading={saving}
      cancelButtonProps={{ disabled: saving }}
      closable={!saving}
      closeOnEsc={!saving}
      maskClosable={false}
      width={620}
      style={{ maxWidth: 'calc(100vw - 24px)' }}
    >
      <div className='group-tag-editor'>
        <div className='group-tag-field'>
          <label htmlFor={`${prefix}-name`}>{t('Tag name')}</label>
          <Input
            id={`${prefix}-name`}
            maxLength={64}
            value={draft.name}
            onChange={(value) => update('name', value)}
            disabled={saving}
          />
        </div>
        <div className='group-tag-field'>
          <label htmlFor={`${prefix}-description`}>
            {t('Tag description')}
          </label>
          <TextArea
            id={`${prefix}-description`}
            maxLength={512}
            value={draft.description}
            onChange={(value) => update('description', value)}
            disabled={saving}
          />
        </div>
        <div className='group-tag-field'>
          <label htmlFor={`${prefix}-order`}>{t('Sort order')}</label>
          <InputNumber
            id={`${prefix}-order`}
            min={-1000000}
            max={1000000}
            precision={0}
            value={draft.sort_order}
            onChange={(value) => update('sort_order', Number(value) || 0)}
            disabled={saving}
          />
        </div>
        <fieldset className='group-tag-filter'>
          <legend className='group-tag-legend'>{t('Tag logos')}</legend>
          <div className='group-tag-editor'>
            <GroupTagLogos icons={draft.icons} />
            {draft.icons.map((icon, index) => (
              <div className='group-tag-logo-row' key={index}>
                <AutoComplete
                  aria-label={t('Logo {{number}}', { number: index + 1 })}
                  data={ICONS}
                  value={icon}
                  maxLength={2048}
                  placeholder={t('Icon name or image URL')}
                  disabled={saving}
                  onChange={(value) =>
                    update(
                      'icons',
                      draft.icons.map((item, i) =>
                        i === index ? value : item,
                      ),
                    )
                  }
                />
                <Button
                  icon={<IconDelete />}
                  aria-label={t('Remove logo {{number}}', {
                    number: index + 1,
                  })}
                  disabled={saving}
                  onClick={() =>
                    update(
                      'icons',
                      draft.icons.filter((_, i) => i !== index),
                    )
                  }
                />
              </div>
            ))}
            <Button
              icon={<IconPlus />}
              disabled={saving || draft.icons.length >= 6}
              onClick={() => update('icons', [...draft.icons, ''])}
            >
              {t('Add logo')}
            </Button>
            <Typography.Text type='tertiary'>
              {t(
                'Combine up to 6 built-in icons or HTTP/HTTPS image URLs. Logos appear in this order.',
              )}
            </Typography.Text>
          </div>
        </fieldset>
        <div className='group-tag-field'>
          <label htmlFor={`${prefix}-groups`}>{t('Bound groups')}</label>
          <div className='group-tag-match-row group-tag-match-keywords'>
            <TagInput
              id={`${prefix}-group-keyword`}
              aria-label={t('Match group display name')}
              value={draft.match_keywords || []}
              showClear
              placeholder={t('Enter keywords and press Enter')}
              disabled={saving}
              onChange={(value) =>
                update(
                  'match_keywords',
                  Array.from(
                    new Set(
                      (value || [])
                        .map((item) => String(item).trim())
                        .filter(Boolean),
                    ),
                  ),
                )
              }
            />
            <Button
              icon={<IconPlus />}
              disabled={saving || matchingGroups.length === 0}
              onClick={() => {
                const matchedIds = matchingGroups.map((group) => group.id);
                update(
                  'group_ids',
                  Array.from(new Set([...draft.group_ids, ...matchedIds])),
                );
              }}
            >
              {t('Add current matches')}
            </Button>
          </div>
          {(draft.match_keywords || []).length > 0 && (
            <Typography.Text type='tertiary'>
              {t('Matching groups: {{total}}', {
                total: matchingGroups.length,
              })}
            </Typography.Text>
          )}
          <Select
            id={`${prefix}-groups`}
            aria-label={t('Bound groups')}
            multiple
            filter
            maxTagCount={3}
            showClear
            disabled={saving}
            style={{ width: '100%' }}
            placeholder={t('Select groups to classify')}
            optionList={props.groups.map((group) => ({
              value: group.id,
              label: group.name || group.code,
            }))}
            value={draft.group_ids}
            onChange={(value) => update('group_ids', value || [])}
          />
          <Typography.Text type='tertiary'>
            {t(
              'Keywords are saved as automatic rules. New or renamed groups are added when any keyword matches the display name.',
            )}
          </Typography.Text>
        </div>
      </div>
    </Modal>
  );
}
