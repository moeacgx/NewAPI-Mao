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

import { ConfirmDialog } from '@/components/confirm-dialog'
import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Alert, AlertDescription } from '@/components/ui/alert'
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

import { listMcpKeys, revokeMcpKey } from '../api'
import type { McpKey } from '../types'
import { CreateKeyForm } from './create-key-form'
import { KeyPermissionsDialog } from './key-permissions-dialog'
import { McpPagination, McpTime } from './shared'

export function KeysPanel() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [page, setPage] = useState(1)
  const [pendingRevoke, setPendingRevoke] = useState<McpKey | null>(null)
  const query = useQuery({
    queryKey: ['mcp', 'keys', page],
    queryFn: () => listMcpKeys(page),
  })
  const revocation = useMutation({
    mutationFn: revokeMcpKey,
    retry: false,
    onSuccess: () => {
      setPendingRevoke(null)
      void client.invalidateQueries({ queryKey: ['mcp', 'keys'] })
    },
  })
  return (
    <div className='space-y-4'>
      <Card>
        <CardHeader className='border-b'>
          <CardTitle>{t('Call keys')}</CardTitle>
          <CardDescription>
            {t('Manage the MCP keys you created.')}
          </CardDescription>
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
            <EmptyState title={t('No MCP keys yet')} />
          )}
          {query.isSuccess && query.data.items.length > 0 && (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Name')}</TableHead>
                  <TableHead>{t('Allowed tools')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                  <TableHead>{t('Created at')}</TableHead>
                  <TableHead>{t('Expiration time')}</TableHead>
                  <TableHead>{t('Last used')}</TableHead>
                  <TableHead>{t('Actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {query.data.items.map((key) => {
                  let status = t('Active')
                  if (
                    key.expires_at > 0 &&
                    key.expires_at <= Date.now() / 1000
                  ) {
                    status = t('Expired')
                  }
                  if (key.revoked_at > 0) status = t('Revoked')
                  return (
                    <TableRow key={key.id}>
                      <TableCell className='max-w-60 break-words whitespace-normal'>
                        <div>{key.name}</div>
                        <code className='text-muted-foreground text-xs'>
                          {key.key_prefix}
                        </code>
                      </TableCell>
                      <TableCell>
                        <div className='flex flex-col items-start gap-1'>
                          <span className='text-sm'>
                            {t('{{total}} authorized tools', {
                              total: key.tools.length,
                            })}
                          </span>
                          <KeyPermissionsDialog mcpKey={key} />
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant='outline'>{status}</Badge>
                        {key.revoked_at > 0 && (
                          <div className='mt-1 text-xs'>
                            <McpTime value={key.revoked_at} />
                          </div>
                        )}
                      </TableCell>
                      <TableCell>
                        <McpTime value={key.created_at} />
                      </TableCell>
                      <TableCell>
                        {key.expires_at > 0 ? (
                          <McpTime value={key.expires_at} />
                        ) : (
                          t('Never expires')
                        )}
                      </TableCell>
                      <TableCell>
                        {key.last_used_at > 0 ? (
                          <McpTime value={key.last_used_at} />
                        ) : (
                          t('Never used')
                        )}
                      </TableCell>
                      <TableCell>
                        <Button
                          type='button'
                          variant='outline'
                          disabled={key.revoked_at > 0 || revocation.isPending}
                          onClick={() => {
                            revocation.reset()
                            setPendingRevoke(key)
                          }}
                        >
                          {t('Revoke')}
                        </Button>
                      </TableCell>
                    </TableRow>
                  )
                })}
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
      <CreateKeyForm />
      <ConfirmDialog
        open={pendingRevoke !== null}
        onOpenChange={(open) => {
          if (!open && !revocation.isPending) setPendingRevoke(null)
        }}
        title={t('Revoke MCP key?')}
        desc={t('Clients using {{name}} will immediately lose access.', {
          name: pendingRevoke?.name,
        })}
        confirmText={t('Confirm revocation')}
        destructive
        isLoading={revocation.isPending}
        handleConfirm={() => {
          if (pendingRevoke && !revocation.isPending) {
            revocation.mutate(pendingRevoke.id)
          }
        }}
      >
        {revocation.isError && (
          <Alert variant='destructive'>
            <AlertDescription>{revocation.error.message}</AlertDescription>
          </Alert>
        )}
      </ConfirmDialog>
    </div>
  )
}
