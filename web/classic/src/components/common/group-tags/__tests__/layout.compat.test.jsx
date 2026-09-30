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
import { readFileSync } from 'node:fs';
import { URL as NodeURL } from 'node:url';
import { afterEach, describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import GroupTagFilter from '../GroupTagFilter';

const styles = readFileSync(
  new NodeURL('../group-tags.css', import.meta.url),
  'utf8',
);

let stylesheet;
afterEach(() => stylesheet?.remove());

describe('标签卡片对齐布局', () => {
  it.each([1, 6])(
    '%s 个 Logo 时默认与自定义标签使用等高图标、名称槽',
    (count) => {
      stylesheet = document.createElement('style');
      stylesheet.textContent = styles;
      document.head.appendChild(stylesheet);
      const name = '多平台联合服务与超长名称布局验证';
      const { container } = render(
        <GroupTagFilter
          tags={[
            {
              id: 1,
              name,
              icons: [
                'OpenAI',
                'Claude.Color',
                'DeepSeek.Color',
                'Moonshot',
                'Gemini.Color',
                'XAI',
              ].slice(0, count),
              group_ids: [11],
            },
          ]}
          groups={[{ id: 11, value: 'internal', label: '当前显示名称' }]}
          value='all'
          onChange={() => {}}
        />,
      );
      const logos = [
        ...container.querySelectorAll('.group-tag-card > .group-tag-logos'),
      ];
      expect(logos.map((node) => getComputedStyle(node).height)).toEqual(
        Array(3).fill(count > 3 ? '66px' : '30px'),
      );
      expect(logos[0].querySelector('.group-tag-logo')).not.toBeNull();
      for (const node of container.querySelectorAll('.group-tag-name')) {
        expect(getComputedStyle(node).height).toBe('40px');
        expect(getComputedStyle(node).overflow).toBe('hidden');
      }
      expect(screen.getByTitle(name).textContent).toBe(name);
      expect(
        screen.getByRole('radio', { name: new RegExp(name) }),
      ).toBeTruthy();
    },
  );
});
