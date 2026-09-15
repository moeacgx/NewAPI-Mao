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
import { useQuery } from '@tanstack/react-query'
import type { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import type { ChannelFormValues } from '@/features/channels/lib/channel-form'

import { getTaskPluginOptions } from '../api'
import { PluginVersionSelect } from './plugin-version-select'

export function TaskPluginBindingField(props: {
  form: UseFormReturn<ChannelFormValues>
  disabled: boolean
}) {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ['task-plugin-options'],
    queryFn: getTaskPluginOptions,
    enabled: !props.disabled,
  })
  if (query.isError) {
    return (
      <ErrorState
        description={query.error.message}
        onRetry={() => void query.refetch()}
      />
    )
  }
  const selectedKey = props.form.watch('task_plugin_key')
  const options = (query.data ?? []).map((plugin) => ({
    value: plugin.key,
    label: `${plugin.name} (${plugin.version})`,
  }))
  if (selectedKey && !options.some((option) => option.value === selectedKey)) {
    options.unshift({ value: selectedKey, label: selectedKey })
  }
  return (
    <FormField
      control={props.form.control}
      name='task_plugin_key'
      render={({ field }) => (
        <FormItem>
          <FormLabel>{t('Task Plugin')}</FormLabel>
          <FormControl>
            <PluginVersionSelect
              aria-label={t('Task Plugin')}
              value={field.value}
              disabled={props.disabled || query.isPending}
              placeholder={t('Select a task plugin')}
              options={options}
              onValueChange={field.onChange}
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      )}
    />
  )
}
