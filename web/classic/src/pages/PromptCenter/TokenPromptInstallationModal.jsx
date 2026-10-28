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

import React from 'react';
import { Modal } from '@douyinfe/semi-ui';
import { ArrowUpRight, KeyRound } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import TokenPromptManager from './TokenPromptManager';
import './style.css';

// 令牌徽标与提示词中心共用的安装弹窗外壳，保证主题变量、宽度和布局一致。
export default function TokenPromptInstallationModal(props) {
  const { t } = useTranslation();
  return (
    <Modal
      visible
      title={
        <span className='prompt-installation-title'>
          <span className='prompt-installation-title-mark' aria-hidden='true'>
            <KeyRound size={17} />
          </span>
          <strong
            className='prompt-installation-title-name'
            title={props.token.name}
          >
            {props.token.name}
          </strong>
        </span>
      }
      footer={null}
      width={760}
      style={{ maxWidth: 'calc(100vw - 24px)' }}
      onCancel={props.onClose}
      className='prompt-installation-modal'
    >
      <div className='prompt-center-page prompt-installation-content'>
        <TokenPromptManager
          token={props.token}
          templateToInstall={props.templateToInstall}
          onChange={props.onChange}
        />
        {props.showCenterLink && (
          <a className='prompt-installation-link' href='/console/prompt-center'>
            {t('Open prompt center')}
            <ArrowUpRight size={14} aria-hidden='true' />
          </a>
        )}
      </div>
    </Modal>
  );
}
