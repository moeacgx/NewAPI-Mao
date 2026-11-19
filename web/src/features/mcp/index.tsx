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

import { Main } from '@/components/layout/components/main'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { ConnectionPanel } from './components/connection-panel'
import { KeysPanel } from './components/keys-panel'
import { LogsPanel } from './components/logs-panel'
import { ToolsPanel } from './components/tools-panel'

export function McpManagement() {
  const { t } = useTranslation()
  const [tab, setTab] = useState('connection')

  return (
    <Main className='bg-muted/20 overflow-y-auto'>
      <div className='mx-auto w-full max-w-7xl space-y-6 px-4 py-6 sm:px-6 sm:py-8'>
        <h1 className='text-xl font-semibold tracking-tight'>
          {t('MCP Management')}
        </h1>
        <Tabs
          value={tab}
          onValueChange={(value) => setTab(String(value))}
          className='gap-6'
        >
          <div className='overflow-x-auto border-b pb-1'>
            <TabsList
              variant='line'
              aria-label={t('MCP Management')}
              className='gap-5'
            >
              <TabsTrigger value='connection'>
                {t('MCP connection')}
              </TabsTrigger>
              <TabsTrigger value='tools'>{t('Tool reference')}</TabsTrigger>
              <TabsTrigger value='keys'>{t('Call keys')}</TabsTrigger>
              <TabsTrigger value='logs'>{t('Call logs')}</TabsTrigger>
            </TabsList>
          </div>
          <TabsContent value='connection'>
            {tab === 'connection' && <ConnectionPanel onNavigate={setTab} />}
          </TabsContent>
          <TabsContent value='tools'>
            {tab === 'tools' && <ToolsPanel />}
          </TabsContent>
          <TabsContent value='keys'>
            {tab === 'keys' && <KeysPanel />}
          </TabsContent>
          <TabsContent value='logs'>
            {tab === 'logs' && <LogsPanel />}
          </TabsContent>
        </Tabs>
      </div>
    </Main>
  )
}
