/*
Copyright (C) 2025 QuantumNous

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

// 分类标识保持服务端契约，仅翻译展示名称。
export function categoryLabel(category, t) {
  const labels = {
    affiliate: t('MCP category: Affiliate'),
    authz: t('MCP category: Authorization'),
    billing: t('MCP category: Billing'),
    channels: t('MCP category: Channels'),
    deployments: t('MCP category: Deployments'),
    extensions: t('MCP category: Extensions'),
    game: t('MCP category: Game'),
    groups: t('MCP category: Groups'),
    marketing: t('MCP category: Marketing'),
    models: t('MCP category: Models'),
    notifications: t('MCP category: Notifications'),
    oauth: t('MCP category: OAuth'),
    prompts: t('MCP category: Prompts'),
    security: t('MCP category: Security'),
    system: t('MCP category: System'),
    tasks: t('MCP category: Tasks'),
    tickets: t('MCP category: Tickets'),
    tokens: t('MCP category: Tokens'),
    usage: t('MCP category: Usage'),
    users: t('MCP category: Users'),
    unavailable: t('MCP category: Unavailable'),
  };
  return labels[category] ?? category;
}
