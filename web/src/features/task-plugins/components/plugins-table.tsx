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
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { handleServerError } from '@/lib/handle-server-error'

import { listTaskPlugins, setTaskPluginStatus } from '../api'
import type { TaskPluginListItem } from '../types'

export function PluginsTable(props: {
  canManage?: boolean
  onDetails: (plugin: TaskPluginListItem) => void
}) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const query = useQuery({
    queryKey: ['task-plugins'],
    queryFn: listTaskPlugins,
  })
  const status = useMutation({
    mutationFn: (request: { key: string; enabled: boolean }) =>
      setTaskPluginStatus(request.key, request.enabled),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['task-plugins'] })
      client.invalidateQueries({ queryKey: ['task-plugin'] })
      client.invalidateQueries({ queryKey: ['task-plugin-versions'] })
      client.invalidateQueries({ queryKey: ['task-plugin-options'] })
    },
    onError: handleServerError,
  })
  if (query.isPending) return <LoadingState />
  if (query.isError) {
    return (
      <ErrorState
        description={query.error.message}
        onRetry={() => void query.refetch()}
      />
    )
  }
  if (!query.data?.length) {
    return <EmptyState title={t('No task plugins found')} />
  }
  return (
    <div className='overflow-auto rounded-lg border'>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Name')}</TableHead>
            <TableHead>{t('Version')}</TableHead>
            <TableHead>{t('Status')}</TableHead>
            <TableHead>{t('Enabled')}</TableHead>
            <TableHead>{t('Usage')}</TableHead>
            <TableHead>{t('Actions')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {query.data.map((plugin) => (
            <TableRow key={plugin.meta.key}>
              <TableCell>
                <div>{plugin.meta.name}</div>
                <div className='text-muted-foreground text-xs'>
                  {plugin.meta.key}
                </div>
              </TableCell>
              <TableCell>{plugin.meta.version}</TableCell>
              <TableCell>
                {plugin.active ? t('Active') : t('Not activated')}
              </TableCell>
              <TableCell>
                <Switch
                  aria-label={t('Enable plugin {{key}}', {
                    key: plugin.meta.key,
                  })}
                  checked={plugin.enabled}
                  disabled={
                    !props.canManage || !plugin.active || status.isPending
                  }
                  onCheckedChange={(enabled) =>
                    status.mutate({ key: plugin.meta.key, enabled })
                  }
                />
              </TableCell>
              <TableCell>
                {t('{{channels}} channels, {{tasks}} in-flight tasks', {
                  channels: plugin.channel_count,
                  tasks: plugin.in_flight_count,
                })}
              </TableCell>
              <TableCell>
                <Button
                  variant='outline'
                  size='sm'
                  onClick={() => props.onDetails(plugin)}
                >
                  {t('Details')}
                </Button>
                <Button variant='ghost' size='sm' disabled>
                  {t('Delete')}
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
