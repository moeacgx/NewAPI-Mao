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
import { Button, Card } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { CopyButton } from './Common';

export default function Access({ onNavigate }) {
  const { t } = useTranslation();
  const [client, setClient] = useState('Codex');
  const url = window.location.origin + '/mcp';
  const codex = `[mcp_servers.newapi_mao]\nurl = "${url}"\nbearer_token_env_var = "NEWAPI_MCP_KEY"`;
  const cursor = JSON.stringify(
    {
      mcpServers: {
        newapi_mao: {
          url,
          headers: { Authorization: 'Bearer ${env:NEWAPI_MCP_KEY}' },
        },
      },
    },
    null,
    2,
  );
  const instructions = t(
    'Set NEWAPI_MCP_KEY in your client environment, then restart the client. Use HTTPS for remote access. Call newapi_context first to verify identity and permissions.',
  );
  return (
    <div className='mcp-stack'>
      <div className='mcp-access-actions'>
        <CopyButton
          value={`${instructions}\n\n${url}\n\n${codex}\n\n${cursor}`}
          label={t('Copy instructions')}
        />
      </div>
      <Card className='mcp-access-card'>
        <div className='mcp-access-grid'>
          <div>
            <h2>{t('Connect your operations agent')}</h2>
            <p className='mcp-muted'>
              {t(
                'Manage your backend from Cursor or Codex with explicitly authorized tools.',
              )}
            </p>
            <ol className='mcp-steps'>
              <li>
                {t('Create a call key and choose its tools and expiry.')}{' '}
                <Button theme='borderless' onClick={() => onNavigate?.('keys')}>
                  {t('Manage keys')}
                </Button>
              </li>
              <li>
                {t(
                  'Copy the client configuration and set NEWAPI_MCP_KEY in your local environment.',
                )}
              </li>
              <li>
                {t(
                  'Call newapi_context first to verify the site, identity and permissions.',
                )}
              </li>
            </ol>
            <div className='mcp-endpoint'>
              <h3>{t('Current MCP endpoint')}</h3>
              <div className='mcp-address'>
                <code>{url}</code>
                <CopyButton value={url} label={t('Copy MCP URL')} />
              </div>
              <p className='mcp-muted'>
                {t(
                  'Configuration does not indicate a connected client. Use HTTPS for remote access.',
                )}
              </p>
            </div>
          </div>
          <div className='mcp-client'>
            <div
              className='mcp-client-switch'
              aria-label={t('Client configuration')}
            >
              {['Cursor', 'Codex'].map((name) => (
                <button
                  key={name}
                  type='button'
                  aria-pressed={client === name}
                  onClick={() => setClient(name)}
                >
                  {name}
                </button>
              ))}
            </div>
            <div className='mcp-config'>
              <div className='mcp-config-heading'>
                <span>
                  {client === 'Codex'
                    ? 'Codex / config.toml'
                    : 'Cursor / .cursor/mcp.json'}
                </span>
                <CopyButton
                  value={client === 'Codex' ? codex : cursor}
                  label={
                    client === 'Codex'
                      ? t('Copy Codex configuration')
                      : t('Copy Cursor configuration')
                  }
                />
              </div>
              <pre className='mcp-code'>
                {client === 'Codex' ? codex : cursor}
              </pre>
            </div>
            <p className='mcp-muted'>
              {t(
                'Configurations contain an environment variable reference, never your secret.',
              )}
            </p>
          </div>
        </div>
        <div className='mcp-rules'>
          <div className='mcp-section-heading'>
            <h3>{t('Tools and execution rules')}</h3>
            <Button theme='borderless' onClick={() => onNavigate?.('tools')}>
              {t('Tool reference')}
            </Button>
          </div>
          <ul>
            <li>
              {t(
                'A key only grants its selected tools within the current owner permissions.',
              )}
            </li>
            <li>
              {t(
                'Write tools require request_id. Check the result before retrying with the same ID and arguments.',
              )}
            </li>
            <li>
              {t(
                'Channel tests and other external operations may incur upstream charges.',
              )}
            </li>
            <li>
              {t(
                'Keep secrets in your local environment. Call logs do not store arguments or secrets.',
              )}
            </li>
          </ul>
        </div>
      </Card>
    </div>
  );
}
