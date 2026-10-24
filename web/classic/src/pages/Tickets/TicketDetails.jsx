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
import { Button, Modal, Spin, Tag, TextArea } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API } from '../../helpers/api';
import {
  ticketError,
  ticketResponse,
  ticketStatuses,
  ticketStatusColors,
  ticketTime,
} from './ticket-utils';

export default function TicketDetails(props) {
  const { t } = useTranslation();
  const id = useId();
  const [detail, setDetail] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [historyLoading, setHistoryLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [reply, setReply] = useState('');
  const [nextStatus, setNextStatus] = useState('open');
  const [writeDenied, setWriteDenied] = useState(false);
  const [revision, setRevision] = useState(0);
  const active = useRef(true);
  const endpoint = `${props.admin ? '/api/ticket-admin' : '/api/tickets'}/${props.ticketId}`;
  const canWrite = props.canWrite && !writeDenied;
  const requiresReopen = ['closed', 'archived'].includes(detail?.ticket.status);

  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);

  useEffect(() => {
    let current = true;
    setLoading(true);
    setError('');
    API.get(endpoint, { skipErrorHandler: true, disableDuplicate: true })
      .then((response) => {
        const result = ticketResponse(response, t('Failed to load ticket'));
        if (current) {
          setDetail(result);
          setNextStatus(result.ticket.status);
        }
      })
      .catch((failure) => {
        if (current)
          setError(t(ticketError(failure, t('Failed to load ticket'))));
      })
      .finally(() => {
        if (current) setLoading(false);
      });
    return () => {
      current = false;
    };
  }, [endpoint, revision, t]);

  const loadEarlier = async () => {
    if (historyLoading || !detail?.messages?.length) return;
    setHistoryLoading(true);
    setError('');
    try {
      const result = ticketResponse(
        await API.get(endpoint, {
          params: { before_message_id: detail.messages[0].id },
          skipErrorHandler: true,
          disableDuplicate: true,
        }),
        t('Failed to load earlier messages'),
      );
      if (!active.current) return;
      setDetail((current) => {
        const existing = new Set(current.messages.map((message) => message.id));
        return {
          ...current,
          messages: [
            ...result.messages.filter((message) => !existing.has(message.id)),
            ...current.messages,
          ],
          has_more: result.has_more,
        };
      });
    } catch (failure) {
      if (active.current)
        setError(t(ticketError(failure, t('Failed to load earlier messages'))));
    } finally {
      if (active.current) setHistoryLoading(false);
    }
  };

  const mutate = async (status) => {
    if (saving || !canWrite) return;
    const content = reply.trim();
    if (!status && (!content || Array.from(content).length > 10000)) {
      setError(t('Enter a reply of 1–10000 characters.'));
      return;
    }
    setSaving(true);
    setError('');
    try {
      const response = status
        ? await API.put(
            `${endpoint}/status`,
            { status },
            { skipErrorHandler: true },
          )
        : await API.post(
            `${endpoint}/messages`,
            { content },
            { skipErrorHandler: true },
          );
      const result = ticketResponse(response, t('Failed to update ticket'));
      if (!active.current) return;
      if (status) {
        setDetail((current) => ({ ...current, ticket: result }));
        setNextStatus(result.status);
      } else {
        setReply('');
        const updatedStatus = props.admin ? 'processing' : 'open';
        setDetail((current) => ({
          ...current,
          ticket: { ...current.ticket, status: updatedStatus },
          messages: [...current.messages, result],
        }));
        setNextStatus(updatedStatus);
      }
      props.onChanged();
    } catch (failure) {
      if (!active.current) return;
      if (props.admin && failure?.response?.status === 403)
        setWriteDenied(true);
      setError(t(ticketError(failure, t('Failed to update ticket'))));
    } finally {
      if (active.current) setSaving(false);
    }
  };

  return (
    <Modal
      title={
        detail
          ? `#${detail.ticket.id} ${detail.ticket.title}`
          : t('Ticket details')
      }
      visible
      onCancel={saving ? undefined : props.onClose}
      closable={!saving}
      closeOnEsc={!saving}
      footer={null}
      className='ticket-modal'
      width={760}
      maskClosable={false}
    >
      <div className='ticket-detail'>
        {loading && (
          <div role='status' className='ticket-state'>
            <Spin />
            {t('Loading tickets…')}
          </div>
        )}
        {error && (
          <div className='ticket-error' role='alert'>
            {error}
            {!detail && !loading && (
              <Button onClick={() => setRevision((value) => value + 1)}>
                {t('Retry')}
              </Button>
            )}
          </div>
        )}
        {detail && !loading && (
          <>
            <div className='ticket-detail-meta'>
              <Tag color={ticketStatusColors[detail.ticket.status]}>
                {t(
                  ticketStatuses[detail.ticket.status] || detail.ticket.status,
                )}
              </Tag>
              <span>
                {t('User ID')}: {detail.ticket.user_id}
              </span>
              <time>{ticketTime(detail.ticket.created_at)}</time>
              <Button
                disabled={saving || historyLoading}
                onClick={() => setRevision((value) => value + 1)}
              >
                {t('Refresh')}
              </Button>
            </div>
            {detail.has_more && (
              <Button
                onClick={loadEarlier}
                loading={historyLoading}
                disabled={saving}
              >
                {t('Load earlier messages')}
              </Button>
            )}
            <div
              className='ticket-messages'
              aria-label={t('Ticket conversation')}
            >
              {detail.messages.map((message) => (
                <article
                  className={`ticket-message ticket-message-${message.author_role}`}
                  key={message.id}
                >
                  <div className='ticket-message-header'>
                    <strong>
                      {message.author_role === 'admin'
                        ? t('Support team')
                        : t('Ticket requester')}
                    </strong>
                    <span>#{message.author_id}</span>
                    <time>{ticketTime(message.created_at)}</time>
                  </div>
                  <p className='ticket-message-content'>{message.content}</p>
                </article>
              ))}
              {detail.messages.length === 0 && (
                <p className='ticket-muted'>{t('No messages yet')}</p>
              )}
            </div>
            {!canWrite && (
              <p className='ticket-muted'>
                {t('You have read-only access to this ticket.')}
              </p>
            )}
            {canWrite && (
              <div className='ticket-status-actions'>
                {props.admin ? (
                  <>
                    <label htmlFor={`${id}-status`}>{t('Ticket status')}</label>
                    <select
                      className='ticket-select'
                      id={`${id}-status`}
                      value={nextStatus}
                      disabled={saving || historyLoading}
                      onChange={(event) => setNextStatus(event.target.value)}
                    >
                      {Object.entries(ticketStatuses).map(([value, label]) => (
                        <option key={value} value={value}>
                          {t(label)}
                        </option>
                      ))}
                    </select>
                    <Button
                      loading={saving}
                      disabled={
                        historyLoading || nextStatus === detail.ticket.status
                      }
                      onClick={() => mutate(nextStatus)}
                    >
                      {t('Save status')}
                    </Button>
                  </>
                ) : (
                  <Button
                    disabled={saving || historyLoading}
                    onClick={() => mutate(requiresReopen ? 'open' : 'closed')}
                  >
                    {requiresReopen ? t('Reopen ticket') : t('Close ticket')}
                  </Button>
                )}
              </div>
            )}
            {canWrite && requiresReopen && (
              <p className='ticket-muted'>
                {t('Reopen this ticket before replying.')}
              </p>
            )}
            {canWrite && !requiresReopen && (
              <form
                className='ticket-form ticket-reply'
                onSubmit={(event) => {
                  event.preventDefault();
                  mutate();
                }}
              >
                <label htmlFor={`${id}-reply`}>{t('Your reply')}</label>
                <TextArea
                  id={`${id}-reply`}
                  value={reply}
                  onChange={setReply}
                  autosize={{ minRows: 3, maxRows: 10 }}
                  disabled={saving || historyLoading}
                />
                <small className='ticket-muted'>
                  {Array.from(reply).length} / 10000
                </small>
                <div className='ticket-actions'>
                  <Button
                    htmlType='submit'
                    theme='solid'
                    loading={saving}
                    disabled={historyLoading}
                  >
                    {t('Send reply')}
                  </Button>
                </div>
              </form>
            )}
          </>
        )}
      </div>
    </Modal>
  );
}
