/*
Copyright (C) 2023-2026 QuantumNous

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

import React, { useState, useEffect } from 'react';
import {
  Form,
  Input,
  Select,
  TextArea,
  Button,
  Banner,
  Space,
  Typography,
  Card,
  Tag,
  TagInput,
} from '@douyinfe/semi-ui';
import { IconPlus, IconDeleteStroked, IconAlertCircle } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

const { Text } = Typography;

const SUPPLIER_MAX_GROUPS = 20;
const SUPPLIER_MAX_MODELS = 20;
const SUPPLIER_RATIO_MIN = 0.0001;
const SUPPLIER_RATIO_MAX = 1000;

const contactTypes = [
  { value: 'wechat', label: 'WeChat' },
  { value: 'qq', label: 'QQ' },
  { value: 'telegram', label: 'Telegram' },
  { value: 'email', label: 'Email' },
  { value: 'phone', label: 'Phone' },
  { value: 'other', label: 'Other' },
];

const platforms = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'grok', label: 'Grok' },
  { value: 'deepseek', label: 'DeepSeek' },
  { value: 'other', label: 'Other' },
];

function isValidUrl(value) {
  if (!value || /\s/.test(value)) return false;
  try {
    const url = new URL(value);
    return (
      (url.protocol === 'http:' || url.protocol === 'https:') &&
      url.hostname !== '' &&
      url.username === '' &&
      url.password === ''
    );
  } catch {
    return false;
  }
}

function parseRatio(value) {
  const text = String(value).trim();
  if (!/^\d+(\.\d{1,4})?$/.test(text)) return null;
  const ratio = Number(text);
  if (!Number.isFinite(ratio)) return null;
  if (ratio < SUPPLIER_RATIO_MIN || ratio > SUPPLIER_RATIO_MAX) return null;
  return ratio;
}

const SupplierApplicationForm = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [formApi, setFormApi] = useState(null);
  const [groups, setGroups] = useState([
    { group_name: '', platform: 'openai', ratio: '1', models: [], description: '' },
  ]);

  const addGroup = () => {
    if (groups.length < SUPPLIER_MAX_GROUPS) {
      setGroups([
        ...groups,
        { group_name: '', platform: 'openai', ratio: '1', models: [], description: '' },
      ]);
    }
  };

  const removeGroup = (index) => {
    if (groups.length > 1) {
      setGroups(groups.filter((_, i) => i !== index));
    }
  };

  const updateGroup = (index, field, value) => {
    const updated = [...groups];
    updated[index] = { ...updated[index], [field]: value };
    setGroups(updated);
  };

  const handleSubmit = async (values) => {
    setLoading(true);
    try {
      // Build payload with numeric ratios
      const payload = {
        site_name: values.site_name?.trim() || '',
        site_url: values.site_url?.trim() || '',
        contact_type: values.contact_type || 'wechat',
        contact: values.contact?.trim() || '',
        remark: values.remark?.trim() || '',
        groups: groups.map((g) => ({
          group_name: g.group_name?.trim() || '',
          platform: g.platform || 'openai',
          ratio: parseRatio(g.ratio) || 1,
          models: Array.isArray(g.models) ? g.models : [],
          description: g.description?.trim() || '',
        })),
      };

      const res = await API.post('/api/supplier-applications', payload);
      const data = res.data;
      if (data.success) {
        showSuccess(t('Application submitted.'));
        // Reset form
        formApi?.reset();
        setGroups([
          { group_name: '', platform: 'openai', ratio: '1', models: [], description: '' },
        ]);
      } else {
        showError(data.message || t('Could not submit the application.'));
      }
    } catch (error) {
      if (error.response?.data?.message) {
        showError(error.response.data.message);
      } else {
        showError(t('Could not submit the application.'));
      }
    } finally {
      setLoading(false);
    }
  };

  const validateSiteUrl = (rule, value) => {
    const val = String(value || '').trim();
    if (!val) return true;
    if (val.length > 500) {
      return t('Site URL must be 500 characters or fewer.');
    }
    if (!isValidUrl(val)) {
      return t('Enter a valid http:// or https:// address.');
    }
    return true;
  };

  const validateRatio = (rule, value) => {
    const val = String(value || '').trim();
    if (!val) return t('Enter a ratio.');
    if (parseRatio(val) === null) {
      return t('Ratio must be between 0.0001 and 1000 with at most 4 decimals.');
    }
    return true;
  };

  return (
    <div className='p-4'>
      <Banner
        type='warning'
        icon={<IconAlertCircle />}
        description={
          <div>
            <div>{t('Do not enter API keys here')}</div>
            <div className='mt-1'>
              {t('Submissions cannot be edited. Review everything before sending.')}
            </div>
          </div>
        }
        className='mb-4'
      />

      <div className='mb-4 text-sm text-gray-600'>
        {t(
          'Tell us your site and the groups you can supply with a ratio for each. We compare quotes across sites and contact you through the details you leave when there is a fit.'
        )}
      </div>

      <Form
        getFormApi={(api) => setFormApi(api)}
        onSubmit={handleSubmit}
        labelPosition='top'
        disabled={loading}
      >
        <div className='mb-6'>
          <Text strong className='text-base'>
            {t('Site information')}
          </Text>
        </div>

        <div className='grid grid-cols-1 md:grid-cols-2 gap-4 mb-4'>
          <Form.Input
            field='site_name'
            label={t('Site name')}
            rules={[
              { required: true, message: t('Enter a site name.') },
              {
                validator: (rule, value) => {
                  const val = String(value || '').trim();
                  if ([...val].length > 100) {
                    return t('Site name must be 100 characters or fewer.');
                  }
                  return true;
                },
              },
            ]}
          />
          <Form.Input
            field='site_url'
            label={t('Site URL')}
            placeholder='https://example.com'
            rules={[{ validator: validateSiteUrl }]}
          />
        </div>

        <div className='grid grid-cols-1 md:grid-cols-2 gap-4 mb-4'>
          <Form.Select
            field='contact_type'
            label={t('Contact type')}
            initValue='wechat'
            optionList={contactTypes.map((ct) => ({ value: ct.value, label: t(ct.label) }))}
          />
          <Form.Input
            field='contact'
            label={t('Contact')}
            rules={[
              { required: true, message: t('Enter your contact details.') },
              {
                validator: (rule, value) => {
                  const val = String(value || '').trim();
                  if ([...val].length > 200) {
                    return t('Contact must be 200 characters or fewer.');
                  }
                  return true;
                },
              },
            ]}
          />
        </div>

        <Form.TextArea
          field='remark'
          label={t('Remarks')}
          placeholder={t('Top-up ratio, settlement method, and any other cooperation terms.')}
          rows={3}
          maxLength={1000}
          showClear
        />

        <div className='mt-6 mb-4'>
          <div className='flex items-center justify-between'>
            <Text strong className='text-base'>
              {t('Groups you can supply')}
            </Text>
            <Text type='tertiary'>
              {groups.length} / {SUPPLIER_MAX_GROUPS} {t('groups')}
            </Text>
          </div>
        </div>

        {groups.map((group, index) => (
          <Card
            key={index}
            className='mb-4'
            title={
              <div className='flex items-center justify-between'>
                <span>
                  {t('Group {{index}}', { index: index + 1 })}
                </span>
                <Button
                  type='danger'
                  theme='borderless'
                  icon={<IconDeleteStroked />}
                  size='small'
                  onClick={() => removeGroup(index)}
                  disabled={groups.length === 1}
                  aria-label={t('Remove group {{index}}', { index: index + 1 })}
                />
              </div>
            }
          >
            <div className='grid grid-cols-1 md:grid-cols-2 gap-4 mb-3'>
              <div>
                <label className='block mb-1 text-sm font-medium'>
                  {t('Group name')} <span className='text-red-500'>*</span>
                </label>
                <Input
                  value={group.group_name}
                  onChange={(val) => updateGroup(index, 'group_name', val)}
                  placeholder={t('Enter a group name.')}
                  maxLength={64}
                />
              </div>
              <div>
                <label className='block mb-1 text-sm font-medium'>{t('Platform')}</label>
                <Select
                  value={group.platform}
                  onChange={(val) => updateGroup(index, 'platform', val)}
                  optionList={platforms.map((p) => ({ value: p.value, label: t(p.label) }))}
                  style={{ width: '100%' }}
                />
              </div>
            </div>

            <div className='mb-3'>
              <label className='block mb-1 text-sm font-medium'>
                {t('Ratio')} <span className='text-red-500'>*</span>
              </label>
              <Input
                value={group.ratio}
                onChange={(val) => updateGroup(index, 'ratio', val)}
                placeholder='1.0'
              />
              <div className='text-xs text-gray-500 mt-1'>
                {t(
                  'Enter each ratio exactly as your own site defines it. It is saved as entered and is not converted to this platform billing ratios.'
                )}
              </div>
            </div>

            <div className='mb-3'>
              <label className='block mb-1 text-sm font-medium'>{t('Main models')}</label>
              <TagInput
                value={group.models}
                onChange={(val) => updateGroup(index, 'models', val)}
                placeholder={t('Type a model name and press Enter')}
                maxLength={SUPPLIER_MAX_MODELS}
                showClear
                allowDuplicates={false}
              />
            </div>

            <div>
              <label className='block mb-1 text-sm font-medium'>{t('Notes')}</label>
              <TextArea
                value={group.description}
                onChange={(val) => updateGroup(index, 'description', val)}
                placeholder={t(
                  'For example: genuine upstream or not, can be re-exposed, injection, distillation, concurrency limit.'
                )}
                rows={2}
                maxLength={500}
                showClear
              />
            </div>
          </Card>
        ))}

        <Button
          icon={<IconPlus />}
          onClick={addGroup}
          disabled={groups.length >= SUPPLIER_MAX_GROUPS}
          block
          className='mb-4'
        >
          {t('Add group')}
        </Button>

        <div className='text-sm text-gray-500 mb-4'>
          {t('Every submission is kept as its own record. To correct one, submit a new version.')}
        </div>

        <Button
          htmlType='submit'
          type='primary'
          theme='solid'
          block
          loading={loading}
          size='large'
        >
          {loading ? t('Submitting...') : t('Submit application')}
        </Button>
      </Form>
    </div>
  );
};

export default SupplierApplicationForm;
