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
import { expect, test } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import EditChannelModal from '../EditChannelModal';
import { API } from '../../../../../helpers/api';

function installChannelFixture({
  multi = false,
  strategy = 'random',
  allowed = true,
} = {}) {
  const adapter = API.defaults.adapter;
  const writes = [];
  API.defaults.adapter = async (config) => {
    let data = [];
    if (config.method === 'put') writes.push(JSON.parse(config.data));
    if (config.url === '/api/user/self') {
      data = {
        role: 100,
        permissions: {
          admin_permissions: { channel: { sensitive_write: allowed } },
        },
      };
    }
    if (config.url === '/api/channel/42') {
      data = {
        id: 42,
        name: '立即提交夹具',
        type: 1,
        models: 'gpt-4',
        group: 'default',
        model_mapping: '',
        status_code_mapping: '',
        key: '',
        setting: '{}',
        other: '{}',
        channel_info: { is_multi_key: multi, multi_key_mode: strategy },
      };
    }
    return {
      data: { success: true, data },
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
    };
  };
  return {
    writes,
    restore: () => {
      API.defaults.adapter = adapter;
    },
  };
}

function ChannelEditor() {
  const [visible, setVisible] = useState(true);
  return (
    <>
      <button onClick={() => setVisible(true)}>重新打开渠道</button>
      <EditChannelModal
        visible={visible}
        editingChannel={{ id: 42 }}
        handleClose={() => setVisible(false)}
        refresh={() => {}}
      />
    </>
  );
}

function selectImmediately(field, label, keyboard = false) {
  const select = document.getElementById(field).closest('.semi-select');
  if (keyboard) {
    fireEvent.keyPress(select, { key: 'Enter', keyCode: 13, charCode: 13 });
    // Semi 展开时的焦点不一定是当前选中值，按真实活动选项导航。
    for (let step = 0; step < 2; step++) {
      const active = document.getElementById(
        select.getAttribute('aria-activedescendant'),
      );
      if (active?.textContent.includes(label)) break;
      fireEvent.keyDown(select, { key: 'ArrowDown', keyCode: 40 });
    }
    expect(
      document.getElementById(select.getAttribute('aria-activedescendant'))
        .textContent,
    ).toContain(label);
    fireEvent.keyDown(select, { key: 'Enter', keyCode: 13 });
  } else {
    fireEvent.click(select);
    fireEvent.click(screen.getByText(label, { exact: true }));
  }
}

for (const scenario of [
  {
    name: '单密钥转轮询',
    multi: false,
    strategy: 'random',
    label: '轮询',
    expected: 'polling',
  },
  {
    name: '已有聚合轮询改随机',
    multi: true,
    strategy: 'polling',
    label: '随机',
    expected: 'random',
  },
  {
    name: '键盘随机改轮询',
    multi: true,
    strategy: 'random',
    label: '轮询',
    expected: 'polling',
    keyboard: true,
  },
  {
    name: '键盘轮询改随机',
    multi: true,
    strategy: 'polling',
    label: '随机',
    expected: 'random',
    keyboard: true,
  },
]) {
  test(`${scenario.name}后立即保存采用选中策略并保留旧密钥`, async () => {
    const fixture = installChannelFixture(scenario);
    try {
      render(<ChannelEditor />);
      await screen.findByDisplayValue('立即提交夹具');
      if (!scenario.multi) {
        fireEvent.click(screen.getByRole('checkbox', { name: '密钥聚合模式' }));
      }
      selectImmediately('multi_key_mode', scenario.label, scenario.keyboard);
      // 不等待下拉退出或控件文本更新，直接走真实表单提交链。
      fireEvent.click(screen.getByRole('button', { name: /提交/ }));
      await waitFor(() => expect(fixture.writes).toHaveLength(1));
      expect(fixture.writes[0].multi_key_mode).toBe(scenario.expected);
      expect(fixture.writes[0].key).toBeUndefined();
      expect(fixture.writes[0].key_mode).toBeUndefined();
      expect(fixture.writes[0].convert_to_multi_key).toBe(
        scenario.multi ? undefined : true,
      );
    } finally {
      fixture.restore();
    }
  });
}

for (const mode of ['replace', 'append']) {
  test(`选择${mode}后立即保存保留准确的密钥更新方式`, async () => {
    const fixture = installChannelFixture({ multi: true });
    try {
      render(<ChannelEditor />);
      await screen.findByDisplayValue('立即提交夹具');
      fireEvent.change(document.getElementById('key'), {
        target: { value: 'fixture-new-key' },
      });
      selectImmediately('key_mode', '覆盖现有密钥');
      if (mode === 'append') selectImmediately('key_mode', '追加到现有密钥');
      fireEvent.click(screen.getByRole('button', { name: /提交/ }));
      await waitFor(() => expect(fixture.writes).toHaveLength(1));
      expect(fixture.writes[0].key_mode).toBe(mode);
      expect(fixture.writes[0].key).toBe('fixture-new-key');
    } finally {
      fixture.restore();
    }
  });
}

test('选择策略后取消重开不会保留转换或模式', async () => {
  const fixture = installChannelFixture();
  try {
    render(<ChannelEditor />);
    await screen.findByDisplayValue('立即提交夹具');
    fireEvent.click(screen.getByRole('checkbox', { name: '密钥聚合模式' }));
    selectImmediately('multi_key_mode', '轮询');
    fireEvent.click(screen.getByRole('button', { name: /取消/ }));
    fireEvent.click(screen.getByRole('button', { name: '重新打开渠道' }));
    await screen.findByDisplayValue('立即提交夹具');
    expect(screen.getByRole('checkbox', { name: '密钥聚合模式' }).checked).toBe(
      false,
    );
    fireEvent.click(screen.getByRole('checkbox', { name: '密钥聚合模式' }));
    fireEvent.click(screen.getByRole('button', { name: /提交/ }));
    await waitFor(() => expect(fixture.writes).toHaveLength(1));
    expect(fixture.writes[0].multi_key_mode).toBe('random');
    expect(fixture.writes[0].key_mode).toBeUndefined();
    expect(fixture.writes[0].key).toBeUndefined();
  } finally {
    fixture.restore();
  }
});

test('没有敏感写权限时不能选择策略且不提交敏感字段', async () => {
  const fixture = installChannelFixture({ multi: true, allowed: false });
  try {
    render(<ChannelEditor />);
    await screen.findByDisplayValue('立即提交夹具');
    for (const field of ['multi_key_mode', 'key_mode']) {
      expect(
        document
          .getElementById(field)
          .closest('.semi-select')
          .getAttribute('aria-disabled'),
      ).toBe('true');
    }
    fireEvent.click(screen.getByRole('button', { name: /提交/ }));
    await waitFor(() => expect(fixture.writes).toHaveLength(1));
    for (const field of [
      'key',
      'key_mode',
      'multi_key_mode',
      'convert_to_multi_key',
    ]) {
      expect(fixture.writes[0][field]).toBeUndefined();
    }
  } finally {
    fixture.restore();
  }
});
