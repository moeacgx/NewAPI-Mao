import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import { API } from '../../../helpers';
import Affiliate from '../index';

const CUSTOM_TITLE = 'Custom <b>title</b>';
const CUSTOM_TEXT = 'Plain <b>body</b> text';
const DEFAULT_TITLE = '推广提示';
const DEFAULT_TEXT = '推广时请遵守返佣规则，滥用邀请可能会受到处罚。';

let notice = { enabled: true, title: CUSTOM_TITLE, text: CUSTOM_TEXT };

const balance = {
  user_id: 1,
  pending_quota: 0,
  available_quota: 0,
  frozen_quota: 0,
  risk_frozen_quota: 0,
  confiscated_quota: 0,
  withdrawn_quota: 0,
  transferred_quota: 0,
  total_quota: 0,
};

function responseFor(url) {
  if (url === '/api/affiliate/summary') {
    return {
      balance,
      aff_code: 'abcd',
      aff_count: 0,
      invite_link: 'https://example.test/register?aff=abcd',
      promotion_text: 'promo text',
      can_invite: true,
      promotion_notice: notice,
      setting: {
        first_level_enabled: true,
        first_level_ratio: 5,
        second_level_enabled: false,
        second_level_ratio: 0,
        payout_methods: ['usdt'],
        usdt_chain: 'TRC20',
      },
    };
  }
  if (url === '/api/affiliate/application-status') {
    return {
      review_enabled: false,
      agreement_enabled: false,
      status: 'approved',
      can_invite: true,
    };
  }
  if (url === '/api/affiliate/agreement') {
    return { agreement_enabled: false, agreement_text: '' };
  }
  if (url === '/api/affiliate/payout-account') return {};
  return { items: [], total: 0, page: 1, page_size: 20 };
}

function summaryCalls() {
  return API.get.mock.calls.filter(([url]) => url === '/api/affiliate/summary')
    .length;
}

// Semi Modal keeps hidden content in the DOM while it animates out, so a
// dismissed notice is one whose element is missing or hidden by an ancestor.
function isShown(text) {
  let node = screen.queryByText(text);
  if (!node) return false;
  while (node) {
    if (node.style && node.style.display === 'none') return false;
    if (String(node.className).includes('semi-modal-content-animate-hide')) {
      return false;
    }
    node = node.parentElement;
  }
  return true;
}

// Both modals keep their footer button mounted, so click the visible one.
function clickVisibleAcknowledge() {
  const visible = screen
    .getAllByRole('button', { name: '我已知悉' })
    .filter((button) => {
      for (let node = button; node; node = node.parentElement) {
        if (node.style && node.style.display === 'none') return false;
        if (
          String(node.className).includes('semi-modal-content-animate-hide')
        ) {
          return false;
        }
      }
      return true;
    });
  expect(visible).toHaveLength(1);
  fireEvent.click(visible[0]);
}

// jsdom never runs CSS animations, so emit the same animationend a browser
// fires when the exit animation of the modal that shows `text` finishes. Semi
// listens on both the content and the mask, so both are signalled.
function finishExitAnimation(text) {
  const target = screen.getByText(text);
  const wrapper = target.closest('.semi-modal-wrap');
  const content = target.closest('.semi-modal-content');
  const mask = wrapper?.parentElement?.querySelector('.semi-modal-mask');
  expect(content).not.toBeNull();
  fireEvent.animationEnd(content);
  if (mask) fireEvent.animationEnd(mask);
}

beforeEach(() => {
  notice = { enabled: true, title: CUSTOM_TITLE, text: CUSTOM_TEXT };
  vi.spyOn(API, 'get').mockImplementation(async (url) => ({
    data: { success: true, data: responseFor(url) },
  }));
});

describe('Classic 推广页提示弹窗', () => {
  test('开启时显示一次纯文本提示，关闭后刷新数据不会再次弹出', async () => {
    render(<Affiliate />);
    await waitFor(() => expect(isShown(CUSTOM_TITLE)).toBe(true));
    expect(isShown(CUSTOM_TEXT)).toBe(true);
    expect(document.querySelector('.semi-modal b')).toBeNull();

    fireEvent.click(screen.getByRole('button', { name: '我已知悉' }));
    await waitFor(() => expect(isShown(CUSTOM_TITLE)).toBe(false));

    const before = summaryCalls();
    fireEvent.click(screen.getByRole('button', { name: '刷新' }));
    await waitFor(() => expect(summaryCalls()).toBe(before + 1));
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(isShown(CUSTOM_TITLE)).toBe(false);
  });

  test('重新进入页面视为新的访问并再次弹出', async () => {
    const first = render(<Affiliate />);
    await waitFor(() => expect(isShown(CUSTOM_TITLE)).toBe(true));
    first.unmount();
    render(<Affiliate />);
    await waitFor(() => expect(isShown(CUSTOM_TITLE)).toBe(true));
  });

  test('开关关闭时不弹出', async () => {
    notice = { enabled: false, title: CUSTOM_TITLE, text: CUSTOM_TEXT };
    render(<Affiliate />);
    await waitFor(() => expect(summaryCalls()).toBeGreaterThan(0));
    await screen.findByText('推广文案');
    expect(isShown(CUSTOM_TITLE)).toBe(false);
  });

  test('标题和正文为空时使用默认文案', async () => {
    notice = { enabled: true, title: '', text: '' };
    render(<Affiliate />);
    await waitFor(() => expect(isShown(DEFAULT_TITLE)).toBe(true));
    expect(isShown(DEFAULT_TEXT)).toBe(true);
  });

  test('同时启用须知时先弹推广提示，关闭后才弹参与返佣须知，且任一时刻只有一个弹窗', async () => {
    const AGREEMENT = '参与须知正文 fixture';
    API.get.mockImplementation(async (url) => {
      const data = responseFor(url);
      if (url === '/api/affiliate/agreement') {
        return {
          data: {
            success: true,
            data: { agreement_enabled: true, agreement_text: AGREEMENT },
          },
        };
      }
      return { data: { success: true, data } };
    });
    render(<Affiliate />);
    await waitFor(() => expect(isShown(CUSTOM_TITLE)).toBe(true));
    expect(isShown(AGREEMENT)).toBe(false);

    clickVisibleAcknowledge();
    await waitFor(() => expect(isShown(CUSTOM_TITLE)).toBe(false));
    // The notice is still in its exit animation: the agreement modal must not
    // be mounted yet, so two animating modals never coexist.
    expect(screen.queryByText(AGREEMENT)).toBeNull();
    await new Promise((resolve) => setTimeout(resolve, 700));
    expect(screen.queryByText(AGREEMENT)).toBeNull();

    // Only the real end of the exit animation lets the agreement open.
    finishExitAnimation(CUSTOM_TITLE);
    await waitFor(() => expect(isShown(AGREEMENT)).toBe(true));
    expect(screen.queryByText(CUSTOM_TITLE)).toBeNull();
  });

  test('未满足邀请资格的访客同样弹出一次提示，申请门槛保持不变', async () => {
    API.get.mockImplementation(async (url) => {
      const data = responseFor(url);
      if (url === '/api/affiliate/application-status') {
        return {
          data: {
            success: true,
            data: { ...data, can_invite: false, status: 'none' },
          },
        };
      }
      if (url === '/api/affiliate/summary') {
        return {
          data: { success: true, data: { ...data, can_invite: false } },
        };
      }
      return { data: { success: true, data } };
    });
    render(<Affiliate />);
    await waitFor(() => expect(isShown(CUSTOM_TITLE)).toBe(true));
    fireEvent.click(screen.getByRole('button', { name: '我已知悉' }));
    await waitFor(() => expect(isShown(CUSTOM_TITLE)).toBe(false));
    expect(screen.queryByText('推广文案')).toBeNull();
  });

  test('汇总请求首次失败后不阻塞后续成功响应决定提示', async () => {
    let attempts = 0;
    API.get.mockImplementation(async (url) => {
      if (url === '/api/affiliate/summary') {
        attempts += 1;
        if (attempts === 1)
          throw new Error('Request failed with status code 429');
      }
      return { data: { success: true, data: responseFor(url) } };
    });
    render(<Affiliate />);
    await waitFor(() => expect(attempts).toBe(1));
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(isShown(CUSTOM_TITLE)).toBe(false);
    fireEvent.click(await screen.findByRole('button', { name: '刷新' }));
    await waitFor(() => expect(isShown(CUSTOM_TITLE)).toBe(true));
  });

  test('较早发出的旧汇总晚返回时不会覆盖新设置或锁定过期开关', async () => {
    const pending = [];
    API.get.mockImplementation((url) => {
      if (url !== '/api/affiliate/summary') {
        return Promise.resolve({
          data: { success: true, data: responseFor(url) },
        });
      }
      return new Promise((resolve) => {
        pending.push((data) => resolve({ data: { success: true, data } }));
      });
    });
    render(<Affiliate />);
    await waitFor(() => expect(pending).toHaveLength(1));
    fireEvent.click(await screen.findByRole('button', { name: '刷新' }));
    await waitFor(() => expect(pending).toHaveLength(2));
    const base = responseFor('/api/affiliate/summary');
    // Newer response first, stale response last; stale one must be ignored.
    pending[1]({
      ...base,
      promotion_notice: { enabled: true, title: 'Fresh title', text: 'fresh' },
    });
    await waitFor(() => expect(isShown('Fresh title')).toBe(true));
    fireEvent.click(screen.getByRole('button', { name: '我已知悉' }));
    await waitFor(() => expect(isShown('Fresh title')).toBe(false));
    pending[0]({
      ...base,
      promotion_notice: { enabled: false, title: 'Stale title', text: 'stale' },
    });
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(isShown('Stale title')).toBe(false);
    expect(isShown('Fresh title')).toBe(false);
  });
});
