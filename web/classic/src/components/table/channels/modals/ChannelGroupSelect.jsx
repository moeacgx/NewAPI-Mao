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

import React, { useRef, useState } from 'react';
import {
  Button,
  Empty,
  Input,
  Popover,
  Select,
  Tag,
  withField,
} from '@douyinfe/semi-ui';
import { IconClose, IconSearch } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import GroupTagFilter from '../../../common/group-tags/GroupTagFilter';
import { filterGroupsByTag } from '../../../../helpers/groupTags';
import './channel-group-select.css';

// 渠道分组选择：值仍是分组 code 数组，提交时由 buildGroupSelectionPayload 生成 group/group_ids。
// 标签只按稳定 Group.Id 收窄候选，不改变已选值；没有标签时保持原多选下拉（属性与旧字段一致）。
// Semi Select 无 allowAdditions/additionLabel 属性，旧字段实际不能新增分组，卡片模式同样只选已有分组。
function ChannelGroupSelect(props) {
  const { t } = useTranslation();
  const { groupOptions = [], tags = [], onChange, disabled } = props;
  const value = Array.isArray(props.value) ? props.value : [];
  const [selectedTag, setSelectedTag] = useState('all');
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const anchor = useRef(null);

  if (tags.length === 0) {
    return (
      <Select
        multiple
        allowAdditions
        additionLabel={t('请在系统设置页面编辑分组倍率以添加新的分组：')}
        placeholder={props.placeholder}
        optionList={groupOptions}
        value={value}
        disabled={disabled}
        style={{ width: '100%' }}
        position='top'
        onChange={onChange}
      />
    );
  }

  const selected = new Set(value);
  const keyword = search.trim().toLocaleLowerCase();
  const candidates = filterGroupsByTag(groupOptions, tags, selectedTag).filter(
    (group) =>
      !keyword ||
      String(group.label || '')
        .toLocaleLowerCase()
        .includes(keyword) ||
      String(group.value || '')
        .toLocaleLowerCase()
        .includes(keyword),
  );
  const labelOf = (code) =>
    groupOptions.find((group) => group.value === code)?.label || code;

  const toggle = (code) =>
    onChange(
      selected.has(code)
        ? value.filter((item) => item !== code)
        : [...value, code],
    );
  const close = () => {
    setOpen(false);
    anchor.current?.focus();
  };

  const picker = (
    <div
      className='channel-group-picker'
      role='dialog'
      aria-label={t('选择分组')}
    >
      <div className='channel-group-picker-search'>
        <Input
          prefix={<IconSearch />}
          placeholder={t('搜索分组...')}
          aria-label={t('搜索分组...')}
          value={search}
          onChange={setSearch}
          autoFocus
          showClear
          size='small'
        />
        <Button
          icon={<IconClose size='small' />}
          aria-label={t('关闭')}
          theme='borderless'
          size='small'
          onClick={close}
        />
      </div>
      <div className='channel-group-picker-options'>
        {candidates.map((group) => (
          <label className='channel-group-picker-option' key={group.value}>
            <input
              type='checkbox'
              disabled={disabled}
              checked={selected.has(group.value)}
              onChange={() => toggle(group.value)}
            />
            <span className='channel-group-picker-option-name'>
              {group.label || group.value}
            </span>
          </label>
        ))}
        {candidates.length === 0 && (
          <Empty description={t('没有可选分组')} style={{ padding: 16 }} />
        )}
      </div>
    </div>
  );

  return (
    <div className='channel-group-select'>
      {/* 浮窗锚定整个卡片区并左对齐，避免右侧卡片的浮窗超出右侧抽屉。 */}
      <Popover
        trigger='custom'
        visible={open}
        motion={false}
        position='bottomLeft'
        autoAdjustOverflow
        closeOnEsc
        onEscKeyDown={close}
        onClickOutSide={() => setOpen(false)}
        content={open ? picker : null}
      >
        <div className='channel-group-tag-anchor'>
          <GroupTagFilter
            tags={tags}
            groups={groupOptions}
            value={selectedTag}
            onChange={setSelectedTag}
            disabled={disabled}
            showUntagged={false}
            hideEmpty
            expanded={open}
            onActivate={(tag, event) => {
              anchor.current = event.currentTarget;
              setSelectedTag(tag);
              setSearch('');
              setOpen(true);
            }}
          />
        </div>
      </Popover>
      <div className='channel-group-selected'>
        <div className='channel-group-selected-heading'>
          <strong>{t('已选分组')}</strong>
          <span className='channel-group-selected-count'>{value.length}</span>
          {value.length > 0 && (
            <Button
              size='small'
              theme='borderless'
              type='danger'
              disabled={disabled}
              onClick={() => onChange([])}
            >
              {t('清空')}
            </Button>
          )}
        </div>
        {value.length === 0 ? (
          <span className='channel-group-selected-empty'>
            {props.placeholder}
          </span>
        ) : (
          <div className='channel-group-selected-list'>
            {value.map((code) => (
              <Tag
                key={code}
                size='large'
                closable={!disabled}
                onClose={() => toggle(code)}
                className='channel-group-selected-tag'
              >
                {labelOf(code)}
              </Tag>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export default withField(ChannelGroupSelect);
