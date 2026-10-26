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
import { ArrowRight, ListChecks, RefreshCw, Ticket } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import {
  pendingTicketsPath,
  usePendingTickets,
} from '../../pages/Tickets/TicketPendingProvider';

export default function PendingTicketsCard() {
  const { t } = useTranslation();
  const summary = usePendingTickets();
  if (!summary?.allowed) return null;
  return (
    <section className='mb-4' aria-label={t('Pending tickets')}>
      <Card
        className='ticket-pending-card'
        title={
          <span className='ticket-pending-card-title'>
            <span className='ticket-pending-card-icon' aria-hidden='true'>
              <Ticket size={16} />
            </span>
            {t('Pending tickets')}
          </span>
        }
      >
        {summary.error ? (
          <div className='ticket-pending-card-body'>
            <span role='status'>{t('Failed to load pending tickets')}</span>
            <Button
              icon={<RefreshCw size={14} aria-hidden='true' />}
              onClick={summary.refresh}
            >
              {t('Retry')}
            </Button>
          </div>
        ) : (
          <div className='ticket-pending-card-body'>
            <div>
              <strong
                className='ticket-pending-total'
                aria-label={t('Pending ticket count')}
              >
                {summary.total === null ? '—' : summary.total.toLocaleString()}
              </strong>
              <p className='ticket-pending-description'>
                {summary.total === 0
                  ? t('No tickets awaiting your response')
                  : t('New tickets and customer follow-ups')}
              </p>
            </div>
            <div className='ticket-pending-card-actions'>
              <Button
                icon={<RefreshCw size={14} aria-hidden='true' />}
                onClick={summary.refresh}
              >
                {t('Refresh')}
              </Button>
              <Link className='ticket-pending-link' to={pendingTicketsPath}>
                <ListChecks size={14} aria-hidden='true' />
                {t('View pending tickets')}
                <ArrowRight size={14} aria-hidden='true' />
              </Link>
            </div>
          </div>
        )}
      </Card>
    </section>
  );
}
