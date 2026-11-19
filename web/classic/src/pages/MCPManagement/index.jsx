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
import { Card } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { isAdmin } from '../../helpers/utils';
import Access from './Access';
import Tools from './Tools';
import Keys from './Keys';
import Logs from './Logs';
import './style.css';

export default function MCPManagement() {
  const { t } = useTranslation();
  const [tab, setTab] = useState('access');
  const tabs = [
    {
      key: 'access',
      label: t('MCP access'),
      content: (
        <Access
          onNavigate={(key) => {
            setTab(key);
            document.getElementById(`mcp-tab-${key}`)?.focus();
          }}
        />
      ),
    },
    { key: 'tools', label: t('Tool reference'), content: <Tools /> },
    { key: 'keys', label: t('Call keys'), content: <Keys /> },
    { key: 'logs', label: t('Call logs'), content: <Logs /> },
  ];
  if (!isAdmin())
    return (
      <main className='mcp-page'>
        <Card>
          <p role='alert'>{t('Administrator access required')}</p>
        </Card>
      </main>
    );
  return (
    <main className='mcp-page mcp-stack'>
      <header className='mcp-heading'>
        <p className='mcp-muted'>
          {t('MCP access, tools, keys and call logs')}
        </p>
        <h1>{t('MCP management')}</h1>
      </header>
      <div className='mcp-tabs' role='tablist' aria-label={t('MCP management')}>
        {tabs.map((item, index) => (
          <button
            type='button'
            role='tab'
            key={item.key}
            id={`mcp-tab-${item.key}`}
            aria-controls={`mcp-panel-${item.key}`}
            aria-selected={tab === item.key}
            tabIndex={tab === item.key ? 0 : -1}
            onClick={() => setTab(item.key)}
            onKeyDown={(event) => {
              let next;
              if (event.key === 'ArrowRight') next = (index + 1) % tabs.length;
              if (event.key === 'ArrowLeft')
                next = (index + tabs.length - 1) % tabs.length;
              if (event.key === 'Home') next = 0;
              if (event.key === 'End') next = tabs.length - 1;
              if (next === undefined) return;
              event.preventDefault();
              setTab(tabs[next].key);
              document.getElementById(`mcp-tab-${tabs[next].key}`)?.focus();
            }}
          >
            {item.label}
          </button>
        ))}
      </div>
      <section
        role='tabpanel'
        id={`mcp-panel-${tab}`}
        aria-labelledby={`mcp-tab-${tab}`}
        tabIndex={0}
      >
        {tabs.find((item) => item.key === tab).content}
      </section>
    </main>
  );
}
