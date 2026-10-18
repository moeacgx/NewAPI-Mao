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
import { api } from '@/lib/api'

import type {
  ExtensionMarketplaceConfig,
  ExtensionCatalogModule,
} from './marketplace-download'
import type { ExtensionListResponse, ExtensionModule } from './types'

export { getExtensionPageUrl } from './urls'

export async function getExtensions(options: { all?: boolean } = {}) {
  const res = await api.get<ExtensionListResponse>('/api/extensions/', {
    params: options.all ? { all: 'true' } : undefined,
    skipBusinessError: true,
  })
  return res.data.data ?? { root: '', modules: [] }
}

export async function getExtensionAdminList() {
  const res = await api.get<ExtensionListResponse>('/api/extension-admin/', {
    params: { all: 'true' },
    skipBusinessError: true,
  })
  return res.data.data ?? { root: '', modules: [] }
}

export async function refreshExtensions() {
  const res = await api.post<ExtensionListResponse>(
    '/api/extension-admin/refresh',
    undefined,
    { skipBusinessError: true }
  )
  return res.data
}

export async function uploadExtension(file: File) {
  const formData = new FormData()
  formData.append('file', file)
  const res = await api.post<ExtensionListResponse>(
    '/api/extension-admin/upload',
    formData,
    { skipBusinessError: true }
  )
  return res.data
}

export async function getExtensionMarketplaceConfig(): Promise<ExtensionMarketplaceConfig> {
  const res = await api.get<{
    success: boolean
    message?: string
    data?: ExtensionMarketplaceConfig
  }>('/api/extension-admin/marketplace', { skipErrorHandler: true })
  if (!res.data.success || !res.data.data) {
    throw new Error(res.data.message || 'Failed to load online modules')
  }
  return res.data.data
}

export async function uploadMarketplaceExtension(
  file: File,
  entry: ExtensionCatalogModule,
  catalogUrl: string
) {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('archiveSha256', entry.sha256.toLowerCase())
  formData.append('expectedId', entry.id)
  formData.append('expectedVersion', entry.version)
  formData.append('catalogUrl', catalogUrl)
  formData.append('archivePath', entry.path)
  const res = await api.post<ExtensionListResponse>(
    '/api/extension-admin/upload',
    formData,
    { skipErrorHandler: true }
  )
  if (!res.data.success) {
    throw new Error(res.data.message || 'Failed to install online module')
  }
  return res.data
}

export async function setExtensionEnabled(id: string, enabled: boolean) {
  const res = await api.put<{
    success: boolean
    message?: string
    data?: ExtensionModule
  }>(
    `/api/extension-admin/${encodeURIComponent(id)}/enabled`,
    { enabled },
    { skipBusinessError: true }
  )
  return res.data
}

export async function uninstallExtension(id: string) {
  const res = await api.delete<ExtensionListResponse>(
    `/api/extension-admin/${encodeURIComponent(id)}`,
    { skipBusinessError: true }
  )
  return res.data
}
