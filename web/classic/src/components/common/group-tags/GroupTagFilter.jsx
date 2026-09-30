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

import React, { useId } from 'react';
import { useTranslation } from 'react-i18next';
import GroupTagLogos from './GroupTagLogos';
import { filterGroupsByTag } from '../../../helpers/groupTags';

export default function GroupTagFilter(props) {
  const { t } = useTranslation();
  const name = useId();
  if (props.tags.length === 0) return null;
  const choices = [
    { id: 'all', name: t('All groups'), icons: [] },
    { id: 'untagged', name: t('Untagged'), icons: [] },
    ...props.tags.map((tag) => ({ ...tag, id: String(tag.id) })),
  ];
  const active = choices.some((choice) => choice.id === props.value)
    ? props.value
    : 'all';

  return (
    <fieldset className='group-tag-filter'>
      <legend className='group-tag-legend'>{t('Group tags')}</legend>
      <div
        className={`group-tag-grid${choices.some((choice) => choice.icons?.length > 3) ? ' group-tag-grid-stacked' : ''}`}
      >
        {choices.map((choice) => {
          const count = filterGroupsByTag(
            props.groups,
            props.tags,
            choice.id,
          ).length;
          return (
            <label className='group-tag-choice' key={choice.id}>
              <input
                type='radio'
                name={name}
                value={choice.id}
                checked={active === choice.id}
                disabled={
                  props.disabled || (count === 0 && choice.id !== 'all')
                }
                onChange={() => props.onChange(choice.id)}
              />
              <span className='group-tag-card' title={choice.description}>
                <GroupTagLogos icons={choice.icons} />
                <span className='group-tag-name' title={choice.name}>
                  {choice.name}
                </span>
                <span className='group-tag-count'>
                  {t('Groups: {{count}}', { count })}
                </span>
              </span>
            </label>
          );
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
