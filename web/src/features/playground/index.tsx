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
import type { PromptToken } from '@/features/prompt-center/types'

import { PlaygroundChat } from './components/chat/playground-chat'
import { PlaygroundInput } from './components/input/playground-input'
import { PlaygroundTokenSwitch } from './components/playground-token-switch'
import {
  useChatHandler,
  usePlaygroundConversation,
  usePlaygroundOptions,
  usePlaygroundState,
} from './hooks'

export function Playground() {
  const { t } = useTranslation()
  const [token, setToken] = useState<PromptToken>()
  const {
    config,
    parameterEnabled,
    messages,
    isLoadingMessages,
    groups,
    updateMessages,
    setModels,
    setGroups,
    updateConfig,
    updateParameterEnabled,
    clearMessages,
  } = usePlaygroundState()

  const { isLoadingModels, models, isModelsError, refetchModels } =
    usePlaygroundOptions({
      currentGroup: config.group,
      currentModel: config.model,
      tokenId: token?.id,
      setGroups,
      setModels,
      updateConfig,
    })
  const requestReady =
    !isLoadingModels &&
    !isModelsError &&
    models.some((model) => model.value === config.model)

  const { sendChat, stopGeneration, isGenerating } = useChatHandler({
    config,
    parameterEnabled,
    onMessageUpdate: updateMessages,
    tokenId: token?.id,
    requestReady,
  })

  const {
    editingMessageKey,
    handleSendMessage,
    handleRegenerateMessage,
    handleEditMessage,
    handleEditOpenChange,
    applyEdit,
    handleDeleteMessage,
  } = usePlaygroundConversation({
    messages,
    updateMessages,
    sendChat,
    canSend: requestReady && !isGenerating,
  })

  const handleClearMessages = () => {
    handleEditOpenChange(false)
    clearMessages()
  }

  return (
    <div className='relative flex size-full min-h-0 flex-col overflow-hidden'>
      {/* Full-width scroll container: scrolling works even over side whitespace */}
      <div className='flex min-h-0 flex-1 flex-col overflow-hidden'>
        <PlaygroundChat
          messages={messages}
          isLoadingMessages={isLoadingMessages}
          onRegenerateMessage={handleRegenerateMessage}
          onEditMessage={handleEditMessage}
          onDeleteMessage={handleDeleteMessage}
          onSelectPrompt={handleSendMessage}
          isGenerating={isGenerating}
          editingKey={editingMessageKey}
          onCancelEdit={handleEditOpenChange}
          onSaveEdit={(newContent) => applyEdit(newContent, false)}
          onSaveEditAndSubmit={(newContent) => applyEdit(newContent, true)}
        />
      </div>

      {/* Input area: center content and constrain to the same container width */}
      <div className='mx-auto w-full max-w-4xl'>
        <PlaygroundTokenSwitch
          token={token}
          disabled={isGenerating}
          onChange={(nextToken) => {
            if (isGenerating) return
            setToken(nextToken)
            setModels([])
          }}
        />
        {token && (
          <p className='text-muted-foreground mb-2 px-1 text-xs'>
            {t(
              'The API key determines the request group and installed prompts.'
            )}
          </p>
        )}
        {isModelsError && (
          <div
            role='alert'
            className='border-destructive/30 bg-destructive/5 text-destructive mb-3 flex items-center justify-between gap-2 rounded-xl border p-3 text-sm'
          >
            {t('Failed to load playground models')}
            <Button
              size='sm'
              variant='outline'
              disabled={isLoadingModels}
              onClick={() => void refetchModels()}
            >
              {t('Retry')}
            </Button>
          </div>
        )}
        <PlaygroundInput
          config={config}
          disabled={isGenerating || !requestReady}
          tokenMode={Boolean(token)}
          groups={groups}
          groupValue={config.group}
          isGenerating={isGenerating}
          isModelLoading={isLoadingModels}
          modelValue={config.model}
          models={models}
          onGroupChange={(value) => updateConfig('group', value)}
          onConfigChange={updateConfig}
          onClearMessages={handleClearMessages}
          onModelChange={(value) => updateConfig('model', value)}
          onParameterEnabledChange={updateParameterEnabled}
          onStop={stopGeneration}
          onSubmit={handleSendMessage}
          parameterEnabled={parameterEnabled}
          hasMessages={messages.length > 0}
        />
      </div>
    </div>
  )
}
