/*
Copyright (C) 2023-2026 QuantumNous

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
  getAuditEventRouteGroupOrigin,
  getAuditEventChannelGroupsOrigin,
  getAuditEventTokenGroupsOrigin,
} from '../event-origin.js';

test('当前名称投影只用于标签，历史名称与别名证据保持原值', () => {
  const group = Object.freeze({
    id: 7,
    code: 'legacy-alias',
    name: '历史名称',
    display_name: '当前名称',
  });
  const event = Object.freeze({
    stage: 'response',
    group_id: 7,
    group_code: 'legacy-alias',
    group_name: '历史名称',
    group_display_name: '当前名称',
    channel_groups: [group],
    token_group_mode: 'explicit',
    token_groups: [group],
  });
  for (const resolve of [
    getAuditEventRouteGroupOrigin,
    getAuditEventChannelGroupsOrigin,
    getAuditEventTokenGroupsOrigin,
  ]) {
    assert.equal(resolve(event).items[0].name, '当前名称');
    assert.equal(resolve(event).items[0].code, 'legacy-alias');
  }
  assert.equal(event.group_name, '历史名称');
  assert.equal(group.name, '历史名称');
});

test('缺失、非字符串或继承的展示字段不能覆盖历史名称', () => {
  for (const group of [
    { id: 7, name: '历史名称', display_name: {} },
    { id: 7, name: '历史名称', display_name: '' },
    Object.assign(Object.create({ display_name: '继承值' }), {
      id: 7,
      name: '历史名称',
    }),
  ]) {
    assert.equal(
      getAuditEventChannelGroupsOrigin({ channel_groups: [group] }).items[0]
        .name,
      '历史名称',
    );
  }
});
