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

import React, { useEffect, useId } from 'react';
import { useTranslation } from 'react-i18next';
import GroupTagLogos from './GroupTagLogos';
import { filterGroupsByTag } from '../../../helpers/groupTags';

export default function GroupTagFilter(props) {
  const { t } = useTranslation();
  const name = useId();
  const choices = [
    { id: 'all', name: t('All groups'), icons: [] },
    ...(props.showUntagged === false
      ? []
      : [{ id: 'untagged', name: t('Untagged'), icons: [] }]),
    ...props.tags.map((tag) => ({ ...tag, id: String(tag.id) })),
  ]
    .map((choice) => ({
      ...choice,
      count: filterGroupsByTag(props.groups, props.tags, choice.id).length,
    }))
    .filter(
      (choice) => !props.hideEmpty || choice.id === 'all' || choice.count > 0,
    );
  const active = choices.some((choice) => choice.id === props.value)
    ? props.value
    : 'all';
  const { value, onChange } = props;
  useEffect(() => {
    if (active !== value) onChange(active);
  }, [active, value, onChange]);

  if (props.tags.length === 0) return null;

  return (
    <fieldset className='group-tag-filter'>
      <legend className='group-tag-legend'>{t('Group tags')}</legend>
      <div
        className={`group-tag-grid${choices.some((choice) => choice.icons?.length > 3) ? ' group-tag-grid-stacked' : ''}${props.compact ? ' group-tag-grid-compact' : ''}`}
      >
        {choices.map((choice) => {
          const option = (
            <label className='group-tag-choice' key={choice.id}>
              <input
                type='radio'
                name={name}
                value={choice.id}
                checked={active === choice.id}
                disabled={props.disabled || choice.count === 0}
                aria-haspopup={props.onActivate ? 'dialog' : undefined}
                aria-expanded={
                  props.onActivate
                    ? active === choice.id && !!props.expanded
                    : undefined
                }
                onChange={(event) => {
                  props.onChange(choice.id);
                  props.onActivate?.(choice.id, event);
                }}
                onClick={(event) => props.onActivate?.(choice.id, event)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' && props.onActivate) {
                    event.preventDefault();
                    props.onActivate(choice.id, event);
                  }
                }}
              />
              <span className='group-tag-card' title={choice.description}>
                <GroupTagLogos icons={choice.icons} />
                <span className='group-tag-name' title={choice.name}>
                  {choice.name}
                </span>
                <span className='group-tag-count'>
                  {t('Groups: {{count}}', { count: choice.count })}
                </span>
              </span>
            </label>
          );
          return props.renderChoice
            ? props.renderChoice(choice, option)
            : option;
        })}
      </div>
      {props.tags.find((tag) => String(tag.id) === active)?.description && (
        <p className='group-tag-description' aria-live='polite'>
          {props.tags.find((tag) => String(tag.id) === active).description}
        </p>
      )}
    </fieldset>
  );
}
