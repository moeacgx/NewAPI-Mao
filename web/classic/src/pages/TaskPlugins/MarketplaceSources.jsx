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

import { Button, Input, Modal, Typography } from '@douyinfe/semi-ui';
import React, { useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { pluginError, taskPluginRequest } from './api';
import {
  resolveMarketplaceSource,
  marketplaceError,
} from './marketplace-utils';

export default function MarketplaceSources(props) {
  const { t } = useTranslation();
  const nextId = useRef(props.sources.length);
  const [rows, setRows] = useState(() =>
    props.sources.map((source, id) => ({ ...source, id })),
  );
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const lock = useRef(false);
  const update = (id, field, value) =>
    setRows((current) =>
      current.map((row) => (row.id === id ? { ...row, [field]: value } : row)),
    );

  const save = async () => {
    if (lock.current) return;
    setError('');
    let sources;
    try {
      sources = rows.map((row) => ({
        name: row.name.trim(),
        index_url: resolveMarketplaceSource(row.index_url.trim()),
      }));
      if (
        sources.length > 16 ||
        sources.some(
          (source) =>
            !source.name ||
            source.name.length > 128 ||
            source.index_url.length > 2048,
        ) ||
        new Set(sources.map((source) => source.index_url)).size !==
          sources.length
      )
        throw new Error(
          'Provide up to 16 unique plugin sources with names and HTTPS index URLs.',
        );
    } catch (err) {
      setError(marketplaceError(err, t));
      return;
    }
    lock.current = true;
    setBusy(true);
    try {
      await taskPluginRequest('put', '/marketplace/sources', sources);
      props.onSaved();
    } catch (err) {
      setError(pluginError(err, t));
      if ([401, 403].includes(err.response?.status)) props.onDenied();
    } finally {
      lock.current = false;
      setBusy(false);
    }
  };

  return (
    <Modal
      visible
      title={t('Manage plugin sources')}
      okText={t('Save plugin sources')}
      cancelText={t('Cancel')}
      onOk={save}
      onCancel={() => {
        if (!busy) props.onClose();
      }}
      confirmLoading={busy}
      closable={!busy}
      maskClosable={!busy}
      okButtonProps={{ 'aria-label': t('Save plugin sources') }}
      cancelButtonProps={{ disabled: busy, 'aria-label': t('Cancel') }}
    >
      <div className='task-plugin-stack'>
        <Typography.Text>
          {t(
            'Sources are shared across administrators. Removing a source does not remove installed plugins.',
          )}
        </Typography.Text>
        {rows.map((row, index) => (
          <div className='task-plugin-source-row' key={row.id}>
            <Input
              aria-label={t('Source name {{number}}', { number: index + 1 })}
              value={row.name}
              maxLength={128}
              disabled={busy}
              onChange={(value) => update(row.id, 'name', value)}
              placeholder={t('Source name')}
            />
            <Input
              aria-label={t('Index URL {{number}}', { number: index + 1 })}
              value={row.index_url}
              maxLength={2048}
              disabled={busy}
              onChange={(value) => update(row.id, 'index_url', value)}
              placeholder={t('HTTPS index URL')}
            />
            <Button
              aria-label={t('Remove source {{number}}', { number: index + 1 })}
              disabled={busy}
              onClick={() =>
                setRows((current) =>
                  current.filter((item) => item.id !== row.id),
                )
              }
            >
              {t('Remove source')}
            </Button>
          </div>
        ))}
        <Button
          disabled={busy || rows.length >= 16}
          onClick={() =>
            setRows((current) => [
              ...current,
              { id: nextId.current++, name: '', index_url: '' },
            ])
          }
        >
          {t('Add plugin source')}
        </Button>
        {error && <div role='alert'>{error}</div>}
      </div>
    </Modal>
  );
}
