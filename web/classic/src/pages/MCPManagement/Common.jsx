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
import { useTranslation } from 'react-i18next';

export function CopyButton({ value, label }) {
  const { t } = useTranslation();
  const [status, setStatus] = useState('');
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      setStatus(t('Copied'));
    } catch {
      setStatus(t('Copy failed. Select and copy the text manually.'));
    }
  };
  return (
    <span className='mcp-copy'>
      <Button onClick={copy}>{label}</Button>
      <span role='status'>{status}</span>
    </span>
  );
}

export function DataState({ data }) {
  const { t } = useTranslation();
  if (data.loading) return <p role='status'>{t('Loading MCP data...')}</p>;
  if (data.error)
    return (
      <p role='alert'>
        {t('Unable to load MCP data. Check your permissions and retry.')}
      </p>
    );
  if (!data.items.length) return <p>{t('No MCP records')}</p>;
  return null;
}

export function Pager({ data }) {
  const { t } = useTranslation();
  return (
    <nav className='mcp-toolbar' aria-label={t('MCP pagination')}>
      <Button
        disabled={data.loading || data.page <= 1}
        onClick={() => data.setPage(data.page - 1)}
      >
        {t('Previous page')}
      </Button>
      <span>
        {t('Page {{page}} / {{total}} records', {
          page: data.page,
          total: data.total,
        })}
      </span>
      <Button
        disabled={data.loading || data.error || data.page * 20 >= data.total}
        onClick={() => data.setPage(data.page + 1)}
      >
        {t('Next page')}
      </Button>
    </nav>
  );
}
