import {
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import type { Model, Vendor } from '../../types'
import { useModelsColumns } from '../models-columns'

// URL 分支不依赖品牌资产，避免 Node 直接解析图标库的 ESM 目录导入。
vi.mock('@lobehub/icons', () => ({}))

const vendor: Vendor = {
  id: 1,
  name: 'Fixture Vendor',
  icon: 'https://cdn.example.com/logo.svg',
  status: 1,
  created_time: 0,
  updated_time: 0,
}
const model: Model = {
  id: 1,
  model_name: 'fixture-model',
  vendor_id: 1,
  icon: '',
  status: 1,
  sync_official: 0,
  created_time: 0,
  updated_time: 0,
  name_rule: 0,
}

function ModelNameCell(props: { model: Model }) {
  const columns = useModelsColumns([vendor])
  const table = useReactTable({
    data: [props.model],
    columns,
    getCoreRowModel: getCoreRowModel(),
  })
  const cell = table
    .getRowModel()
    .rows[0].getVisibleCells()
    .find((entry) => entry.column.id === 'model_name')
  return cell ? flexRender(cell.column.columnDef.cell, cell.getContext()) : null
}

describe('模型列表 Logo', () => {
  it('模型无独立图标时向共享加载器传递供应商原始 URL', () => {
    const { container } = render(<ModelNameCell model={model} />)
    expect(container.querySelector('img')).toHaveAttribute('src', vendor.icon)
    expect(container.querySelector('img')).toHaveAttribute('width', '20')
    expect(screen.getByText('fixture-model')).toBeInTheDocument()
  })

  it('模型独立 URL 优先于供应商 URL 且不追加 Avatar 参数', () => {
    const url = 'https://models.example.com/model.svg?version=1.2'
    const { container } = render(
      <ModelNameCell model={{ ...model, icon: url }} />
    )
    expect(container.querySelector('img')).toHaveAttribute('src', url)
  })
})
