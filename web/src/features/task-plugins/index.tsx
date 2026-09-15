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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { handleServerError } from '@/lib/handle-server-error'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { getTaskPluginEnabledOption, setTaskPluginEnabledOption } from './api'
import { PluginDetailSheet } from './components/plugin-detail-sheet'
import { PluginsTable } from './components/plugins-table'
import { UploadPluginDialog } from './components/upload-plugin-dialog'
import type { TaskPluginListItem } from './types'

export function TaskPlugins() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const canManage = useAuthStore(
    (state) => (state.auth.user?.role ?? 0) >= ROLE.SUPER_ADMIN
  )
  const [detail, setDetail] = useState<TaskPluginListItem | null>(null)
  const [uploadOpen, setUploadOpen] = useState(false)
  const runtime = useQuery({
    queryKey: ['task-plugin-enabled'],
    queryFn: getTaskPluginEnabledOption,
  })
  const enabledMutation = useMutation({
    mutationFn: setTaskPluginEnabledOption,
    onSuccess: (_, enabled) => {
      queryClient.setQueryData(['task-plugin-enabled'], enabled)
      toast.success(t('Task plugin setting updated'))
    },
    onError: handleServerError,
  })
  return (
    <>
      <SectionPageLayout fixedContent>
        <SectionPageLayout.Title>{t('Task Plugins')}</SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <Switch
            id='task-plugin-enabled-switch'
            checked={runtime.data ?? false}
            disabled={
              !canManage || !runtime.isSuccess || enabledMutation.isPending
            }
            onCheckedChange={(checked) => enabledMutation.mutate(checked)}
          />
          <Label htmlFor='task-plugin-enabled-switch'>
            {t('Enable task plugins')}
          </Label>
          <Button disabled>{t('Marketplace')}</Button>
          <Button disabled={!canManage} onClick={() => setUploadOpen(true)}>
            {t('Upload plugin')}
          </Button>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='bg-card flex min-h-0 flex-1 flex-col gap-3 rounded-xl border p-4'>
            <p className='text-muted-foreground text-sm'>
              {t(
                'Built-in and Root-uploaded custom plugins are supported. Marketplace, remote resources and sandbox are unavailable in this version.'
              )}
            </p>
            <p className='text-muted-foreground text-sm'>
              {t(
                'Disabling task plugins stops new submissions. Existing tasks continue with their pinned version.'
              )}
            </p>
            {runtime.isError && (
              <ErrorState
                description={runtime.error.message}
                onRetry={() => void runtime.refetch()}
              />
            )}
            <PluginsTable onDetails={setDetail} canManage={canManage} />
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>
      <PluginDetailSheet
        canManage={canManage}
        plugin={detail}
        onOpenChange={(open) => {
          if (!open) setDetail(null)
        }}
      />
      <UploadPluginDialog open={uploadOpen} onOpenChange={setUploadOpen} />
    </>
  )
}
