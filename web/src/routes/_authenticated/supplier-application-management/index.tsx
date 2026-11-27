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
import { createFileRoute, redirect } from '@tanstack/react-router'

import { SupplierApplicationManagement } from '@/features/supplier-applications/admin'
import { supplierManagementSearchSchema } from '@/features/supplier-applications/schema'
import { hasPermission } from '@/lib/admin-permissions'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute(
  '/_authenticated/supplier-application-management/'
)({
  validateSearch: supplierManagementSearchSchema,
  beforeLoad: () => {
    const user = useAuthStore.getState().auth.user
    if (
      !user ||
      user.role < ROLE.ADMIN ||
      !hasPermission(user, 'supplier_applications', 'read')
    ) {
      throw redirect({ to: '/403' })
    }
  },
  component: SupplierApplicationManagementRoute,
})

function SupplierApplicationManagementRoute() {
  const { status } = Route.useSearch()
  const navigate = Route.useNavigate()
  return (
    <SupplierApplicationManagement
      status={status}
      onStatusChange={(nextStatus) => {
        void navigate({ search: { status: nextStatus }, replace: true })
      }}
    />
  )
}
