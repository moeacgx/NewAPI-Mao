import assert from 'node:assert/strict';
import fs from 'node:fs';
import { createRequire } from 'node:module';
import test from 'node:test';
import vm from 'node:vm';

import { transformSync } from 'esbuild';
import React from 'react';
import { act, create } from 'react-test-renderer';

import {
  describeLogSkillUsage,
  getLogSkillUsage,
} from '../../../../helpers/log.js';

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
    getLogSkillUsage,
    describeLogSkillUsage,
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

for (const isAdminUser of [true, false]) {
  test(`技能条件跨分页并同步统计，清除后恢复全部，管理员=${isAdminUser}`, async () => {
    adminMode = isAdminUser;
    const errors = [];
    const { api, calls } = createApiMock(errors);
    const harness = await mountHook(api, errors);
    const values = formValues({
      skill: 'used',
      username: '101',
      userSearchType: 'id',
    });
    await setForm(harness.result, values);
    calls.length = 0;
    await act(async () => {
      await harness.result().refresh();
      await harness.result().handlePageChange(2);
    });
    assert.ok(calls.some((call) => call.url.endsWith('/stat')));
    assert.ok(calls.some((call) => call.config.params.p === 2));
    for (const call of calls) {
      assert.equal(call.config.params.skill, 'used');
      assert.equal(call.config.params.user_id, isAdminUser ? '101' : undefined);
      if (!isAdminUser) assert.ok(call.url.startsWith('/api/log/self/'));
    }
    values.skill = '';
    calls.length = 0;
    await act(async () => {
      await harness.result().refresh();
    });
    assert.equal(calls.length, 2);
    assert.ok(calls.every((call) => call.config.params.skill === undefined));
    assert.equal(
      calls.find((call) => !call.url.endsWith('/stat')).config.params.p,
      1,
    );
    harness.renderer.unmount();
  });
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

test('当前连接只跟随管理员已提交的用户条件，清空提交恢复本人', async () => {
  adminMode = true;
  const errors = [];
  const { api } = createApiMock(errors);
  const harness = await mountHook(api, errors);
  assert.equal(harness.result().activeRequestSubject, null);

  await setForm(harness.result, formValues({ username: 'alice' }));
  assert.equal(harness.result().activeRequestSubject, null);
  await act(async () => {
    await harness.result().loadLogs(1, 20);
  });
  assert.deepEqual(harness.result().activeRequestSubject, {
    username: 'alice',
  });

  await setForm(
    harness.result,
    formValues({ username: '5215', userSearchType: 'id' }),
  );
  await act(async () => {
    await harness.result().handleEyeClick();
  });
  assert.deepEqual(harness.result().activeRequestSubject, {
    username: 'alice',
  });
  await act(async () => {
    await harness.result().loadLogs(1, 20);
  });
  assert.deepEqual(harness.result().activeRequestSubject, { userId: '5215' });
  const submitted = harness.result().activeRequestSubject;
  await act(async () => {
    await harness.result().loadLogs(2, 20);
  });
  assert.equal(harness.result().activeRequestSubject, submitted);

  await setForm(harness.result, formValues({ username: '' }));
  assert.deepEqual(harness.result().activeRequestSubject, { userId: '5215' });
  await act(async () => {
    await harness.result().loadLogs(1, 20);
  });
  assert.equal(harness.result().activeRequestSubject, null);
  harness.renderer.unmount();
});

test('提交查询后即使历史列表失败，当前连接仍对应所查用户', async () => {
  adminMode = true;
  const errors = [];
  const { api, queue } = createApiMock(errors);
  const harness = await mountHook(api, errors);
  await setForm(
    harness.result,
    formValues({ username: 'invalid', userSearchType: 'id' }),
  );
  queue.push({ reject: new Error('bad request') });
  await act(async () => {
    await harness.result().loadLogs(1, 20);
  });
  assert.deepEqual(harness.result().activeRequestSubject, {
    userId: 'invalid',
  });
  harness.renderer.unmount();
});

test('refresh 提交时立即切换当前连接，统计失败也不回退为本人', async () => {
  adminMode = true;
  const errors = [];
  const { api, queue } = createApiMock(errors);
  const harness = await mountHook(api, errors);
  await setForm(harness.result, formValues({ username: 'alice' }));
  queue.push({ data: { success: false, message: '查询失败' } });
  await act(async () => {
    await harness.result().refresh();
  });
  assert.deepEqual(harness.result().activeRequestSubject, {
    username: 'alice',
  });
  harness.renderer.unmount();
});

test('refresh 等待统计时编辑输入不改变已提交对象或列表用户条件', async () => {
  adminMode = true;
  const errors = [];
  const { api, calls, queue } = createApiMock(errors);
  const harness = await mountHook(api, errors);
  const values = formValues({ username: '5215', userSearchType: 'id' });
  await setForm(harness.result, values);
  let resolveStat;
  queue.push(
    new Promise((resolve) => {
      resolveStat = resolve;
    }),
  );
  calls.length = 0;
  let refresh;
  await act(async () => {
    refresh = harness.result().refresh();
    await Promise.resolve();
  });
  assert.deepEqual(harness.result().activeRequestSubject, { userId: '5215' });
  values.username = 'alice';
  values.userSearchType = 'username';
  await act(async () => {
    resolveStat(successStat());
    await refresh;
  });
  assert.equal(calls[1].config.params.user_id, '5215');
  assert.equal(calls[1].config.params.username, undefined);
  assert.deepEqual(harness.result().activeRequestSubject, { userId: '5215' });
  harness.renderer.unmount();
});

test('普通用户即使带有跨用户表单条件，当前连接始终为本人', async () => {
  adminMode = false;
  const errors = [];
  const { api } = createApiMock(errors);
  const harness = await mountHook(api, errors);
  await setForm(
    harness.result,
    formValues({ username: '5215', userSearchType: 'id' }),
  );
  await act(async () => {
    await harness.result().loadLogs(1, 20);
    await harness.result().refresh();
  });
  assert.equal(harness.result().activeRequestSubject, null);
  harness.renderer.unmount();
});

test('较早 refresh 的统计完成时不能覆盖后来提交的连接对象', async () => {
  adminMode = true;
  const errors = [];
  const { api, calls, queue } = createApiMock(errors);
  const harness = await mountHook(api, errors);
  await setForm(harness.result, formValues({ username: 'alice' }));
  let resolveStat;
  queue.push(
    new Promise((resolve) => {
      resolveStat = resolve;
    }),
  );
  let refresh;
  await act(async () => {
    refresh = harness.result().refresh();
    await Promise.resolve();
  });
  await setForm(
    harness.result,
    formValues({ username: '5215', userSearchType: 'id' }),
  );
  await act(async () => {
    await harness.result().loadLogs(1, 20);
  });
  const callsBeforeLateStat = calls.length;
  assert.deepEqual(harness.result().activeRequestSubject, { userId: '5215' });
  await act(async () => {
    resolveStat(successStat());
    await refresh;
  });
  assert.deepEqual(harness.result().activeRequestSubject, { userId: '5215' });
  assert.equal(calls.length, callsBeforeLateStat);
  assert.deepEqual(harness.result().stat, { quota: 0, token: 0 });
  harness.renderer.unmount();
});

test('较早列表的迟到响应不覆盖新用户日志与分页状态', async () => {
  adminMode = true;
  const errors = [];
  const { api, queue } = createApiMock(errors);
  const harness = await mountHook(api, errors);
  await setForm(harness.result, formValues({ username: 'alice' }));
  let resolveOldList;
  queue.push(
    new Promise((resolve) => {
      resolveOldList = resolve;
    }),
  );
  let oldList;
  await act(async () => {
    oldList = harness.result().loadLogs(1, 20);
    await Promise.resolve();
  });
  await setForm(harness.result, formValues({ username: 'bob' }));
  queue.push(successList([{ id: 21, created_at: 1, type: 2, other: {} }]));
  await act(async () => {
    await harness.result().loadLogs(1, 20);
  });
  await act(async () => {
    resolveOldList(
      successList([{ id: 11, created_at: 1, type: 2, other: {} }]),
    );
    await oldList;
  });
  assert.equal(harness.result().logs[0].id, 21);
  assert.equal(harness.result().logCount, 1);
  assert.equal(harness.result().loading, false);
  assert.deepEqual(harness.result().activeRequestSubject, { username: 'bob' });
  harness.renderer.unmount();
});

test('等待旧统计时仍可提交新用户，迟到统计不能覆盖新统计或重新加载旧列表', async () => {
  adminMode = true;
  const errors = [];
  const { api, calls, queue } = createApiMock(errors);
  const harness = await mountHook(api, errors);
  await setForm(harness.result, formValues({ username: 'alice' }));
  let resolveOldStat;
  queue.push(
    new Promise((resolve) => {
      resolveOldStat = resolve;
    }),
  );
  let oldRefresh;
  await act(async () => {
    oldRefresh = harness.result().refresh();
    await Promise.resolve();
  });
  await setForm(harness.result, formValues({ username: 'bob' }));
  queue.push(successStat({ quota: 20, token: 40 }));
  queue.push(successList([{ id: 21, created_at: 1, type: 2, other: {} }]));
  await act(async () => {
    await harness.result().refresh();
  });
  assert.equal(harness.result().logs[0]?.id, 21);
  const callsBeforeLateStat = calls.length;
  await act(async () => {
    resolveOldStat(successStat({ quota: 10, token: 30 }));
    await oldRefresh;
  });
  assert.deepEqual(harness.result().stat, { quota: 20, token: 40 });
  assert.deepEqual(harness.result().activeRequestSubject, { username: 'bob' });
  assert.equal(harness.result().logs[0].id, 21);
  assert.equal(harness.result().loadingStat, false);
  assert.equal(calls.length, callsBeforeLateStat);
  harness.renderer.unmount();
});

test('相同查询条件翻页保留统计，切换用户时隐藏旧统计', async () => {
  adminMode = true;
  const errors = [];
  const { api } = createApiMock(errors);
  const harness = await mountHook(api, errors);
  const values = formValues({ username: 'alice' });
  await setForm(harness.result, values);
  assert.equal(harness.result().showStat, true);
  await act(async () => {
    await harness.result().loadLogs(2, 20);
  });
  assert.equal(harness.result().showStat, true);
  values.username = 'bob';
  await act(async () => {
    await harness.result().loadLogs(1, 20);
  });
  assert.equal(harness.result().showStat, false);
  harness.renderer.unmount();
});
