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

import {
  Button,
  Card,
  Empty,
  Input,
  Modal,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { isAdmin, isRoot } from '../../helpers/utils';
import { taskPluginRequest, taskPluginPath, pluginError } from './api';
import PluginDetails from './PluginDetails';

import './style.css';

export default function TaskPlugins() {
  const { t } = useTranslation();
  const [plugins, setPlugins] = useState([]);
  const [runtime, setRuntime] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [runtimeError, setRuntimeError] = useState('');
  const [mutationError, setMutationError] = useState('');
  const [selected, setSelected] = useState(null);
  const [revision, setRevision] = useState(0);
  const [pending, setPending] = useState(null);
  const [uploadOpen, setUploadOpen] = useState(false);
  const [uploadSource, setUploadSource] = useState('');
  const [uploadError, setUploadError] = useState('');
  const [busy, setBusy] = useState(false);
  const [denied, setDenied] = useState(false);
  const sequence = useRef(0);
  const lock = useRef(false);
  const admin = isAdmin();
  const canManage = isRoot() && !denied && !error;
  const load = useCallback(async () => {
    if (!admin) {
      setLoading(false);
      return;
    }
    const id = ++sequence.current;
    setLoading(true);
    setError('');
    setRuntimeError('');
    const results = await Promise.allSettled([
      taskPluginRequest('get'),
      taskPluginRequest('get', '/runtime/status'),
    ]);
    if (sequence.current !== id) return;
    if (
      results[0].status === 'fulfilled' &&
      Array.isArray(results[0].value) &&
      results[0].value.every((row) => typeof row?.key === 'string')
    ) {
      setPlugins(results[0].value);
    } else {
      setPlugins([]);
      setError(
        pluginError(
          results[0].reason || new Error(t('Task plugin request failed')),
          t,
        ),
      );
    }
    if (
      results[1].status === 'fulfilled' &&
      typeof results[1].value?.enabled === 'boolean'
    ) {
      setRuntime(results[1].value);
    } else {
      setRuntime(null);
      setRuntimeError(
        pluginError(
          results[1].reason || new Error(t('Task plugin request failed')),
          t,
        ),
      );
    }
    setLoading(false);
  }, [admin, t]);

  useEffect(() => {
    load();
    return () => {
      sequence.current++;
    };
  }, [load]);

  const mutate = async () => {
    if (!canManage || !pending || lock.current) return;
    lock.current = true;
    setBusy(true);
    setMutationError('');
    try {
      await taskPluginRequest(pending.method, pending.path, pending.body);
      setPending(null);
      await load();
      setRevision((value) => value + 1);
    } catch (err) {
      setMutationError(pluginError(err, t));
      if ([401, 403].includes(err.response?.status)) setDenied(true);
      setPending(null);
    } finally {
      lock.current = false;
      setBusy(false);
    }
  };
  const activate = (key, version) =>
    setPending({
      method: 'post',
      path: taskPluginPath(key) + '/activate',
      body: { version },
      label: t('Activate version {{version}} for {{key}}?', { key, version }),
    });

  const upload = async () => {
    if (!canManage || !uploadSource.trim() || busy) return;
    setBusy(true);
    setUploadError('');
    try {
      await taskPluginRequest('post', '', { source: uploadSource });
      setUploadSource('');
      setUploadOpen(false);
      await load();
      setRevision((value) => value + 1);
    } catch (err) {
      setUploadError(pluginError(err, t('Plugin upload failed')));
    } finally {
      setBusy(false);
    }
  };

  if (!admin)
    return (
      <div className='task-plugin-page'>
        <Card>
          <div role='alert'>{t('No permission to access task plugins')}</div>
        </Card>
      </div>
    );
  return (
    <div className='task-plugin-page task-plugin-stack'>
      <div className='task-plugin-toolbar'>
        <Typography.Title heading={3}>{t('Task Plugins')}</Typography.Title>
        <Button disabled={busy || loading} onClick={load}>
          {t('Refresh')}
        </Button>
        {canManage && (
          <Button
            disabled={busy || loading}
            onClick={() => setUploadOpen(true)}
          >
            {t('Upload custom task plugin')}
          </Button>
        )}
      </div>
      <Card title={t('New task submissions')}>
        <div className='task-plugin-stack'>
          <Typography.Text>
            {t(
              'Disabling blocks new submissions only. Existing tasks continue with their pinned versions.',
            )}
          </Typography.Text>
          {runtimeError && <div role='alert'>{runtimeError}</div>}
          {runtime && (
            <Space wrap>
              <Tag color={runtime.enabled ? 'green' : 'grey'}>
                {runtime.enabled ? t('Enabled') : t('Disabled')}
              </Tag>
              {runtime.builtin_only && <Tag>{t('Built-in plugins only')}</Tag>}
              {runtime.production_ready === false && (
                <Tag color='orange'>
                  {t('Production readiness not verified')}
                </Tag>
              )}
              {canManage && (
                <Button
                  disabled={loading || busy}
                  onClick={() =>
                    setPending({
                      method: 'put',
                      path: '/runtime/status',
                      body: { enabled: !runtime.enabled },
                      label: runtime.enabled
                        ? t('Disable new task submissions')
                        : t('Enable new task submissions'),
                    })
                  }
                >
                  {runtime.enabled
                    ? t('Disable new task submissions')
                    : t('Enable new task submissions')}
                </Button>
              )}
            </Space>
          )}
          <Typography.Text type='tertiary'>
            {t(
              'Root users can upload custom plugins. Marketplace, remote resources and dry-run are unavailable.',
            )}
          </Typography.Text>
        </div>
      </Card>
      {mutationError && (
        <Card>
          <div role='alert'>{mutationError}</div>
        </Card>
      )}
      <Card title={t('Installed')}>
        {loading ? (
          <div role='status' aria-label={t('Loading plugins')}>
            <Spin />
          </div>
        ) : error ? (
          <div role='alert'>
            {error}
            <Button onClick={load}>{t('Retry')}</Button>
          </div>
        ) : (
          <div className='task-plugin-table'>
            <Table
              rowKey='key'
              pagination={{ pageSize: 10 }}
              dataSource={plugins}
              empty={<Empty description={t('No task plugins found')} />}
              columns={[
                {
                  title: t('Plugin'),
                  render: (_, row) => (
                    <div>
                      {row.meta?.name || row.key}
                      <div>
                        <Typography.Text type='tertiary'>
                          {row.key}
                        </Typography.Text>
                      </div>
                    </div>
                  ),
                },
                {
                  title: t('Version'),
                  dataIndex: 'version',
                  render: (value) => value ?? t('Not provided'),
                },
                {
                  title: t('Source'),
                  dataIndex: 'source_kind',
                  render: (value) =>
                    value === 'builtin'
                      ? t('Built-in')
                      : value || t('Not provided'),
                },
                {
                  title: t('Status'),
                  render: (_, row) => (
                    <Space wrap>
                      <Tag>
                        {row.active === true ? t('Active') : t('Inactive')}
                      </Tag>
                      <Tag>
                        {row.enabled === true ? t('Enabled') : t('Disabled')}
                      </Tag>
                    </Space>
                  ),
                },
                {
                  title: t('Enabled channels'),
                  dataIndex: 'channel_count',
                  render: (value) => value ?? t('Not provided'),
                },
                {
                  title: t('In-flight tasks'),
                  dataIndex: 'in_flight_count',
                  render: (value) => value ?? t('Not provided'),
                },
                {
                  title: t('Actions'),
                  render: (_, row) => (
                    <Space wrap>
                      <Button disabled={busy} onClick={() => setSelected(row)}>
                        {t('Details')}
                      </Button>
                      {canManage &&
                        row.active === true &&
                        typeof row.enabled === 'boolean' && (
                          <Button
                            disabled={busy}
                            onClick={() =>
                              setPending({
                                method: 'post',
                                path: taskPluginPath(row.key) + '/status',
                                body: { enabled: !row.enabled },
                                label: row.enabled
                                  ? t('Disable plugin')
                                  : t('Enable plugin'),
                              })
                            }
                          >
                            {row.enabled
                              ? t('Disable plugin')
                              : t('Enable plugin')}
                          </Button>
                        )}
                    </Space>
                  ),
                },
              ]}
            />
          </div>
        )}
      </Card>
      {selected && (
        <PluginDetails
          key={selected.key}
          plugin={selected}
          revision={revision}
          canManage={canManage}
          busy={busy}
          onActivate={activate}
          onChanged={() => {
            setRevision((value) => value + 1);
            void load();
          }}
          onClose={() => setSelected(null)}
        />
      )}
      <Modal
        visible={Boolean(pending)}
        title={pending?.label}
        okText={t('Confirm')}
        cancelText={t('Cancel')}
        onOk={mutate}
        onCancel={() => {
          if (!busy) setPending(null);
        }}
        confirmLoading={busy}
        okButtonProps={{ 'aria-label': t('Confirm') }}
        cancelButtonProps={{ disabled: busy, 'aria-label': t('Cancel') }}
        closable={!busy}
        maskClosable={!busy}
      >
        {t(
          'Disabling blocks new submissions only. Existing tasks continue with their pinned versions.',
        )}
      </Modal>
      <Modal
        visible={uploadOpen}
        title={t('Upload custom task plugin')}
        okText={t('Upload')}
        cancelText={t('Cancel')}
        confirmLoading={busy}
        onOk={upload}
        onCancel={() => {
          if (!busy) setUploadOpen(false);
        }}
        okButtonProps={{
          disabled: !uploadSource.trim() || uploadSource.length > 1024 * 1024,
        }}
        cancelButtonProps={{ disabled: busy }}
      >
        <Typography.Paragraph type='tertiary'>
          {t(
            'Only Root users can upload. The source is compiled and stored disabled and inactive.',
          )}
        </Typography.Paragraph>
        <Typography.Text>{t('Plugin source')}</Typography.Text>
        <Input.TextArea
          value={uploadSource}
          onChange={setUploadSource}
          rows={14}
          maxLength={1024 * 1024}
          placeholder={t('Paste JavaScript plugin source')}
          style={{ fontFamily: 'monospace', marginTop: 8 }}
        />
        <Typography.Text type='tertiary'>
          {t(
            'Maximum source size: 1 MiB. The plugin key and version come from its metadata.',
          )}
        </Typography.Text>
        {uploadError && <div role='alert'>{uploadError}</div>}
      </Modal>
    </div>
  );
}
