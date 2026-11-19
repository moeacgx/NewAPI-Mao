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
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { expect, test } from 'vitest';
import { API } from '../../../helpers/api';
import Tools from '../Tools';

test('展开工具参考完整呈现服务端动态body属性、必填字段及协议描述', async () => {
  const schema = {
    type: 'object',
    required: ['request_id', 'body'],
    properties: {
      request_id: { type: 'string', minLength: 1 },
      body: {
        type: 'object',
        additionalProperties: false,
        required: [
          'id',
          'name',
          'endpoint_type',
          'stream',
          'vendor_id',
          'tags',
        ],
        properties: {
          id: { type: 'integer', minimum: 1 },
          name: { type: 'string', maxLength: 100 },
          endpoint_type: {
            type: 'string',
            enum: ['chat_completions', 'responses'],
            description: 'Select endpoint_type for the upstream protocol.',
          },
          stream: {
            type: 'boolean',
            default: false,
            description: 'Set stream explicitly, including false.',
          },
          vendor_id: { type: 'integer', minimum: 1 },
          tags: {
            type: 'array',
            items: {
              type: 'object',
              required: ['id', 'name'],
              properties: { id: { type: 'integer' }, name: { type: 'string' } },
            },
          },
        },
      },
    },
  };
  const description = 'Update a model. Specify endpoint_type and stream.';
  const originalAdapter = API.defaults.adapter;
  API.defaults.adapter = async (config) => {
    expect(config.method).toBe('get');
    expect(config.url).toBe('/api/mcp/tools');
    return {
      data: {
        success: true,
        data: [
          {
            name: 'models_update',
            description,
            category: 'models',
            write: true,
            min_role: 10,
            inputSchema: schema,
          },
        ],
      },
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
    };
  };
  try {
    const user = userEvent.setup();
    render(<Tools />);
    expect(await screen.findByText(description)).toBeTruthy();
    await user.click(screen.getByText('Input schema'));
    const details = screen.getByText('Input schema').closest('details');
    expect(details.open).toBe(true);
    // 对照显式接口样例核对全部内容，避免后续视图只投影顶层字段或丢弃嵌套约束。
    expect(JSON.parse(details.querySelector('pre').textContent)).toEqual(
      schema,
    );
    expect(details.textContent).toContain(
      'Select endpoint_type for the upstream protocol.',
    );
    expect(details.textContent).toContain(
      'Set stream explicitly, including false.',
    );
  } finally {
    API.defaults.adapter = originalAdapter;
  }
});
