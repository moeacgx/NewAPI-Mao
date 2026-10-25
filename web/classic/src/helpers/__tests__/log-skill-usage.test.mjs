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

import assert from 'node:assert/strict';
import test from 'node:test';

import {
  describeLogSkillUsage,
  getLogOther,
  getLogSkillUsage,
} from '../log.js';

const t = (key, vars = {}) =>
  key.replace(/{{(\w+)}}/g, (_match, name) => String(vars[name]));

const alwaysLoadedTemplate = { id: 7, title: 'Code Review Rules', version: 3 };
const bundleFiles = [
  {
    template_id: 12,
    skill_id: 'go-testing',
    title: 'Go Testing',
    version: 2,
    path: 'SKILL.md',
    offset: 0,
    bytes: 4096,
  },
  {
    template_id: 12,
    skill_id: 'go-testing',
    title: 'Go Testing',
    version: 2,
    path: 'references/table-tests.md',
    offset: 0,
    bytes: 1024,
  },
  {
    template_id: 12,
    skill_id: 'sql-style',
    title: 'SQL Style',
    version: 1,
    path: 'SKILL.md',
    offset: 0,
    bytes: 2048,
  },
];

test('Classic 使用日志统计始终装载模板与最终生成按需读取的去重技能', () => {
  assert.deepEqual(
    getLogSkillUsage({
      prompt_templates: [alwaysLoadedTemplate],
      gateway_skills: {
        stage: 'generation',
        round: 2,
        catalogue_count: 5,
        catalogue_queries: [{ query_bytes: 12, offset: 0 }],
        files: bundleFiles,
      },
    }),
    {
      kind: 'used',
      count: 3,
      alwaysLoaded: ['Code Review Rules'],
      onDemand: ['Go Testing', 'SQL Style'],
    },
  );
});

test('Classic 使用日志的私有发现轮次优先于已用技能并列出已读技能', () => {
  assert.deepEqual(
    getLogSkillUsage({
      prompt_templates: [alwaysLoadedTemplate],
      gateway_skills: {
        stage: 'discovery',
        round: 2,
        catalogue_count: 5,
        files: bundleFiles,
      },
    }),
    { kind: 'discovery', round: 2, readSkills: ['Go Testing', 'SQL Style'] },
  );
});

test('Classic 使用日志最终生成未读取技能时返回可用技能数', () => {
  assert.deepEqual(
    getLogSkillUsage({
      gateway_skills: { stage: 'generation', round: 1, catalogue_count: 5 },
    }),
    { kind: 'unused', catalogueCount: 5 },
  );
});

test('Classic 使用日志仅有始终装载模板时也计为已用技能', () => {
  assert.deepEqual(
    getLogSkillUsage({ prompt_templates: [alwaysLoadedTemplate] }),
    {
      kind: 'used',
      count: 1,
      alwaysLoaded: ['Code Review Rules'],
      onDemand: [],
    },
  );
});

test('Classic 使用日志技能缺少标题时回退为技能标识', () => {
  const usage = getLogSkillUsage({
    gateway_skills: {
      stage: 'generation',
      catalogue_count: 1,
      files: [{ template_id: 12, skill_id: 'go-testing', title: ' ' }],
    },
  });
  assert.deepEqual(usage.onDemand, ['go-testing']);
});

test('Classic 使用日志无技能数据或数据格式错误时不显示技能', () => {
  const originalConsoleError = console.error;
  console.error = () => {};
  try {
    for (const other of [
      null,
      {},
      { prompt_templates: [] },
      { prompt_templates: 'x' },
      { gateway_skills: 'x' },
      { gateway_skills: { stage: 'generation', files: {} } },
      {
        prompt_templates: [null, { id: '7', title: 'Code Review Rules' }],
        gateway_skills: {
          stage: 'generation',
          files: [null, { template_id: '12', skill_id: 'go-testing' }],
        },
      },
      { gateway_skills: { stage: 'discovery', files: bundleFiles } },
      42,
      getLogOther('{"gateway_skills":'),
    ]) {
      assert.equal(getLogSkillUsage(other), null, JSON.stringify(other));
    }
  } finally {
    console.error = originalConsoleError;
  }
});

test('Classic 使用日志技能文案按装载方式分组列出名称', () => {
  assert.deepEqual(
    describeLogSkillUsage(
      {
        kind: 'used',
        count: 3,
        alwaysLoaded: ['Code Review Rules'],
        onDemand: ['Go Testing', 'SQL Style'],
      },
      t,
    ),
    {
      label: '技能 3',
      lines: ['始终装载：Code Review Rules', '按需读取：Go Testing, SQL Style'],
    },
  );
});

test('Classic 使用日志技能发现文案说明单独计费并列出已读技能', () => {
  assert.deepEqual(
    describeLogSkillUsage(
      { kind: 'discovery', round: 2, readSkills: ['Go Testing'] },
      t,
    ),
    {
      label: '技能发现 · 第 2 轮',
      lines: ['私有技能发现轮次，单独计费', '已读技能：Go Testing'],
    },
  );
  assert.deepEqual(
    describeLogSkillUsage({ kind: 'discovery', round: 1, readSkills: [] }, t)
      .lines,
    ['私有技能发现轮次，单独计费'],
  );
});

test('Classic 使用日志未用技能文案给出可用技能数', () => {
  assert.deepEqual(
    describeLogSkillUsage({ kind: 'unused', catalogueCount: 5 }, t),
    {
      label: '未用技能',
      lines: ['可用 5 个技能，本次未读取'],
    },
  );
});
