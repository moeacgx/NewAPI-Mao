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
import React, { useState } from 'react';
import { Button } from '@douyinfe/semi-ui';
import { MessageSquarePlus } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { useLocation, useNavigate } from 'react-router-dom';
import CreateTicketModal from './CreateTicketModal';

function TicketFloatingSession(props) {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const [visible, setVisible] = useState(false);
  if (
    !props.user?.id ||
    !(
      location.pathname === '/console' ||
      location.pathname.startsWith('/console/')
    )
  )
    return null;
  return (
    <>
      <Button
        className='ticket-floating-button'
        theme='solid'
        aria-label={t('Create support ticket')}
        icon={<MessageSquarePlus size={20} aria-hidden='true' />}
        onClick={() => setVisible(true)}
      >
        {t('Support')}
      </Button>
      <CreateTicketModal
        visible={visible}
        onClose={() => setVisible(false)}
        onCreated={(ticket) => navigate(`/console/tickets?ticket=${ticket.id}`)}
      />
    </>
  );
}

export default function TicketFloatingButton(props) {
  return (
    <TicketFloatingSession
      key={props.user?.id || 'anonymous'}
      user={props.user}
    />
  );
}
