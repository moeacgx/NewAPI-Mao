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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import { listMcpLogs } from '../api'
import { McpPagination, McpTime } from './shared'

export function LogsPanel() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const query = useQuery({
    queryKey: ['mcp', 'logs', page],
    queryFn: () => listMcpLogs(page),
  })
  return (
    <Card>
      <CardHeader className='border-b'>
        <CardTitle>{t('Call logs')}</CardTitle>
        <CardDescription>
          {t('Review tool calls made with your MCP keys.')}
        </CardDescription>
        <Button
          type='button'
          variant='outline'
          className='w-fit'
          disabled={query.isFetching}
          onClick={() => void query.refetch()}
        >
          {t('Refresh')}
        </Button>
      </CardHeader>
      <CardContent>
        {query.isPending && <LoadingState />}
        {query.isError && (
          <ErrorState
            description={query.error.message}
            onRetry={() => void query.refetch()}
          />
        )}
        {query.isSuccess && query.data.items.length === 0 && (
          <EmptyState title={t('No MCP calls yet')} />
        )}
        {query.isSuccess && query.data.items.length > 0 && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Created at')}</TableHead>
                <TableHead>{t('Tool')}</TableHead>
                <TableHead>{t('Key ID')}</TableHead>
                <TableHead>{t('Status')}</TableHead>
                <TableHead>{t('Request ID')}</TableHead>
                <TableHead>{t('Duration (ms)')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {query.data.items.map((log) => (
                <TableRow key={log.id}>
                  <TableCell>
                    <McpTime value={log.created_at} />
                  </TableCell>
                  <TableCell className='max-w-64 font-mono break-all whitespace-normal'>
                    {log.tool_name}
                  </TableCell>
                  <TableCell>{log.key_id}</TableCell>
                  <TableCell>
                    <Badge variant={log.success ? 'secondary' : 'destructive'}>
                      {log.success ? t('Success') : t('Failed')}
                    </Badge>
                  </TableCell>
                  <TableCell className='max-w-72 font-mono break-all whitespace-normal'>
                    {log.request_id || '—'}
                  </TableCell>
                  <TableCell>{log.duration_ms}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
        <McpPagination
          page={page}
          total={query.data?.total ?? 0}
          busy={query.isFetching}
          onChange={setPage}
        />
      </CardContent>
    </Card>
  )
}
