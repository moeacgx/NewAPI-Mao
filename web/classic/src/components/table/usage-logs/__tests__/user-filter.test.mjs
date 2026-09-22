import assert from 'node:assert/strict';
import fs from 'node:fs';
import { createRequire } from 'node:module';
import test from 'node:test';
import vm from 'node:vm';

import { transformSync } from 'esbuild';
import React from 'react';
import { act, create } from 'react-test-renderer';

const requireFromTest = createRequire(import.meta.url);
const hookSource = fs.readFileSync(
  new URL('../../../../hooks/usage-logs/useUsageLogsData.jsx', import.meta.url),
  'utf8',
);
const hookCode = transformSync(hookSource, {
  loader: 'jsx',
  format: 'cjs',
  platform: 'node',
}).code;

const localStorageData = new Map();
globalThis.localStorage = {
  getItem: (key) => localStorageData.get(key) ?? null,
  setItem: (key, value) => localStorageData.set(key, String(value)),
  removeItem: (key) => localStorageData.delete(key),
};

let adminMode = true;

function loadUseLogsData(api, errors) {
  const helpers = {
    API: api,
    getTodayStartTimestamp: () => 0,
    isAdmin: () => adminMode,
    showError: (message) => errors.push(message),
    showSuccess: () => {},
    timestamp2string: (value) => String(value),
    renderQuota: (value) => String(value),
    renderNumber: (value) => String(value),
    getLogOther: (value) => value || {},
    copy: async () => true,
    renderClaudeLogContent: () => '',
    renderLogContent: () => '',
    renderAudioModelPrice: () => '',
    renderClaudeModelPrice: () => '',
    renderModelPrice: () => '',
    renderTieredModelPrice: () => '',
    renderTaskBillingProcess: () => '',
  };
  const module = { exports: {} };
  const mockedRequire = (specifier) => {
    if (specifier === '../../helpers') return helpers;
    if (specifier === '../../constants') return { ITEMS_PER_PAGE: 20 };
    if (specifier === '../common/useTableCompactMode') {
      return { useTableCompactMode: () => [false, () => {}] };
    }
    if (
      specifier ===
      '../../components/table/usage-logs/components/ParamOverrideEntry'
    ) {
      return { default: () => null };
    }
    if (
      specifier ===
      '../../components/table/usage-logs/modals/channel-affinity-usage-cache'
    ) {
      return { buildChannelAffinityUsageCacheTarget: (value) => value };
    }
    if (specifier === '../../components/table/usage-logs/login-log-presenter') {
      return { getLoginLogDetailItems: () => [], LOG_TYPE_LOGIN: 7 };
    }
    if (specifier === 'react-i18next') {
      return { useTranslation: () => ({ t: (value) => value }) };
    }
    if (specifier === '@douyinfe/semi-ui') {
      return { Modal: { error: () => {} } };
    }
    return requireFromTest(specifier);
  };
  const script = new vm.Script(
    `(function (exports, require, module) {${hookCode}\n})`,
  );
  script.runInThisContext()(module.exports, mockedRequire, module);
  return module.exports.useLogsData;
}

function successList(items = []) {
  return {
    data: {
      success: true,
      data: { items, page: 1, page_size: 20, total: items.length },
    },
  };
}

function successStat(stat = { quota: 0, token: 0 }) {
  return { data: { success: true, data: stat } };
}

function createApiMock(errors) {
  const calls = [];
  const queue = [];
  const api = {
    get: (url, config) => {
      calls.push({ url, config });
      const next = queue.shift();
      if (next?.reject) {
        if (next.toast) errors.push(next.message || '请求失败');
        return Promise.reject(next.reject);
      }
      if (next) return Promise.resolve(next);
      return Promise.resolve(
        url.includes('/stat') ? successStat() : successList(),
      );
    },
  };
  return { api, calls, queue };
}

async function settle() {
  await act(async () => {
    await Promise.resolve();
  });
}

async function mountHook(api, errors) {
  const useLogsData = loadUseLogsData(api, errors);
  let result;
  function Harness() {
    result = useLogsData();
    return null;
  }
  let renderer;
  await act(async () => {
    renderer = create(React.createElement(Harness));
    await Promise.resolve();
  });
  await settle();
  return { result: () => result, renderer };
}

async function setForm(result, values) {
  await act(async () => {
    result().setFormApi({ getValues: () => values });
    await Promise.resolve();
  });
  await settle();
}

function formValues(overrides = {}) {
  return {
    username: '',
    userSearchType: 'username',
    token_name: '',
    model_name: '',
    channel: '',
    group: '',
    request_id: '',
    dateRange: ['2026-09-23 00:00:00', '2026-09-23 01:00:00'],
    logType: '0',
    ...overrides,
  };
}

test('管理员列表和统计通过 params 保留特殊字符，并切换用户 ID 空值语义', async () => {
  adminMode = true;
  const errors = [];
  const { api, calls } = createApiMock(errors);
  const harness = await mountHook(api, errors);
  await setForm(
    harness.result,
    formValues({
      username: 'alice&user_id=101',
      token_name: 'token&x',
      model_name: 'model?x',
      channel: 'channel#x',
      group: 'group+1',
      request_id: 'request?x',
      logType: '2',
    }),
  );
  calls.length = 0;

  await act(async () => {
    await harness.result().loadLogs(1, 20);
    await harness.result().handleEyeClick();
  });

  assert.deepEqual(
    calls.map((call) => call.url),
    ['/api/log/', '/api/log/stat'],
  );
  for (const call of calls) {
    const query = new URLSearchParams(call.config.params);
    assert.equal(query.get('username'), 'alice&user_id=101');
    assert.equal(query.get('user_id'), null);
    assert.equal(query.get('token_name'), 'token&x');
    assert.equal(query.get('model_name'), 'model?x');
    assert.equal(query.get('group'), 'group+1');
  }

  await setForm(
    harness.result,
    formValues({ username: '101', userSearchType: 'id' }),
  );
  calls.length = 0;
  await act(async () => {
    await harness.result().loadLogs(1, 20);
    await harness.result().handleEyeClick();
  });
  assert.equal(
    new URLSearchParams(calls[0].config.params).get('user_id'),
    '101',
  );
  assert.equal(
    new URLSearchParams(calls[0].config.params).get('username'),
    null,
  );

  await setForm(
    harness.result,
    formValues({ username: '', userSearchType: 'id' }),
  );
  calls.length = 0;
  await act(async () => {
    await harness.result().loadLogs(1, 20);
  });
  const emptyQuery = new URLSearchParams(calls[0].config.params);
  assert.equal(emptyQuery.get('user_id'), null);
  assert.equal(emptyQuery.get('username'), '');
  harness.renderer.unmount();
});

test('管理员与普通用户使用各自列表和统计端点', async () => {
  for (const admin of [true, false]) {
    adminMode = admin;
    const errors = [];
    const { api, calls } = createApiMock(errors);
    const harness = await mountHook(api, errors);
    await setForm(harness.result, formValues({ username: 'alice' }));
    calls.length = 0;
    await act(async () => {
      await harness.result().loadLogs(1, 20);
      await harness.result().handleEyeClick();
    });
    assert.deepEqual(
      calls.map((call) => call.url),
      admin
        ? ['/api/log/', '/api/log/stat']
        : ['/api/log/self/', '/api/log/self/stat'],
    );
    if (!admin) {
      assert.equal(calls[0].config.params.username, undefined);
      assert.equal(calls[1].config.params.username, undefined);
    }
    harness.renderer.unmount();
  }
});

test('HTTP 400 或网络失败后 loading 复位，下一次成功请求更新数据', async () => {
  adminMode = true;
  const errors = [];
  const { api, calls, queue } = createApiMock(errors);
  const harness = await mountHook(api, errors);
  await setForm(harness.result, formValues({ username: 'alice' }));
  calls.length = 0;

  queue.push({ reject: new Error('network'), toast: true });
  await act(async () => {
    await harness.result().handleEyeClick();
  });
  assert.equal(harness.result().loadingStat, false);
  queue.push(successStat({ quota: 12, token: 34 }));
  await act(async () => {
    await harness.result().handleEyeClick();
  });
  assert.equal(harness.result().loadingStat, false);
  assert.deepEqual(harness.result().stat, { quota: 12, token: 34 });

  const http400 = Object.assign(new Error('bad request'), {
    response: { status: 400 },
  });
  queue.push({ reject: http400, toast: true });
  await act(async () => {
    await harness.result().loadLogs(1, 20);
  });
  assert.equal(harness.result().loading, false);
  const log = { id: 9, created_at: 1, type: 2, other: {} };
  queue.push(successList([log]));
  await act(async () => {
    await harness.result().loadLogs(1, 20);
  });
  assert.equal(harness.result().loading, false);
  assert.equal(harness.result().logs[0].id, 9);
  assert.equal(harness.result().logCount, 1);
  assert.equal(errors.length, 2);
  harness.renderer.unmount();
});

test('refresh 统计失败只提示一次并停止列表请求，下一次成功时更新两类数据', async () => {
  adminMode = true;
  const errors = [];
  const { api, calls, queue } = createApiMock(errors);
  const harness = await mountHook(api, errors);
  await setForm(harness.result, formValues({ username: 'alice' }));
  calls.length = 0;

  queue.push({ data: { success: false, message: '非法用户 ID' } });
  await act(async () => {
    await harness.result().refresh();
  });
  assert.equal(errors.length, 1);
  assert.deepEqual(
    calls.map((call) => call.url),
    ['/api/log/stat'],
  );
  assert.equal(harness.result().loadingStat, false);
  assert.equal(harness.result().loading, false);

  queue.push(successStat({ quota: 20, token: 40 }));
  queue.push(successList([{ id: 10, created_at: 2, type: 2, other: {} }]));
  await act(async () => {
    await harness.result().refresh();
  });
  assert.deepEqual(
    calls.map((call) => call.url),
    ['/api/log/stat', '/api/log/stat', '/api/log/'],
  );
  assert.deepEqual(harness.result().stat, { quota: 20, token: 40 });
  assert.equal(harness.result().logs[0].id, 10);
  assert.equal(harness.result().loadingStat, false);
  assert.equal(harness.result().loading, false);
  harness.renderer.unmount();
});
