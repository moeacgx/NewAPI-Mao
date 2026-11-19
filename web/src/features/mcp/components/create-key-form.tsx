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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'

import { createMcpKey, listMcpTools } from '../api'
import { mcpKeySchema, type McpKeyFormValues } from '../schema'
import { CopyButton } from './shared'
import { ToolCatalog } from './tool-catalog'

export function CreateKeyForm() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [secret, setSecret] = useState<string | null>(null)
  const creationLocked = useRef(false)
  const query = useQuery({ queryKey: ['mcp', 'tools'], queryFn: listMcpTools })
  const form = useForm<McpKeyFormValues>({
    resolver: zodResolver(mcpKeySchema),
    defaultValues: { name: '', expires: '', tools: [] },
  })
  const selected = form.watch('tools')
  const creation = useMutation({
    // 完整密钥只进入本组件状态，不作为 mutation 返回值进入共享缓存。
    mutationFn: async (values: McpKeyFormValues) => {
      const receipt = await createMcpKey({
        name: values.name,
        tools: values.tools,
        expires_at: Math.floor(new Date(values.expires).getTime() / 1000),
      })
      setSecret(receipt.secret)
    },
    retry: false,
    gcTime: 0,
    onSuccess: () => {
      form.reset()
      void client.invalidateQueries({ queryKey: ['mcp', 'keys'] })
    },
    onSettled: () => {
      creationLocked.current = false
    },
  })

  return (
    <Card>
      <CardHeader className='border-b'>
        <CardTitle>{t('Create MCP key')}</CardTitle>
        <CardDescription>
          {t(
            'Grant only the tools this client needs. Write access requires explicit selection.'
          )}
        </CardDescription>
      </CardHeader>
      <CardContent>
        {secret !== null ? (
          <Alert>
            <AlertTitle>{t('Save your MCP key')}</AlertTitle>
            <AlertDescription className='min-w-0 space-y-3'>
              <p>
                {t(
                  'This key is shown once. Copy it before closing or leaving this tab.'
                )}
              </p>
              <code className='bg-muted block rounded-md border p-3 break-all select-all'>
                {secret}
              </code>
              <div className='flex flex-wrap gap-2'>
                <CopyButton text={secret} label={t('Copy MCP key')} />
                <Button
                  type='button'
                  variant='outline'
                  onClick={() => setSecret(null)}
                >
                  {t('Close key receipt')}
                </Button>
              </div>
            </AlertDescription>
          </Alert>
        ) : (
          <form
            noValidate
            onSubmit={form.handleSubmit((values) => {
              // 异步校验可能让同轮提交共享旧状态，必须先同步加锁再发起请求。
              if (creationLocked.current) return
              creationLocked.current = true
              creation.mutate(values)
            })}
          >
            <FieldGroup>
              <div className='grid max-w-3xl gap-4 sm:grid-cols-2'>
                <Field data-invalid={Boolean(form.formState.errors.name)}>
                  <FieldLabel htmlFor='mcp-key-name'>
                    {t('Key name')}
                  </FieldLabel>
                  <Input
                    id='mcp-key-name'
                    autoComplete='off'
                    placeholder={t('For example, Cursor on my laptop')}
                    disabled={creation.isPending}
                    aria-invalid={Boolean(form.formState.errors.name)}
                    aria-describedby='mcp-name-error'
                    {...form.register('name')}
                  />
                  <FieldError id='mcp-name-error'>
                    {form.formState.errors.name &&
                      (form.formState.errors.name.message ===
                      'Use no more than 100 characters'
                        ? t('Use no more than 100 characters')
                        : t('Enter a key name'))}
                  </FieldError>
                </Field>
                <Field data-invalid={Boolean(form.formState.errors.expires)}>
                  <FieldLabel htmlFor='mcp-key-expiration'>
                    {t('Expiration time')}
                  </FieldLabel>
                  <Input
                    id='mcp-key-expiration'
                    type='datetime-local'
                    disabled={creation.isPending}
                    aria-invalid={Boolean(form.formState.errors.expires)}
                    aria-describedby='mcp-expiration-error'
                    {...form.register('expires')}
                  />
                  <FieldDescription>
                    {t('Choose a future time in your local timezone.')}
                  </FieldDescription>
                  <FieldError id='mcp-expiration-error'>
                    {form.formState.errors.expires &&
                      t('Choose a future expiration time')}
                  </FieldError>
                </Field>
              </div>
              <FieldSet
                disabled={creation.isPending}
                aria-describedby='mcp-tools-error'
                aria-invalid={Boolean(form.formState.errors.tools)}
              >
                <FieldLegend>{t('Allowed tools')}</FieldLegend>
                {query.isPending && <LoadingState />}
                {query.isError && (
                  <ErrorState
                    description={query.error.message}
                    onRetry={() => void query.refetch()}
                  />
                )}
                {query.isSuccess && (
                  <ToolCatalog
                    tools={query.data}
                    selected={selected}
                    disabled={creation.isPending}
                    onChange={(names) =>
                      form.setValue('tools', names, {
                        shouldValidate: true,
                      })
                    }
                  />
                )}
                <FieldError id='mcp-tools-error'>
                  {form.formState.errors.tools && t('Select at least one tool')}
                </FieldError>
              </FieldSet>
              {creation.isError && (
                <Alert variant='destructive'>
                  <AlertTitle>{t('Request failed')}</AlertTitle>
                  <AlertDescription>{creation.error.message}</AlertDescription>
                </Alert>
              )}
              <Button
                type='submit'
                className='w-fit'
                disabled={
                  creation.isPending ||
                  !query.isSuccess ||
                  query.data.length === 0
                }
              >
                {creation.isPending ? t('Creating...') : t('Create MCP key')}
              </Button>
            </FieldGroup>
          </form>
        )}
      </CardContent>
    </Card>
  )
}
