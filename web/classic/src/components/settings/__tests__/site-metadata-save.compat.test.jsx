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

import { Toast } from '@douyinfe/semi-ui';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useReducer } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, expect, test, vi } from 'vitest';

import { StatusContext } from '../../../context/Status';
import { reducer } from '../../../context/Status/reducer';
import { UserProvider } from '../../../context/User';
import { API, setStatusData } from '../../../helpers';
import { useHeaderBar } from '../../../hooks/common/useHeaderBar';
import OtherSetting from '../OtherSetting';

afterEach(() => {
  document.head.innerHTML = '';
});

const initialStatus = {
  system_name: 'Configured site',
  system_description: 'Initial description',
  logo: '/logo.png',
  quota_per_unit: 500000,
};

function HeaderConsumer() {
  const { systemName, statusState } = useHeaderBar({});
  return (
    <>
      <h1>{systemName}</h1>
      <output aria-label='Status description'>
        {statusState.status.system_description}
      </output>
    </>
  );
}

function SettingsFixture() {
  const status = useReducer(reducer, { status: initialStatus });
  return (
    <MemoryRouter>
      <UserProvider>
        <StatusContext.Provider value={status}>
          <HeaderConsumer />
          <OtherSetting />
        </StatusContext.Provider>
      </UserProvider>
    </MemoryRouter>
  );
}

test.each([
  {
    key: 'SystemName',
    label: '系统名称',
    button: '设置系统名称',
    value: 'Updated site',
    expectedTitle: 'Updated site',
    expectedDescription: 'Initial description',
  },
  {
    key: 'SystemDescription',
    label: 'Website Description',
    button: 'Save website description',
    value: 'Updated description',
    expectedTitle: 'Configured site',
    expectedDescription: 'Updated description',
  },
  {
    key: 'SystemDescription',
    label: 'Website Description',
    button: 'Save website description',
    value: '',
    expectedTitle: 'Configured site',
    expectedDescription: null,
  },
])(
  '业务拒绝 $key=$value 时不提示成功，保留元数据并允许重试',
  async (scenario) => {
    vi.spyOn(API, 'get').mockResolvedValue({
      data: {
        success: true,
        data: [
          { key: 'SystemName', value: 'Configured site' },
          { key: 'SystemDescription', value: 'Initial description' },
        ],
      },
    });
    const put = vi
      .spyOn(API, 'put')
      .mockResolvedValueOnce({ data: { success: false, message: 'Rejected' } })
      .mockResolvedValueOnce({ data: { success: true } });
    const successToast = vi.spyOn(Toast, 'success');
    const errorToast = vi.spyOn(Toast, 'error');
    document.head.innerHTML = `
    <title>Configured site</title>
    <meta name="description" content="Initial description">
    <meta property="og:description" content="Initial description">
  `;
    setStatusData(initialStatus);
    render(<SettingsFixture />);

    const input = screen.getByRole('textbox', { name: scenario.label });
    await waitFor(() =>
      expect(input.value).toBe(
        scenario.key === 'SystemName'
          ? 'Configured site'
          : 'Initial description',
      ),
    );
    await userEvent.clear(input);
    if (scenario.value) await userEvent.type(input, scenario.value);
    await userEvent.click(
      screen.getByRole('button', { name: scenario.button }),
    );

    await waitFor(() =>
      expect(errorToast).toHaveBeenCalledWith('错误：Rejected'),
    );
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: scenario.button }).disabled,
      ).toBe(false),
    );
    expect(errorToast).toHaveBeenCalledTimes(1);
    expect(successToast).not.toHaveBeenCalled();
    expect(input.value).toBe(scenario.value);
    expect(document.title).toBe('Configured site');
    expect(screen.getByRole('heading', { level: 1 }).textContent).toBe(
      'Configured site',
    );
    expect(
      screen.getByRole('status', { name: 'Status description' }).textContent,
    ).toBe('Initial description');
    expect(JSON.parse(localStorage.getItem('status'))).toEqual(initialStatus);
    for (const selector of [
      'meta[name="description"]',
      'meta[property="og:description"]',
    ]) {
      expect(
        document.head.querySelector(selector)?.getAttribute('content'),
      ).toBe('Initial description');
    }

    await userEvent.click(
      screen.getByRole('button', { name: scenario.button }),
    );

    await waitFor(() => expect(successToast).toHaveBeenCalledTimes(1));
    expect(put).toHaveBeenCalledTimes(2);
    expect(put).toHaveBeenNthCalledWith(2, '/api/option/', {
      key: scenario.key,
      value: scenario.value,
    });
    expect(document.title).toBe(scenario.expectedTitle);
    await waitFor(() => {
      expect(screen.getByRole('heading', { level: 1 }).textContent).toBe(
        scenario.expectedTitle,
      );
      expect(
        screen.getByRole('status', { name: 'Status description' }).textContent,
      ).toBe(scenario.expectedDescription ?? '');
    });
    expect(JSON.parse(localStorage.getItem('status'))).toEqual({
      ...initialStatus,
      system_name: scenario.expectedTitle,
      system_description: scenario.expectedDescription ?? '',
    });
    expect(localStorage.getItem('system_name')).toBe(scenario.expectedTitle);
    expect(API.get).toHaveBeenCalledTimes(1);
    expect(API.get).toHaveBeenCalledWith('/api/option/');
    for (const selector of [
      'meta[name="description"]',
      'meta[property="og:description"]',
    ]) {
      const meta = document.head.querySelector(selector);
      if (scenario.expectedDescription === null) {
        expect(meta).toBeNull();
      } else {
        expect(meta?.getAttribute('content')).toBe(
          scenario.expectedDescription,
        );
      }
    }
  },
);
