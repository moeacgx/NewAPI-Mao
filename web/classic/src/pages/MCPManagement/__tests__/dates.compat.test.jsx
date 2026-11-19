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
import { act, render, screen, within } from '@testing-library/react';
import { afterEach, beforeEach, expect, test } from 'vitest';
import { createInstance } from 'i18next';
import { I18nextProvider, initReactI18next } from 'react-i18next';
import { API } from '../../../helpers/api';
import en from '../../../i18n/locales/en.json';
import zhCN from '../../../i18n/locales/zh-CN.json';
import { formatTime } from '../api';
import Keys from '../Keys';
import Logs from '../Logs';

const originalAdapter = API.defaults.adapter;
const timestamp = 1700000000;
let language;

beforeEach(async () => {
  language = createInstance();
  await language.use(initReactI18next).init({
    lng: 'zh-CN',
    fallbackLng: 'en',
    resources: { en, 'zh-CN': zhCN },
    interpolation: { escapeValue: false },
  });
  API.defaults.adapter = async (config) => {
    let data;
    if (config.url === '/api/mcp/tools') data = [];
    else if (config.url.startsWith('/api/mcp/keys?')) {
      data = {
        items: [
          {
            id: 1,
            name: 'Laptop',
            tools: [],
            created_at: timestamp,
            expires_at: 253402300799,
            last_used_at: timestamp + 10,
            revoked_at: 0,
          },
        ],
        total: 1,
      };
    } else if (config.url.startsWith('/api/mcp/logs?')) {
      data = {
        items: [
          {
            id: 1,
            tool_name: 'users_list',
            created_at: timestamp,
            success: true,
          },
        ],
        total: 1,
      };
    } else throw new Error(`Unexpected request: ${config.url}`);
    return {
      data: { success: true, data },
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
    };
  };
});

afterEach(() => {
  API.defaults.adapter = originalAdapter;
});

test.each([
  ['密钥', Keys, [timestamp, 253402300799, timestamp + 10]],
  ['日志', Logs, [timestamp]],
])(
  '%s页面切换站点中英文后，已加载日期即时按当前语言展示',
  async (_name, Component, timestamps) => {
    render(
      <I18nextProvider i18n={language}>
        <Component />
      </I18nextProvider>,
    );
    const table = await screen.findByRole('table');
    const chinese = timestamps.map((value) =>
      new Date(value * 1000).toLocaleString('zh-CN'),
    );
    const english = timestamps.map((value) =>
      new Date(value * 1000).toLocaleString('en'),
    );
    expect(chinese).not.toEqual(english);
    for (const text of chinese)
      expect(within(table).getByText(text)).toBeTruthy();

    await act(async () => {
      await language.changeLanguage('en');
    });
    for (const text of english)
      expect(within(table).getByText(text)).toBeTruthy();
    for (const text of chinese)
      expect(within(table).queryByText(text)).toBeNull();

    await act(async () => {
      await language.changeLanguage('zhCN');
    });
    for (const text of chinese)
      expect(within(table).getByText(text)).toBeTruthy();
  },
);

test.each([
  ['zh-CN', 'zh-CN'],
  ['zhCN', 'zh-CN'],
  ['zhTW', 'zh-TW'],
  ['en', 'en'],
  ['invalid_locale_!', 'zh-CN'],
  [undefined, 'zh-CN'],
])('时间格式化按语言 %s 及已有映射安全显示', (locale, expectedLocale) => {
  expect(formatTime(timestamp, locale)).toBe(
    new Date(timestamp * 1000).toLocaleString(expectedLocale),
  );
});

test.each([
  0,
  -1,
  undefined,
  null,
  '1700000000',
  NaN,
  Infinity,
  -Infinity,
  Number.MAX_VALUE,
  8640000000001,
])('历史异常时间 %s 回退短横线，不输出 Invalid Date', (seconds) => {
  expect(formatTime(seconds, 'en')).toBe('-');
});

test.each([253402300799, 8640000000000])(
  '有效上界时间 %s 仍可格式化',
  (seconds) => {
    expect(formatTime(seconds, 'en')).toBe(
      new Date(seconds * 1000).toLocaleString('en'),
    );
  },
);
