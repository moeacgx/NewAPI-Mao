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

import React, { useId, useState } from 'react';
import { Button } from '@douyinfe/semi-ui';
import { Github } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { promptError, promptRequest } from './api';

export default function RepositoryImport(props) {
  const { t } = useTranslation();
  const prefix = useId();
  const [expanded, setExpanded] = useState(false);
  const [url, setUrl] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const disabled = busy || props.disabled;
  const importRepository = async () => {
    if (disabled || !url.trim()) return;
    setBusy(true);
    props.onBusyChange?.(true);
    setError('');
    try {
      props.onImport(
        await promptRequest(
          'post',
          '/api/prompt-center/admin/import/repository',
          { url: url.trim() },
        ),
      );
    } catch (reason) {
      setError(promptError(reason, t));
    } finally {
      setBusy(false);
      props.onBusyChange?.(false);
    }
  };
  return (
    <div className='prompt-archive-import prompt-repository-import'>
      <Button
        icon={<Github size={14} />}
        disabled={disabled}
        aria-expanded={expanded}
        aria-controls={`${prefix}-repository`}
        onClick={() => setExpanded((value) => !value)}
      >
        {t('Import GitHub repository')}
      </Button>
      {expanded && (
        <div id={`${prefix}-repository`} className='prompt-repository-fields'>
          <label htmlFor={`${prefix}-url`}>
            {t('Public GitHub repository URL')}
          </label>
          <div className='prompt-center-row'>
            <input
              id={`${prefix}-url`}
              type='url'
              maxLength={2048}
              placeholder='https://github.com/owner/repository'
              value={url}
              disabled={disabled}
              onChange={(event) => setUrl(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') {
                  event.preventDefault();
                  importRepository();
                }
              }}
            />
            <Button
              loading={busy}
              disabled={disabled || !url.trim()}
              onClick={importRepository}
            >
              {t('Import repository and edit')}
            </Button>
          </div>
          <small className='prompt-center-muted'>
            {t(
              'Supports public GitHub repositories, folders and SKILL.md links. Only text is imported; scripts and attachments are not executed.',
            )}
          </small>
          {error && (
            <p className='prompt-center-error' role='alert'>
              {error}
            </p>
          )}
        </div>
      )}
    </div>
  );
}
