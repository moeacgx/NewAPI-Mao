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

import React, { useEffect, useRef, useState } from 'react';
import { Button, Modal, Spin } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import TelegramLoginButton from 'react-telegram-login';
import { API } from '../../../../helpers/api';
import { showError, showSuccess } from '../../../../helpers/utils';

const TelegramBindModal = ({ visible, onCancel, botName, onSuccess }) => {
  const { t } = useTranslation();
  const [flow, setFlow] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [attempt, setAttempt] = useState(0);
  const pendingFlow = useRef('');

  useEffect(() => {
    let active = true;
    pendingFlow.current = '';
    setFlow(null);
    setError('');
    if (!visible) return;
    setLoading(true);
    void (async () => {
      try {
        const response = await API.post('/api/oauth/telegram/bind/start');
        if (!active) return;
        const { success, data, message } = response.data;
        if (!success || !data?.flow_token || !data?.callback_url) {
          throw new Error(message || t('授权失败'));
        }
        const callback = new URL(data.callback_url, window.location.origin);
        if (callback.origin !== window.location.origin) {
          throw new Error(t('授权失败'));
        }
        pendingFlow.current = data.flow_token;
        setFlow({ token: data.flow_token, url: callback.toString() });
      } catch (error) {
        if (active) setError(error.message || t('授权失败'));
      } finally {
        if (active) setLoading(false);
      }
    })();
    return () => {
      active = false;
      pendingFlow.current = '';
    };
  }, [visible, attempt, t]);

  useEffect(() => {
    if (!visible || !flow) return;
    const handleResult = async (event) => {
      const result = event.data;
      if (
        event.origin !== window.location.origin ||
        !result ||
        result.type !== 'telegram:binding:result' ||
        !pendingFlow.current ||
        result.flow_token !== pendingFlow.current
      )
        return;
      // 一次性结果只处理一次，失败也需重新发起绑定。
      pendingFlow.current = '';
      setFlow(null);
      if (result.success !== true) {
        setError(t('授权失败'));
        return;
      }
      try {
        await onSuccess?.();
        showSuccess(t('绑定成功！'));
        onCancel();
      } catch (error) {
        showError(error.message || t('操作失败'));
        setError(t('操作失败'));
      }
    };
    window.addEventListener('message', handleResult);
    return () => window.removeEventListener('message', handleResult);
  }, [visible, flow, onSuccess, onCancel, t]);

  return (
    <Modal
      title={t('绑定 Telegram')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
    >
      <div className='my-3 text-sm text-gray-600'>
        {t('点击下方按钮通过 Telegram 完成绑定')}
      </div>
      <div className='flex flex-col items-center gap-3 rounded-lg border p-4'>
        {loading && <Spin />}
        {error && (
          <>
            <p role='alert'>{error}</p>
            <Button onClick={() => setAttempt((value) => value + 1)}>
              {t('重试')}
            </Button>
          </>
        )}
        {flow && (
          <TelegramLoginButton
            key={flow.token}
            dataAuthUrl={flow.url}
            botName={botName}
          />
        )}
      </div>
    </Modal>
  );
};

export default TelegramBindModal;
