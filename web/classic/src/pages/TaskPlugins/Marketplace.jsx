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
  Modal,
  Space,
  Spin,
  Table,
  Typography,
} from '@douyinfe/semi-ui';
import React, { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { pluginError, taskPluginRequest } from './api';
import {
  downloadVerifiedPlugin,
  fetchMarketplaceText,
  parseMarketplaceIndex,
  resolveMarketplaceSource,
  marketplaceError,
} from './marketplace-utils';
import MarketplaceSources from './MarketplaceSources';

export default function Marketplace(props) {
  const { t } = useTranslation();
  const [sources, setSources] = useState([]);
  const [indexUrl, setIndexUrl] = useState('');
  const [plugins, setPlugins] = useState([]);
  const [versions, setVersions] = useState(new Map());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [sourceError, setSourceError] = useState('');
  const [success, setSuccess] = useState(false);
  const [revision, setRevision] = useState(0);
  const [manageSources, setManageSources] = useState(false);
  const [pending, setPending] = useState(null);
  const [preview, setPreview] = useState(null);
  const [previewError, setPreviewError] = useState('');
  const [busy, setBusy] = useState(false);
  const [denied, setDenied] = useState(false);
  const lock = useRef(false);
  const installAbort = useRef(null);
  const canManage = props.canManage && !denied;

  useEffect(() => () => installAbort.current?.abort(), []);

  useEffect(() => {
    setPreview(null);
    setPreviewError('');
    if (!pending) return;
    const controller = new AbortController();
    downloadVerifiedPlugin(pending.indexUrl, pending.version, controller.signal)
      .then((source) => {
        if (!controller.signal.aborted) setPreview(source);
      })
      .catch((err) => {
        if (!controller.signal.aborted)
          setPreviewError(marketplaceError(err, t));
      });
    return () => controller.abort();
  }, [pending, t]);

  useEffect(() => {
    const controller = new AbortController();
    setSourceError('');
    taskPluginRequest(
      'get',
      '/marketplace/sources',
      undefined,
      controller.signal,
    )
      .then((data) => {
        if (controller.signal.aborted) return;
        if (
          !Array.isArray(data) ||
          !data.every(
            (source) =>
              typeof source?.name === 'string' &&
              typeof source?.index_url === 'string',
          )
        )
          throw new Error(t('Unsupported plugin source configuration.'));
        setSources(data);
        setIndexUrl((current) =>
          data.some((source) => source.index_url === current)
            ? current
            : data[0]?.index_url || '',
        );
        if (!data.length) setLoading(false);
      })
      .catch((err) => {
        if (controller.signal.aborted) return;
        setSourceError(pluginError(err, t));
        setLoading(false);
      });
    return () => controller.abort();
  }, [revision, t]);

  useEffect(() => {
    if (!indexUrl) {
      setPlugins([]);
      setPending(null);
      return;
    }
    const controller = new AbortController();
    setPlugins([]);
    setVersions(new Map());
    setPending(null);
    setSuccess(false);
    setError('');
    setLoading(true);
    Promise.resolve()
      .then(() => {
        const url = resolveMarketplaceSource(indexUrl);
        return fetchMarketplaceText(url, 2 * 1024 * 1024, controller.signal);
      })
      .then((text) => {
        if (!controller.signal.aborted)
          setPlugins(parseMarketplaceIndex(JSON.parse(text)));
      })
      .catch((err) => {
        if (!controller.signal.aborted) setError(marketplaceError(err, t));
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [indexUrl, revision, t]);

  const install = async () => {
    if (
      !pending ||
      preview === null ||
      previewError ||
      !canManage ||
      lock.current
    )
      return;
    lock.current = true;
    setBusy(true);
    setError('');
    const controller = new AbortController();
    installAbort.current = controller;
    try {
      await taskPluginRequest(
        'post',
        '',
        {
          source: preview,
          sourceSha256: pending.version.sha256,
          expectedKey: pending.key,
          expectedVersion: pending.version.version,
          marketplace: {
            name: pending.sourceName,
            index_url: pending.indexUrl,
            path: pending.version.path,
          },
        },
        controller.signal,
      );
      setPending(null);
      setSuccess(true);
      props.onInstalled();
    } catch (err) {
      if (controller.signal.aborted) return;
      setError(t(pluginError(err, t)));
      if ([401, 403].includes(err.response?.status)) setDenied(true);
      setPending(null);
    } finally {
      lock.current = false;
      setBusy(false);
    }
  };

  return (
    <Card title={t('Marketplace')}>
      <div className='task-plugin-stack'>
        <Typography.Text>
          {t(
            'Install stable versions from your configured repositories, or upload temporary plugins directly.',
          )}
        </Typography.Text>
        <Space wrap>
          <label htmlFor='task-plugin-marketplace-source'>
            {t('Plugin source repository')}
          </label>
          <select
            id='task-plugin-marketplace-source'
            className='task-plugin-select'
            disabled={busy || !sources.length || Boolean(sourceError)}
            value={indexUrl}
            onChange={(event) => setIndexUrl(event.target.value)}
          >
            {sources.map((source) => (
              <option key={source.index_url} value={source.index_url}>
                {source.name}
              </option>
            ))}
          </select>
          <Button
            disabled={busy || loading}
            onClick={() => setRevision((value) => value + 1)}
          >
            {t('Refresh marketplace')}
          </Button>
          {canManage && !sourceError && (
            <Button
              disabled={busy || loading}
              onClick={() => setManageSources(true)}
            >
              {t('Manage plugin sources')}
            </Button>
          )}
        </Space>
        {indexUrl && (
          <Typography.Text className='task-plugin-source-url' type='tertiary'>
            {indexUrl}
          </Typography.Text>
        )}
        {sourceError && <div role='alert'>{sourceError}</div>}
        {error && <div role='alert'>{error}</div>}
        {success && (
          <div role='status'>
            {t(
              'Plugin installed disabled and inactive. Activate it from Installed when ready.',
            )}
          </div>
        )}
        <Typography.Text type='tertiary'>
          {t(
            'SHA-256 checks source integrity, not publisher identity. Install only from sources you trust.',
          )}
        </Typography.Text>
        {loading ? (
          <div role='status' aria-label={t('Loading marketplace')}>
            <Spin />
          </div>
        ) : (
          !error &&
          !sourceError && (
            <div className='task-plugin-table'>
              <Table
                rowKey='key'
                pagination={{ pageSize: 10 }}
                dataSource={plugins}
                empty={
                  <Empty
                    description={t('No compatible task plugins in this source')}
                  />
                }
                columns={[
                  {
                    title: t('Plugin'),
                    render: (_, row) => (
                      <div>
                        {row.name}
                        <div>{row.key}</div>
                      </div>
                    ),
                  },
                  {
                    title: t('Version'),
                    render: (_, row) => (
                      <select
                        className='task-plugin-select'
                        aria-label={t('Version for {{key}}', { key: row.key })}
                        value={versions.get(row.key) || row.latest}
                        disabled={busy}
                        onChange={(event) =>
                          setVersions((current) =>
                            new Map(current).set(row.key, event.target.value),
                          )
                        }
                      >
                        {row.versions.map((version) => (
                          <option key={version.version} value={version.version}>
                            {version.version}
                          </option>
                        ))}
                      </select>
                    ),
                  },
                  {
                    title: t('Actions'),
                    render: (_, row) => {
                      const version = row.versions.find(
                        (entry) =>
                          entry.version ===
                          (versions.get(row.key) || row.latest),
                      );
                      const hasHash = /^[a-f0-9]{64}$/i.test(
                        version?.sha256 || '',
                      );
                      return (
                        <Space vertical align='start'>
                          {!hasHash && (
                            <Typography.Text type='warning'>
                              {t(
                                'A valid SHA-256 is required to install this version.',
                              )}
                            </Typography.Text>
                          )}
                          {canManage && (
                            <Button
                              disabled={
                                busy || !hasHash || Boolean(sourceError)
                              }
                              onClick={() =>
                                setPending({
                                  key: row.key,
                                  version,
                                  indexUrl,
                                  sourceName:
                                    sources.find(
                                      (source) => source.index_url === indexUrl,
                                    )?.name || '',
                                })
                              }
                            >
                              {t('Review and install')}
                            </Button>
                          )}
                        </Space>
                      );
                    },
                  },
                ]}
              />
            </div>
          )
        )}
      </div>
      {manageSources && (
        <MarketplaceSources
          sources={sources}
          onClose={() => setManageSources(false)}
          onSaved={() => {
            setManageSources(false);
            setRevision((value) => value + 1);
          }}
          onDenied={() => {
            setManageSources(false);
            setDenied(true);
            setSourceError(t('No permission to access task plugins'));
          }}
        />
      )}
      <Modal
        visible={Boolean(pending)}
        title={t('Install plugin version?')}
        okText={t('Install version')}
        cancelText={t('Cancel')}
        onOk={install}
        onCancel={() => {
          if (!busy) setPending(null);
        }}
        confirmLoading={busy}
        closable={!busy}
        maskClosable={!busy}
        okButtonProps={{
          'aria-label': t('Install version'),
          disabled: !canManage || preview === null || Boolean(previewError),
        }}
        cancelButtonProps={{ disabled: busy, 'aria-label': t('Cancel') }}
      >
        <div className='task-plugin-stack'>
          <Typography.Text>
            {pending?.key} {pending?.version.version}
          </Typography.Text>
          <Typography.Text className='task-plugin-source-url'>
            {pending?.indexUrl}
          </Typography.Text>
          <Typography.Text className='task-plugin-source-url'>
            SHA-256: {pending?.version.sha256}
          </Typography.Text>
          <Typography.Text>
            {t(
              'Installation does not activate the plugin or change your channels.',
            )}
          </Typography.Text>
          {previewError ? (
            <div role='alert'>{previewError}</div>
          ) : preview === null ? (
            <div role='status'>{t('Verifying plugin source...')}</div>
          ) : (
            <pre className='task-plugin-json' aria-label={t('Plugin source')}>
              {preview}
            </pre>
          )}
        </div>
      </Modal>
    </Card>
  );
}
