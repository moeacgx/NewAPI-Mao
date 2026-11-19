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

import React, { useEffect, useRef, useState } from 'react';
import { Button, Card, Modal } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import useMCPData from './useMCPData';
import { CopyButton, DataState, Pager } from './Common';
import { formatTime, mcpRequest } from './api';
import { ToolCatalog } from './ToolCatalog';

export default function Keys() {
  const { t, i18n } = useTranslation();
  const data = useMCPData('keys', true);
  const tools = useMCPData('tools');
  const [name, setName] = useState('');
  const [expires, setExpires] = useState('');
  const [selected, setSelected] = useState([]);
  const [secret, setSecret] = useState('');
  const [pending, setPending] = useState(null);
  const [inspecting, setInspecting] = useState(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const lock = useRef(false);
  const controller = useRef(null);
  const secretKeyId = useRef(null);
  useEffect(() => () => controller.current?.abort(), []);

  const create = async (event) => {
    event.preventDefault();
    if (lock.current) return;
    const expiresAt = Math.floor(new Date(expires).getTime() / 1000);
    if (!name.trim() || Array.from(name.trim()).length > 100) {
      setError(t('Enter a key name of 1-100 characters.'));
      return;
    }
    if (
      !Number.isFinite(expiresAt) ||
      expiresAt <= Date.now() / 1000 ||
      expiresAt > 253402300799
    ) {
      setError(t('Choose a future expiry time.'));
      return;
    }
    if (
      !selected.length ||
      tools.loading ||
      tools.error ||
      selected.some((tool) => !tools.items.some((item) => item.name === tool))
    ) {
      setError(t('Select at least one available tool.'));
      return;
    }
    lock.current = true;
    setBusy(true);
    setError('');
    setSecret('');
    const request = new AbortController();
    controller.current = request;
    try {
      const result = await mcpRequest(
        'post',
        'keys',
        { name: name.trim(), tools: selected, expires_at: expiresAt },
        request.signal,
      );
      if (request.signal.aborted) return;
      if (typeof result?.secret !== 'string' || !result.secret)
        throw new Error('Missing secret');
      // 完整密钥仅存于当前页签内存，关闭或卸载后不再保留。
      setSecret(result.secret);
      secretKeyId.current = result.key?.id ?? null;
      setName('');
      setExpires('');
      setSelected([]);
      if (data.page === 1) data.refresh();
      else data.setPage(1);
    } catch {
      if (!request.signal.aborted)
        setError(
          t(
            'MCP operation failed. Check permissions and refresh before retrying.',
          ),
        );
    } finally {
      lock.current = false;
      if (!request.signal.aborted) setBusy(false);
    }
  };

  const revoke = async () => {
    if (!pending || lock.current) return;
    lock.current = true;
    setBusy(true);
    setError('');
    const request = new AbortController();
    controller.current = request;
    try {
      await mcpRequest(
        'delete',
        `keys/${encodeURIComponent(pending.id)}`,
        undefined,
        request.signal,
      );
      if (!request.signal.aborted) {
        setPending(null);
        if (pending.id === secretKeyId.current) setSecret('');
        data.refresh();
      }
    } catch {
      if (!request.signal.aborted) {
        setPending(null);
        setError(
          t(
            'MCP operation failed. Check permissions and refresh before retrying.',
          ),
        );
      }
    } finally {
      lock.current = false;
      if (!request.signal.aborted) setBusy(false);
    }
  };

  return (
    <div className='mcp-stack'>
      <Card
        title={t('Call keys')}
        headerExtraContent={
          <Button disabled={data.loading || busy} onClick={data.refresh}>
            {t('Refresh')}
          </Button>
        }
      >
        <DataState data={data} />
        {data.items.length > 0 && (
          <div className='mcp-table'>
            <table>
              <caption className='sr-only'>{t('Call keys')}</caption>
              <thead>
                <tr>
                  {[
                    t('Key name'),
                    t('Key prefix'),
                    t('Allowed tools'),
                    t('Status'),
                    t('Created at'),
                    t('Expires at'),
                    t('Last used at'),
                    t('Actions'),
                  ].map((label) => (
                    <th scope='col' key={label}>
                      {label}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {data.items.map((key) => {
                  let status = t('Active');
                  if (key.revoked_at > 0) status = t('Revoked');
                  else if (
                    key.expires_at > 0 &&
                    key.expires_at <= Date.now() / 1000
                  )
                    status = t('Expired');
                  return (
                    <tr key={key.id}>
                      <td>{key.name}</td>
                      <td>{key.key_prefix}</td>
                      <td>
                        <div className='mcp-permission-summary'>
                          <span>
                            {t('{{count}} authorized tools', {
                              count: key.tools?.length ?? 0,
                            })}
                          </span>
                          <Button
                            size='small'
                            theme='borderless'
                            onClick={() => setInspecting(key)}
                          >
                            {t('View permissions')}
                          </Button>
                        </div>
                      </td>
                      <td>{status}</td>
                      <td>{formatTime(key.created_at, i18n.language)}</td>
                      <td>{formatTime(key.expires_at, i18n.language)}</td>
                      <td>{formatTime(key.last_used_at, i18n.language)}</td>
                      <td>
                        <Button
                          type='danger'
                          disabled={busy || key.revoked_at > 0}
                          onClick={() => setPending(key)}
                        >
                          {t('Revoke')}
                        </Button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
        <Pager data={data} />
      </Card>
      <Card title={t('Create call key')}>
        <form className='mcp-stack' onSubmit={create} noValidate>
          <div className='mcp-form-fields'>
            <label>
              {t('Key name')}
              <input
                value={name}
                disabled={busy}
                onChange={(event) => setName(event.target.value)}
                autoComplete='off'
              />
            </label>
            <label>
              {t('Expires at')}
              <input
                type='datetime-local'
                value={expires}
                disabled={busy}
                onChange={(event) => setExpires(event.target.value)}
              />
            </label>
          </div>
          <fieldset disabled={busy || tools.loading || tools.error}>
            <legend>{t('Allowed tools')}</legend>
            <DataState data={tools} />
            {!tools.loading && !tools.error && (
              <ToolCatalog
                tools={tools.items}
                selected={selected}
                onChange={setSelected}
                disabled={busy || !!secret}
              />
            )}
          </fieldset>
          {tools.error && (
            <Button onClick={tools.refresh}>{t('Retry tools')}</Button>
          )}
          <Button
            className='mcp-create-button'
            htmlType='submit'
            theme='solid'
            loading={busy}
            disabled={busy || tools.loading || tools.error || !!secret}
          >
            {t('Create call key')}
          </Button>
        </form>
        {error && <p role='alert'>{error}</p>}
      </Card>
      {secret && (
        <Card title={t('Save your key now')}>
          <p>
            {t(
              'This secret is shown only once. Closing it or leaving this tab clears it.',
            )}
          </p>
          <pre className='mcp-code'>{secret}</pre>
          <div className='mcp-toolbar'>
            <CopyButton value={secret} label={t('Copy secret')} />
            <Button onClick={() => setSecret('')}>{t('Close secret')}</Button>
          </div>
        </Card>
      )}
      <Modal
        title={t('View permissions')}
        visible={!!inspecting}
        width={880}
        style={{ maxWidth: 'calc(100vw - 32px)' }}
        bodyStyle={{ maxHeight: 'calc(100dvh - 240px)', overflowY: 'auto' }}
        onCancel={() => setInspecting(null)}
        footer={
          <Button onClick={() => setInspecting(null)}>{t('Close')}</Button>
        }
      >
        <div className='mcp-permissions'>
          <p className='mcp-break'>{inspecting?.name}</p>
          <ToolCatalog
            permissions
            tools={(inspecting?.tools ?? []).map(
              (name) =>
                tools.items.find((tool) => tool.name === name) ?? {
                  name,
                  category: 'unavailable',
                  description: t(
                    'This saved tool is absent from the current catalog.',
                  ),
                },
            )}
          />
        </div>
      </Modal>
      <Modal
        title={t('Revoke call key?')}
        visible={!!pending}
        onCancel={() => {
          if (!busy) setPending(null);
        }}
        onOk={revoke}
        okText={t('Confirm revoke')}
        cancelText={t('Cancel')}
        confirmLoading={busy}
        okButtonProps={{ 'aria-label': t('Confirm revoke') }}
        cancelButtonProps={{ disabled: busy, 'aria-label': t('Cancel') }}
        closable={!busy}
        maskClosable={!busy}
      >
        <p className='mcp-break'>{pending?.name}</p>
        <p>{t('Clients using this key will immediately lose access.')}</p>
      </Modal>
    </div>
  );
}
