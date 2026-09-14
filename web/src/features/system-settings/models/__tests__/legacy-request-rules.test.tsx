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
import { useState } from 'react'
import { describe, expect, test, vi } from 'vitest'

import { TieredPricingEditor } from '../tiered-pricing-editor'

const billingExpr = 'tier("base", p * 2 + c * 4)'

function EditorFixture(props: {
  rules: string
  onRulesChange: (value: string) => void
}) {
  const [billing, setBilling] = useState(billingExpr)
  const [rules, setRules] = useState(props.rules)
  return (
    <TieredPricingEditor
      billingExpr={billing}
      requestRuleExpr={rules}
      onBillingExprChange={setBilling}
      onRequestRuleExprChange={(value) => {
        props.onRulesChange(value)
        setRules(value)
      }}
    />
  )
}

describe('历史请求规则保留', () => {
  test.each([
    '(custom_probe() == true ? 2 : 1)',
    '(hour("UTC") >= 9 || hour("UTC") < 12 ? 2 : 1)',
  ])('打开无法可视化的规则 %s 时不清空已存表达式', async (rules) => {
    const onRulesChange = vi.fn()
    const user = userEvent.setup()
    render(<EditorFixture rules={rules} onRulesChange={onRulesChange} />)
    expect(
      screen.getByText(
        'This expression is too complex for the visual editor. Please switch to expression mode to edit.'
      )
    ).toBeVisible()
    expect(onRulesChange).not.toHaveBeenCalled()
    await user.click(screen.getByRole('combobox', { name: 'Editor mode' }))
    await user.click(screen.getByRole('option', { name: 'Expression editor' }))
    expect(screen.getByRole('textbox')).toHaveValue(
      `(${billingExpr}) * ${rules}`
    )
    expect(onRulesChange).not.toHaveBeenCalled()
    await user.click(screen.getByRole('combobox', { name: 'Editor mode' }))
    await user.click(screen.getByRole('option', { name: 'Visual editor' }))
    expect(
      screen.getByText(
        'This expression is too complex for the visual editor. Please switch to expression mode to edit.'
      )
    ).toBeVisible()
    expect(onRulesChange).not.toHaveBeenCalledWith('')
    await user.click(screen.getByRole('combobox', { name: 'Editor mode' }))
    await user.click(screen.getByRole('option', { name: 'Expression editor' }))
    expect(screen.getByRole('textbox')).toHaveValue(
      `(${billingExpr}) * ${rules}`
    )
  })
})

describe('编辑草稿行的稳定标识', () => {
  test('编辑及删除前一规则组后保留后一组输入节点与内容', async () => {
    const user = userEvent.setup()
    render(
      <EditorFixture
        rules='(param("first") == "a" ? 2 : 1) * (param("second") == "b" ? 3 : 1)'
        onRulesChange={vi.fn()}
      />
    )
    const secondPath = screen.getByDisplayValue('second')
    await user.type(secondPath, '_edited')
    expect(screen.getByDisplayValue('second_edited')).toBe(secondPath)
    expect(secondPath).toHaveFocus()
    await user.click(
      screen.getAllByRole('button', { name: 'Remove rule group' })[0]
    )
    expect(screen.queryByDisplayValue('first')).not.toBeInTheDocument()
    expect(screen.getByDisplayValue('second_edited')).toBe(secondPath)
    await user.click(screen.getByRole('button', { name: 'Add rule group' }))
    expect(screen.getByDisplayValue('second_edited')).toBe(secondPath)
  })

  test('删除前一请求条件后保留后一条件并可继续输入', async () => {
    const user = userEvent.setup()
    render(
      <EditorFixture
        rules='(param("first") == "a" && param("second") == "b" ? 2 : 1)'
        onRulesChange={vi.fn()}
      />
    )
    const secondPath = screen.getByDisplayValue('second')
    await user.click(
      screen.getAllByRole('button', { name: 'Remove condition' })[0]
    )
    expect(screen.getByDisplayValue('second')).toBe(secondPath)
    await user.type(secondPath, '_edited')
    expect(secondPath).toHaveFocus()
    expect(secondPath).toHaveValue('second_edited')
    await user.click(
      screen.getAllByRole('button', { name: 'Add condition' })[1]
    )
    expect(screen.getByDisplayValue('second_edited')).toBe(secondPath)
  })

  test('删除前一阶梯后保留后一阶梯输入节点与名称', async () => {
    const user = userEvent.setup()
    render(<EditorFixture rules='' onRulesChange={vi.fn()} />)
    await user.click(screen.getByRole('button', { name: 'Add tier' }))
    const secondTier = screen.getByDisplayValue('tier_2')
    await user.type(secondTier, '_edited')
    expect(secondTier).toHaveFocus()
    await user.click(screen.getAllByRole('button', { name: 'Remove tier' })[0])
    expect(screen.getByDisplayValue('tier_2_edited')).toBe(secondTier)
    await user.click(screen.getByRole('button', { name: 'Add tier' }))
    expect(screen.getByDisplayValue('tier_2_edited')).toBe(secondTier)
  })

  test('删除前一阶梯条件后保留后一条件的输入草稿', async () => {
    const user = userEvent.setup()
    render(<EditorFixture rules='' onRulesChange={vi.fn()} />)
    await user.click(screen.getByRole('button', { name: 'Add condition' }))
    await user.click(screen.getByRole('button', { name: 'Add condition' }))
    const secondValue = screen.getAllByPlaceholderText('tokens')[1]
    await user.clear(secondValue)
    await user.type(secondValue, '123')
    expect(secondValue).toHaveFocus()
    await user.click(screen.getAllByRole('button', { name: 'remove' })[0])
    expect(screen.getByPlaceholderText('tokens')).toBe(secondValue)
    expect(secondValue).toHaveValue(123)
    await user.click(screen.getByRole('button', { name: 'Add condition' }))
    expect(screen.getAllByPlaceholderText('tokens')[0]).toBe(secondValue)
  })
})
