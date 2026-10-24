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
import { MessageSquarePlus } from 'lucide-react'
import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { useAuthStore } from '@/stores/auth-store'

import { CreateTicketDialog } from './create-ticket-dialog'

// 此入口仅挂载于已登录控制台布局，公开页面不显示。
export function FloatingTicketButton() {
  const { t } = useTranslation()
  const userId = useAuthStore((state) => state.auth.user?.id)
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLButtonElement>(null)
  if (!userId) return null

  return (
    <>
      <Button
        ref={triggerRef}
        className='fixed right-4 bottom-[max(5rem,env(safe-area-inset-bottom))] z-30 size-11 rounded-full shadow-lg sm:right-6'
        size='icon-lg'
        aria-label={t('Create ticket')}
        title={t('Create ticket')}
        aria-haspopup='dialog'
        aria-expanded={open}
        onClick={() => setOpen(true)}
      >
        <MessageSquarePlus aria-hidden='true' />
      </Button>
      <CreateTicketDialog
        key={userId}
        open={open}
        onOpenChange={setOpen}
        returnFocus={triggerRef}
      />
    </>
  )
}
