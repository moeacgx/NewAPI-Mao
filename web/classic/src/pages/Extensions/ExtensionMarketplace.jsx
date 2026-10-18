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
import {
  Button,
  Card,
  Empty,
  Modal,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { Download } from 'lucide-react';
import { API } from '../../helpers/api';
import {
  parseExtensionMarketplaceMetadata,
  loadExtensionCatalog,
  downloadExtensionArchive,
  extensionMarketplaceError,
} from './marketplace-utils';

const { Text } = Typography;

export default function ExtensionMarketplace(props) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [metadata, setMetadata] = useState(null);
  const [entries, setEntries] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [revision, setRevision] = useState(0);
  const [pending, setPending] = useState(null);
  const [busy, setBusy] = useState(false);
  const [installError, setInstallError] = useState('');
  const [success, setSuccess] = useState(false);
  const installAbort = useRef(null);
  const installLocked = useRef(false);

  useEffect(() => () => installAbort.current?.abort(), []);

  useEffect(() => {
    if (!props.canManage || !open) return undefined;
    const controller = new AbortController();
    setLoading(true);
    setError('');
    setEntries([]);
    setMetadata(null);
    const load = async () => {
      let source;
      try {
        const response = await API.get('/api/extension-admin/marketplace', {
          signal: controller.signal,
        });
        if (!response?.data?.success) throw new Error('metadata');
        source = parseExtensionMarketplaceMetadata(response.data.data);
      } catch {
        if (!controller.signal.aborted)
          setError(t('Could not load the extension repository.'));
        return;
      }
      try {
        const modules = await loadExtensionCatalog(
          source.catalog_url,
          controller.signal,
        );
        if (!controller.signal.aborted) {
          setMetadata(source);
          setEntries(modules);
        }
      } catch (err) {
        if (!controller.signal.aborted)
          setError(extensionMarketplaceError(err, t));
      }
    };
    load().finally(() => {
      if (!controller.signal.aborted) setLoading(false);
    });
    return () => controller.abort();
  }, [props.canManage, open, revision, t]);

  const install = async () => {
    if (
      !open ||
      !pending ||
      !metadata ||
      !props.canManage ||
      installLocked.current
    )
      return;
    installLocked.current = true;
    setBusy(true);
    setInstallError('');
    setSuccess(false);
    const controller = new AbortController();
    installAbort.current = controller;
    try {
      const archive = await downloadExtensionArchive(
        metadata.catalog_url,
        pending,
        controller.signal,
        metadata.max_archive_bytes,
      );
      if (controller.signal.aborted) return;
      const form = new FormData();
      form.append('file', archive, `${pending.id}-${pending.version}.zip`);
      form.append('archiveSha256', pending.sha256);
      form.append('expectedId', pending.id);
      form.append('expectedVersion', pending.version);
      form.append('catalogUrl', metadata.catalog_url);
      form.append('archivePath', pending.path);
      const response = await API.post('/api/extension-admin/upload', form, {
        signal: controller.signal,
      });
      if (!response?.data?.success) {
        setInstallError(
          response?.data?.message || t('Extension installation failed.'),
        );
        return;
      }
      if (controller.signal.aborted) return;
      setPending(null);
      setSuccess(true);
      await props.onInstalled?.();
    } catch (err) {
      if (!controller.signal.aborted) {
        setInstallError(
          err.response?.data?.message || extensionMarketplaceError(err, t),
        );
      }
    } finally {
      installLocked.current = false;
      if (!controller.signal.aborted) setBusy(false);
    }
  };

  if (!props.canManage) return null;

  const columns = [
    {
      title: t('模块'),
      dataIndex: 'name',
      width: 240,
      render: (name, entry) => (
        <div style={{ overflowWrap: 'anywhere' }}>
          <Text strong>{name}</Text>
          {entry.description && (
            <div>
              <Text type='secondary' size='small'>
                {entry.description}
              </Text>
            </div>
          )}
        </div>
      ),
    },
    { title: t('版本'), dataIndex: 'version', width: 110 },
    {
      title: t('Host requirements'),
      width: 300,
      render: (_, entry) => (
        <div style={{ overflowWrap: 'anywhere' }}>
          {entry.host.min && (
            <Text size='small'>
              {t('Minimum host: {{version}}', { version: entry.host.min })}
            </Text>
          )}
          {entry.host.max && (
            <div>
              <Text size='small'>
                {t('Maximum host: {{version}}', { version: entry.host.max })}
              </Text>
            </div>
          )}
          {entry.compatibility && (
            <div>
              <Text type='secondary' size='small'>
                {entry.compatibility}
              </Text>
            </div>
          )}
          {!entry.host.min && !entry.host.max && !entry.compatibility && (
            <Text type='tertiary'>{t('Unspecified')}</Text>
          )}
        </div>
      ),
    },
    {
      title: t('状态'),
      width: 175,
      render: (_, entry) => {
        let label = entry.status || t('Unspecified');
        if (entry.status === 'builtin-snapshot') label = t('Built-in snapshot');
        if (entry.status === 'requires-host-upgrade')
          label = t('Requires host support');
        return (
          <Tag
            style={{ maxWidth: '100%', whiteSpace: 'normal', height: 'auto' }}
          >
            {label}
          </Tag>
        );
      },
    },
    {
      title: t('操作'),
      width: 160,
      render: (_, entry) => (
        <Button
          size='small'
          theme='outline'
          disabled={busy || entry.size > metadata.max_archive_bytes}
          onClick={() => {
            setPending(entry);
            setInstallError('');
            setSuccess(false);
          }}
        >
          {t('Install from repository')}
        </Button>
      ),
    },
  ];

  return (
    <>
      <Button
        theme='outline'
        icon={<Download size={16} />}
        onClick={() => setOpen(true)}
      >
        {t('Online modules')}
      </Button>
      <Modal
        title={
          pending ? t('Confirm extension installation') : t('Online modules')
        }
        visible={open}
        width='min(1120px, calc(100vw - 32px))'
        centered
        bodyStyle={{ maxHeight: 'calc(85dvh - 140px)', overflowY: 'auto' }}
        maskClosable={!busy}
        closeOnEsc={!busy}
        closable={!busy}
        onCancel={() => {
          if (installLocked.current) return;
          setOpen(false);
          setPending(null);
          setInstallError('');
          setSuccess(false);
        }}
        footer={
          pending ? (
            <Space>
              <Button
                disabled={busy}
                onClick={() => {
                  setPending(null);
                  setInstallError('');
                }}
              >
                {t('Cancel')}
              </Button>
              <Button
                theme='solid'
                loading={busy}
                disabled={busy}
                onClick={install}
              >
                {t('Confirm installation')}
              </Button>
            </Space>
          ) : null
        }
      >
        {!pending ? (
          <Space vertical align='start' style={{ width: '100%' }} spacing={12}>
            <div
              style={{
                display: 'flex',
                flexWrap: 'wrap',
                gap: 12,
                justifyContent: 'space-between',
                width: '100%',
              }}
            >
              <Text type='secondary'>
                {t('Browse published modules from the maintained repository.')}
              </Text>
              <Button
                size='small'
                theme='outline'
                loading={loading}
                disabled={busy}
                onClick={() => setRevision((value) => value + 1)}
              >
                {t('Reload repository')}
              </Button>
            </div>
            {metadata && (
              <Space wrap>
                <a
                  href={metadata.repository_url}
                  target='_blank'
                  rel='noopener noreferrer'
                >
                  {t('Extension repository')}
                </a>
                <Text type='tertiary'>
                  {t('Current host: {{version}}', {
                    version: metadata.host_version,
                  })}
                </Text>
              </Space>
            )}
            {success && (
              <div role='status'>
                <Text type='success'>
                  {t(
                    'Extension installed. Review its enabled state in the module list.',
                  )}
                </Text>
              </div>
            )}
            {error && (
              <div role='alert'>
                <Text type='danger'>{error}</Text>
              </div>
            )}
            <Spin spinning={loading} style={{ width: '100%' }}>
              <Card bodyStyle={{ padding: 12 }} style={{ width: '100%' }}>
                {!loading && !error && entries.length === 0 && (
                  <Empty description={t('No online modules are available.')} />
                )}
                {entries.length > 0 && (
                  <Table
                    rowKey={(entry) => `${entry.id}:${entry.version}`}
                    columns={columns}
                    dataSource={entries}
                    pagination={false}
                    scroll={{ x: 985 }}
                  />
                )}
              </Card>
            </Spin>
          </Space>
        ) : (
          <Space vertical align='start' spacing={12} style={{ width: '100%' }}>
            <Space wrap>
              <Text strong>{pending.name}</Text>
              <Tag>{pending.version}</Tag>
            </Space>
            <Text type='secondary'>
              {t(
                'New installations start disabled. Updates preserve the current enabled state.',
              )}
            </Text>
            <Text type='secondary'>
              {t(
                'Install only modules you trust. SHA-256 verifies file integrity.',
              )}
            </Text>
            {pending.compatibility && <Text>{pending.compatibility}</Text>}
            {installError && (
              <div role='alert'>
                <Text type='danger'>{installError}</Text>
              </div>
            )}
          </Space>
        )}
      </Modal>
    </>
  );
}
