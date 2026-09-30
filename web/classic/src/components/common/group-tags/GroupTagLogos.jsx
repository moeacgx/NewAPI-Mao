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
import { IconApps } from '@douyinfe/semi-icons';
import { getLobeHubIcon } from '../../../helpers/render';
import './group-tags.css';

export default function GroupTagLogos(props) {
  const icons = props.icons || [];
  return (
    <span className='group-tag-logos' aria-hidden='true'>
      {icons.length === 0 && (
        <span className='group-tag-logo'>
          <IconApps size='large' />
        </span>
      )}
      {icons.map((icon, index) => (
        <span className='group-tag-logo' key={`${index}:${icon}`}>
          {getLobeHubIcon(icon, 24)}
        </span>
      ))}
    </span>
  );
}
