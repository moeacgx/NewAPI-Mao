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
import { describe, expect, it, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import PricingTemplateHeader from '../PricingTemplateHeader';

const links = [
  { itemKey: 'home', to: '/', text: '主页' },
  { itemKey: 'console', to: '/console', text: '控制台' },
  { itemKey: 'pricing', to: '/pricing', text: '模型广场' },
  {
    itemKey: 'private',
    to: '/extensions/private',
    text: '需要登录的扩展',
    requiresAuth: true,
  },
  {
    itemKey: 'docs',
    text: '图片异步生成文档与渠道可观测性使用说明',
    isExternal: true,
    externalLink: 'https://example.invalid/docs',
  },
];

function renderHeader(user = { username: 'admin', role: 10 }) {
  const onNoticeOpen = vi.fn();
  render(
    <PricingTemplateHeader
      systemName='本地测试站点'
      logoLoaded={false}
      mainNavLinks={links}
      userState={{ user }}
      pricingRequireAuth
      unreadCount={2}
      onNoticeOpen={onNoticeOpen}
      theme='light'
      onThemeToggle={vi.fn()}
      currentLang='zh'
      onLanguageChange={vi.fn()}
      isLoading={false}
      logout={vi.fn()}
      t={(key) => key}
    />,
  );
  return { header: within(screen.getByRole('banner')), onNoticeOpen };
}

describe('Classic 长导航与操作的行为契约', () => {
  it('保留长链接、外链保护、正常 Tab 顺序和可操作的通知入口', async () => {
    const user = userEvent.setup();
    const { header, onNoticeOpen } = renderHeader();
    const longLink = header.getByRole('link', { name: links[4].text });
    expect(longLink.getAttribute('href')).toBe(links[4].externalLink);
    expect(longLink.getAttribute('target')).toBe('_blank');
    expect(longLink.getAttribute('rel')).toBe('noopener noreferrer');
    header.getByRole('link', { name: '需要登录的扩展' }).focus();
    await user.tab();
    expect(document.activeElement).toBe(longLink);
    await user.tab();
    expect(document.activeElement.tagName).toBe('BUTTON');
    await user.click(header.getByRole('button', { name: '系统公告' }));
    expect(onNoticeOpen).toHaveBeenCalledOnce();
    expect(header.getByRole('button', { name: /admin/ })).toBeTruthy();
  });

  it('访客的后台、模型广场和受保护自定义入口继续转向登录', () => {
    const { header } = renderHeader(null);
    for (const name of ['控制台', '模型广场', '需要登录的扩展']) {
      expect(header.getByRole('link', { name }).getAttribute('href')).toBe(
        '/login',
      );
    }
    expect(
      header.getByRole('link', { name: links[4].text }).getAttribute('href'),
    ).toBe(links[4].externalLink);
    expect(header.getByRole('link', { name: '登录' })).toBeTruthy();
  });

  it('移动菜单支持键盘展开、完整长链接和选择后关闭', async () => {
    const user = userEvent.setup();
    renderHeader();
    const toggle = screen.getByRole('button', { name: '切换导航菜单' });
    toggle.focus();
    await user.keyboard('{Enter}');
    expect(toggle.getAttribute('aria-expanded')).toBe('true');
    const menu = document.getElementById(toggle.getAttribute('aria-controls'));
    expect(menu.getAttribute('aria-hidden')).toBe('false');
    expect(document.body.style.overflow).toBe('hidden');
    const longLink = within(menu).getByRole('link', { name: links[4].text });
    // 阻止 jsdom 实际导航，保留真实链接事件与组件关闭逻辑。
    longLink.addEventListener('click', (event) => event.preventDefault());
    longLink.focus();
    await user.keyboard('{Enter}');
    expect(toggle.getAttribute('aria-expanded')).toBe('false');
    expect(menu.getAttribute('aria-hidden')).toBe('true');
    expect(document.body.style.overflow).not.toBe('hidden');
  });
});
