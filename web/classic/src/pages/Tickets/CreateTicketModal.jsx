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
import React, { useEffect, useId, useRef, useState } from 'react';
import { Button, Input, Modal, TextArea } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API } from '../../helpers/api';
import {
  ticketCategories,
  ticketPriorities,
  ticketError,
  ticketResponse,
} from './ticket-utils';
import './tickets.css';

const initialForm = {
  title: '',
  category: 'general',
  priority: 'normal',
  content: '',
};

export default function CreateTicketModal(props) {
  const { t } = useTranslation();
  const id = useId();
  const [form, setForm] = useState(initialForm);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const active = useRef(true);
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);
  const submit = async (event) => {
    event.preventDefault();
    if (saving) return;
    const payload = {
      ...form,
      title: form.title.trim(),
      content: form.content.trim(),
    };
    if (
      !payload.title ||
      !payload.content ||
      Array.from(payload.title).length > 120 ||
      Array.from(payload.content).length > 10000
    ) {
      setError(
        t(
          'Enter a title of 1–120 characters and a message of 1–10000 characters.',
        ),
      );
      return;
    }
    setSaving(true);
    setError('');
    try {
      const ticket = ticketResponse(
        await API.post('/api/tickets', payload, { skipErrorHandler: true }),
        t('Failed to create ticket'),
      );
      if (!active.current) return;
      setForm(initialForm);
      window.dispatchEvent(new Event('classic-tickets-changed'));
      props.onCreated?.(ticket);
      props.onClose();
    } catch (failure) {
      if (!active.current) return;
      setError(t(ticketError(failure, t('Failed to create ticket'))));
    } finally {
      if (active.current) setSaving(false);
    }
  };
  if (!props.visible) return null;
  return (
    <Modal
      title={t('Create ticket')}
      visible={props.visible}
      onCancel={saving ? undefined : props.onClose}
      closable={!saving}
      maskClosable={false}
      closeOnEsc={!saving}
      footer={null}
      className='ticket-modal'
      width={600}
    >
      <form className='ticket-form' onSubmit={submit}>
        <p className='ticket-muted'>
          {t('Do not include passwords, API keys, or other secrets.')}
        </p>
        <div>
          <label htmlFor={`${id}-title`}>{t('Ticket title')}</label>
          <Input
            id={`${id}-title`}
            value={form.title}
            onChange={(title) => setForm({ ...form, title })}
            disabled={saving}
            aria-required={true}
          />
          <small className='ticket-muted'>
            {Array.from(form.title).length} / 120
          </small>
        </div>
        <div className='ticket-form-row'>
          <div>
            <label htmlFor={`${id}-category`}>{t('Category')}</label>
            <select
              id={`${id}-category`}
              className='ticket-select'
              value={form.category}
              disabled={saving}
              onChange={(event) =>
                setForm({ ...form, category: event.target.value })
              }
            >
              {Object.entries(ticketCategories).map(([value, label]) => (
                <option key={value} value={value}>
                  {t(label)}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label htmlFor={`${id}-priority`}>{t('Priority')}</label>
            <select
              id={`${id}-priority`}
              className='ticket-select'
              value={form.priority}
              disabled={saving}
              onChange={(event) =>
                setForm({ ...form, priority: event.target.value })
              }
            >
              {Object.entries(ticketPriorities).map(([value, label]) => (
                <option key={value} value={value}>
                  {t(label)}
                </option>
              ))}
            </select>
          </div>
        </div>
        <div>
          <label htmlFor={`${id}-content`}>{t('Describe your issue')}</label>
          <TextArea
            id={`${id}-content`}
            value={form.content}
            disabled={saving}
            autosize={{ minRows: 5, maxRows: 12 }}
            onChange={(content) => setForm({ ...form, content })}
            aria-required={true}
          />
          <small className='ticket-muted'>
            {Array.from(form.content).length} / 10000
          </small>
        </div>
        {error && (
          <div className='ticket-error' role='alert'>
            {error}
          </div>
        )}
        <div className='ticket-actions'>
          <Button onClick={props.onClose} disabled={saving}>
            {t('Cancel')}
          </Button>
          <Button htmlType='submit' theme='solid' loading={saving}>
            {t('Submit ticket')}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
