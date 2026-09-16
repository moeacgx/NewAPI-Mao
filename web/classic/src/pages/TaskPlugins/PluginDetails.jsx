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
  SideSheet,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { taskPluginRequest, taskPluginPath, pluginError } from './api';

export default function PluginDetails(props) {
  const { t } = useTranslation();
  const [detail, setDetail] = useState(null);
  const [versions, setVersions] = useState([]);
  const [error, setError] = useState('');
  const [versionError, setVersionError] = useState('');
  const [loading, setLoading] = useState(true);
  const [retry, setRetry] = useState(0);
  const [version, setVersion] = useState('');
  useEffect(() => {
    const controller = new AbortController();
    setDetail(null);
    setVersions([]);
    setError('');
    setVersionError('');
    setLoading(true);
    const path = taskPluginPath(props.plugin.key);
    const load = async () => {
      const results = await Promise.allSettled([
        taskPluginRequest(
          'get',
          path + (version ? '?version=' + encodeURIComponent(version) : ''),
          undefined,
          controller.signal,
        ),
        taskPluginRequest(
          'get',
          path + '/versions',
          undefined,
          controller.signal,
        ),
      ]);
      if (controller.signal.aborted) return;
      if (results[0].status === 'fulfilled' && results[0].value?.key)
        setDetail(results[0].value);
      else
        setError(
          pluginError(
            results[0].reason || new Error(t('Task plugin request failed')),
            t,
          ),
        );
      if (results[1].status === 'fulfilled' && Array.isArray(results[1].value))
        setVersions(results[1].value);
      else
        setVersionError(
          pluginError(
            results[1].reason || new Error(t('Task plugin request failed')),
            t,
          ),
        );
      setLoading(false);
    };
    load();
    return () => controller.abort();
  }, [props.plugin.key, props.revision, retry, version, t]);

  const metadata = detail?.meta || {};
  return (
    <SideSheet
      visible
      title={t('Task plugin details')}
      width='min(680px, 100vw)'
      onCancel={props.onClose}
    >
      <div className='task-plugin-stack'>
        <Button onClick={props.onClose}>{t('Close details')}</Button>
        {loading && (
          <Card>
            <div role='status' aria-label={t('Loading plugin details')}>
              <Spin />
            </div>
          </Card>
        )}
        {error && (
          <Card>
            <div role='alert'>{error}</div>
            <Button onClick={() => setRetry(retry + 1)}>{t('Retry')}</Button>
          </Card>
        )}
        {detail && (
          <>
            <Card title={metadata.name || detail.key}>
              <dl className='task-plugin-facts'>
                <dt>{t('Plugin key')}</dt>
                <dd>{detail.key}</dd>
                <dt>{t('Version')}</dt>
                <dd>{detail.version || t('Not provided')}</dd>
                <dt>{t('API version')}</dt>
                <dd>{detail.api_version ?? t('Not provided')}</dd>
                <dt>{t('Source hash')}</dt>
                <dd>{detail.source_hash || t('Not provided')}</dd>
                {detail.marketplace && (
                  <>
                    <dt>{t('Plugin source repository')}</dt>
                    <dd>
                      {detail.marketplace.name}
                      <div>{detail.marketplace.index_url}</div>
                      <div>{detail.marketplace.path}</div>
                    </dd>
                  </>
                )}
                <dt>{t('Enabled channels')}</dt>
                <dd>{detail.channel_count ?? t('Not provided')}</dd>
                <dt>{t('In-flight tasks')}</dt>
                <dd>{detail.in_flight_count ?? t('Not provided')}</dd>
              </dl>
            </Card>
            <Card title={t('Capabilities and usage schema')}>
              {[
                'description',
                'author',
                'models',
                'protocols',
                'routes',
                'usageSchema',
              ].map((field) => (
                <div key={field}>
                  <Typography.Text strong>{field}</Typography.Text>
                  <pre className='task-plugin-json'>
                    {metadata[field] == null
                      ? t('Not provided')
                      : JSON.stringify(metadata[field], null, 2)}
                  </pre>
                </div>
              ))}
            </Card>
            {props.canManage && typeof detail.source === 'string' && (
              <Card title={t('Plugin source')}>
                <pre className='task-plugin-json'>{detail.source}</pre>
              </Card>
            )}
          </>
        )}
        {!loading && (
          <Card title={t('Plugin versions')}>
            {versionError ? (
              <div role='alert'>
                {versionError}
                <Button onClick={() => setRetry(retry + 1)}>
                  {t('Retry')}
                </Button>
              </div>
            ) : (
              <Table
                rowKey='version'
                pagination={false}
                dataSource={versions}
                empty={<Empty description={t('No version data')} />}
                columns={[
                  { title: t('Version'), dataIndex: 'version' },
                  {
                    title: t('Status'),
                    render: (_, row) =>
                      row.active ? <Tag>{t('Active')}</Tag> : t('Inactive'),
                  },
                  {
                    title: t('Actions'),
                    render: (_, row) => (
                      <Space wrap>
                        <Button
                          disabled={props.busy}
                          onClick={() => setVersion(row.version)}
                        >
                          {t('View version')}
                        </Button>
                        {props.canManage && row.active === false && (
                          <Button
                            disabled={props.busy}
                            onClick={() =>
                              props.onActivate(props.plugin.key, row.version)
                            }
                          >
                            {t('Activate')}
                          </Button>
                        )}
                        {props.canManage &&
                          row.source_kind === 'custom' &&
                          row.active === false && (
                            <Button
                              disabled={props.busy}
                              onClick={async () => {
                                if (
                                  !window.confirm(
                                    t(
                                      'Delete this custom plugin version? Historical tasks referencing it cannot be deleted.',
                                    ),
                                  )
                                )
                                  return;
                                try {
                                  await taskPluginRequest(
                                    'delete',
                                    `${taskPluginPath(props.plugin.key)}/versions/${encodeURIComponent(row.version)}`,
                                  );
                                  props.onChanged?.();
                                } catch (err) {
                                  setVersionError(pluginError(err, t));
                                }
                              }}
                            >
                              {t('Delete')}
                            </Button>
                          )}
                      </Space>
                    ),
                  },
                ]}
              />
            )}
          </Card>
        )}
      </div>
    </SideSheet>
  );
}
