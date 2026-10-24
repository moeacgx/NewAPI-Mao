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
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { getTokenModels, sendChatCompletion } from '../api'
import { createStreamRequestController } from '../hooks/use-stream-request'

vi.mock('@/lib/api', () => ({ api: { get: vi.fn(), post: vi.fn() } }))

beforeEach(() => vi.clearAllMocks())

describe('操练场令牌请求', () => {
  it('选中令牌后携带 ID 并省略账号分组，保留原始参数', async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { choices: [] } })
    const payload = {
      model: 'model-a',
      group: 'account-only',
      messages: [],
      stream: false,
      temperature: 0,
    }
    await sendChatCompletion(payload, undefined, 7)
    expect(api.post).toHaveBeenCalledWith(
      '/pg/chat/completions',
      {
        model: 'model-a',
        messages: [],
        stream: false,
        temperature: 0,
      },
      expect.objectContaining({
        headers: {
          'X-New-API-Skills-Display': 'metadata',
          'X-Playground-Token-Id': '7',
        },
      })
    )
    expect(payload.group).toBe('account-only')
  })

  it('默认账号调用保留分组且不设置令牌头', async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { choices: [] } })
    const payload = {
      model: 'model-a',
      group: 'account-group',
      messages: [],
      stream: false,
    }
    await sendChatCompletion(payload)
    expect(api.post).toHaveBeenCalledWith(
      '/pg/chat/completions',
      payload,
      expect.objectContaining({
        headers: { 'X-New-API-Skills-Display': 'metadata' },
      })
    )
  })

  it('非流式响应读取网关实际装载的技能元数据', async () => {
    const metadata = {
      version: 1,
      total: 1,
      skills: [
        {
          template_id: 15,
          skill_id: 'review',
          version: 2,
          title: 'Code review',
        },
      ],
      truncated: false,
    }
    const encoded = btoa(JSON.stringify(metadata))
      .replaceAll('+', '-')
      .replaceAll('/', '_')
      .replaceAll('=', '')
    vi.mocked(api.post).mockResolvedValue({
      data: { choices: [] },
      headers: { 'x-new-api-gateway-skills': encoded },
    })

    const result = await sendChatCompletion({
      model: 'model-a',
      messages: [],
      stream: false,
    })

    expect(result.response).toEqual({ choices: [] })
    expect(result.gatewaySkills).toEqual({
      version: 1,
      total: 1,
      skills: [
        {
          templateId: 15,
          skillId: 'review',
          version: 2,
          title: 'Code review',
        },
      ],
      truncated: false,
    })
  })

  it('模型发现使用同一令牌身份，接口失败不回退账号模型', async () => {
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { data: [{ id: 'token-model' }] },
    })
    await expect(getTokenModels(7)).resolves.toEqual([
      { label: 'token-model', value: 'token-model' },
    ])
    expect(api.get).toHaveBeenCalledWith(
      '/pg/token/models',
      expect.objectContaining({ headers: { 'X-Playground-Token-Id': '7' } })
    )
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { error: { message: 'Token disabled' } },
    })
    await expect(getTokenModels(7)).rejects.toThrow('Token disabled')
    expect(api.get).toHaveBeenCalledTimes(2)
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { success: false, message: 'Token expired', data: [] },
    })
    await expect(getTokenModels(7)).rejects.toThrow('Token expired')
  })

  it('流式请求保留登录态并应用同一令牌头和分组省略规则', async () => {
    const createSource = vi.fn(() => ({
      addEventListener: vi.fn(),
      close: vi.fn(),
      stream: vi.fn(),
    }))
    const controller = createStreamRequestController({
      getHeaders: async () => ({ Authorization: 'Bearer session-auth' }),
      createSource,
      setStreaming: vi.fn(),
    })
    const callbacks = {
      onUpdate: vi.fn(),
      onError: vi.fn(),
      onComplete: vi.fn(),
    }
    await controller.send(
      {
        model: 'token-model',
        messages: [],
        group: 'account-only',
        stream: true,
      },
      callbacks,
      7
    )
    expect(createSource).toHaveBeenCalledWith(
      { model: 'token-model', messages: [], stream: true },
      {
        Authorization: 'Bearer session-auth',
        'X-New-API-Skills-Display': 'metadata',
        'X-Playground-Token-Id': '7',
      }
    )
    controller.dispose()
  })
})
