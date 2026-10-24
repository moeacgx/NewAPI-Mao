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
import { Button, Modal } from '@douyinfe/semi-ui';
import { RefreshCw } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { promptError, promptRequest } from './api';
import RepositorySource from './RepositorySource';

export default function RepositoryUpdate(props) {
  const { t } = useTranslation();
  const [confirm, setConfirm] = useState(false);
  const [busy, setBusy] = useState(false);
  const busyRef = useRef(false);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  const update = async (overwriteLocal) => {
    if (busyRef.current) return;
    busyRef.current = true;
    setBusy(true);
    setError('');
    setMessage('');
    try {
      const result = await promptRequest(
        'post',
        `/api/prompt-center/admin/${props.template.id}/update-repository`,
        {
          version: props.template.version,
          overwrite_local: overwriteLocal,
        },
      );
      if (!result?.template || typeof result.updated !== 'boolean')
        throw new Error('Prompt request failed');
      props.onUpdated(result.template);
      setMessage(
        result.updated
          ? t('Skills updated from GitHub')
          : t('Skills are already up to date'),
      );
      setConfirm(false);
    } catch (reason) {
      const code = reason.response?.data?.code || reason.code;
      if (code === 'repository_local_changes') setConfirm(true);
      else {
        setConfirm(false);
        setError(promptError(reason, t));
      }
    } finally {
      busyRef.current = false;
      setBusy(false);
    }
  };
  return (
    <div className='prompt-repository-update'>
      <RepositorySource source={props.template.repository_source} />
      <div className='prompt-repository-update-actions'>
        <Button
          className='prompt-center-secondary'
          icon={<RefreshCw size={14} aria-hidden='true' />}
          loading={busy}
          disabled={busy}
          aria-label={t('Update {{title}} from GitHub', {
            title: props.template.title,
          })}
          onClick={() => {
            if (props.template.repository_source.locally_modified)
              setConfirm(true);
            else update(false);
          }}
        >
          {t('Update from GitHub')}
        </Button>
        {message && <span role='status'>{message}</span>}
      </div>
      {error && (
        <p role='alert' className='prompt-repository-update-error'>
          {error}
        </p>
      )}
      {confirm && (
        <Modal
          visible
          title={t('Overwrite local skill changes?')}
          closable={!busy}
          closeOnEsc={!busy}
          maskClosable={false}
          onCancel={() => {
            if (!busyRef.current) setConfirm(false);
          }}
          footer={
            <div className='prompt-center-row'>
              <Button disabled={busy} onClick={() => setConfirm(false)}>
                {t('Cancel')}
              </Button>
              <Button
                theme='solid'
                loading={busy}
                disabled={busy}
                onClick={() => update(true)}
              >
                {t('Update and overwrite')}
              </Button>
            </div>
          }
        >
          <p>
            {t(
              'Updating replaces local instructions and reference files. The title, description, category, allowed groups, publication state and sort order are preserved.',
            )}
          </p>
        </Modal>
      )}
    </div>
  );
}
