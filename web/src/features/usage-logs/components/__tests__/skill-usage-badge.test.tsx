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
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, test, vi } from 'vitest'

import type { UsageLog } from '../../data/schema'
import { useCommonLogsColumns } from '../columns/common-logs-columns'

vi.mock('@lobehub/icons', () => ({}))
vi.mock('@/assets/custom/icon-sub2api', () => ({ IconSub2api: () => null }))

const baseLog: UsageLog = {
  id: 1,
  user_id: 2,
  created_at: 1_761_350_400,
  type: 2,
  content: '',
  username: 'alice',
  token_name: 'token',
  model_name: 'claude-sonnet',
  quota: 1,
  prompt_tokens: 1,
  completion_tokens: 1,
  use_time: 1,
  is_stream: false,
  channel: 3,
  channel_name: 'channel',
  token_id: 4,
  group: 'default',
  group_name: 'Default',
  ip: '',
  other: '{}',
  request_id: 'request-1',
  upstream_request_id: 'upstream-1',
}

const onDemandBundleOther = {
  gateway_skills: {
    stage: 'generation',
    round: 1,
    catalogue_count: 5,
    files: [
      { template_id: 12, skill_id: 'go-testing', title: 'Go Testing' },
      {
        template_id: 12,
        skill_id: 'go-testing',
        title: 'Go Testing',
        path: 'references/table-tests.md',
      },
      { template_id: 12, skill_id: 'sql-style', title: 'SQL Style' },
    ],
  },
}

function TimeCell(props: { other: unknown }) {
  const columns = useCommonLogsColumns(false)
  const table = useReactTable({
    data: [{ ...baseLog, other: JSON.stringify(props.other) }],
    columns,
    getCoreRowModel: getCoreRowModel(),
  })
  const cell = table
    .getRowModel()
    .rows[0].getVisibleCells()
    .find((visibleCell) => visibleCell.column.id === 'created_at')
  if (!cell) throw new Error('usage logs must render a time column')
  return <>{flexRender(cell.column.columnDef.cell, cell.getContext())}</>
}

describe('usage log time column skill badge', () => {
  test('shows the distinct on-demand skill count and names both skills', () => {
    render(<TimeCell other={onDemandBundleOther} />)

    expect(screen.getByText('Skills 2')).toBeInTheDocument()
    expect(
      screen.getByLabelText('Skills 2. Read on demand: Go Testing, SQL Style')
    ).toBeInTheDocument()
  })

  test('lists the skill names in a tooltip on hover', async () => {
    render(<TimeCell other={onDemandBundleOther} />)

    await userEvent.setup().hover(screen.getByText('Skills 2'))

    expect(
      await screen.findByText('Read on demand: Go Testing, SQL Style')
    ).toBeInTheDocument()
  })

  test('shows an unused badge when the final answer read no skills', () => {
    render(
      <TimeCell
        other={{
          gateway_skills: { stage: 'generation', round: 1, catalogue_count: 5 },
        }}
      />
    )

    expect(screen.getByText('Skills unused')).toBeInTheDocument()
    expect(
      screen.getByLabelText(
        'Skills unused. Available skills: 5, none read in this request'
      )
    ).toBeInTheDocument()
  })

  test('shows the discovery round for a private skill-discovery log', () => {
    render(
      <TimeCell
        other={{
          gateway_skills: {
            stage: 'discovery',
            round: 2,
            catalogue_count: 5,
            files: [
              { template_id: 12, skill_id: 'go-testing', title: 'Go Testing' },
            ],
          },
        }}
      />
    )

    expect(screen.getByText('Skill discovery · round 2')).toBeInTheDocument()
    expect(
      screen.getByLabelText(
        'Skill discovery · round 2. Private skill-discovery round, billed separately. Skills read: Go Testing'
      )
    ).toBeInTheDocument()
  })

  test('shows no skill badge for a log without skill data', () => {
    render(<TimeCell other={{ ws: true }} />)

    expect(screen.getByText('WS')).toBeInTheDocument()
    expect(screen.queryByText(/^Skill/)).not.toBeInTheDocument()
  })
})
