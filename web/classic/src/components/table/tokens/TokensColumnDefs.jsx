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
  Space,
  Tag,
  AvatarGroup,
  Avatar,
  Tooltip,
  Progress,
  Popover,
  Typography,
  Modal,
} from '@douyinfe/semi-ui';
import {
  timestamp2string,
  renderGroup,
  renderQuota,
  getModelCategories,
  showError,
} from '../../../helpers';
import {
  IconTreeTriangleDown,
  IconCopy,
  IconEyeOpened,
  IconEyeClosed,
} from '@douyinfe/semi-icons';

// progress color helper
const getProgressColor = (pct) => {
  if (pct === 100) return 'var(--semi-color-success)';
  if (pct <= 10) return 'var(--semi-color-danger)';
  if (pct <= 30) return 'var(--semi-color-warning)';
  return undefined;
};

// Render functions
function renderTimestamp(timestamp) {
  return <>{timestamp2string(timestamp)}</>;
}

// Render status column only (no usage)
const renderStatus = (text, record, t) => {
  const enabled = text === 1;

  let tagColor = 'black';
  let tagText = t('未知状态');
  if (enabled) {
    tagColor = 'green';
    tagText = t('已启用');
  } else if (text === 2) {
    tagColor = 'red';
    tagText = t('已禁用');
  } else if (text === 3) {
    tagColor = 'yellow';
    tagText = t('已过期');
  } else if (text === 4) {
    tagColor = 'grey';
    tagText = t('已耗尽');
  }

  return (
    <span
      style={{
        color: enabled
          ? 'var(--tokens-success, #008f69)'
          : 'var(--semi-color-text-1)',
        fontWeight: 500,
      }}
    >
      {tagText}
    </span>
  );
};

// Render group column
const renderGroupColumn = (text, record, t, groupRatios = {}) => {
  const groupDetails = Array.isArray(record?.group_details)
    ? record.group_details
    : [];
  const groupLabels = groupDetails.reduce((labels, group) => {
    if (group?.code && group?.name && group.name !== group.code) {
      labels[group.code] = group.name;
    }
    return labels;
  }, {});
  if (text === 'auto') {
    return (
      <Tooltip
        content={t(
          '当前分组为 auto，会自动选择最优分组，当一个组不可用时自动降级到下一个组（熔断机制）',
        )}
        position='top'
      >
        <Tag color='white' shape='circle'>
          {t('智能熔断')}
          {record && record.cross_group_retry ? `(${t('跨分组')})` : ''}
        </Tag>
      </Tooltip>
    );
  }
  // Multi-group: show first group + count badge with tooltip
  if (text && text.includes(',')) {
    const groupList = text
      .split(',')
      .map((g) => g.trim())
      .filter(Boolean);
    const firstRatio = groupRatios[groupList[0]];
    return (
      <Tooltip
        content={
          <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            {groupList.map((g, i) => (
              <span key={g}>
                {i + 1}. {groupLabels[g] || g}
              </span>
            ))}
          </div>
        }
        position='top'
      >
        <span className='tokens-group'>
          {renderGroup(groupList[0], groupLabels)}
          {firstRatio !== undefined && (
            <Tag className='tokens-ratio' size='small' shape='circle'>
              {firstRatio}x
            </Tag>
          )}
          {groupList.length > 1 && (
            <Tag size='small' color='blue' shape='circle'>
              +{groupList.length - 1}
            </Tag>
          )}
        </span>
      </Tooltip>
    );
  }
  const ratio = groupRatios[text];
  return (
    <span className='tokens-group'>
      {renderGroup(text, groupLabels)}
      {ratio !== undefined && (
        <Tag className='tokens-ratio' size='small' shape='circle'>
          {ratio}x
        </Tag>
      )}
    </span>
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
      <Button
        className='tokens-key-value'
        theme='borderless'
        type='tertiary'
        size='small'
        loading={loading}
        aria-label={t('查看密钥')}
        onClick={() => toggleTokenVisibility(record)}
      >
        {displayedKey}
      </Button>
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
const renderModelLimits = (text, record, t) => {
  if (record.model_limits_enabled && text) {
    const models = text.split(',').filter(Boolean);
    const categories = getModelCategories(t);

    const vendorAvatars = [];
    const matchedModels = new Set();
    Object.entries(categories).forEach(([key, category]) => {
      if (key === 'all') return;
      if (!category.icon || !category.filter) return;
      const vendorModels = models.filter((m) =>
        category.filter({ model_name: m }),
      );
      if (vendorModels.length > 0) {
        vendorAvatars.push(
          <Tooltip
            key={key}
            content={vendorModels.join(', ')}
            position='top'
            showArrow
          >
            <Avatar
              size='extra-extra-small'
              alt={category.label}
              color='transparent'
            >
              {category.icon}
            </Avatar>
          </Tooltip>,
        );
        vendorModels.forEach((m) => matchedModels.add(m));
      }
    });

    const unmatchedModels = models.filter((m) => !matchedModels.has(m));
    if (unmatchedModels.length > 0) {
      vendorAvatars.push(
        <Tooltip
          key='unknown'
          content={unmatchedModels.join(', ')}
          position='top'
          showArrow
        >
          <Avatar size='extra-extra-small' alt='unknown'>
            {t('其他')}
          </Avatar>
        </Tooltip>,
      );
    }

    return <AvatarGroup size='extra-extra-small'>{vendorAvatars}</AvatarGroup>;
  } else {
    return <span className='tokens-muted'>{t('无限制')}</span>;
  }
};

// Render IP restrictions column
const renderAllowIps = (text, t) => {
  if (!text || text.trim() === '') {
    return <span className='tokens-muted'>{t('无限制')}</span>;
  }

  const ips = text
    .split('\n')
    .map((ip) => ip.trim())
    .filter(Boolean);

  const displayIps = ips.slice(0, 1);
  const extraCount = ips.length - displayIps.length;

  const ipTags = displayIps.map((ip, idx) => (
    <Tag key={idx} shape='circle'>
      {ip}
    </Tag>
  ));

  if (extraCount > 0) {
    ipTags.push(
      <Tooltip
        key='extra'
        content={ips.slice(1).join(', ')}
        position='top'
        showArrow
      >
        <Tag shape='circle'>{'+' + extraCount}</Tag>
      </Tooltip>,
    );
  }

  return <Space wrap>{ipTags}</Space>;
};

// Render separate quota usage column
const renderQuotaUsage = (text, record, t) => {
  const { Paragraph } = Typography;
  const used = parseInt(record.used_quota) || 0;
  const remain = parseInt(record.remain_quota) || 0;
  const total = used + remain;
  if (record.unlimited_quota) {
    const popoverContent = (
      <div className='text-xs p-2'>
        <Paragraph copyable={{ content: renderQuota(used) }}>
          {t('已用额度')}: {renderQuota(used)}
        </Paragraph>
      </div>
    );
    return (
      <Popover content={popoverContent} position='top'>
        <div className='tokens-quota'>
          <span>{t('无限额度')}</span>
          <span className='tokens-muted'>{renderQuota(used)}</span>
        </div>
      </Popover>
    );
  }
  const percent = total > 0 ? (remain / total) * 100 : 0;
  const popoverContent = (
    <div className='text-xs p-2'>
      <Paragraph copyable={{ content: renderQuota(used) }}>
        {t('已用额度')}: {renderQuota(used)}
      </Paragraph>
      <Paragraph copyable={{ content: renderQuota(remain) }}>
        {t('剩余额度')}: {renderQuota(remain)} ({percent.toFixed(0)}%)
      </Paragraph>
      <Paragraph copyable={{ content: renderQuota(total) }}>
        {t('总额度')}: {renderQuota(total)}
      </Paragraph>
    </div>
  );
  return (
    <Popover content={popoverContent} position='top'>
      <Tag color='white' shape='circle'>
        <div className='flex flex-col items-end'>
          <span className='text-xs leading-none'>{`${renderQuota(remain)} / ${renderQuota(total)}`}</span>
          <Progress
            percent={percent}
            stroke={getProgressColor(percent)}
            aria-label='quota usage'
            format={() => `${percent.toFixed(0)}%`}
            style={{ width: '100%', marginTop: '1px', marginBottom: 0 }}
          />
        </div>
      </Tag>
    </Popover>
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
}) => {
  const columns = [
    {
      title: t('名称'),
      dataIndex: 'name',
      width: 190,
      render: (text) => <span className='tokens-name'>{text}</span>,
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 110,
      key: 'status',
      render: (text, record) => renderStatus(text, record, t),
    },
    {
      title: t('额度'),
      width: 210,
      key: 'quota_usage',
      render: (text, record) => renderQuotaUsage(text, record, t),
    },
    {
      title: t('分组'),
      dataIndex: 'group',
      width: 270,
      key: 'group',
      render: (text, record) => renderGroupColumn(text, record, t, groupRatios),
    },
    {
      title: t('API key'),
      width: 240,
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
      title: t('可用模型'),
      dataIndex: 'model_limits',
      width: 140,
      render: (text, record) => renderModelLimits(text, record, t),
    },
    {
      title: t('IP限制'),
      dataIndex: 'allow_ips',
      width: 140,
      render: (text) => renderAllowIps(text, t),
    },
    {
      title: t('时间'),
      dataIndex: 'activity_time',
      width: 270,
      render: (_text, record) => (
        <div className='tokens-time'>
          <span>{t('创建时间')}</span>
          <span>{renderTimestamp(record.created_time)}</span>
          <span>{t('最后使用时间')}</span>
          <span>
            {record.accessed_time ? renderTimestamp(record.accessed_time) : '-'}
          </span>
        </div>
      ),
    },
    {
      title: t('过期时间'),
      dataIndex: 'expired_time',
      width: 160,
      render: (text, record, index) => {
        return (
          <div>
            {record.expired_time === -1 ? t('永不过期') : renderTimestamp(text)}
          </div>
        );
      },
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
