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
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

import { listMcpTools } from '../api'
import { ToolCatalog } from './tool-catalog'

export function ToolsPanel() {
  const { t } = useTranslation()
  const query = useQuery({ queryKey: ['mcp', 'tools'], queryFn: listMcpTools })
  return (
    <Card>
      <CardHeader className='border-b'>
        <CardTitle>{t('Tool reference')}</CardTitle>
        <CardDescription>
          {t('Browse available tools and their input schemas.')}
        </CardDescription>
      </CardHeader>
      <CardContent className='space-y-4'>
        {query.isPending && <LoadingState />}
        {query.isError && (
          <ErrorState
            description={query.error.message}
            onRetry={() => void query.refetch()}
          />
        )}
        {query.isSuccess && <ToolCatalog tools={query.data} />}
      </CardContent>
    </Card>
  )
}
