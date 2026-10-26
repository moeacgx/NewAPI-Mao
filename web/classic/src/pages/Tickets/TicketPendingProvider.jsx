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
import React, {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
} from 'react';
import { Button } from '@douyinfe/semi-ui';
import { X, Ticket } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Link, useLocation } from 'react-router-dom';
import { API } from '../../helpers/api';
import { ticketResponse } from './ticket-utils';
import './ticket-pending.css';

const TicketPendingContext = createContext(null);
export const pendingTicketsPath = '/console/ticket-management?status=open';
export const usePendingTickets = () => useContext(TicketPendingContext);

function TicketPendingSession({ children }) {
  const { t } = useTranslation();
  const location = useLocation();
  const [summary, setSummary] = useState({
    allowed: false,
    total: null,
    error: false,
  });
  const [dismissed, setDismissed] = useState(false);
  const refreshRef = useRef(null);

  useEffect(() => {
    let active = true;
    let running = false;
    let queued = false;
    const controller = new AbortController();
    const config = {
      signal: controller.signal,
      timeout: 10000,
      skipErrorHandler: true,
    };
    const refresh = async () => {
      if (!active || document.visibilityState === 'hidden') return;
      if (running) {
        queued = true;
        return;
      }
      running = true;
      try {
        // 定期复核权限；只在授权后读取全站工单总数。
        const user = ticketResponse(await API.get('/api/user/self', config));
        if (!active) return;
        if (user?.permissions?.admin_permissions?.tickets?.read !== true) {
          setSummary({ allowed: false, total: null, error: false });
          return;
        }
        setSummary((current) => ({ ...current, allowed: true }));
        const data = ticketResponse(
          await API.get('/api/ticket-admin', {
            ...config,
            params: { p: 1, page_size: 1, status: 'open' },
          }),
        );
        if (!Number.isSafeInteger(data?.total) || data.total < 0) {
          throw new Error('Invalid ticket total');
        }
        if (active)
          setSummary({ allowed: true, total: data.total, error: false });
      } catch (error) {
        if (!active) return;
        const forbidden = [401, 403].includes(error?.response?.status);
        setSummary((current) => ({
          allowed: forbidden ? false : current.allowed,
          total: null,
          error: true,
        }));
      } finally {
        running = false;
        if (queued && active) {
          queued = false;
          void refresh();
        }
      }
    };
    refreshRef.current = refresh;
    void refresh();
    const interval = window.setInterval(refresh, 60000);
    window.addEventListener('classic-tickets-changed', refresh);
    window.addEventListener('focus', refresh);
    document.addEventListener('visibilitychange', refresh);
    return () => {
      active = false;
      refreshRef.current = null;
      controller.abort();
      window.clearInterval(interval);
      window.removeEventListener('classic-tickets-changed', refresh);
      window.removeEventListener('focus', refresh);
      document.removeEventListener('visibilitychange', refresh);
    };
  }, []);

  const showNotice =
    summary.allowed &&
    summary.total > 0 &&
    !dismissed &&
    location.pathname !== '/console/ticket-management';
  return (
    <TicketPendingContext.Provider
      value={{ ...summary, refresh: () => refreshRef.current?.() }}
    >
      {children}
      {showNotice && (
        <aside
          className='ticket-pending-notice'
          aria-label={t('Pending ticket reminder')}
        >
          <div className='ticket-pending-notice-heading'>
            <Ticket size={18} aria-hidden='true' />
            <strong>{t('Pending tickets')}</strong>
            <Button
              theme='borderless'
              size='small'
              icon={<X size={16} aria-hidden='true' />}
              aria-label={t('Dismiss ticket reminder')}
              onClick={() => setDismissed(true)}
            />
          </div>
          <p role='status'>
            {t('Tickets awaiting your response: {{total}}', {
              total: summary.total,
            })}
          </p>
          <Link className='ticket-pending-link' to={pendingTicketsPath}>
            {t('View pending tickets')}
          </Link>
        </aside>
      )}
    </TicketPendingContext.Provider>
  );
}

export default function TicketPendingProvider({ user, active, children }) {
  if (!active || !user?.id || user.role < 10 || !Number.isFinite(user.role)) {
    return (
      <TicketPendingContext.Provider value={null}>
        {children}
      </TicketPendingContext.Provider>
    );
  }
  return (
    <TicketPendingSession key={`${user.id}:${user.role}`}>
      {children}
    </TicketPendingSession>
  );
}
