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

import React, { useEffect, useMemo, useState } from 'react';
import { Empty } from '@douyinfe/semi-ui';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import TokensMobileList from './TokensMobileList';

import { getTokensColumns } from './TokensColumnDefs';

const TokensTable = (tokensData) => {
  const isMobile = useIsMobile();
  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 30000);
    return () => window.clearInterval(timer);
  }, []);
  const {
    tokens,
    loading,
    showKeys,
    resolvedTokenKeys,
    loadingTokenKeys,
    toggleTokenVisibility,
    copyTokenKey,
    copyTokenConnectionString,
    openCCSwitchForRecord,
    manageToken,
    onOpenLink,
    setEditingToken,
    setShowEdit,
    refresh,
    groupRatios,
    t,
  } = tokensData;

  // Get all columns
  const columns = useMemo(() => {
    return getTokensColumns({
      t,
      showKeys,
      resolvedTokenKeys,
      loadingTokenKeys,
      toggleTokenVisibility,
      copyTokenKey,
      copyTokenConnectionString,
      openCCSwitchForRecord,
      manageToken,
      onOpenLink,
      setEditingToken,
      setShowEdit,
      refresh,
      groupRatios,
      isMobile,
      now,
    });
  }, [
    t,
    showKeys,
    resolvedTokenKeys,
    loadingTokenKeys,
    toggleTokenVisibility,
    copyTokenKey,
    copyTokenConnectionString,
    openCCSwitchForRecord,
    manageToken,
    onOpenLink,
    setEditingToken,
    setShowEdit,
    refresh,
    groupRatios,
    isMobile,
    now,
  ]);

  if (isMobile) {
    return (
      <TokensMobileList
        tokens={tokens}
        columns={columns}
        loading={loading}
        searching={tokensData.searching}
        t={t}
      />
    );
  }

  const tableColumns = columns.filter(
    (column) =>
      !['model_limits', 'allow_ips', 'activity_time', 'expired_time'].includes(
        column.dataIndex,
      ) || (tokensData.visibleColumns || []).includes(column.dataIndex),
  );

  const pageIds = new Set(tokens.map((token) => token.id));
  const selectedOnPage = (tokensData.selectedKeys || []).filter((token) =>
    pageIds.has(token.id),
  );
  const selectedIds = new Set(selectedOnPage.map((token) => token.id));
  const allSelected =
    tokens.length > 0 && tokens.every((token) => selectedIds.has(token.id));
  const someSelected = tokens.some((token) => selectedIds.has(token.id));
  const totalSize =
    40 +
    tableColumns
      .filter((column) => column.dataIndex !== 'operate')
      .reduce((sum, column) => sum + (column.width || 160), 0);
  return (
    <div className='tokens-table' aria-busy={loading || tokensData.searching}>
      <div
        className='tokens-table-scroll'
        tabIndex={0}
        aria-label={t('API keys')}
      >
        <table
          data-slot='table'
          style={{
            minWidth: 'max(100%, ' + totalSize + 'px)',
            tableLayout: 'auto',
            width: '100%',
          }}
        >
          <colgroup>
            <col style={{ width: (40 / totalSize) * 100 + '%' }} />
            {tableColumns.map((column) => (
              <col
                key={column.key || column.dataIndex}
                style={{
                  width:
                    column.dataIndex === 'operate'
                      ? '1%'
                      : ((column.width || 160) / totalSize) * 100 + '%',
                }}
              />
            ))}
          </colgroup>
          <thead data-slot='table-header'>
            <tr>
              <th data-column-id='select'>
                <input
                  type='checkbox'
                  aria-label={t('全选')}
                  checked={allSelected}
                  ref={(node) => {
                    if (node) node.indeterminate = someSelected && !allSelected;
                  }}
                  onChange={(event) =>
                    tokensData.setSelectedKeys(
                      event.target.checked ? tokens : [],
                    )
                  }
                />
              </th>
              {tableColumns.map((column) => (
                <th
                  key={column.key || column.dataIndex}
                  data-column-id={column.dataIndex || column.key}
                  className={
                    column.dataIndex === 'operate' ? 'tokens-pinned' : ''
                  }
                >
                  {column.title}
                </th>
              ))}
            </tr>
          </thead>
          <tbody data-slot='table-body'>
            {tokens.map((record, index) => (
              <tr
                key={record.id}
                data-state={selectedIds.has(record.id) ? 'selected' : undefined}
                className={record.status !== 1 ? 'tokens-disabled-row' : ''}
              >
                <td data-column-id='select'>
                  <input
                    type='checkbox'
                    aria-label={t('选择') + ' ' + record.name}
                    checked={selectedIds.has(record.id)}
                    onChange={(event) =>
                      tokensData.setSelectedKeys(
                        event.target.checked
                          ? [...selectedOnPage, record]
                          : selectedOnPage.filter(
                              (token) => token.id !== record.id,
                            ),
                      )
                    }
                  />
                </td>
                {tableColumns.map((column) => (
                  <td
                    key={column.key || column.dataIndex}
                    data-column-id={column.dataIndex || column.key}
                    className={
                      column.dataIndex === 'operate' ? 'tokens-pinned' : ''
                    }
                  >
                    {column.render
                      ? column.render(record[column.dataIndex], record, index)
                      : record[column.dataIndex]}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
        {!tokens.length && (
          <div className='tokens-table-empty'>
            {loading ? (
              t('加载中...')
            ) : (
              <Empty description={t('搜索无结果')} image={null} />
            )}
          </div>
        )}
      </div>
    </div>
  );
};
export default TokensTable;
