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
import { GitBranch, Link2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import type { SkillRepositorySource } from '../types'

export function SkillRepositoryDetails(props: {
  source: SkillRepositorySource
}) {
  const { t } = useTranslation()
  let repository: URL | undefined
  try {
    const parsed = new URL(props.source.url)
    if (
      parsed.protocol === 'https:' &&
      parsed.hostname === 'github.com' &&
      !parsed.port &&
      !parsed.username &&
      !parsed.password &&
      !parsed.search &&
      !parsed.hash
    ) {
      repository = parsed
    }
  } catch {
    // 来源文本仍可展示，只有通过校验的 GitHub HTTPS 地址生成链接。
  }
  const repositoryName = repository?.pathname.split('/').slice(1, 3).join('/')
  return (
    <section
      aria-label={t('GitHub source')}
      className='border-border/60 bg-muted/20 space-y-2 rounded-xl border p-3 text-xs'
    >
      <div className='flex min-w-0 items-center gap-2'>
        <Link2
          className='text-muted-foreground size-3.5 shrink-0'
          aria-hidden='true'
        />
        {repository ? (
          <a
            href={repository.href}
            target='_blank'
            rel='noopener noreferrer'
            className='text-primary min-w-0 truncate font-medium hover:underline'
            title={props.source.url}
          >
            {repositoryName}
          </a>
        ) : (
          <span className='min-w-0 break-all'>{props.source.url}</span>
        )}
      </div>
      <dl className='text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1.5'>
        <div className='flex max-w-full min-w-0 items-center gap-1'>
          <dt className='sr-only'>{t('Reference')}</dt>
          <GitBranch className='size-3 shrink-0' aria-hidden='true' />
          <dd className='truncate' title={props.source.ref}>
            {props.source.ref}
          </dd>
        </div>
        <div className='max-w-full min-w-0'>
          <dt className='sr-only'>{t('Directory')}</dt>
          <dd
            className='truncate'
            title={props.source.path || t('Repository root')}
          >
            {props.source.path || t('Repository root')}
          </dd>
        </div>
        <div>
          <dt className='sr-only'>{t('Commit')}</dt>
          <dd>
            <code title={props.source.commit}>
              {props.source.commit.slice(0, 7)}
            </code>
          </dd>
        </div>
      </dl>
    </section>
  )
}
