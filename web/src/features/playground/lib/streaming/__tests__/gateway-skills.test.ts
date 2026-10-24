/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { describe, expect, it } from 'vitest'

import {
  GATEWAY_SKILLS_RESPONSE_HEADER,
  parseGatewaySkillsHeader,
} from '../gateway-skills'

function encodeHeader(value: unknown): string {
  const bytes = new TextEncoder().encode(JSON.stringify(value))
  let binary = ''
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return btoa(binary)
    .replaceAll('+', '-')
    .replaceAll('/', '_')
    .replaceAll('=', '')
}

const validPayload = {
  version: 1,
  total: 1,
  skills: [
    {
      template_id: 21,
      skill_id: 'code-review',
      version: 4,
      title: 'Code review',
    },
  ],
  truncated: false,
}

describe('gateway skills response header', () => {
  it('parses the base64url metadata from Axios-style headers', async () => {
    const metadata = parseGatewaySkillsHeader({
      [GATEWAY_SKILLS_RESPONSE_HEADER]: [encodeHeader(validPayload)],
    })

    expect(metadata).toEqual({
      version: 1,
      total: 1,
      skills: [
        {
          templateId: 21,
          skillId: 'code-review',
          version: 4,
          title: 'Code review',
        },
      ],
      truncated: false,
    })
  })

  it('accepts a Fetch Headers object and ignores malformed metadata', () => {
    const encoded = encodeHeader(validPayload)
    const headers = new Headers({ 'X-New-API-Gateway-Skills': encoded })
    expect(parseGatewaySkillsHeader(headers)?.total).toBe(1)

    expect(
      parseGatewaySkillsHeader({
        'x-new-api-gateway-skills': ['not-base64'],
      })
    ).toBeUndefined()
    expect(
      parseGatewaySkillsHeader({
        'x-new-api-gateway-skills': [encodeHeader({ ...validPayload, total: 2 })],
      })
    ).toBeUndefined()
  })

  it('rejects ambiguous, oversized, and invalid skill entries', () => {
    const encoded = encodeHeader(validPayload)
    expect(
      parseGatewaySkillsHeader({
        'x-new-api-gateway-skills': [encoded, encoded],
      })
    ).toBeUndefined()
    expect(
      parseGatewaySkillsHeader({
        'x-new-api-gateway-skills': [`${encoded}${'a'.repeat(6 * 1024)}`],
      })
    ).toBeUndefined()
    expect(
      parseGatewaySkillsHeader({
        'x-new-api-gateway-skills': [
          encodeHeader({
            ...validPayload,
            skills: [{ ...validPayload.skills[0], version: 0 }],
          }),
        ],
      })
    ).toBeUndefined()
  })
})
