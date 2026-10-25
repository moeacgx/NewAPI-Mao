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
import i18next from 'i18next'
import { describe, expect, test, vi } from 'vitest'

import type { LogOtherData } from '../../types'
import { parseLogOther } from '../format'
import { describeLogSkillUsage, getLogSkillUsage } from '../skill-usage'

const alwaysLoadedTemplate = { id: 7, title: 'Code Review Rules', version: 3 }
const bundleFiles = [
  {
    template_id: 12,
    skill_id: 'go-testing',
    title: 'Go Testing',
    version: 2,
    path: 'SKILL.md',
    offset: 0,
    bytes: 4096,
  },
  {
    template_id: 12,
    skill_id: 'go-testing',
    title: 'Go Testing',
    version: 2,
    path: 'references/table-tests.md',
    offset: 0,
    bytes: 1024,
  },
  {
    template_id: 12,
    skill_id: 'sql-style',
    title: 'SQL Style',
    version: 1,
    path: 'SKILL.md',
    offset: 0,
    bytes: 2048,
  },
]

describe('getLogSkillUsage', () => {
  test('counts always-loaded templates and distinct on-demand skills of the final answer', () => {
    const other: LogOtherData = {
      prompt_templates: [alwaysLoadedTemplate],
      gateway_skills: {
        stage: 'generation',
        round: 2,
        catalogue_count: 5,
        catalogue_queries: [{ query_bytes: 12, offset: 0 }],
        files: bundleFiles,
      },
    }

    expect(getLogSkillUsage(other)).toEqual({
      kind: 'used',
      count: 3,
      alwaysLoaded: ['Code Review Rules'],
      onDemand: ['Go Testing', 'SQL Style'],
    })
  })

  test('reports a discovery round before any used skills and lists distinct pages read so far', () => {
    const other: LogOtherData = {
      prompt_templates: [alwaysLoadedTemplate],
      gateway_skills: {
        stage: 'discovery',
        round: 2,
        catalogue_count: 5,
        files: bundleFiles,
      },
    }

    expect(getLogSkillUsage(other)).toEqual({
      kind: 'discovery',
      round: 2,
      readSkills: ['Go Testing', 'SQL Style'],
    })
  })

  test('reports unused skills when the final answer read no skill pages', () => {
    const other: LogOtherData = {
      gateway_skills: { stage: 'generation', round: 1, catalogue_count: 5 },
    }

    expect(getLogSkillUsage(other)).toEqual({
      kind: 'unused',
      catalogueCount: 5,
    })
  })

  test('counts always-loaded templates without any progressive skill audit', () => {
    expect(
      getLogSkillUsage({ prompt_templates: [alwaysLoadedTemplate] })
    ).toEqual({
      kind: 'used',
      count: 1,
      alwaysLoaded: ['Code Review Rules'],
      onDemand: [],
    })
  })

  test('falls back to the skill id when a read skill has no title', () => {
    const other: LogOtherData = {
      gateway_skills: {
        stage: 'generation',
        catalogue_count: 1,
        files: [{ template_id: 12, skill_id: 'go-testing', title: ' ' }],
      },
    }

    expect(getLogSkillUsage(other)).toMatchObject({
      kind: 'used',
      onDemand: ['go-testing'],
    })
  })

  test('returns null for logs without skill data', () => {
    expect(getLogSkillUsage(null)).toBeNull()
    expect(getLogSkillUsage({})).toBeNull()
    expect(getLogSkillUsage({ prompt_templates: [] })).toBeNull()
  })

  test.each([
    ['prompt templates that are not an array', { prompt_templates: 'x' }],
    ['a gateway audit that is not an object', { gateway_skills: 'x' }],
    [
      'files that are not an array',
      { gateway_skills: { stage: 'generation', files: {} } },
    ],
    [
      'entries without usable identifiers',
      {
        prompt_templates: [null, { id: '7', title: 'Code Review Rules' }],
        gateway_skills: {
          stage: 'generation',
          files: [null, { template_id: '12', skill_id: 'go-testing' }],
        },
      },
    ],
    [
      'a discovery round without a round number',
      { gateway_skills: { stage: 'discovery', files: bundleFiles } },
    ],
    ['a log other value that is not an object', 42],
  ])('returns null for %s', (_name, other) => {
    expect(getLogSkillUsage(other as unknown as LogOtherData)).toBeNull()
  })

  test('returns null when the log other JSON cannot be parsed', () => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined)

    expect(getLogSkillUsage(parseLogOther('{"gateway_skills":'))).toBeNull()
  })
})

describe('describeLogSkillUsage', () => {
  test('groups used skill names by loading mode', () => {
    expect(
      describeLogSkillUsage(
        {
          kind: 'used',
          count: 3,
          alwaysLoaded: ['Code Review Rules'],
          onDemand: ['Go Testing', 'SQL Style'],
        },
        i18next.t
      )
    ).toEqual({
      label: 'Skills 3',
      lines: [
        'Always loaded: Code Review Rules',
        'Read on demand: Go Testing, SQL Style',
      ],
    })
  })

  test('explains a discovery round and the skills it read', () => {
    expect(
      describeLogSkillUsage(
        { kind: 'discovery', round: 2, readSkills: ['Go Testing'] },
        i18next.t
      )
    ).toEqual({
      label: 'Skill discovery · round 2',
      lines: [
        'Private skill-discovery round, billed separately',
        'Skills read: Go Testing',
      ],
    })
  })

  test('omits the read line for a discovery round that read nothing', () => {
    expect(
      describeLogSkillUsage(
        { kind: 'discovery', round: 1, readSkills: [] },
        i18next.t
      ).lines
    ).toEqual(['Private skill-discovery round, billed separately'])
  })

  test('states how many skills were available when none were read', () => {
    expect(
      describeLogSkillUsage({ kind: 'unused', catalogueCount: 5 }, i18next.t)
    ).toEqual({
      label: 'Skills unused',
      lines: ['Available skills: 5, none read in this request'],
    })
  })
})
