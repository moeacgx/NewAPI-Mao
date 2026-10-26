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

import React, { useState } from 'react';
import { Tooltip } from '@douyinfe/semi-ui';
import { Sparkles } from 'lucide-react';
import {
  describeLogSkillUsage,
  getLogSkillUsage,
  renderGroup,
} from '../../../helpers';
import './skill-usage-group.css';

export default function SkillUsageGroup({ group, other, t }) {
  const [hovered, setHovered] = useState(false);
  const [focused, setFocused] = useState(false);
  const usage = getLogSkillUsage(other);
  const groupTag = typeof group === 'string' ? renderGroup(group) : null;
  if (!usage) return groupTag;
  const description = describeLogSkillUsage(usage, t);

  return (
    <span className='usage-log-skill-group'>
      {groupTag || <span>—</span>}
      <Tooltip
        trigger='custom'
        visible={hovered || focused}
        content={
          <div className='usage-log-skill-details'>
            <strong>{description.label}</strong>
            {description.lines.map((line) => (
              <div key={line}>{line}</div>
            ))}
          </div>
        }
      >
        <button
          type='button'
          className={`usage-log-skill-badge${usage.kind === 'used' ? ' usage-log-skill-badge-used' : ''}`}
          aria-label={[description.label, ...description.lines].join('；')}
          onMouseEnter={() => setHovered(true)}
          onMouseLeave={() => setHovered(false)}
          onFocus={() => setFocused(true)}
          onBlur={() => setFocused(false)}
          onClick={(event) => event.stopPropagation()}
          onKeyDown={(event) => {
            if (event.key === 'Escape') {
              setHovered(false);
              setFocused(false);
              event.stopPropagation();
            }
          }}
        >
          <Sparkles size={11} strokeWidth={2.2} aria-hidden='true' />
        </button>
      </Tooltip>
    </span>
  );
}
