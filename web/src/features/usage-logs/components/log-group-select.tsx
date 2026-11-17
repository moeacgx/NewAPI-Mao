import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import { useLogGroupOptions } from '../hooks/use-log-group-options'

type LogGroupSelectProps = {
  value: string
  isAdmin: boolean
  sensitiveVisible?: boolean
  onValueChange: (value: string) => void
}

export function LogGroupSelect({
  value,
  isAdmin,
  sensitiveVisible = true,
  onValueChange,
}: LogGroupSelectProps) {
  const { t } = useTranslation()
  const query = useLogGroupOptions(isAdmin)
  const options = query.data?.options ?? []
  const selected = options.find((option) => option.value === value)
  let label = t('All Groups')
  if (value) {
    label = selected?.label || value
    const names = query.data?.names
    // 历史筛选值可能与原型属性同名；仅将映射自有的非空字符串用于展示。
    if (!selected && names && Object.hasOwn(names, value)) {
      const name: unknown = names[value]
      if (typeof name === 'string' && name) {
        label = name
      }
    }
  }
  if (query.isLoading) {
    label = t('Loading...')
  }
  if (value && !sensitiveVisible) {
    label = '••••••'
  }
  const items = [
    { value: '', label: t('All Groups'), disabled: false },
    ...options.map((option) => ({
      ...option,
      label: sensitiveVisible ? option.label : '••••••',
      disabled: false,
    })),
  ]
  // 旧链接保留原始筛选语义；解析名称不会把 alias 改写为当前 code，也不会增加可用选项。
  if (value && !selected) {
    items.push({ value, label, disabled: true })
  }
  const namesUnavailable = query.isSuccess && query.data.names === undefined

  return (
    <div className='min-w-0 space-y-1'>
      <div className='flex min-w-0 items-center gap-1'>
        <Select
          items={items}
          value={value}
          onValueChange={(next) => onValueChange(next ?? '')}
        >
          <SelectTrigger
            aria-label={t('Group')}
            aria-busy={query.isLoading}
            className='min-w-0 flex-1'
          >
            <SelectValue className='min-w-0'>
              <span
                className='truncate'
                title={sensitiveVisible ? label : undefined}
              >
                {label}
              </span>
            </SelectValue>
          </SelectTrigger>
          <SelectContent
            alignItemWithTrigger={false}
            className='max-w-[calc(100vw-2rem)]'
          >
            {items.map((item) => (
              <SelectItem
                key={item.value}
                value={item.value}
                disabled={item.disabled}
              >
                <span
                  className='block min-w-0 truncate'
                  title={sensitiveVisible ? item.label : undefined}
                >
                  {item.label}
                </span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button
          type='button'
          variant='ghost'
          size='icon'
          className='size-8 shrink-0'
          aria-label={t('Clear group filter')}
          disabled={!value}
          onClick={() => onValueChange('')}
        >
          <X className='size-4' />
        </Button>
      </div>
      {query.isError || namesUnavailable ? (
        <div
          role='alert'
          className='text-destructive flex flex-wrap items-center gap-1 text-xs'
        >
          <span>
            {t(
              query.isError
                ? 'Failed to load groups'
                : 'Failed to load group names'
            )}
          </span>
          <Button
            type='button'
            variant='ghost'
            size='sm'
            disabled={query.isFetching}
            onClick={() => void query.refetch()}
          >
            {t('Retry')}
          </Button>
        </div>
      ) : null}
    </div>
  )
}
