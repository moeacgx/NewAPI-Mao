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
import TokenTimestamp from './TokenTimestamp';
import { TokenStatusBadge, TokenGroupBadge } from './TokenDefaultBadges';
import {
  Power,
  PowerOff,
  SquarePen,
  MoreHorizontal,
  Copy,
  Link,
  ArrowLeftRight,
  Trash2,
} from 'lucide-react';
import {
  Button,
  Dropdown,
  Tooltip,
  Progress,
  Popover,
  Modal,
} from '@douyinfe/semi-ui';
import { getCurrencyConfig, renderQuota, showError } from '../../../helpers';

// progress color helper
const getProgressColor = (pct) => {
  if (pct === 100) return 'var(--semi-color-success)';
  if (pct <= 10) return 'var(--semi-color-danger)';
  if (pct <= 30) return 'var(--semi-color-warning)';
  return undefined;
};

// Render functions
function renderTimestamp(timestamp, now) {
  return <TokenTimestamp timestamp={timestamp} now={now} />;
}

// Render status column only (no usage)
const renderStatus = (text, record, t) => {
  const config = {
    1: ['已启用', 'success'],
    2: ['已禁用', 'neutral'],
    3: ['已过期', 'warning'],
    4: ['已耗尽', 'danger'],
  }[text] || ['未知状态', 'neutral'];
  return <TokenStatusBadge label={t(config[0])} variant={config[1]} />;
};

const renderGroupColumn = (text, record, t, groupRatios = {}) => {
  const details = Array.isArray(record.group_details)
    ? record.group_details
    : [];
  const labels = Object.fromEntries(
    details
      .filter((group) => group.code && group.name)
      .map((group) => [group.code, group.name]),
  );
  if (text === 'auto')
    return (
      <Tooltip
        content={t(
          '当前分组为 auto，会自动选择最优分组，当一个组不可用时自动降级到下一个组（熔断机制）',
        )}
      >
        <span>
          <TokenStatusBadge label={t('跨分组')} variant='info' />
        </span>
      </Tooltip>
    );
  const codes = (text || '')
    .split(',')
    .map((code) => code.trim())
    .filter(Boolean);
  if (!codes.length) return <TokenStatusBadge label={t('User Group')} />;
  const code = codes[0],
    name = labels[code] || code;
  return (
    <Tooltip
      content={
        <div>
          {codes.map((item, index) => (
            <div key={item}>
              {codes.length > 1 ? index + 1 + '. ' : ''}
              {labels[item] || item}
            </div>
          ))}
        </div>
      }
    >
      <span className='tokens-group'>
        <TokenGroupBadge code={code} name={name} ratio={groupRatios[code]} />
        {codes.length > 1 && (
          <TokenStatusBadge variant='info' label={'+' + (codes.length - 1)} />
        )}
      </span>
    </Tooltip>
  );
};

// Render token key column with show/hide and copy functionality
const renderTokenKey = (
  text,
  record,
  showKeys,
  resolvedTokenKeys,
  loadingTokenKeys,
  toggleTokenVisibility,
  copyTokenKey,
  copyTokenConnectionString,
  t,
) => {
  const revealed = !!showKeys[record.id];
  const loading = !!loadingTokenKeys[record.id];
  const keyValue =
    revealed && resolvedTokenKeys[record.id]
      ? resolvedTokenKeys[record.id]
      : record.key || '';
  const displayedKey = keyValue ? `sk-${keyValue}` : '';

  return (
    <div className='tokens-key'>
      <Popover
        trigger='custom'
        visible={revealed}
        position='bottomLeft'
        onClickOutSide={() => {
          if (revealed) toggleTokenVisibility(record);
        }}
        content={
          <div className='tokens-key-popover'>
            <div>{t('Full API Key')}</div>
            <input
              autoFocus
              aria-label={t('Full API Key')}
              readOnly
              value={displayedKey}
              onFocus={(event) => event.target.select()}
            />
          </div>
        }
      >
        <Button
          className='tokens-key-value'
          theme='borderless'
          type='tertiary'
          size='small'
          loading={loading}
          aria-label={t('查看密钥')}
          onClick={() => toggleTokenVisibility(record)}
        >
          {record.key ? 'sk-' + record.key : ''}
        </Button>
      </Popover>
      <Button
        theme='borderless'
        type='tertiary'
        size='small'
        icon={<Copy size={14} />}
        loading={loading}
        aria-label={t('复制密钥')}
        onClick={() => copyTokenKey(record)}
      />
    </div>
  );
};

// Render model limits column
const renderModelLimits = (text, record, t, isMobile) => {
  const models = record.model_limits_enabled
    ? (text || '').split(',').filter(Boolean)
    : [];
  if (!models.length) return <TokenStatusBadge label={t('无限制')} />;
  return (
    <Tooltip
      trigger={isMobile ? 'click' : 'hover'}
      content={
        <div>
          {models.map((model) => (
            <div key={model}>{model}</div>
          ))}
        </div>
      }
    >
      <span>
        <TokenStatusBadge
          label={t('{{count}} models', { count: models.length })}
        />
      </span>
    </Tooltip>
  );
};
const renderAllowIps = (text, t, isMobile) => {
  const ips = (text || '')
    .split('\n')
    .map((ip) => ip.trim())
    .filter(Boolean);
  if (!ips.length) return <TokenStatusBadge label={t('无限制')} />;
  return (
    <Tooltip
      trigger={isMobile ? 'click' : 'hover'}
      content={
        <div>
          {ips.map((ip) => (
            <div key={ip}>{ip}</div>
          ))}
        </div>
      }
    >
      <span>
        <TokenStatusBadge label={t('{{count}} IP(s)', { count: ips.length })} />
      </span>
    </Tooltip>
  );
};

// Render separate quota usage column
const renderQuotaUsage = (text, record, t, isMobile) => {
  const used = Number(record.used_quota) || 0,
    remain = Number(record.remain_quota) || 0,
    total = used + remain;
  const { symbol, type } = getCurrencyConfig();
  const amount = (value) => {
    const formatted = renderQuota(value);
    if (type === 'TOKENS' || !formatted.startsWith(symbol)) return formatted;
    return Number(formatted.slice(symbol.length)).toLocaleString(undefined, {
      maximumFractionDigits: 2,
    });
  };
  const percent =
    total > 0 ? Math.max(0, Math.min(100, (remain / total) * 100)) : 0;
  return (
    <div className={isMobile ? 'tokens-quota-card' : 'tokens-quota-cell'}>
      <Popover
        trigger='click'
        position='top'
        content={
          <div className='tokens-default-overlay' style={{ padding: 12 }}>
            <div>
              {t('已用额度')}: {renderQuota(used)}
            </div>
            {!record.unlimited_quota && (
              <>
                <div>
                  {t('剩余额度')}: {renderQuota(remain)}
                </div>
                <div>
                  {t('总额度')}: {renderQuota(total)}
                </div>
              </>
            )}
          </div>
        }
      >
        <button
          type='button'
          className='tokens-quota-trigger'
          aria-label={t('已用额度') + ' ' + amount(used)}
        >
          <span className='tokens-quota'>
            {isMobile && (
              <span className='tokens-quota-label'>
                {t('剩余额度')} ({type === 'TOKENS' ? t('Tokens') : symbol})
              </span>
            )}
            <span>{record.unlimited_quota ? t('无限制') : amount(remain)}</span>
            {isMobile && (
              <span className='tokens-quota-label'>{t('已用额度')}</span>
            )}
            <span>{amount(used)}</span>
          </span>
        </button>
      </Popover>
      {!record.unlimited_quota && (
        <Progress
          size='small'
          percent={percent}
          showInfo={false}
          stroke={getProgressColor(percent)}
          aria-label={t('剩余额度')}
        />
      )}
    </div>
  );
};

// Render operations column
const renderOperations = (
  text,
  record,
  onOpenLink,
  openCCSwitchForRecord,
  setEditingToken,
  setShowEdit,
  manageToken,
  refresh,
  t,
  copyTokenKey,
  copyTokenConnectionString,
) => {
  let chatsArray = [];
  try {
    const raw = localStorage.getItem('chats');
    const parsed = JSON.parse(raw);
    if (Array.isArray(parsed)) {
      for (let i = 0; i < parsed.length; i++) {
        const item = parsed[i];
        const name = Object.keys(item)[0];
        if (!name) continue;
        chatsArray.push({
          node: 'item',
          key: i,
          name,
          value: item[name],
          onClick: () => onOpenLink(name, item[name], record),
        });
      }
    }
  } catch (_) {
    showError(t('聊天链接配置错误，请联系管理员'));
  }

  return (
    <div className='tokens-row-actions'>
      <Tooltip content={t(record.status === 1 ? '禁用' : '启用')}>
        <Button
          theme='borderless'
          type={record.status === 1 ? 'danger' : 'tertiary'}
          size='small'
          icon={
            record.status === 1 ? <PowerOff size={16} /> : <Power size={16} />
          }
          aria-label={t(record.status === 1 ? '禁用' : '启用')}
          onClick={async () => {
            await manageToken(
              record.id,
              record.status === 1 ? 'disable' : 'enable',
              record,
            );
            await refresh();
          }}
        />
      </Tooltip>
      <Tooltip content={t('编辑')}>
        <Button
          theme='borderless'
          type='tertiary'
          size='small'
          icon={<SquarePen size={16} />}
          aria-label={t('编辑')}
          onClick={() => {
            setEditingToken(record);
            setShowEdit(true);
          }}
        />
      </Tooltip>
      <Dropdown
        trigger='click'
        position='bottomRight'
        render={
          <Dropdown.Menu className='tokens-more-menu'>
            <Dropdown.Item onClick={() => copyTokenKey(record)}>
              {t('复制密钥')}
              <Copy size={15} />
            </Dropdown.Item>
            <Dropdown.Item onClick={() => copyTokenConnectionString(record)}>
              {t('复制连接信息')}
              <Link size={15} />
            </Dropdown.Item>
            <Dropdown.Divider />
            <Dropdown.Item onClick={() => openCCSwitchForRecord(record)}>
              CC Switch
              <ArrowLeftRight size={15} />
            </Dropdown.Item>
            <Dropdown trigger='hover' position='leftTop' menu={chatsArray}>
              <Dropdown.Item
                onClick={() => {
                  if (!chatsArray.length)
                    showError(t('请联系管理员配置聊天链接'));
                }}
              >
                {t('聊天')}
              </Dropdown.Item>
            </Dropdown>
            <Dropdown.Divider />
            <Dropdown.Item
              type='danger'
              onClick={() =>
                Modal.confirm({
                  title: t('确定是否要删除此令牌？'),
                  content: t('此修改将不可逆'),
                  onOk: async () => {
                    await manageToken(record.id, 'delete', record);
                    await refresh();
                  },
                })
              }
            >
              {t('删除')}
              <Trash2 size={15} />
            </Dropdown.Item>
          </Dropdown.Menu>
        }
      >
        <Button
          theme='borderless'
          type='tertiary'
          size='small'
          icon={<MoreHorizontal size={17} />}
          aria-label={t('更多')}
        />
      </Dropdown>
    </div>
  );
};

export const getTokensColumns = ({
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
  groupRatios = {},
  isMobile = false,
  now = Date.now(),
}) => {
  const columns = [
    {
      title: t('名称'),
      dataIndex: 'name',
      width: 180,
      render: (text) => <span className='tokens-name'>{text}</span>,
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 120,
      key: 'status',
      render: (text, record) => renderStatus(text, record, t),
    },
    {
      title:
        t('额度') +
        ' (' +
        (getCurrencyConfig().type === 'TOKENS'
          ? t('Tokens')
          : getCurrencyConfig().symbol) +
        ')',
      width: 260,
      key: 'quota_usage',
      render: (text, record) => renderQuotaUsage(text, record, t, isMobile),
    },
    {
      title: t('分组'),
      dataIndex: 'group',
      width: 220,
      key: 'group',
      render: (text, record) => renderGroupColumn(text, record, t, groupRatios),
    },
    {
      title: t('API key'),
      width: 260,
      key: 'token_key',
      render: (text, record) =>
        renderTokenKey(
          text,
          record,
          showKeys,
          resolvedTokenKeys,
          loadingTokenKeys,
          toggleTokenVisibility,
          copyTokenKey,
          copyTokenConnectionString,
          t,
        ),
    },
    {
      title: t('Models'),
      dataIndex: 'model_limits',
      width: 160,
      render: (text, record) => renderModelLimits(text, record, t, isMobile),
    },
    {
      title: t('IP限制'),
      dataIndex: 'allow_ips',
      width: 160,
      render: (text) => renderAllowIps(text, t, isMobile),
    },
    {
      title: t('时间'),
      dataIndex: 'activity_time',
      width: 220,
      render: (_text, record) => (
        <div className='tokens-time' data-table-text='secondary'>
          <span>{t('创建时间')}</span>
          <span>{renderTimestamp(record.created_time, now)}</span>
          <span>{t('最后使用时间')}</span>
          <span>
            {record.accessed_time
              ? renderTimestamp(record.accessed_time, now)
              : '-'}
          </span>
        </div>
      ),
    },
    {
      title: t('过期时间'),
      dataIndex: 'expired_time',
      width: 180,
      render: (text) =>
        text === -1 ? (
          <TokenStatusBadge label={t('永不过期')} />
        ) : (
          <span
            className={text * 1000 <= now ? 'tokens-expired' : 'tokens-expiry'}
          >
            <TokenTimestamp timestamp={text} now={now} />
          </span>
        ),
    },
    {
      title: t('操作'),
      width: 110,
      dataIndex: 'operate',
      fixed: 'right',
      render: (text, record, index) =>
        renderOperations(
          text,
          record,
          onOpenLink,
          openCCSwitchForRecord,
          setEditingToken,
          setShowEdit,
          manageToken,
          refresh,
          t,
          copyTokenKey,
          copyTokenConnectionString,
        ),
    },
  ];
  const keyColumn = columns.find((column) => column.key === 'token_key');
  return [
    columns[0],
    columns[1],
    keyColumn,
    ...columns.slice(2).filter((column) => column !== keyColumn),
  ];
};
