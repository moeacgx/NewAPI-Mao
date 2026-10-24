/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/
import type {
  GatewaySkillUsage,
  GatewaySkillsMetadata,
} from '../../types'

export const GATEWAY_SKILLS_RESPONSE_HEADER = 'x-new-api-gateway-skills'
export const SKILLS_DISPLAY_REQUEST_HEADER = 'X-New-API-Skills-Display'
export const SKILLS_DISPLAY_REQUEST_VALUE = 'metadata'

const MAX_HEADER_VALUE_BYTES = 6 * 1024
const MAX_SKILLS = 256
const MAX_TOTAL = 1_000_000
const MAX_SKILL_ID_LENGTH = 256
const MAX_TITLE_LENGTH = 512

type HeaderRecord = Record<string, unknown>

function isRecord(value: unknown): value is HeaderRecord {
  return typeof value === 'object' && value !== null
}

function readHeaderValue(headers: unknown): string | undefined {
  if (!isRecord(headers)) return undefined

  const get = headers.get
  if (typeof get === 'function') {
    try {
      const value = get.call(headers, GATEWAY_SKILLS_RESPONSE_HEADER)
      if (typeof value === 'string') return value
    } catch {
      return undefined
    }
  }

  const key = Object.keys(headers).find(
    (candidate) => candidate.toLowerCase() === GATEWAY_SKILLS_RESPONSE_HEADER
  )
  if (!key) return undefined

  const value = headers[key]
  if (typeof value === 'string') return value
  if (Array.isArray(value) && value.length === 1 && typeof value[0] === 'string') {
    return value[0]
  }
  return undefined
}

function decodeBase64Url(value: string): string | undefined {
  if (
    value.length === 0 ||
    value.length > MAX_HEADER_VALUE_BYTES ||
    !/^[A-Za-z0-9_-]+$/.test(value) ||
    value.length % 4 === 1
  ) {
    return undefined
  }

  try {
    const base64 = value.replaceAll('-', '+').replaceAll('_', '/')
    const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), '=')
    const binary = atob(padded)
    const bytes = Uint8Array.from(binary, (character) => character.charCodeAt(0))
    return new TextDecoder('utf-8', { fatal: true }).decode(bytes)
  } catch {
    return undefined
  }
}

function parseSkill(value: unknown): GatewaySkillUsage | undefined {
  if (!isRecord(value)) return undefined
  const templateId = value.template_id
  const skillId = value.skill_id
  const version = value.version
  const title = value.title

  if (
    typeof templateId !== 'number' ||
    !Number.isSafeInteger(templateId) ||
    templateId < 1 ||
    typeof skillId !== 'string' ||
    skillId.length === 0 ||
    skillId.length > MAX_SKILL_ID_LENGTH ||
    typeof version !== 'number' ||
    !Number.isSafeInteger(version) ||
    version < 1 ||
    typeof title !== 'string' ||
    title.trim().length === 0 ||
    title.length > MAX_TITLE_LENGTH
  ) {
    return undefined
  }

  return { templateId, skillId, version, title }
}

export function parseGatewaySkillsHeader(
  headers: unknown
): GatewaySkillsMetadata | undefined {
  const encoded = readHeaderValue(headers)
  if (!encoded) return undefined

  const decoded = decodeBase64Url(encoded)
  if (!decoded) return undefined

  try {
    const value: unknown = JSON.parse(decoded)
    if (!isRecord(value)) return undefined
    const total = value.total
    const skills = value.skills

    if (
      value.version !== 1 ||
      typeof total !== 'number' ||
      !Number.isSafeInteger(total) ||
      total < 0 ||
      total > MAX_TOTAL ||
      typeof value.truncated !== 'boolean' ||
      !Array.isArray(skills) ||
      skills.length > MAX_SKILLS
    ) {
      return undefined
    }

    const parsedSkills = skills.map(parseSkill)
    if (
      parsedSkills.some((skill): skill is undefined => skill === undefined) ||
      total < skills.length ||
      (!value.truncated && total !== skills.length)
    ) {
      return undefined
    }

    return {
      version: 1,
      total,
      skills: parsedSkills as GatewaySkillUsage[],
      truncated: value.truncated,
    }
  } catch {
    return undefined
  }
}
