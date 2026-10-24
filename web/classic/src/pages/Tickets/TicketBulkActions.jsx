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
import { Button, Modal } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API } from '../../helpers/api';
import { ticketError, ticketResponse, ticketStatuses } from './ticket-utils';

export default function TicketBulkActions(props) {
  const { t } = useTranslation();
  const id = useId();
  const active = useRef(true);
  const [status, setStatus] = useState('processing');
  const [action, setAction] = useState(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);

  const submit = async () => {
    if (saving || !props.selected.length || !props.canWrite) return;
    setSaving(true);
    setError('');
    props.onPendingChange(true);
    const tickets = props.selected.map((ticket) => ({
      id: ticket.id,
      version: ticket.version,
    }));
    try {
      const response =
        action === 'delete'
          ? await API.post(
              '/api/ticket-admin/batch/delete',
              { tickets },
              { skipErrorHandler: true },
            )
          : await API.put(
              '/api/ticket-admin/batch/status',
              { tickets, status },
              { skipErrorHandler: true },
            );
      ticketResponse(response, t('Failed to update selected tickets'));
      if (!active.current) return;
      setAction(null);
      props.onComplete();
    } catch (failure) {
      if (active.current)
        setError(
          t(ticketError(failure, t('Failed to update selected tickets'))),
        );
    } finally {
      if (active.current) {
        setSaving(false);
        props.onPendingChange(false);
      }
    }
  };

  if (!props.canWrite) return null;
  return (
    <>
      <div className='ticket-bulk-actions'>
        <label className='ticket-selection'>
          <input
            type='checkbox'
            ref={(input) => {
              if (input)
                input.indeterminate =
                  props.selected.length > 0 &&
                  props.selected.length < props.items.length;
            }}
            checked={
              props.items.length > 0 &&
              props.selected.length === props.items.length
            }
            disabled={props.disabled || saving || !props.items.length}
            onChange={(event) => props.onSelectAll(event.target.checked)}
          />
          {t('Select all tickets on this page')}
        </label>
        <span role='status'>
          {t('{{count}} selected', { count: props.selected.length })}
        </span>
        <label htmlFor={`${id}-status`}>{t('Batch status')}</label>
        <select
          className='ticket-select'
          id={`${id}-status`}
          value={status}
          disabled={saving || props.disabled}
          onChange={(event) => setStatus(event.target.value)}
        >
          {Object.entries(ticketStatuses).map(([value, label]) => (
            <option key={value} value={value}>
              {t(label)}
            </option>
          ))}
        </select>
        <Button
          disabled={!props.selected.length || saving || props.disabled}
          onClick={() => {
            setError('');
            setAction('status');
          }}
        >
          {t('Change selected status')}
        </Button>
        <Button
          type='danger'
          disabled={!props.selected.length || saving || props.disabled}
          onClick={() => {
            setError('');
            setAction('delete');
          }}
        >
          {t('Delete selected tickets')}
        </Button>
      </div>
      <Modal
        visible={Boolean(action)}
        title={
          action === 'delete'
            ? t('Delete selected tickets')
            : t('Change selected status')
        }
        className='ticket-modal'
        width={560}
        footer={null}
        maskClosable={false}
        closable={!saving}
        closeOnEsc={!saving}
        onCancel={saving ? undefined : () => setAction(null)}
      >
        <div className='ticket-form'>
          <p>
            {action === 'delete'
              ? t(
                  'Delete {{count}} selected tickets? They will disappear from both lists. Conversations and audit records are retained; there is no restore action.',
                  { count: props.selected.length },
                )
              : t('Change {{count}} selected tickets to {{status}}?', {
                  count: props.selected.length,
                  status: t(ticketStatuses[status]),
                })}
          </p>
          {error && (
            <div role='alert' className='ticket-error'>
              <div>
                {error}
                <p>{t('Refresh the list before retrying.')}</p>
              </div>
            </div>
          )}
          <div className='ticket-actions'>
            <Button disabled={saving} onClick={() => setAction(null)}>
              {t('Cancel')}
            </Button>
            <Button
              type={action === 'delete' ? 'danger' : 'primary'}
              theme='solid'
              loading={saving}
              disabled={saving || !props.selected.length || props.disabled}
              onClick={submit}
            >
              {action === 'delete'
                ? t('Confirm deletion')
                : t('Confirm status change')}
            </Button>
          </div>
        </div>
      </Modal>
    </>
  );
}
