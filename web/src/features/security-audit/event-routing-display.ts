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
import type {
  SecurityAuditChannelGroup,
  SecurityAuditEvent,
  SecurityAuditTokenGroup,
} from './types'

type AuditRouteEvent = Pick<
  SecurityAuditEvent,
  | 'stage'
  | 'channel_id'
  | 'channel_name'
  | 'channel_groups'
  | 'group_id'
  | 'group_code'
  | 'group_name'
  | 'group_display_name'
>

type AuditTokenGroupEvent = Pick<
  SecurityAuditEvent,
  'token_group_mode' | 'token_groups'
>

export type AuditChannelReference =
  | { kind: 'assigned'; id: number; name: string }
  | { kind: 'unassigned' }
  | { kind: 'historical' }

export type AuditGroupReference = SecurityAuditChannelGroup & {
  source: 'channel' | 'route' | 'token'
}

export type AuditTokenGroupReference =
  | { kind: 'historical' }
  | { kind: 'unbound' }
  | { kind: 'auto' }
  | {
      kind: 'configured'
      mode: 'explicit' | 'inherit'
      groups: AuditGroupReference[]
    }

function isPreRoutingStage(stage: string): boolean {
  return !stage.includes('response')
}

function getAuditGroupName(group: SecurityAuditChannelGroup): string {
  const current =
    Object.hasOwn(group, 'display_name') &&
    typeof group.display_name === 'string'
      ? group.display_name.trim()
      : ''
  return current || String(group.name ?? '').trim()
}

export function getAuditChannelReference(
  event: AuditRouteEvent
): AuditChannelReference {
  const channelId = Number(event.channel_id)
  if (Number.isInteger(channelId) && channelId > 0) {
    return {
      kind: 'assigned',
      id: channelId,
      name: String(event.channel_name ?? '').trim(),
    }
  }
  return isPreRoutingStage(
    String(event.stage ?? '')
      .trim()
      .toLowerCase()
  )
    ? { kind: 'unassigned' }
    : { kind: 'historical' }
}

export function getAuditRouteGroupReference(
  event: AuditRouteEvent
): AuditGroupReference | null {
  const routeId = Number(event.group_id)
  const routeCode = String(event.group_code ?? '').trim()
  const currentName =
    Object.hasOwn(event, 'group_display_name') &&
    typeof event.group_display_name === 'string'
      ? event.group_display_name.trim()
      : ''
  const routeName = currentName || String(event.group_name ?? '').trim()
  if (
    (!Number.isInteger(routeId) || routeId <= 0) &&
    !routeCode &&
    !routeName
  ) {
    return null
  }

  return {
    id: Number.isInteger(routeId) && routeId > 0 ? routeId : 0,
    code: routeCode,
    name: routeName,
    source: 'route',
  }
}

export function getAuditChannelGroupReferences(
  event: AuditRouteEvent
): AuditGroupReference[] {
  const seen = new Set<string>()
  return (event.channel_groups ?? [])
    .map((group) => ({
      id: Number(group.id) || 0,
      code: String(group.code ?? '').trim(),
      name: getAuditGroupName(group),
      source: 'channel' as const,
    }))
    .filter((group) => {
      if (group.id <= 0 && !group.code && !group.name) return false
      const key = `${group.id}\u0000${group.code}\u0000${group.name}`
      if (seen.has(key)) return false
      seen.add(key)
      return true
    })
}

function normalizeTokenGroups(
  groups: readonly SecurityAuditTokenGroup[] | null | undefined
): AuditGroupReference[] {
  const seen = new Set<string>()
  return (groups ?? [])
    .map((group) => ({
      id: Number(group.id) || 0,
      code: String(group.code ?? '').trim(),
      name: getAuditGroupName(group),
      source: 'token' as const,
    }))
    .filter((group) => {
      if (group.id <= 0 && !group.code && !group.name) return false
      const key = `${group.id}\u0000${group.code}\u0000${group.name}`
      if (seen.has(key)) return false
      seen.add(key)
      return true
    })
}

export function getAuditTokenGroupReference(
  event: AuditTokenGroupEvent
): AuditTokenGroupReference {
  const mode = String(event.token_group_mode ?? '')
    .trim()
    .toLowerCase()

  if (!mode) return { kind: 'historical' }
  if (mode === 'none') return { kind: 'unbound' }
  if (mode === 'auto') return { kind: 'auto' }

  if (mode !== 'explicit' && mode !== 'inherit') {
    return { kind: 'historical' }
  }

  return {
    kind: 'configured',
    mode,
    groups: normalizeTokenGroups(event.token_groups),
  }
}

export function formatAuditGroupReference(group: AuditGroupReference): string {
  return group.name || group.code || (group.id > 0 ? `#${group.id}` : '')
}
