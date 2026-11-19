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
import { createContext } from 'react'

export interface ApiKeyActivityValue {
  byToken: Record<string, number>
  stale: boolean
}

// 活动计数经上下文传给名称单元格，避免进入每次渲染都会重建的列定义闭包。
export const ApiKeyActivityContext = createContext<ApiKeyActivityValue>({
  byToken: {},
  stale: false,
})
