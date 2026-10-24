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
import { FileText, Plus, Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export default function SkillResourcesEditor(props) {
  const { t } = useTranslation();
  const resources = props.resources || [];
  const change = (index, key, value) =>
    props.onChange(
      resources.map((resource, position) =>
        position === index ? { ...resource, [key]: value } : resource,
      ),
    );
  return (
    <fieldset className='prompt-resource-editor' disabled={props.disabled}>
      <legend>
        <FileText size={14} aria-hidden='true' /> {t('Text resources')}
      </legend>
      <p className='prompt-center-muted'>
        {t(
          'Resources are read individually when needed. Scripts are not executed.',
        )}
      </p>
      {resources.map((resource, index) => (
        <section className='prompt-bundle-skill' key={index}>
          <label htmlFor={`${props.prefix}-resource-${index}-path`}>
            {props.skillNumber
              ? t('Skill {{skill}} resource {{number}} path', {
                  skill: props.skillNumber,
                  number: index + 1,
                })
              : t('Resource {{number}} path', { number: index + 1 })}
          </label>
          <input
            id={`${props.prefix}-resource-${index}-path`}
            required
            value={resource.path}
            placeholder='references/examples.md'
            onChange={(event) => change(index, 'path', event.target.value)}
          />
          <label htmlFor={`${props.prefix}-resource-${index}-content`}>
            {props.skillNumber
              ? t('Skill {{skill}} resource {{number}} text', {
                  skill: props.skillNumber,
                  number: index + 1,
                })
              : t('Resource {{number}} text', { number: index + 1 })}
          </label>
          <textarea
            id={`${props.prefix}-resource-${index}-content`}
            rows={5}
            value={resource.content || ''}
            onChange={(event) => change(index, 'content', event.target.value)}
          />
          <Button
            icon={<Trash2 size={14} />}
            disabled={props.disabled}
            aria-label={
              props.skillNumber
                ? t('Remove skill {{skill}} resource {{number}}', {
                    skill: props.skillNumber,
                    number: index + 1,
                  })
                : t('Remove resource {{number}}', { number: index + 1 })
            }
            onClick={() =>
              props.onChange(
                resources.filter((_, position) => position !== index),
              )
            }
          >
            {t('Remove resource')}
          </Button>
        </section>
      ))}
      <Button
        icon={<Plus size={14} />}
        disabled={props.disabled}
        onClick={() =>
          props.onChange([...resources, { path: '', content: '' }])
        }
      >
        {t('Add text resource')}
      </Button>
    </fieldset>
  );
}
