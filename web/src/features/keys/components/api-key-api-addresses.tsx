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
import { isAxiosError, isCancel } from 'axios'
import { Copy, Globe } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from '@/components/ui/popover'
import { api } from '@/lib/api'
import { copyToClipboard } from '@/lib/copy-to-clipboard'

type ApiAddress = { url: string; route?: string; description?: string }

export function ApiKeyApiAddresses() {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const addresses = useQuery({
    queryKey: ['keys', 'api-addresses'],
    enabled: open,
    retry: false,
    staleTime: 0,
    refetchOnWindowFocus: false,
    queryFn: async ({ signal }) => {
      const response = await api
        .get<{
          success?: boolean
          data?: Record<string, unknown>
        }>('/api/status', { signal, skipErrorHandler: true })
        .catch((error: unknown) => {
          // 局部故障不能触发全局 Axios 500 跳转；认证与取消保留原有语义。
          if (
            isAxiosError(error) &&
            !isCancel(error) &&
            ![401, 403].includes(error.response?.status ?? 0)
          ) {
            throw new Error('Failed to load API addresses', { cause: error })
          }
          throw error
        })
      if (
        response.data.success === false ||
        !response.data.data ||
        typeof response.data.data !== 'object'
      ) {
        throw new Error('Failed to load API addresses')
      }
      const status = response.data.data
      const items: ApiAddress[] = []
      if (status.api_info_enabled !== false && Array.isArray(status.api_info)) {
        for (const item of status.api_info) {
          if (!item || typeof item.url !== 'string' || !item.url) continue
          items.push({
            url: item.url,
            route: typeof item.route === 'string' ? item.route : undefined,
            description:
              typeof item.description === 'string'
                ? item.description
                : undefined,
          })
        }
      }
      const serverAddress =
        typeof status.server_address === 'string'
          ? status.server_address.trim()
          : ''
      return { items, serverAddress }
    },
  })
  const items = addresses.data?.items.length
    ? addresses.data.items
    : [
        {
          url: addresses.data?.serverAddress || window.location.origin,
          route: t(
            addresses.data?.serverAddress
              ? 'Default API address'
              : 'Current domain'
          ),
        },
      ]

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger render={<Button variant='outline' size='sm' />}>
        <Globe className='size-4' aria-hidden='true' />
        {t('API addresses')}
      </PopoverTrigger>
      <PopoverContent
        align='end'
        collisionPadding={12}
        className='w-96 max-w-[calc(100vw-1.5rem)] gap-3 border p-4'
      >
        <PopoverHeader className='border-b pb-3'>
          <PopoverTitle>{t('API addresses')}</PopoverTitle>
        </PopoverHeader>
        {addresses.isFetching && (
          <div role='status' className='text-muted-foreground py-4 text-center'>
            {t('Loading...')}
          </div>
        )}
        {!addresses.isFetching && addresses.isError && (
          <div className='space-y-3'>
            <p role='alert' className='text-destructive'>
              {t('Failed to load API addresses')}
            </p>
            <Button
              variant='outline'
              size='sm'
              onClick={() => void addresses.refetch()}
            >
              {t('Retry')}
            </Button>
          </div>
        )}
        {!addresses.isFetching && !addresses.isError && (
          <ul className='max-h-[min(24rem,60dvh)] min-w-0 space-y-3 overflow-y-auto'>
            {items.map((item, index) => (
              // 配置允许重复线路，快照内用序号区分完全相同的地址；列表没有本地编辑状态。
              <li
                // eslint-disable-next-line react/no-array-index-key
                key={`${index}-${item.url}`}
                aria-label={item.route || item.url}
                className='bg-muted/30 min-w-0 space-y-2 rounded-md border p-3'
              >
                <p className='font-medium [overflow-wrap:anywhere]'>
                  {item.route || item.url}
                </p>
                <div className='flex min-w-0 items-start gap-2'>
                  <code className='min-w-0 flex-1 text-xs [overflow-wrap:anywhere]'>
                    {item.url}
                  </code>
                  <Button
                    variant='ghost'
                    size='icon-sm'
                    className='shrink-0'
                    aria-label={t('Copy address')}
                    onClick={async (event) => {
                      const button = event.currentTarget
                      const copied = await copyToClipboard(item.url)
                      // 兼容复制会临时聚焦隐藏文本框，完成后恢复当前按钮焦点。
                      if (button.isConnected) button.focus()
                      if (copied) toast.success(t('Copied successfully'))
                      else toast.error(t('Copy failed'))
                    }}
                  >
                    <Copy className='size-4' aria-hidden='true' />
                  </Button>
                </div>
                {item.description && (
                  <p className='text-muted-foreground text-xs [overflow-wrap:anywhere]'>
                    {item.description}
                  </p>
                )}
              </li>
            ))}
          </ul>
        )}
      </PopoverContent>
    </Popover>
  )
}
