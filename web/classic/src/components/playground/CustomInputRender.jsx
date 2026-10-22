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

import React, { useEffect, useId } from 'react';
import { Button, Select } from '@douyinfe/semi-ui';
import { Bug, Send, SlidersHorizontal, Square } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { prioritizePlaygroundGroupOptions, selectFilter } from '../../helpers';

const CustomInputRender = (props) => {
  const { t } = useTranslation();
  const selectorId = useId();
  const { detailProps } = props;
  const { clearContextNode, uploadNode, inputNode, sendNode, onClick } =
    detailProps;

  const textarea = React.Children.toArray(inputNode.props.children)[0];
  useEffect(() => {
    if (!props.starterPrompt) return;
    textarea.props.onChange(props.starterPrompt.text);
  }, [props.starterPrompt, textarea.props.onChange]);

  const styledClearNode = clearContextNode
    ? React.cloneElement(clearContextNode, {
        className: 'playground-tool-button',
        'aria-label': t('Clear chat history'),
        title: t('Clear chat history'),
        disabled: props.isRequesting || !props.hasMessages,
      })
    : null;
  const styledSendNode = React.cloneElement(sendNode, {
    className: 'playground-send-button',
    'aria-label': t('Send'),
    disabled:
      sendNode.props.disabled ||
      !props.modelsReady ||
      (!props.customRequestMode && !props.inputs.model),
    icon: <Send size={16} />,
    children: <span>{t('Send')}</span>,
  });
  return (
    <div className='playground-composer-wrap'>
      {props.tokenSelector}
      <div className='playground-composer' onClick={onClick}>
        <div className='playground-composer-input'>
          {React.cloneElement(
            inputNode,
            {},
            React.Children.map(inputNode.props.children, (child, index) =>
              index === 0
                ? React.cloneElement(child, {
                    disabled: props.isRequesting || !props.modelsReady,
                  })
                : child,
            ),
          )}
        </div>
        <div className='playground-composer-footer'>
          <div className='playground-composer-tools'>
            <span title={t('上传图片')}>{uploadNode}</span>
            <Button
              className='playground-tool-button'
              icon={<SlidersHorizontal size={16} />}
              theme='borderless'
              aria-label={t('Parameters')}
              title={t('Parameters')}
              disabled={props.isRequesting}
              onClick={props.onOpenSettings}
            />
            {styledClearNode}
            <Button
              className='playground-tool-button'
              icon={<Bug size={16} />}
              theme='borderless'
              aria-label={t('Debug')}
              title={t('Debug')}
              onClick={props.onToggleDebugPanel}
            />
          </div>
          <div className='playground-composer-controls'>
            <div className='playground-model-selectors'>
              <span className='sr-only' id={`${selectorId}-group`}>
                {t('分组')}
              </span>
              <span className='sr-only' id={`${selectorId}-model`}>
                {t('模型')}
              </span>
              {!props.selectedToken && (
                <Select
                  aria-labelledby={`${selectorId}-group`}
                  placeholder={t('请选择分组')}
                  optionList={prioritizePlaygroundGroupOptions(
                    props.groups,
                    props.inputs.group,
                  )}
                  value={props.inputs.group}
                  onChange={(value) => props.onInputChange('group', value)}
                  filter={selectFilter}
                  disabled={props.isRequesting || props.customRequestMode}
                  className='playground-group-select'
                />
              )}
              <Select
                aria-labelledby={`${selectorId}-model`}
                placeholder={t('请选择模型')}
                optionList={props.models}
                value={props.inputs.model || undefined}
                onChange={(value) => props.onInputChange('model', value)}
                filter={selectFilter}
                loading={!props.modelsReady && !props.modelError}
                disabled={
                  props.isRequesting ||
                  props.customRequestMode ||
                  !props.modelsReady
                }
                className='playground-model-select'
              />
            </div>
            {props.isRequesting ? (
              <Button
                className='playground-send-button'
                icon={<Square size={15} />}
                theme='solid'
                onClick={props.onStopGenerator}
                aria-label={t('Stop generating')}
              >
                {t('Stop')}
              </Button>
            ) : (
              styledSendNode
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default CustomInputRender;
