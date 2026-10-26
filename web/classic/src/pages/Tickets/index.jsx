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
import React, { useContext, useEffect, useId, useState } from 'react';
import { Button, Card, Empty, Input, Spin, Tag } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { useSearchParams } from 'react-router-dom';
import { API } from '../../helpers/api';
import { readStoredUser } from '../../helpers/auth-data';
import { UserContext } from '../../context/User';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useUserPermissions } from '../../hooks/common/useUserPermissions';
import CreateTicketModal from './CreateTicketModal';
import TicketDetails from './TicketDetails';
import TicketAutomationSettings from './TicketAutomationSettings';
import TicketArchiveSettings from './TicketArchiveSettings';
import TicketBulkActions from './TicketBulkActions';
import {
  ticketCategories,
  ticketError,
  ticketPriorities,
  ticketResponse,
  ticketStatuses,
  ticketStatusColors,
  ticketTime,
} from './ticket-utils';
import './tickets.css';

function TicketAdmin() {
  const { t } = useTranslation();
  const { permissions, loading, error, loadPermissions } = useUserPermissions();
  if (loading)
    return (
      <Card className='ticket-panel'>
        <div role='status' className='ticket-state'>
          <Spin />
          {t('Loading ticket permissions…')}
        </div>
      </Card>
    );
  if (error)
    return (
      <Card className='ticket-panel'>
        <div role='alert' className='ticket-error'>
          {t('Failed to load ticket permissions')}
          <Button onClick={loadPermissions}>{t('Retry')}</Button>
        </div>
      </Card>
    );
  if (permissions?.admin_permissions?.tickets?.read !== true)
    return (
      <Card className='ticket-panel'>
        <div role='alert' className='ticket-error'>
          {t('You do not have permission to view ticket management.')}
        </div>
      </Card>
    );
  const canWrite = permissions?.admin_permissions?.tickets?.write === true;
  return (
    <>
      <TicketAutomationSettings canWrite={canWrite} />
      <TicketArchiveSettings canWrite={canWrite} />
      <TicketList admin canWrite={canWrite} />
    </>
  );
}

function TicketList(props) {
  const { t } = useTranslation();
  const id = useId();
  const isMobile = useIsMobile();
  const [searchParams, setSearchParams] = useSearchParams();
  const selected = Number(searchParams.get('ticket')) || null;
  const [items, setItems] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [createVisible, setCreateVisible] = useState(false);
  const requestedStatus = searchParams.get('status') || '';
  const status = Object.hasOwn(ticketStatuses, requestedStatus)
    ? requestedStatus
    : '';
  const setStatus = (value) => {
    const next = new URLSearchParams(searchParams);
    if (value) next.set('status', value);
    else next.delete('status');
    next.delete('ticket');
    setSearchParams(next, { replace: true });
  };
  const [keyword, setKeyword] = useState('');
  const [userId, setUserId] = useState('');
  const [filters, setFilters] = useState({ keyword: '', userId: '' });
  const [revision, setRevision] = useState(0);
  const [selectedIds, setSelectedIds] = useState([]);
  const [batchPending, setBatchPending] = useState(false);
  const canSelect = props.admin && props.canWrite;
  const selectedTickets = items.filter((ticket) =>
    selectedIds.includes(ticket.id),
  );
  const endpoint = props.admin ? '/api/ticket-admin' : '/api/tickets';
  const openTicket = (ticketId) => {
    const next = new URLSearchParams(searchParams);
    if (ticketId) next.set('ticket', String(ticketId));
    else next.delete('ticket');
    setSearchParams(next, { replace: true });
  };

  useEffect(() => {
    setPage(1);
  }, [status]);

  useEffect(() => {
    const refresh = () => setRevision((value) => value + 1);
    window.addEventListener('classic-tickets-changed', refresh);
    return () => window.removeEventListener('classic-tickets-changed', refresh);
  }, []);

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError('');
    setSelectedIds([]);
    const params = { p: page, page_size: 20, status, keyword: filters.keyword };
    if (props.admin && filters.userId) params.user_id = filters.userId;
    API.get(endpoint, {
      params,
      skipErrorHandler: true,
      disableDuplicate: true,
    })
      .then((response) => {
        const result = ticketResponse(response, t('Failed to load tickets'));
        if (active) {
          setItems(result.items || []);
          setTotal(result.total || 0);
        }
      })
      .catch((failure) => {
        if (active) {
          setItems([]);
          setTotal(0);
          setError(t(ticketError(failure, t('Failed to load tickets'))));
        }
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [endpoint, page, status, filters, revision, props.admin, t]);

  const title = props.admin ? t('Ticket management') : t('Ticket center');
  return (
    <section aria-label={title}>
      <Card className='ticket-panel'>
        <div className='ticket-heading'>
          <div>
            <h2>{title}</h2>
            <p className='ticket-muted'>
              {props.admin
                ? t('Review and respond to customer tickets.')
                : t(
                    'Create a ticket and follow the conversation with support.',
                  )}
            </p>
          </div>
          <div className='ticket-actions'>
            <Button
              onClick={() => setRevision((value) => value + 1)}
              disabled={loading || batchPending}
            >
              {t('Refresh')}
            </Button>
            {!props.admin && (
              <Button theme='solid' onClick={() => setCreateVisible(true)}>
                {t('Create ticket')}
              </Button>
            )}
          </div>
        </div>
        <form
          className='ticket-filters'
          onSubmit={(event) => {
            event.preventDefault();
            if (batchPending) return;
            setPage(1);
            setFilters({ keyword: keyword.trim(), userId: userId.trim() });
          }}
        >
          <div>
            <label htmlFor={`${id}-search`}>{t('Search tickets')}</label>
            <Input
              id={`${id}-search`}
              value={keyword}
              disabled={batchPending}
              onChange={setKeyword}
              showClear
              placeholder={t('Search by title')}
            />
          </div>
          <div>
            <label htmlFor={`${id}-status`}>{t('Ticket status')}</label>
            <select
              className='ticket-select'
              id={`${id}-status`}
              value={status}
              disabled={batchPending}
              onChange={(event) => {
                setStatus(event.target.value);
                setPage(1);
              }}
            >
              <option value=''>{t('All statuses')}</option>
              {Object.entries(ticketStatuses).map(([value, label]) => (
                <option key={value} value={value}>
                  {t(label)}
                </option>
              ))}
            </select>
          </div>
          {props.admin && (
            <div>
              <label htmlFor={`${id}-user`}>{t('User ID')}</label>
              <Input
                id={`${id}-user`}
                type='number'
                min={1}
                value={userId}
                disabled={batchPending}
                onChange={setUserId}
                showClear
              />
            </div>
          )}
          <Button htmlType='submit' disabled={batchPending}>
            {t('Search')}
          </Button>
        </form>
        {canSelect && (
          <TicketBulkActions
            canWrite={props.canWrite}
            items={items}
            selected={selectedTickets}
            disabled={loading || Boolean(error)}
            onSelectAll={(checked) =>
              setSelectedIds(checked ? items.map((ticket) => ticket.id) : [])
            }
            onPendingChange={setBatchPending}
            onComplete={() => {
              setSelectedIds([]);
              window.dispatchEvent(new Event('classic-tickets-changed'));
            }}
          />
        )}
        {loading && (
          <div role='status' className='ticket-state'>
            <Spin />
            {t('Loading tickets…')}
          </div>
        )}
        {!loading && error && (
          <div role='alert' className='ticket-error'>
            {error}
            <Button onClick={() => setRevision((value) => value + 1)}>
              {t('Retry')}
            </Button>
          </div>
        )}
        {!loading && !error && items.length === 0 && (
          <Empty
            className='ticket-state'
            title={t('No tickets found')}
            description={t('Create a ticket or adjust your filters.')}
          />
        )}
        {!loading &&
          !error &&
          items.length > 0 &&
          (isMobile ? (
            <div className='ticket-mobile-list'>
              {items.map((ticket) => (
                <article className='ticket-mobile-card' key={ticket.id}>
                  {canSelect && (
                    <label className='ticket-selection'>
                      <input
                        type='checkbox'
                        checked={selectedIds.includes(ticket.id)}
                        disabled={batchPending}
                        onChange={(event) =>
                          setSelectedIds((current) =>
                            event.target.checked
                              ? [...current, ticket.id]
                              : current.filter((value) => value !== ticket.id),
                          )
                        }
                      />
                      {t('Select ticket #{{id}}', { id: ticket.id })}
                    </label>
                  )}
                  <Button
                    theme='borderless'
                    className='ticket-title-button'
                    onClick={() => openTicket(ticket.id)}
                  >
                    {ticket.title}
                  </Button>
                  <div className='ticket-detail-meta'>
                    <span>#{ticket.id}</span>
                    <Tag color={ticketStatusColors[ticket.status]}>
                      {t(ticketStatuses[ticket.status] || ticket.status)}
                    </Tag>
                    <span>
                      {t(ticketCategories[ticket.category] || ticket.category)}
                    </span>
                    <span>
                      {t(ticketPriorities[ticket.priority] || ticket.priority)}
                    </span>
                  </div>
                  {props.admin && (
                    <p>
                      {t('User ID')}: {ticket.user_id}
                    </p>
                  )}
                  <time className='ticket-muted'>
                    {ticketTime(ticket.updated_at)}
                  </time>
                </article>
              ))}
            </div>
          ) : (
            <div className='ticket-table-scroll'>
              <table className='ticket-table'>
                <thead>
                  <tr>
                    {canSelect && <th>{t('Select')}</th>}
                    <th>{t('Ticket')}</th>
                    {props.admin && <th>{t('User ID')}</th>}
                    <th>{t('Category')}</th>
                    <th>{t('Priority')}</th>
                    <th>{t('Status')}</th>
                    <th>{t('Updated')}</th>
                  </tr>
                </thead>
                <tbody>
                  {items.map((ticket) => (
                    <tr key={ticket.id}>
                      {canSelect && (
                        <td>
                          <input
                            type='checkbox'
                            aria-label={t('Select ticket #{{id}}', {
                              id: ticket.id,
                            })}
                            checked={selectedIds.includes(ticket.id)}
                            disabled={batchPending}
                            onChange={(event) =>
                              setSelectedIds((current) =>
                                event.target.checked
                                  ? [...current, ticket.id]
                                  : current.filter(
                                      (value) => value !== ticket.id,
                                    ),
                              )
                            }
                          />
                        </td>
                      )}
                      <td>
                        <span className='ticket-muted'>#{ticket.id}</span>
                        <Button
                          theme='borderless'
                          className='ticket-title-button'
                          onClick={() => openTicket(ticket.id)}
                        >
                          {ticket.title}
                        </Button>
                      </td>
                      {props.admin && <td>{ticket.user_id}</td>}
                      <td>
                        {t(
                          ticketCategories[ticket.category] || ticket.category,
                        )}
                      </td>
                      <td>
                        {t(
                          ticketPriorities[ticket.priority] || ticket.priority,
                        )}
                      </td>
                      <td>
                        <Tag color={ticketStatusColors[ticket.status]}>
                          {t(ticketStatuses[ticket.status] || ticket.status)}
                        </Tag>
                      </td>
                      <td>{ticketTime(ticket.updated_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ))}
        <nav className='ticket-pagination' aria-label={t('Ticket pagination')}>
          <span>{t('{{count}} tickets', { count: total })}</span>
          <Button
            disabled={loading || batchPending || page <= 1}
            onClick={() => setPage((value) => value - 1)}
          >
            {t('Previous')}
          </Button>
          <span>
            {page} / {Math.max(1, Math.ceil(total / 20))}
          </span>
          <Button
            disabled={loading || batchPending || page * 20 >= total}
            onClick={() => setPage((value) => value + 1)}
          >
            {t('Next')}
          </Button>
        </nav>
      </Card>
      {!props.admin && (
        <CreateTicketModal
          visible={createVisible}
          onClose={() => setCreateVisible(false)}
          onCreated={(ticket) => openTicket(ticket.id)}
        />
      )}
      {selected > 0 && (
        <TicketDetails
          key={`${endpoint}:${selected}`}
          ticketId={selected}
          admin={props.admin}
          canWrite={props.canWrite}
          onClose={() => openTicket(null)}
          onChanged={() =>
            window.dispatchEvent(new Event('classic-tickets-changed'))
          }
        />
      )}
    </section>
  );
}

export default function Tickets(props) {
  const context = useContext(UserContext);
  const user = Array.isArray(context) ? context[0]?.user : readStoredUser();
  if (!user?.id) return null;
  const identity = `${user.id}:${user.role}`;
  return (
    <div className='classic-console-page ticket-page'>
      <div className='classic-console-page-container'>
        {props.admin ? (
          <TicketAdmin key={identity} />
        ) : (
          <TicketList key={identity} canWrite />
        )}
      </div>
    </div>
  );
}
