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

import React, { useEffect, useState } from 'react';
import { Button, Select, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API } from '../../../../helpers/api';
import { pluginError } from '../../../../pages/TaskPlugins/api';

export default function TaskPluginSelect(props) {
  const { t } = useTranslation();
  const [options, setOptions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [retry, setRetry] = useState(0);
  useEffect(() => {
    if (props.disabled) {
      setLoading(false);
      return;
    }
    const controller = new AbortController();
    setLoading(true);
    setOptions([]);
    setError('');
    API.get('/api/task_plugin_options', {
      signal: controller.signal,
      skipErrorHandler: true,
    })
      .then((response) => {
        if (
          response.data?.success !== true ||
          !Array.isArray(response.data.data)
        )
          throw new Error(
            response.data?.message || t('Task plugin request failed'),
          );
        setOptions(
          response.data.data.filter(
            (item) => item.channel_type === 62 && typeof item.key === 'string',
          ),
        );
      })
      .catch((err) => {
        if (!controller.signal.aborted) setError(pluginError(err, t));
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [props.disabled, retry, t]);
  const list = options.map((item) => ({
    value: item.key,
    label: (item.name || item.key) + ' (' + item.key + ')',
  }));
  if (props.value && !list.some((item) => item.value === props.value))
    list.push({ value: props.value, label: props.value });
  return (
    <div style={{ marginBottom: 16 }}>
      <label id='task-plugin-label'>{t('Task plugin')}</label>
      <Select
        aria-labelledby='task-plugin-label'
        style={{ width: '100%' }}
        value={props.value || undefined}
        loading={loading}
        disabled={
          props.disabled || loading || Boolean(error) || options.length === 0
        }
        placeholder={t('Select task plugin')}
        optionList={list}
        onChange={(value) =>
          props.onChange(
            value,
            options.find((item) => item.key === value)?.models,
          )
        }
      />
      <Typography.Text type='tertiary'>
        {t(
          'Selecting a plugin fills models only when the model list is empty.',
        )}
      </Typography.Text>
      {props.disabled && (
        <div>{t('No permission to change task plugin binding')}</div>
      )}
      {error && (
        <div role='alert'>
          {error}
          <Button onClick={() => setRetry(retry + 1)}>{t('Retry')}</Button>
        </div>
      )}
      {!loading && !props.disabled && !error && options.length === 0 && (
        <div>{t('No task plugins available for binding')}</div>
      )}
    </div>
  );
}
