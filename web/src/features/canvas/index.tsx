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
import { useQuery } from '@tanstack/react-query'
import { ArrowUpRight, Brush } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { getUserGroupsWithDefault } from '@/features/playground/api'
import type { GroupOption } from '@/features/playground/types'
import { useStatus } from '@/hooks/use-status'
import { getCanvasSettingsFromSidebarModules } from '@/lib/canvas-settings'
import { getCustomNavIcon } from '@/lib/custom-nav'

import {
  buildCanvasLaunchUrl,
  prioritizeCanvasAutoGroup,
  resolveCanvasCapabilityGroup,
  resolveCanvasDefaultGroup,
} from './lib'

const FOLLOW_DEFAULT_GROUP = '__follow_default__'

const capabilityFields = [
  { key: 'textGroup', label: 'Text Group' },
  { key: 'imageGroup', label: 'Image Group' },
  { key: 'audioGroup', label: 'Audio Group' },
  { key: 'videoGroup', label: 'Video Group' },
] as const

type CapabilityGroupKey = (typeof capabilityFields)[number]['key']
type CapabilityGroupSelection = Record<CapabilityGroupKey, string>

function groupLabel(group: GroupOption, translate: (key: string) => string) {
  return group.value === 'auto' ? translate('Automatic selection') : group.label
}

function CanvasGroupSelect({
  label,
  value,
  groups,
  disabled,
  allowFollowDefault,
  onChange,
}: {
  label: string
  value: string
  groups: GroupOption[]
  disabled: boolean
  allowFollowDefault: boolean
  onChange: (value: string) => void
}) {
  const { t } = useTranslation()
  const selectableGroups = groups.filter(
    (group) => group.value !== FOLLOW_DEFAULT_GROUP
  )
  const items = [
    ...(allowFollowDefault
      ? [{ value: FOLLOW_DEFAULT_GROUP, label: t('Follow default group') }]
      : []),
    ...selectableGroups.map((group) => ({
      value: group.value,
      label: groupLabel(group, t),
    })),
  ]
  const selectedValue = allowFollowDefault
    ? value || FOLLOW_DEFAULT_GROUP
    : value

  return (
    <div className='space-y-2'>
      <p className='text-sm font-medium'>{label}</p>
      <Select
        value={selectedValue}
        items={items}
        disabled={disabled || items.length === 0}
        onValueChange={(next) =>
          onChange(next && next !== FOLLOW_DEFAULT_GROUP ? next : '')
        }
      >
        <SelectTrigger aria-label={label} className='w-full'>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {items.map((item) => (
            <SelectItem key={item.value} value={item.value}>
              {item.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

export function CanvasLauncher() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const [selectedGroup, setSelectedGroup] = useState('')
  const [capabilityGroups, setCapabilityGroups] =
    useState<CapabilityGroupSelection | null>(null)
  const canvasSettings = useMemo(
    () =>
      getCanvasSettingsFromSidebarModules(
        (status as Record<string, unknown> | null)?.SidebarModulesAdmin
      ),
    [status]
  )
  const CanvasIcon = getCustomNavIcon(canvasSettings.canvasIcon) ?? Brush

  const { data: groupsData, isLoading } = useQuery({
    queryKey: ['canvas-groups'],
    queryFn: async () => {
      try {
        return await getUserGroupsWithDefault()
      } catch (error) {
        toast.error(
          error instanceof Error
            ? error.message
            : t('Failed to load playground groups')
        )
        return {
          groups: [],
          defaultGroup: '',
          textGroup: '',
          imageGroup: '',
          audioGroup: '',
          videoGroup: '',
        }
      }
    },
  })
  const groups = useMemo(
    () => prioritizeCanvasAutoGroup(groupsData?.groups ?? []),
    [groupsData?.groups]
  )

  useEffect(() => {
    if (selectedGroup || groups.length === 0) return
    setSelectedGroup(
      resolveCanvasDefaultGroup(groups, groupsData?.defaultGroup ?? '')
    )
  }, [groups, groupsData?.defaultGroup, selectedGroup])

  const launchUrl = useMemo(() => {
    if (!selectedGroup || typeof window === 'undefined') return ''
    return buildCanvasLaunchUrl({
      canvasOrigin: canvasSettings.canvasOrigin,
      newApiOrigin: window.location.origin,
      group: selectedGroup,
      textGroup: capabilityGroups?.textGroup,
      imageGroup: capabilityGroups?.imageGroup,
      audioGroup: capabilityGroups?.audioGroup,
      videoGroup: capabilityGroups?.videoGroup,
    })
  }, [canvasSettings.canvasOrigin, capabilityGroups, selectedGroup])

  useEffect(() => {
    if (capabilityGroups || groups.length === 0) return
    setCapabilityGroups({
      textGroup: resolveCanvasCapabilityGroup(
        groups,
        groupsData?.textGroup ?? ''
      ),
      imageGroup: resolveCanvasCapabilityGroup(
        groups,
        groupsData?.imageGroup ?? ''
      ),
      audioGroup: resolveCanvasCapabilityGroup(
        groups,
        groupsData?.audioGroup ?? ''
      ),
      videoGroup: resolveCanvasCapabilityGroup(
        groups,
        groupsData?.videoGroup ?? ''
      ),
    })
  }, [
    capabilityGroups,
    groups,
    groupsData?.audioGroup,
    groupsData?.imageGroup,
    groupsData?.textGroup,
    groupsData?.videoGroup,
  ])

  const handleOpenCanvas = () => {
    if (!launchUrl) return
    window.open(launchUrl, '_blank', 'noopener')
  }

  return (
    <div className='flex min-h-full items-center justify-center p-4 sm:p-8'>
      <Card className='w-full max-w-xl'>
        <CardHeader>
          <div className='bg-muted mb-2 flex h-10 w-10 items-center justify-center rounded-lg'>
            <CanvasIcon className='h-5 w-5' aria-hidden='true' />
          </div>
          <CardTitle>{t('Infinite Canvas')}</CardTitle>
          <CardDescription>
            {t(
              'Choose a group and open Infinite Canvas. The canvas uses your current NewAPI-Mao login session to call models.'
            )}
          </CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          <CanvasGroupSelect
            label={t('Model Group')}
            value={selectedGroup}
            groups={groups}
            disabled={isLoading || groups.length === 0}
            allowFollowDefault={false}
            onChange={setSelectedGroup}
          />
          <div className='grid grid-cols-1 gap-3 sm:grid-cols-2'>
            {capabilityFields.map((field) => (
              <CanvasGroupSelect
                key={field.key}
                label={t(field.label)}
                value={capabilityGroups?.[field.key] ?? ''}
                groups={groups}
                disabled={isLoading || groups.length === 0 || !capabilityGroups}
                allowFollowDefault
                onChange={(value) =>
                  setCapabilityGroups((current) =>
                    current ? { ...current, [field.key]: value } : current
                  )
                }
              />
            ))}
          </div>

          <Button
            className='w-full'
            onClick={handleOpenCanvas}
            disabled={!launchUrl || isLoading}
          >
            <ArrowUpRight className='h-4 w-4' aria-hidden='true' />
            {t('Open in new tab')}
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
