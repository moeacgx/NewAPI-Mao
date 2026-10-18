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
import { expect, test } from 'vitest'

import { CHANNEL_TYPES } from '@/features/channels/constants'
import {
  CHANNEL_FORM_DEFAULT_VALUES,
  buildSettingJSON,
  channelFormSchema,
  transformChannelToFormDefaults,
  transformFormDataToCreatePayload,
} from '@/features/channels/lib/channel-form'
import type { Channel } from '@/features/channels/types'

test('plugin selection roundtrips through channel setting and never uses AtlasCloud 61', () => {
  const form = {
    ...CHANNEL_FORM_DEFAULT_VALUES,
    name: 'plugin',
    type: 62,
    key: 'key',
    models: 'sora-2',
    task_plugin_key: 'sora',
  }
  expect(CHANNEL_TYPES[61]).toBe('AtlasCloud')
  expect(CHANNEL_TYPES[62]).toBe('Task Plugin')
  const payload = transformFormDataToCreatePayload(form).channel
  expect(JSON.parse(payload.setting ?? '{}')).toMatchObject({
    task_plugin_key: 'sora',
  })
  const restored = transformChannelToFormDefaults({
    ...payload,
    channel_info: {},
    id: 123,
  } as Channel)
  expect(restored.task_plugin_key).toBe('sora')
  expect(
    JSON.parse(buildSettingJSON({ ...form, type: 61 }))
  ).not.toHaveProperty('task_plugin_key')
})

test('plugin channel validation rejects missing binding while native channels remain valid', () => {
  const form = {
    ...CHANNEL_FORM_DEFAULT_VALUES,
    name: 'plugin',
    key: 'key',
    models: 'sora-2',
  }
  const result = channelFormSchema.safeParse({ ...form, type: 62 })
  expect(result.success).toBe(false)
  if (!result.success) {
    expect(
      result.error.issues.some((issue) => issue.path[0] === 'task_plugin_key')
    ).toBe(true)
  }
  expect(
    channelFormSchema.safeParse({ ...form, type: 62, task_plugin_key: 'sora' })
      .success
  ).toBe(true)
  expect(channelFormSchema.safeParse({ ...form, type: 61 }).success).toBe(true)
})
