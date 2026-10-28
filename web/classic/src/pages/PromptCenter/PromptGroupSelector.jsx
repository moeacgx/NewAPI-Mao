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
import { Button, Popover } from '@douyinfe/semi-ui';
import { X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import GroupTagFilter from '../../components/common/group-tags/GroupTagFilter';
import { filterGroupsByTag } from '../../helpers/groupTags';
import useGroupTags from '../../hooks/common/useGroupTags';

export default function PromptGroupSelector(props) {
  const { t } = useTranslation();
  const { tags, error, reload } = useGroupTags();
  const [selectedTag, setSelectedTag] = useState('all');
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const anchor = useRef(null);
  // 标签只收窄候选组；白名单始终保存实体分组 ID，不能应用令牌的互斥和自动分组规则。
  const groups = props.groups.filter((group) => group.id > 0);
  const availableTags = error ? [] : tags;
  const selected = new Set(props.selected);
  const candidates = filterGroupsByTag(
    groups,
    availableTags,
    selectedTag,
  ).filter((group) =>
    (group.name || group.code || '')
      .toLocaleLowerCase()
      .includes(search.trim().toLocaleLowerCase()),
  );
  const close = () => {
    setOpen(false);
    anchor.current?.focus();
  };
  const toggle = (id) =>
    props.onChange(
      selected.has(id)
        ? props.selected.filter((value) => value !== id)
        : [...props.selected, id],
    );
  const options = (
    <div className='prompt-center-group-options'>
      {candidates.map((group) => (
        <label key={group.id}>
          <input
            type='checkbox'
            disabled={props.disabled}
            checked={selected.has(group.id)}
            onChange={() => toggle(group.id)}
          />
          {group.name || group.code}
        </label>
      ))}
      {!candidates.length && (
        <p className='prompt-center-muted'>{t('No eligible groups')}</p>
      )}
    </div>
  );

  return (
    <div className='prompt-group-selector'>
      {error && (
        <div role='status' className='prompt-center-muted'>
          {t('Failed to load group tags')}
          <Button
            htmlType='button'
            size='small'
            disabled={props.disabled}
            onClick={reload}
          >
            {t('Retry')}
          </Button>
        </div>
      )}
      <GroupTagFilter
        tags={availableTags}
        groups={groups}
        value={selectedTag}
        onChange={setSelectedTag}
        disabled={props.disabled}
        showUntagged={false}
        hideEmpty
        expanded={open}
        onActivate={(tag, event) => {
          anchor.current = event.currentTarget;
          setSelectedTag(tag);
          setSearch('');
          setOpen(true);
        }}
        renderChoice={(choice, option) => (
          <Popover
            key={choice.id}
            trigger='custom'
            visible={open && selectedTag === choice.id}
            motion={false}
            position='bottomLeft'
            autoAdjustOverflow
            closeOnEsc
            onEscKeyDown={close}
            onClickOutSide={() => {
              if (selectedTag === choice.id) setOpen(false);
            }}
            content={
              open && selectedTag === choice.id ? (
                <div
                  className='prompt-group-picker'
                  role='dialog'
                  aria-label={t('Allowed groups')}
                >
                  <div className='prompt-group-picker-search'>
                    <input
                      type='search'
                      autoFocus
                      disabled={props.disabled}
                      aria-label={t('搜索分组...')}
                      placeholder={t('搜索分组...')}
                      value={search}
                      onChange={(event) => setSearch(event.target.value)}
                    />
                    <Button
                      htmlType='button'
                      theme='borderless'
                      icon={<X size={16} />}
                      aria-label={t('Close')}
                      onClick={close}
                    />
                  </div>
                  {options}
                </div>
              ) : null
            }
          >
            {option}
          </Popover>
        )}
      />
      {!availableTags.length && options}
      {availableTags.length > 0 && props.selected.length > 0 && (
        <div className='prompt-group-selected'>
          <strong>{t('已选分组')}</strong>
          {groups
            .filter((group) => selected.has(group.id))
            .map((group) => (
              <div className='prompt-group-selected-row' key={group.id}>
                <span>{group.name || group.code}</span>
                <Button
                  htmlType='button'
                  size='small'
                  theme='borderless'
                  disabled={props.disabled}
                  icon={<X size={14} />}
                  aria-label={t('Remove group {{name}}', {
                    name: group.name || group.code,
                  })}
                  onClick={() => toggle(group.id)}
                />
              </div>
            ))}
        </div>
      )}
      {props.selected
        .filter((id) => !groups.some((group) => group.id === id))
        .map((id) => (
          <label className='prompt-group-unavailable' key={id}>
            <input
              type='checkbox'
              checked
              disabled={props.disabled}
              onChange={() => toggle(id)}
            />
            {t('Unavailable group')} · {id}
          </label>
        ))}
    </div>
  );
}
