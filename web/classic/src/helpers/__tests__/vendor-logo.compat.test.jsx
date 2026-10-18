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

import { describe, expect, it } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { getLobeHubIcon } from '../render';

describe('供应商图标', () => {
  it.each([
    'https://cdn.example.com/typesafe.svg',
    'http://cdn.example.com/jev.png',
  ])('图片地址 %s 保持尺寸并隐藏来源页面', (url) => {
    render(getLobeHubIcon(`  ${url}  `, 32));
    const img = screen.getByRole('presentation');
    expect(img.tagName).toBe('IMG');
    expect(img.getAttribute('src')).toBe(url);
    expect(img.getAttribute('width')).toBe('32');
    expect(img.getAttribute('height')).toBe('32');
    expect(img.style.objectFit).toBe('contain');
    expect(img.style.width).toBe('32px');
    expect(img.style.height).toBe('32px');
    expect(img.getAttribute('referrerpolicy')).toBe('no-referrer');
  });
  it('加载失败显示兜底且更换地址可以重新加载', () => {
    const { rerender } = render(
      getLobeHubIcon('https://cdn.example.com/missing.png'),
    );
    fireEvent.error(screen.getByRole('presentation'));
    expect(screen.queryByRole('presentation')).toBeNull();
    expect(screen.getByText('?')).toBeTruthy();
    rerender(getLobeHubIcon('https://cdn.example.com/valid.png'));
    expect(screen.getByRole('presentation').getAttribute('src')).toBe(
      'https://cdn.example.com/valid.png',
    );
  });
  it.each([
    'javascript:alert(1)',
    'data:image/svg+xml,<svg/>',
    '//cdn.example.com/logo.png',
    'https://',
    'https://user:secret@cdn.example.com/logo.png',
    '',
  ])('输入 %s 不创建图片请求', (input) => {
    const { container } = render(getLobeHubIcon(input));
    expect(container.querySelector('img')).toBeNull();
  });
  it.each(['OpenAI', 'Claude.Color', "OpenAI.Avatar.type={'platform'}"])(
    '内置图标 %s 仍渲染品牌图形',
    (input) => {
      const { container } = render(getLobeHubIcon(input, 24));
      expect(container.querySelector('svg')).not.toBeNull();
      expect(container.querySelector('img')).toBeNull();
    },
  );
});
