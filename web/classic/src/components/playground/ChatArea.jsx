/*
Copyright (C) 2025 QuantumNous

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

import React, { useState } from 'react';
import { Button, Chat } from '@douyinfe/semi-ui';
import {
  BarChart,
  CodeSquare,
  GraduationCap,
  ImagePlus,
  MessageSquarePlus,
  NotepadText,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import CustomInputRender from './CustomInputRender';
import TokenSourceSelector from './TokenSourceSelector';
import './playground-layout.css';

const starterPrompts = [
  { icon: BarChart, text: 'Analyze data' },
  { icon: NotepadText, text: 'Summarize text' },
  { icon: CodeSquare, text: 'Code' },
  { icon: GraduationCap, text: 'Get advice' },
];

const ChatArea = (props) => {
  const { t } = useTranslation();
  const [starterPrompt, setStarterPrompt] = useState(null);
  const isEmpty = props.message.length === 0;
  const tokenSelector = (
    <>
      <TokenSourceSelector
        token={props.selectedToken}
        onSelect={props.onTokenChange}
        disabled={props.isRequesting}
      />
      {props.modelError && (
        <div className='playground-model-error' role='alert'>
          <span>{props.modelError}</span>
          <Button
            size='small'
            disabled={props.isRequesting}
            onClick={props.onRetryModels}
          >
            {t('Retry')}
          </Button>
        </div>
      )}
    </>
  );
  return (
    <Chat
      ref={props.chatRef}
      className={`playground-chat ${isEmpty ? 'playground-chat-empty' : ''}`}
      style={{ height: '100%', maxWidth: '100%' }}
      topSlot={
        isEmpty && (
          <div className='playground-empty-state'>
            <div className='playground-empty-content'>
              <span className='playground-empty-icon'>
                <MessageSquarePlus size={20} aria-hidden='true' />
              </span>
              <div>
                <h2>{t('Start a playground chat')}</h2>
                <p>
                  {t(
                    'Test a model with a starter prompt, or write your own request below.',
                  )}
                </p>
              </div>
              <div className='playground-starter-prompts'>
                {starterPrompts.map(({ icon: Icon, text }) => (
                  <button
                    type='button'
                    key={text}
                    disabled={props.isRequesting}
                    onClick={() => setStarterPrompt({ text: t(text) })}
                  >
                    <Icon size={16} aria-hidden='true' />
                    <span>{t(text)}</span>
                  </button>
                ))}
              </div>
            </div>
          </div>
        )
      }
      renderInputArea={(inputProps) => (
        <CustomInputRender
          {...inputProps}
          {...props}
          hasMessages={!isEmpty}
          starterPrompt={starterPrompt}
          tokenSelector={tokenSelector}
        />
      )}
      chatBoxRenderConfig={{
        renderChatBoxContent: props.renderCustomChatContent,
        renderChatBoxAction: props.renderChatBoxAction,
        renderChatBoxTitle: () => null,
      }}
      uploadProps={{
        action: '',
        accept: 'image/*',
        multiple: true,
        uploadTrigger: 'custom',
        showUploadList: false,
        disabled: props.customRequestMode || props.isRequesting,
        children: (
          <Button
            aria-label={t('上传图片')}
            title={t('上传图片')}
            icon={<ImagePlus size={16} />}
            theme='borderless'
            type='tertiary'
            className='playground-tool-button'
            disabled={props.customRequestMode || props.isRequesting}
          />
        ),
      }}
      roleConfig={props.roleInfo}
      chats={props.message}
      onMessageSend={props.onMessageSend}
      onMessageCopy={props.onMessageCopy}
      onMessageReset={props.onMessageReset}
      onMessageDelete={props.onMessageDelete}
      showClearContext
      showStopGenerate={false}
      onStopGenerator={props.onStopGenerator}
      onClear={props.onClearMessages}
      placeholder={t('Ask anything')}
    />
  );
};

export default ChatArea;
