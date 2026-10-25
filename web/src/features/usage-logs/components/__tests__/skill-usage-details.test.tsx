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
import { describe, expect, test, vi } from 'vitest'

import type { UsageLog } from '../../data/schema'
import { DetailsDialog } from '../dialogs/details-dialog'

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

function renderDetails(other: unknown) {
  render(
    <DetailsDialog
      log={{ ...baseLog, other: JSON.stringify(other) }}
      isAdmin={false}
      open
      onOpenChange={() => undefined}
    />
  )
}

describe('usage log details skill line', () => {
  test('lists always-loaded and on-demand skills of the request', () => {
    renderDetails({
      prompt_templates: [{ id: 7, title: 'Code Review Rules', version: 3 }],
      gateway_skills: {
        stage: 'generation',
        round: 2,
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
    })

    expect(screen.getByText('Skills')).toBeInTheDocument()
    expect(screen.getByText('Skills 3')).toBeInTheDocument()
    expect(
      screen.getByText('Always loaded: Code Review Rules')
    ).toBeInTheDocument()
    expect(
      screen.getByText('Read on demand: Go Testing, SQL Style')
    ).toBeInTheDocument()
  })

  test('explains a private skill-discovery round', () => {
    renderDetails({
      gateway_skills: { stage: 'discovery', round: 1, catalogue_count: 5 },
    })

    expect(screen.getByText('Skill discovery · round 1')).toBeInTheDocument()
    expect(
      screen.getByText('Private skill-discovery round, billed separately')
    ).toBeInTheDocument()
  })

  test('omits the skill line for a log without skill data', () => {
    renderDetails({})

    expect(screen.getByText('Request ID')).toBeInTheDocument()
    expect(screen.queryByText('Skills')).not.toBeInTheDocument()
  })
})
