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

import { afterEach, expect, test } from 'vitest';

import { applySiteMetadataFromStatus } from '../siteMetadata';

afterEach(() => {
  document.head.innerHTML = '';
});

test('服务端状态到达后更新 Classic 主标题和 SEO 元数据', () => {
  document.head.innerHTML = `
    <title>Server site name</title>
    <meta name="title" content="Server site name">
    <meta name="description" content="Old static description">
    <meta property="og:title" content="Server site name">
    <meta property="og:site_name" content="Server site name">
    <meta property="og:description" content="Old static description">
  `;

  applySiteMetadataFromStatus({
    system_name: 'Current site name',
    system_description: '当前站点描述',
  });

  expect(document.title).toBe('Current site name');
  expect(document.head.querySelector('meta[name="title"]').content).toBe(
    'Current site name',
  );
  expect(document.head.querySelector('meta[property="og:title"]').content).toBe(
    'Current site name',
  );
  expect(
    document.head.querySelector('meta[property="og:site_name"]').content,
  ).toBe('Current site name');
  expect(document.head.querySelector('meta[name="description"]').content).toBe(
    '当前站点描述',
  );
  expect(
    document.head.querySelector('meta[property="og:description"]').content,
  ).toBe('当前站点描述');
});

test('描述清空后移除 description 与 Open Graph 描述但保留主标题', () => {
  document.head.innerHTML = `
    <title>Current site name</title>
    <meta name="description" content="Old static description">
    <meta property="og:description" content="Old static description">
  `;

  applySiteMetadataFromStatus({
    system_name: 'Current site name',
    system_description: '',
  });

  expect(document.title).toBe('Current site name');
  expect(document.head.querySelector('meta[name="description"]')).toBeNull();
  expect(
    document.head.querySelector('meta[property="og:description"]'),
  ).toBeNull();
});
