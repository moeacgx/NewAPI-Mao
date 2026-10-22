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
import { Layers3 } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export default function SkillSummary(props) {
  const { t } = useTranslation();
  const template = props.template;
  const bundle = template.kind === 'bundle';
  return (
    <div className='prompt-skill-summary'>
      <div className='prompt-center-group-chips'>
        <span className='prompt-center-group-chip'>
          {bundle && <Layers3 size={12} aria-hidden='true' />}
          {bundle ? t('Skill bundle') : t('Single skill')}
        </span>
        <span className='prompt-center-group-chip'>
          {template.category_name || t('Uncategorized')}
        </span>
      </div>
      {bundle && (
        <details>
          <summary>
            {t('{{count}} skills', {
              count: template.skill_count || template.skills?.length || 0,
            })}
          </summary>
          <ol>
            {(template.skills || []).map((skill, index) => (
              <li key={index}>{skill.title}</li>
            ))}
          </ol>
        </details>
      )}
    </div>
  );
}
