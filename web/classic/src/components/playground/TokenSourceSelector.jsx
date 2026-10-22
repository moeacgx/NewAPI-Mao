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
import { KeyRound, UserRound } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import TokenPicker from '../../pages/PromptCenter/TokenPicker';
import '../../pages/PromptCenter/style.css';
import './token-source.css';

export default function TokenSourceSelector(props) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  return (
    <div className='playground-token-section'>
      <div className='playground-token-source'>
        <span className='playground-token-identity'>
          {props.token ? (
            <KeyRound size={16} aria-hidden='true' />
          ) : (
            <UserRound size={16} aria-hidden='true' />
          )}
          <strong title={props.token?.name}>
            {props.token?.name || t('Account mode')}
          </strong>
          {props.token && (
            <span className='playground-token-skills'>
              {t('Skills · {{count}}', {
                count: props.token.prompt_count || 0,
              })}
            </span>
          )}
        </span>
        {props.token && (
          <button
            type='button'
            className='playground-account-button'
            disabled={props.disabled}
            onClick={() => props.onSelect(null)}
          >
            {t('Use account mode')}
          </button>
        )}
        <button
          type='button'
          className='playground-switch-button'
          aria-label={t('Switch API token')}
          aria-haspopup='dialog'
          aria-expanded={open}
          disabled={props.disabled}
          onClick={() => setOpen(true)}
        >
          <KeyRound size={14} aria-hidden='true' />
          {t('Switch API token')}
        </button>
      </div>
      {props.token && (
        <p className='playground-token-hint'>
          {t('Uses this token’s group, permissions and installed prompts')}
        </p>
      )}
      {open && (
        <TokenPicker
          open
          hideTrigger
          allowAccount
          summaryOnly
          showGroup
          token={props.token}
          disabled={props.disabled}
          introduction={t(
            'Select your API token without exposing its secret key',
          )}
          onClose={() => setOpen(false)}
          onSelect={(token) => {
            props.onSelect(token);
            setOpen(false);
          }}
        />
      )}
    </div>
  );
}
