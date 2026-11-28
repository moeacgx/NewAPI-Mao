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
import { MessageSquare } from 'lucide-react'
import { useTranslation } from 'react-i18next'

// 只展示列表接口带回的对话数量。0 或缺失时不渲染，内部备注不在这里计数。
export function SupplierMessageCount(props: { count?: number }) {
  const { t } = useTranslation()
  const count = props.count ?? 0
  if (count <= 0) return null
  return (
    <span
      role='img'
      aria-label={t('{{count}} messages', { count })}
      className='text-muted-foreground inline-flex shrink-0 items-center gap-1 text-xs tabular-nums'
    >
      <MessageSquare aria-hidden='true' className='size-3.5' />
      <span>{count}</span>
    </span>
  )
}
