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

import React, { useRef, useState } from 'react';
import { Button } from '@douyinfe/semi-ui';
import { Upload } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { promptError, promptRequest } from './api';

export default function ArchiveImport(props) {
  const { t } = useTranslation();
  const input = useRef(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const upload = async (event) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;
    const form = new FormData();
    form.append('file', file);
    setBusy(true);
    props.onBusyChange?.(true);
    setError('');
    try {
      props.onImport(
        await promptRequest(
          'post',
          '/api/prompt-center/admin/import/archive',
          form,
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
    <div className='prompt-archive-import'>
      <input
        ref={input}
        hidden
        type='file'
        accept='.zip,application/zip'
        aria-label={t('Upload skill archive')}
        disabled={busy || props.disabled}
        onChange={upload}
      />
      <Button
        icon={<Upload size={14} />}
        loading={busy}
        disabled={busy || props.disabled}
        onClick={() => input.current?.click()}
      >
        {t('Import ZIP archive')}
      </Button>
      <small className='prompt-center-muted'>
        {t(
          'Imports create editable drafts. Nothing is published automatically.',
        )}
      </small>
      {error && (
        <p className='prompt-center-error' role='alert'>
          {error}
        </p>
      )}
    </div>
  );
}
