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
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { expect, test, vi } from 'vitest'

import { getModelCategory } from '@/features/channels/lib/model-categories'

import { ModelBadge } from '../model-badge'

// 图标包依赖浏览器专用 JSON 资源；此处仅隔离第三方渲染资源。
vi.mock('@lobehub/icons', () =>
  Object.fromEntries(
    ['OpenAI', 'Meta', 'Ai360', 'Qwen', 'Stepfun', 'Perplexity'].map((name) => [
      name,
      () => <svg aria-hidden='true' />,
    ])
  )
)

test.each([
  ['360gpt-pro', '360 AI', '360 AI'],
  ['wan2.2-t2v-plus', 'Wan', 'Wan'],
  ['text-embedding-v3', 'Qwen', 'Qwen'],
  ['step-tts-mini', 'StepFun', 'StepFun'],
  ['sonar-llama-3', 'Perplexity', 'Perplexity'],
])('模型 %s 的渠道分类和日志供应商一致', (model, category, label) => {
  render(<ModelBadge modelName={model} />)
  expect(getModelCategory(model)).toBe(category)
  expect(screen.getByLabelText(label)).toBeInTheDocument()
  expect(screen.getByText(model)).toBeInTheDocument()
})

test('未知模型保留原始名称且不伪造供应商', () => {
  render(<ModelBadge modelName='private-model-42' />)
  expect(getModelCategory('private-model-42')).toBe('Other')
  expect(screen.getByText('private-model-42')).toBeInTheDocument()
  expect(screen.queryByLabelText('OpenAI')).not.toBeInTheDocument()
})

test('映射模型继续在详情中显示且不替换请求模型', async () => {
  render(
    <ModelBadge modelName='text-embedding-v3' actualModel='private-embedding' />
  )
  await userEvent.click(screen.getByRole('button'))
  expect(await screen.findByText('private-embedding')).toBeVisible()
  expect(screen.getByText('Request Model:')).toBeVisible()
  expect(screen.getByText('Actual Model:')).toBeVisible()
})
