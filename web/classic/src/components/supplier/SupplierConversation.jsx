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

import React, { useEffect, useMemo, useState } from 'react';
import { Button, Empty, Input, Select, Tag, TextArea, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, showError, timestamp2string } from '../../helpers';

const { Text, Paragraph } = Typography;

const MESSAGE_TYPES = [
  ['message', 'Plain message'],
  ['negotiation', 'Negotiation'],
  ['offer', 'Offer'],
  ['test_quota', 'Test quota'],
];

const KNOWN_ERRORS = new Set([
  'Authentication required',
  'Permission denied',
  'Supplier application service is unavailable',
  'Supplier application not found',
  'The same supplier application was already submitted',
  'Too many supplier applications in 24 hours',
  'Supplier application changed, please refresh',
  'Supplier application request exceeds 64 KiB',
  'Some fields are invalid. Check the form and try again.',
]);

function textLength(value) {
  return Array.from(value).length;
}

function decimal(value, min, max) {
  const text = value.trim();
  if (!/^\d+(\.\d{1,4})?$/.test(text)) return null;
  const number = Number(text);
  if (!Number.isFinite(number) || number < min || number > max) return null;
  return number;
}

// 只组装当前类型需要的字段。议价、优惠和测试额度都只是对话记录。
function buildMessageInput(type, fields) {
  const body = fields.body.trim();
  if (type === 'message' || type === 'offer') {
    if (body === '' || textLength(body) > 2000) return null;
    return { type, body };
  }
  if (type === 'negotiation') {
    const groupName = fields.groupName.trim();
    const ratio = decimal(fields.ratio, 0.0001, 1000);
    if (groupName === '' || textLength(groupName) > 64 || ratio === null || body === '' || textLength(body) > 1000) {
      return null;
    }
    return { type, body, group_name: groupName, ratio };
  }
  const quantity = decimal(fields.quantity, 0.0001, 1000000000);
  const unit = fields.unit.trim();
  if (quantity === null || unit === '' || textLength(unit) > 32 || body === '' || textLength(body) > 1000) {
    return null;
  }
  return {
    type,
    body,
    quantity,
    unit,
    quota_status: fields.quotaStatus,
  };
}

function publicError(message) {
  if (typeof message === 'string' && message.startsWith('invalid supplier application input')) {
    return 'Some fields are invalid. Check the form and try again.';
  }
  return KNOWN_ERRORS.has(message) ? message : 'Could not send the message.';
}

export function useSupplierSheetWidth(desktopWidth) {
  const query = '(max-width: 767px)';
  const [width, setWidth] = useState(() =>
    window.matchMedia(query).matches ? '100%' : desktopWidth,
  );
  useEffect(() => {
    const media = window.matchMedia(query);
    const update = () => setWidth(media.matches ? '100%' : desktopWidth);
    update();
    media.addEventListener('change', update);
    return () => media.removeEventListener('change', update);
  }, [desktopWidth]);
  return width;
}

const SupplierConversation = ({ applicationId, messages = [], canSend, mode, onSent }) => {
  const { t } = useTranslation();
  const [type, setType] = useState('message');
  const [body, setBody] = useState('');
  const [groupName, setGroupName] = useState('');
  const [ratio, setRatio] = useState('');
  const [quantity, setQuantity] = useState('');
  const [unit, setUnit] = useState('');
  const [quotaStatus, setQuotaStatus] = useState('pending');
  const [sent, setSent] = useState([]);
  const [pending, setPending] = useState(false);
  const [notice, setNotice] = useState('');
  const ordered = useMemo(() => {
    const merged = new Map();
    for (const message of [...messages, ...sent]) merged.set(message.id, message);
    return [...merged.values()].sort(
      (left, right) => left.created_at - right.created_at || left.id - right.id,
    );
  }, [messages, sent]);

  const typeLabel = {
    message: t('Plain message'),
    negotiation: t('Negotiation'),
    offer: t('Offer'),
    test_quota: t('Test quota'),
  };
  const roleLabel = { applicant: t('Applicant'), admin: t('Admin') };
  const quotaLabel = { pending: t('Not issued'), issued: t('Issued') };
  let bodyLabel = t('Message text');
  if (type === 'offer') bodyLabel = t('Offer plan');
  else if (type === 'negotiation') bodyLabel = t('Negotiation notes');
  else if (type === 'test_quota') bodyLabel = t('Test quota description');

  const submit = async () => {
    if (!canSend || pending) return;
    const input = buildMessageInput(type, { body, groupName, ratio, quantity, unit, quotaStatus });
    if (!input) {
      setNotice('');
      showError(t('Some fields are invalid. Check the form and try again.'));
      return;
    }
    const path =
      mode === 'admin'
        ? `/api/supplier-application-admin/${applicationId}/messages`
        : `/api/supplier-applications/${applicationId}/messages`;
    setPending(true);
    setNotice('');
    try {
      const res = await API.post(path, input);
      if (!res.data?.success) {
        showError(t(publicError(res.data?.message)));
        return;
      }
      setSent((current) => [...current, res.data.data]);
      setBody('');
      setGroupName('');
      setRatio('');
      setQuantity('');
      setUnit('');
      setQuotaStatus('pending');
      setNotice('Message sent');
      onSent?.(res.data.data);
    } catch (error) {
      showError(t(publicError(error?.response?.data?.message)));
    } finally {
      setPending(false);
    }
  };

  return (
    <div style={{ minWidth: 0, maxWidth: '100%', overflowX: 'hidden' }}>
      <Text strong>{t('Conversation')}</Text>
      {ordered.length === 0 ? (
        <Empty description={t('No messages yet')} style={{ padding: '24px 0' }} />
      ) : (
        <div className='mt-3' style={{ display: 'grid', gap: 12, minWidth: 0 }}>
          {ordered.map((message) => (
            <div
              key={message.id}
              style={{
                minWidth: 0,
                border: '1px solid var(--semi-color-border)',
                borderRadius: 8,
                padding: 12,
                overflowWrap: 'anywhere',
              }}
            >
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, marginBottom: 8 }}>
                <Text strong style={{ minWidth: 0, overflowWrap: 'anywhere' }}>
                  {message.author_name || roleLabel[message.author_role]}
                </Text>
                <Tag>{roleLabel[message.author_role]}</Tag>
                <Tag color='blue'>{typeLabel[message.type] || message.type}</Tag>
                <Text type='tertiary' size='small'>
                  {timestamp2string(message.created_at)}
                </Text>
              </div>
              {message.type === 'negotiation' && (
                <Paragraph style={{ overflowWrap: 'anywhere' }}>
                  {t('Group name')}: {message.group_name} · {t('Suggested ratio')} ×{message.ratio}
                </Paragraph>
              )}
              {message.type === 'test_quota' && (
                <Paragraph style={{ overflowWrap: 'anywhere' }}>
                  {t('Test quota')}: {message.quantity} {message.unit}
                  {message.quota_status ? ` · ${quotaLabel[message.quota_status] || message.quota_status}` : ''}
                </Paragraph>
              )}
              <Paragraph style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere', marginBottom: 0 }}>
                {message.body}
              </Paragraph>
            </div>
          ))}
        </div>
      )}
      {canSend && (
        <div className='mt-4' style={{ display: 'grid', gap: 12, minWidth: 0 }}>
          <div>
            <Text type='tertiary'>{t('Message type')}</Text>
            <Select value={type} onChange={(value) => setType(String(value))} style={{ width: '100%', marginTop: 4 }}>
              {MESSAGE_TYPES.map(([value, label]) => (
                <Select.Option key={value} value={value}>
                  {t(label)}
                </Select.Option>
              ))}
            </Select>
          </div>
          {type === 'negotiation' && (
            <>
              <Input
                value={groupName}
                maxLength={64}
                placeholder={t('Group name')}
                aria-label={t('Group name')}
                onChange={setGroupName}
              />
              <Input
                value={ratio}
                inputMode='decimal'
                placeholder={t('Suggested ratio')}
                aria-label={t('Suggested ratio')}
                onChange={setRatio}
              />
            </>
          )}
          {type === 'test_quota' && (
            <>
              <Input
                value={quantity}
                inputMode='decimal'
                placeholder={t('Quantity')}
                aria-label={t('Quantity')}
                onChange={setQuantity}
              />
              <Input value={unit} maxLength={32} placeholder={t('Unit')} aria-label={t('Unit')} onChange={setUnit} />
              <Select
                value={quotaStatus}
                aria-label={t('Issuance status')}
                onChange={(value) => setQuotaStatus(String(value))}
                style={{ width: '100%' }}
              >
                <Select.Option value='pending'>{t('Not issued')}</Select.Option>
                <Select.Option value='issued'>{t('Issued')}</Select.Option>
              </Select>
            </>
          )}
          <TextArea
            value={body}
            rows={4}
            maxLength={type === 'message' || type === 'offer' ? 2000 : 1000}
            placeholder={bodyLabel}
            aria-label={bodyLabel}
            onChange={setBody}
          />
          <Text type='tertiary' size='small' style={{ overflowWrap: 'anywhere' }}>
            {t('Messages cannot be edited after sending.')} {t('Do not include API keys or access credentials.')}
          </Text>
          {notice && <Text role='status'>{t(notice)}</Text>}
          <Button theme='solid' type='primary' loading={pending} onClick={submit} style={{ maxWidth: '100%' }}>
            {pending ? t('Sending...') : t('Send message')}
          </Button>
        </div>
      )}
    </div>
  );
};

export default SupplierConversation;
