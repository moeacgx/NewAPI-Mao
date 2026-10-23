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
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'

import type { PromptTemplate } from '../types'

export function SkillSummary(props: { template: PromptTemplate }) {
  const { t } = useTranslation()
  const bundle = props.template.kind === 'bundle'
  return (
    <div className='space-y-2'>
      <div className='flex flex-wrap gap-1.5'>
        <Badge variant='secondary' className='rounded-full'>
          {bundle ? t('Skill bundle') : t('Single skill')}
        </Badge>
        <Badge variant='outline' className='rounded-full'>
          {props.template.load_mode === 'progressive'
            ? t('Read on demand')
            : t('Always loaded')}
        </Badge>
        <Badge
          variant='outline'
          className='max-w-full rounded-full [overflow-wrap:anywhere] whitespace-normal'
        >
          {props.template.category_name || t('Uncategorized skills')}
        </Badge>
      </div>
      {bundle && (
        <details className='text-muted-foreground text-xs'>
          <summary className='cursor-pointer py-1 font-medium'>
            {t('{{count}} skills in this bundle', {
              count:
                props.template.skill_count ??
                props.template.skills?.length ??
                0,
            })}
          </summary>
          <ol className='bg-muted/20 mt-1 list-inside list-decimal space-y-1 rounded-lg border p-3'>
            {props.template.skills?.map((skill) => (
              <li key={skill.title} className='[overflow-wrap:anywhere]'>
                {skill.title}
              </li>
            ))}
          </ol>
        </details>
      )}
    </div>
  )
}
