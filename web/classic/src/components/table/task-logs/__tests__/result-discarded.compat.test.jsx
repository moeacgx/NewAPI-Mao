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
import { expect, test, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { getTaskLogsColumns } from '../TaskLogsColumnDefs';

const title = 'Result not retained';
const explanation =
  'The synchronous result was returned through the API and was not saved for later viewing';

test('同步结果未保留时显示说明且不提供预览入口', async () => {
  const openVideoModal = vi.fn();
  const openImagePreview = vi.fn();
  const columns = getTaskLogsColumns({
    t: (key) => key,
    COLUMN_KEYS: {},
    openVideoModal,
    openImagePreview,
  });
  const details = columns.find((column) => column.dataIndex === 'fail_reason');
  const record = {
    task_id: 'task-discarded',
    action: 'text_to_video',
    status: 'SUCCESS',
    result_discarded: true,
    image_urls: ['/api/task/task-discarded/content/0'],
  };
  render(<>{details.render('', record)}</>);
  const label = screen.getByText(title);
  expect(screen.queryByRole('link')).toBeNull();
  await userEvent.setup().hover(label);
  expect(await screen.findByText(explanation)).toBeTruthy();
  expect(openVideoModal).not.toHaveBeenCalled();
  expect(openImagePreview).not.toHaveBeenCalled();
});

test.each([
  ['FAILURE', true, '供应商任务失败'],
  ['IN_PROGRESS', true, '无'],
  ['SUCCESS', false, '无'],
])('状态%s保留现有详情且不误报结果未保留', (status, discarded, expected) => {
  const columns = getTaskLogsColumns({ t: (key) => key, COLUMN_KEYS: {} });
  const details = columns.find((column) => column.dataIndex === 'fail_reason');
  render(
    <>
      {details.render(status === 'FAILURE' ? expected : '', {
        status,
        result_discarded: discarded,
      })}
    </>,
  );
  expect(screen.getByText(expected)).toBeTruthy();
  expect(screen.queryByText(title)).toBeNull();
});

test('保留的图片结果仍可打开原有预览', async () => {
  const openImagePreview = vi.fn();
  const columns = getTaskLogsColumns({
    t: (key) => key,
    COLUMN_KEYS: {},
    openImagePreview,
  });
  const details = columns.find((column) => column.dataIndex === 'fail_reason');
  const imageUrls = ['/api/task/task-retained/content/0'];
  render(
    <>
      {details.render('', {
        task_id: 'task-retained',
        status: 'SUCCESS',
        result_discarded: false,
        image_urls: imageUrls,
      })}
    </>,
  );
  await userEvent.setup().click(screen.getByRole('link', { name: '查看图片' }));
  expect(openImagePreview).toHaveBeenCalledWith(imageUrls, 'task-retained');
  expect(screen.queryByText(title)).toBeNull();
});
