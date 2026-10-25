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
import { Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { showSuccess } from '../../helpers/utils';
import { promptError, promptRequest } from './api';

export default function TemplateDelete(props) {
  const { t } = useTranslation();
  const [confirm, setConfirm] = useState(false);
  const [busy, setBusy] = useState(false);
  const busyRef = useRef(false);
  const [error, setError] = useState('');
  const close = () => {
    if (busyRef.current) return;
    setConfirm(false);
    setError('');
  };
  const remove = async () => {
    if (busyRef.current) return;
    busyRef.current = true;
    setBusy(true);
    setError('');
    try {
      const result = await promptRequest(
        'delete',
        `/api/prompt-center/admin/${props.template.id}?version=${encodeURIComponent(props.template.version)}`,
      );
      showSuccess(
        t('Deleted {{title}} and uninstalled it from {{tokens}} tokens', {
          title: props.template.title,
          tokens: result?.uninstalled_tokens ?? 0,
        }),
      );
      setConfirm(false);
      props.onDeleted(props.template.id);
    } catch (reason) {
      setError(promptError(reason, t));
    } finally {
      busyRef.current = false;
      setBusy(false);
    }
  };
  return (
    <>
      <Button
        className='prompt-center-delete-button'
        type='danger'
        theme='borderless'
        icon={<Trash2 size={14} aria-hidden='true' />}
        disabled={busy}
        aria-label={t('Delete {{title}}', { title: props.template.title })}
        onClick={() => {
          setError('');
          setConfirm(true);
        }}
      >
        {t('Delete')}
      </Button>
      {confirm && (
        <Modal
          visible
          title={t('Delete skill?')}
          closable={!busy}
          closeOnEsc={!busy}
          maskClosable={false}
          onCancel={close}
          footer={
            <div className='prompt-center-row'>
              <Button disabled={busy} onClick={close}>
                {t('Cancel')}
              </Button>
              <Button
                type='danger'
                theme='solid'
                loading={busy}
                disabled={busy}
                onClick={remove}
              >
                {t('Delete')}
              </Button>
            </div>
          }
        >
          <p>
            {t(
              'Deleting “{{title}}” automatically uninstalls it from every token that installed it. This action cannot be undone.',
              { title: props.template.title },
            )}
          </p>
          {error && (
            <p role='alert' className='prompt-template-delete-error'>
              {error}
            </p>
          )}
        </Modal>
      )}
    </>
  );
}
