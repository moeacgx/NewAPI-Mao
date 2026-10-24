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
import { Button } from '@douyinfe/semi-ui';
import { ArrowDown, ArrowUp, Plus, Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import SkillResourcesEditor from './SkillResourcesEditor';

export default function BundleSkillsEditor(props) {
  const { t } = useTranslation();
  const change = (index, key, value) =>
    props.onChange(
      props.skills.map((item, position) =>
        position === index ? { ...item, [key]: value } : item,
      ),
    );
  const move = (index, offset) => {
    const next = [...props.skills];
    [next[index], next[index + offset]] = [next[index + offset], next[index]];
    props.onChange(next);
  };
  return (
    <fieldset disabled={props.disabled} className='prompt-bundle-editor'>
      <legend>{t('Bundle skills')}</legend>
      <p className='prompt-center-muted'>
        {props.progressive
          ? t(
              'A bundle contains 2 to 128 independent skills and installs as one package.',
            )
          : t(
              'A bundle contains 2 to 16 text skills and installs as one item.',
            )}
      </p>
      {props.skills.map((skill, index) => (
        <section className='prompt-bundle-skill' key={skill.id || index}>
          <div className='prompt-center-row'>
            <strong>{t('Skill {{number}}', { number: index + 1 })}</strong>
            <Button
              icon={<ArrowUp size={14} />}
              disabled={props.disabled || index === 0}
              aria-label={t('Move skill {{number}} up', { number: index + 1 })}
              onClick={() => move(index, -1)}
            />
            <Button
              icon={<ArrowDown size={14} />}
              disabled={props.disabled || index === props.skills.length - 1}
              aria-label={t('Move skill {{number}} down', {
                number: index + 1,
              })}
              onClick={() => move(index, 1)}
            />
            <Button
              icon={<Trash2 size={14} />}
              disabled={props.disabled || props.skills.length <= 2}
              aria-label={t('Remove skill {{number}}', { number: index + 1 })}
              onClick={() =>
                props.onChange(
                  props.skills.filter((_, position) => position !== index),
                )
              }
            />
          </div>
          <label htmlFor={`${props.prefix}-skill-${index}-title`}>
            {t('Skill {{number}} title', { number: index + 1 })}
          </label>
          <input
            id={`${props.prefix}-skill-${index}-title`}
            required
            maxLength={80}
            value={skill.title}
            onChange={(event) => change(index, 'title', event.target.value)}
          />
          {props.progressive && (
            <>
              <label htmlFor={`${props.prefix}-skill-${index}-description`}>
                {t('Skill {{number}} description', { number: index + 1 })}
              </label>
              <textarea
                id={`${props.prefix}-skill-${index}-description`}
                rows={2}
                maxLength={512}
                value={skill.description || ''}
                onChange={(event) =>
                  change(index, 'description', event.target.value)
                }
              />
              <label htmlFor={`${props.prefix}-skill-${index}-path`}>
                {t('Skill {{number}} file path', { number: index + 1 })}
              </label>
              <input
                id={`${props.prefix}-skill-${index}-path`}
                value={skill.path || ''}
                placeholder='skills/review/SKILL.md'
                onChange={(event) => change(index, 'path', event.target.value)}
              />
            </>
          )}
          <label htmlFor={`${props.prefix}-skill-${index}-text`}>
            {t('Skill {{number}} text', { number: index + 1 })}
          </label>
          <textarea
            id={`${props.prefix}-skill-${index}-text`}
            required
            rows={5}
            value={skill.content || ''}
            onChange={(event) => change(index, 'content', event.target.value)}
          />
          {props.progressive && (
            <SkillResourcesEditor
              prefix={`${props.prefix}-skill-${index}`}
              skillNumber={index + 1}
              resources={skill.resources}
              disabled={props.disabled}
              onChange={(resources) => change(index, 'resources', resources)}
            />
          )}
        </section>
      ))}
      <Button
        icon={<Plus size={14} />}
        disabled={
          props.disabled ||
          props.skills.length >= (props.progressive ? 128 : 16)
        }
        onClick={() =>
          props.onChange([...props.skills, { title: '', content: '' }])
        }
      >
        {t('Add skill')}
      </Button>
    </fieldset>
  );
}
