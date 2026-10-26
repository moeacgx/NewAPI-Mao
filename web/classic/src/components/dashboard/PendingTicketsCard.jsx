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
import React from 'react';
import { Button, Card } from '@douyinfe/semi-ui';
import {
  ArrowRight,
  CheckCircle2,
  Clock3,
  Loader2,
  RefreshCw,
  Ticket,
  XCircle,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import {
  ticketStatusPath,
  usePendingTickets,
} from '../../pages/Tickets/TicketPendingProvider';

const overviewItems = [
  { status: 'open', label: 'Pending', Icon: Clock3 },
  { status: 'processing', label: 'In progress', Icon: Loader2 },
  { status: 'resolved', label: 'Resolved', Icon: CheckCircle2 },
  { status: 'closed', label: 'Closed', Icon: XCircle },
];

export default function PendingTicketsCard() {
  const { t } = useTranslation();
  const summary = usePendingTickets();
  if (!summary?.allowed) return null;
  return (
    <section className='mb-4' aria-label={t('Ticket overview')}>
      <Card
        className='ticket-pending-card'
        title={
          <span className='ticket-pending-card-title'>
            <span className='ticket-pending-card-icon' aria-hidden='true'>
              <Ticket size={16} />
            </span>
            {t('Ticket overview')}
          </span>
        }
        headerExtraContent={
          <span className='ticket-pending-card-actions'>
            <Button
              theme='borderless'
              icon={<RefreshCw size={14} aria-hidden='true' />}
              aria-label={t('Refresh')}
              onClick={summary.refresh}
            />
            <Link
              className='ticket-pending-link'
              to='/console/ticket-management'
            >
              {t('View tickets')}
              <ArrowRight size={14} aria-hidden='true' />
            </Link>
          </span>
        }
      >
        {summary.error ? (
          <div className='ticket-pending-card-body'>
            <span role='status'>{t('Failed to load ticket overview')}</span>
            <Button
              icon={<RefreshCw size={14} aria-hidden='true' />}
              onClick={summary.refresh}
            >
              {t('Retry')}
            </Button>
          </div>
        ) : (
          <ul className='ticket-overview-grid'>
            {overviewItems.map(({ status, label, Icon }) => {
              const count = summary.counts?.[status];
              return (
                <li key={status}>
                  <Link
                    className='ticket-overview-item'
                    to={ticketStatusPath(status)}
                  >
                    <Icon size={18} aria-hidden='true' />
                    <span className='ticket-overview-label'>{t(label)}</span>
                    <strong className='ticket-overview-count'>
                      {count === null || count === undefined
                        ? '—'
                        : count.toLocaleString()}
                    </strong>
                  </Link>
                </li>
              );
            })}
          </ul>
        )}
      </Card>
    </section>
  );
}
