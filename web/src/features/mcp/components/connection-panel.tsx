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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { CopyButton } from './shared'

export function ConnectionPanel(props: { onNavigate?: (tab: string) => void }) {
  const { t } = useTranslation()
  const [client, setClient] = useState('cursor')
  const url = `${window.location.origin}/mcp`
  const codex = `[mcp_servers.newapi_mao]\nurl = ${JSON.stringify(url)}\nbearer_token_env_var = "NEWAPI_MCP_KEY"`
  const cursor = JSON.stringify(
    {
      mcpServers: {
        newapi_mao: {
          url,
          headers: { Authorization: 'Bearer ${env:NEWAPI_MCP_KEY}' },
        },
      },
    },
    null,
    2
  )
  const instructions = t(
    'Create a scoped key in Call keys. Set NEWAPI_MCP_KEY in your client environment, then restart the client. Connect and call newapi_context to check your identity and permissions.'
  )

  return (
    <Card className='gap-0 overflow-hidden'>
      <CardHeader className='border-b pb-6'>
        <CardTitle>{t('MCP connection')}</CardTitle>
        <CardDescription>
          {t('Connect Codex or Cursor to manage your NewAPI-Mao instance.')}
        </CardDescription>
      </CardHeader>
      <CardContent className='grid gap-0 p-0 lg:grid-cols-2'>
        <div className='min-w-0 space-y-6 p-5 sm:p-6'>
          <ol className='space-y-5'>
            <li className='flex gap-3'>
              <span
                aria-hidden='true'
                className='bg-primary/10 text-primary flex size-7 shrink-0 items-center justify-center rounded-full text-xs font-semibold'
              >
                1
              </span>
              <div className='min-w-0 space-y-1'>
                <h3 className='font-medium'>{t('Create a scoped key')}</h3>
                <p className='text-muted-foreground text-sm'>
                  {t(
                    'Grant only the tools this client needs. Write access requires explicit selection.'
                  )}
                </p>
                {props.onNavigate && (
                  <Button
                    variant='link'
                    className='h-auto px-0'
                    onClick={() => props.onNavigate?.('keys')}
                  >
                    {t('Create MCP key')}
                  </Button>
                )}
              </div>
            </li>
            <li className='flex gap-3'>
              <span
                aria-hidden='true'
                className='bg-primary/10 text-primary flex size-7 shrink-0 items-center justify-center rounded-full text-xs font-semibold'
              >
                2
              </span>
              <div className='min-w-0 space-y-1'>
                <h3 className='font-medium'>{t('Configure your client')}</h3>
                <p className='text-muted-foreground text-sm'>
                  {t(
                    'Set NEWAPI_MCP_KEY in your client environment, then restart the client.'
                  )}
                </p>
              </div>
            </li>
            <li className='flex gap-3'>
              <span
                aria-hidden='true'
                className='bg-primary/10 text-primary flex size-7 shrink-0 items-center justify-center rounded-full text-xs font-semibold'
              >
                3
              </span>
              <div className='min-w-0 space-y-1'>
                <h3 className='font-medium'>{t('Verify the connection')}</h3>
                <p className='text-muted-foreground text-sm'>
                  {t(
                    'Call newapi_context to check your identity and permissions.'
                  )}
                </p>
              </div>
            </li>
          </ol>
          <div className='space-y-2 border-t pt-5'>
            <h3 className='text-sm font-medium'>{t('MCP URL')}</h3>
            <code className='bg-muted/50 block rounded-md border px-3 py-2 text-sm break-all'>
              {url}
            </code>
            <div className='flex flex-wrap gap-2'>
              <CopyButton text={url} label={t('Copy MCP URL')} />
              <CopyButton
                text={`${url}\n\n${instructions}`}
                label={t('Copy connection instructions')}
              />
            </div>
          </div>
        </div>
        <div className='bg-muted/20 min-w-0 border-t p-5 sm:p-6 lg:border-t-0 lg:border-l'>
          <Tabs
            value={client}
            onValueChange={(value) => setClient(String(value))}
            className='gap-4'
          >
            <TabsList aria-label={t('Client configuration')}>
              <TabsTrigger value='cursor'>Cursor</TabsTrigger>
              <TabsTrigger value='codex'>Codex</TabsTrigger>
            </TabsList>
            <TabsContent value='cursor' className='min-w-0 space-y-4'>
              <p className='text-muted-foreground text-sm'>
                {t('Add this to ~/.cursor/mcp.json.')}
              </p>
              <pre className='bg-background overflow-x-auto rounded-lg border p-4 text-xs leading-relaxed'>
                <code>{cursor}</code>
              </pre>
              <CopyButton
                text={cursor}
                label={t('Copy Cursor configuration')}
              />
            </TabsContent>
            <TabsContent value='codex' className='min-w-0 space-y-4'>
              <p className='text-muted-foreground text-sm'>
                {t('Add this to ~/.codex/config.toml.')}
              </p>
              <pre className='bg-background overflow-x-auto rounded-lg border p-4 text-xs leading-relaxed'>
                <code>{codex}</code>
              </pre>
              <CopyButton text={codex} label={t('Copy Codex configuration')} />
            </TabsContent>
          </Tabs>
          <p className='text-muted-foreground mt-5 text-xs leading-relaxed'>
            {t(
              'Use an accessible HTTPS address for remote clients. Keep the key in your local environment, not in configuration files.'
            )}
          </p>
        </div>
      </CardContent>
      <div className='space-y-3 border-t px-5 py-5 sm:px-6'>
        <div className='flex flex-wrap items-center justify-between gap-2'>
          <h3 className='text-sm font-semibold'>{t('Execution rules')}</h3>
          {props.onNavigate && (
            <Button
              variant='link'
              className='h-auto px-0'
              onClick={() => props.onNavigate?.('tools')}
            >
              {t('Tool reference')}
            </Button>
          )}
        </div>
        <ul className='text-muted-foreground list-disc space-y-2 pl-4 text-sm'>
          <li>
            {t(
              'Only explicitly authorized tools can be called. New tools are not added to existing keys automatically.'
            )}
          </li>
          <li>
            {t(
              'Write calls require a unique request_id. Reuse it only when retrying the same operation.'
            )}
          </li>
          <li>
            {t(
              'Expired or revoked keys cannot make calls. Review activity in Call logs.'
            )}
          </li>
        </ul>
      </div>
    </Card>
  )
}
