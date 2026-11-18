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
import type { CellContext } from '@tanstack/react-table'
import { useContext } from 'react'

import { TokenPromptBadge } from '@/features/prompt-center/components/token-prompt-badge'

import type { ApiKey } from '../types'
import { ApiKeyActivityContext } from './api-key-activity-context'
import { TokenActivityBadge } from './token-activity'

// 名称单元格必须是稳定的组件类型：列定义每次渲染都会重建，内联函数会让 flexRender 把它当成新组件，
// 从而卸载徽标及其已打开的技能弹窗。活动计数经上下文读取，不进入列定义闭包。
export function ApiKeyNameCell(props: CellContext<ApiKey, unknown>) {
  const activity = useContext(ApiKeyActivityContext)
  const apiKey = props.row.original
  return (
    <div className='flex min-w-0 flex-wrap items-center gap-1.5'>
      <span className='min-w-0 font-medium [overflow-wrap:anywhere]'>
        {apiKey.name}
      </span>
      <TokenPromptBadge token={apiKey} />
      <TokenActivityBadge
        tokenId={apiKey.id}
        count={activity.byToken[String(apiKey.id)] || 0}
        stale={activity.stale}
      />
    </div>
  )
}
