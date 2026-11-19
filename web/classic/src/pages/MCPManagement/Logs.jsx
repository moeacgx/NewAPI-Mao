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
import { Button, Card, Tag } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import useMCPData from './useMCPData';
import { DataState, Pager } from './Common';
import { formatTime } from './api';

export default function Logs() {
  const { t, i18n } = useTranslation();
  const data = useMCPData('logs', true);
  return (
    <Card
      title={t('Call logs')}
      headerExtraContent={
        <Button disabled={data.loading} onClick={data.refresh}>
          {t('Refresh')}
        </Button>
      }
    >
      <DataState data={data} />
      {data.items.length > 0 && (
        <div className='mcp-table'>
          <table>
            <caption className='sr-only'>{t('Call logs')}</caption>
            <thead>
              <tr>
                {[
                  t('Time'),
                  t('Key ID'),
                  t('Tool name'),
                  t('Result'),
                  t('Request ID'),
                  t('Duration (ms)'),
                ].map((label) => (
                  <th key={label} scope='col'>
                    {label}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data.items.map((log) => (
                <tr key={log.id}>
                  <td>{formatTime(log.created_at, i18n.language)}</td>
                  <td>{log.key_id}</td>
                  <td>{log.tool_name}</td>
                  <td>
                    <Tag color={log.success ? 'green' : 'red'}>
                      {log.success ? t('Succeeded') : t('Failed')}
                    </Tag>
                  </td>
                  <td>{log.request_id || '-'}</td>
                  <td>{log.duration_ms}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <Pager data={data} />
    </Card>
  );
}
