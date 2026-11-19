import {
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import {
  transformFormDataToModelPayload,
  transformModelToFormDefaults,
} from '../../lib/model-form'
import type { Model } from '../../types'
import { useModelsColumns } from '../models-columns'

vi.mock('@lobehub/icons', () => ({}))

const model: Model & { enable_group_names: Record<string, string> } = {
  id: 1,
  model_name: 'fixture-model',
  status: 1,
  sync_official: 0,
  created_time: 0,
  updated_time: 0,
  name_rule: 0,
  enable_groups: ['g_7', 'legacy', 'deleted'],
  enable_group_names: {
    g_7: '当前名称',
    legacy: '历史别名对应名称',
    deleted: 'deleted',
  },
}

function GroupCell(props: { model: Model }) {
  const columns = useModelsColumns([])
  const table = useReactTable({
    data: [props.model],
    columns,
    getCoreRowModel: getCoreRowModel(),
  })
  const cell = table
    .getRowModel()
    .rows[0].getVisibleCells()
    .find((entry) => entry.column.id === 'enable_groups')
  return cell ? flexRender(cell.column.columnDef.cell, cell.getContext()) : null
}

describe('模型可用分组名称', () => {
  it('名称与编码不同时显示当前名称和历史别名名称，删除分组回退编码', () => {
    render(<GroupCell model={model} />)
    expect(screen.getByText('当前名称')).toBeInTheDocument()
    expect(screen.getByText('历史别名对应名称')).toBeInTheDocument()
    expect(screen.getByText('deleted')).toBeInTheDocument()
    expect(screen.queryByText('g_7')).not.toBeInTheDocument()
    expect(screen.queryByText('legacy')).not.toBeInTheDocument()
  })

  it('刷新改名响应后更新名称，移除映射时不残留旧名称', () => {
    const view = render(<GroupCell model={model} />)
    view.rerender(
      <GroupCell
        model={{
          ...model,
          enable_group_names: { g_7: '改名后', legacy: '别名改名后' },
        }}
      />
    )
    expect(screen.getByText('改名后')).toBeInTheDocument()
    expect(screen.getByText('别名改名后')).toBeInTheDocument()
    expect(screen.queryByText('当前名称')).not.toBeInTheDocument()
    view.rerender(<GroupCell model={{ ...model, enable_group_names: {} }} />)
    expect(screen.getByText('g_7')).toBeInTheDocument()
  })

  it('原型属性名仅使用响应自有映射，未知编码正常回退', () => {
    render(
      <GroupCell
        model={{
          ...model,
          enable_groups: ['__proto__', 'constructor', 'unknown'],
          enable_group_names: JSON.parse('{"__proto__":"原型别名名称"}'),
        }}
      />
    )
    expect(screen.getByText('原型别名名称')).toBeInTheDocument()
    expect(screen.getByText('constructor')).toBeInTheDocument()
    expect(screen.getByText('unknown')).toBeInTheDocument()
  })

  it('模型表单提交保留原始编码，不提交只读名称映射', () => {
    const payload = transformFormDataToModelPayload(
      transformModelToFormDefaults(model)
    )
    expect(payload.enable_groups).toEqual(['g_7', 'legacy', 'deleted'])
    expect(payload).not.toHaveProperty('enable_group_names')
  })
})
