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
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

// 仅本地化展示名称；筛选和提交始终使用服务端分类标识。
export function useMcpCategoryLabels(): Record<string, string> {
  const { t } = useTranslation()
  return useMemo(
    () => ({
      affiliate: t('Affiliates'),
      authz: t('Permissions'),
      billing: t('Billing'),
      channels: t('Channels'),
      deployments: t('Deployments'),
      extensions: t('Extensions'),
      game: t('Rewards'),
      groups: t('Groups'),
      marketing: t('Marketing'),
      models: t('Models'),
      notifications: t('Notifications'),
      oauth: t('OAuth'),
      prompts: t('Prompts'),
      security: t('Security'),
      system: t('System'),
      tasks: t('Tasks'),
      tickets: t('Tickets'),
      tokens: t('Token management'),
      usage: t('Usage'),
      users: t('Users'),
      __unknown: t('Unavailable tools'),
    }),
    [t]
  )
}
