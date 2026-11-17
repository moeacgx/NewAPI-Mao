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
import { Link } from '@tanstack/react-router'
import { Menu } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { getCustomNavIcon } from '@/lib/custom-nav'
import { cn } from '@/lib/utils'

import type { TopNavLink } from '../types'

type TopNavProps = React.HTMLAttributes<HTMLElement> & {
  links: TopNavLink[]
}

/**
 * 顶部导航栏组件
 * 桌面链接只在导航区域滚动，所有尺寸均可通过菜单访问完整导航。
 */
export function TopNav({ className, links, ...props }: TopNavProps) {
  const { t } = useTranslation()
  // 规范化链接，确保所有可选属性都有默认值
  const normalizedLinks = useMemo(
    () =>
      links.map((link) => ({
        isActive: false,
        disabled: false,
        external: false,
        ...link,
      })),
    [links]
  )

  if (normalizedLinks.length === 0) return null

  return (
    <nav
      aria-label={t('Navigation')}
      className={cn('flex min-w-0 items-center justify-end gap-2', className)}
      {...props}
    >
      {/* 链接不压缩、不换行；溢出只发生在这个可滚动区域。 */}
      <div className='hidden min-w-0 items-center gap-4 overflow-x-auto overscroll-x-contain py-1 lg:flex'>
        {normalizedLinks.map(
          ({ title, href, isActive, disabled, external, icon }) => {
            const Icon = getCustomNavIcon(icon)
            const content = (
              <>
                {Icon ? <Icon className='size-4 shrink-0' /> : null}
                <span>{title}</span>
              </>
            )
            const linkClassName = cn(
              'hover:text-primary focus-visible:ring-ring/40 inline-flex shrink-0 items-center gap-1.5 rounded-sm text-sm font-medium whitespace-nowrap transition-colors outline-none focus-visible:ring-2 focus-visible:ring-inset',
              !isActive && 'text-muted-foreground',
              disabled && 'pointer-events-none opacity-50'
            )

            return external ? (
              <a
                key={`${title}-${href}`}
                href={href}
                target='_blank'
                rel='noopener noreferrer'
                aria-disabled={disabled}
                aria-current={isActive ? 'page' : undefined}
                tabIndex={disabled ? -1 : undefined}
                className={linkClassName}
              >
                {content}
              </a>
            ) : (
              <Link
                key={`${title}-${href}`}
                to={href}
                disabled={disabled}
                aria-current={isActive ? 'page' : undefined}
                className={linkClassName}
              >
                {content}
              </Link>
            )
          }
        )}
      </div>

      {/* 菜单不依赖断点隐藏，长导航也能通过键盘完整访问。 */}
      <div className='shrink-0'>
        <DropdownMenu modal={false}>
          <DropdownMenuTrigger
            aria-label={t('Navigation menu')}
            render={<Button size='icon' variant='ghost' className='size-8' />}
          >
            <Menu aria-hidden='true' />
          </DropdownMenuTrigger>
          <DropdownMenuContent
            side='bottom'
            align='end'
            className='w-72 max-w-[calc(100vw-2rem)]'
          >
            {normalizedLinks.map(
              ({ title, href, isActive, disabled, external, icon }) => {
                const Icon = getCustomNavIcon(icon)
                const content = (
                  <>
                    {Icon ? <Icon className='size-4' /> : null}
                    <span className='min-w-0 wrap-anywhere whitespace-normal'>
                      {title}
                    </span>
                  </>
                )
                const linkClassName = cn(
                  'flex w-full items-center gap-2',
                  !isActive && 'text-muted-foreground'
                )
                return (
                  <DropdownMenuItem
                    key={`${title}-${href}`}
                    disabled={disabled}
                    aria-current={isActive ? 'page' : undefined}
                    render={
                      external ? (
                        <a
                          href={href}
                          target='_blank'
                          rel='noopener noreferrer'
                          className={linkClassName}
                        >
                          {content}
                        </a>
                      ) : (
                        <Link
                          to={href}
                          className={linkClassName}
                          disabled={disabled}
                        >
                          {content}
                        </Link>
                      )
                    }
                  />
                )
              }
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </nav>
  )
}
