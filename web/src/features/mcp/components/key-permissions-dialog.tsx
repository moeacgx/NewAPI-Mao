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

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'

import { listMcpTools } from '../api'
import type { McpKey } from '../types'
import { ToolCatalog } from './tool-catalog'

export function KeyPermissionsDialog(props: { mcpKey: McpKey }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const query = useQuery({
    queryKey: ['mcp', 'tools'],
    queryFn: listMcpTools,
    enabled: open,
  })
  // 失败时不使用缓存推断权限，原始白名单始终可见。
  const catalog = new Map(
    (open && query.isSuccess ? query.data : []).map((tool) => [tool.name, tool])
  )
  const tools = (open ? props.mcpKey.tools : []).map(
    (name) =>
      catalog.get(name) ?? {
        name,
        category: '__unknown',
        description: t(
          'This tool is not in the current catalog. Its access type is unknown.'
        ),
      }
  )

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger
        render={<Button variant='link' size='sm' className='h-auto px-0' />}
      >
        {t('View permissions')}
      </DialogTrigger>
      <DialogContent
        showCloseButton={false}
        className='max-h-[85dvh] overflow-y-auto sm:max-w-3xl'
      >
        <DialogHeader>
          <div className='flex items-start justify-between gap-3'>
            <DialogTitle className='min-w-0 leading-relaxed break-words'>
              {t('Permissions for {{name}}', { name: props.mcpKey.name })}
            </DialogTitle>
            <DialogClose render={<Button variant='outline' size='sm' />}>
              {t('Close')}
            </DialogClose>
          </div>
          <DialogDescription>
            {t('{{total}} authorized tools', {
              total: props.mcpKey.tools.length,
            })}
          </DialogDescription>
        </DialogHeader>
        {query.isPending && (
          <p role='status'>
            {t('Loading tool details. Saved tool names are available below.')}
          </p>
        )}
        {query.isError && (
          <Alert>
            <AlertDescription>
              <p>
                {t(
                  'Tool catalog unavailable. Saved tool names are still shown; access types cannot be verified.'
                )}
              </p>
              <Button
                variant='outline'
                size='sm'
                disabled={query.isFetching}
                onClick={() => void query.refetch()}
              >
                {t('Retry')}
              </Button>
            </AlertDescription>
          </Alert>
        )}
        {open && <ToolCatalog tools={tools} permissions />}
      </DialogContent>
    </Dialog>
  )
}
