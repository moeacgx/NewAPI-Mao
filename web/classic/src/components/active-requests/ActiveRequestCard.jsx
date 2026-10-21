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
import { useTranslation } from 'react-i18next';
import { KeyRound } from 'lucide-react';

export const ACTIVE_STATUS_KEYS = {
  waiting_response: 'Waiting for response',
  receiving: 'Receiving response',
  processing: 'Processing request',
  retrying: 'Retrying request',
};

export default function ActiveRequestCard(props) {
  const { t } = useTranslation();
  const request = props.request;
  const elapsed = Math.max(0, props.serverNow - request.started_at) / 1000;
  const protocolNames = {
    openai: 'OpenAI Chat Completions',
    claude: 'Claude Messages',
    gemini: 'Gemini',
    openai_responses: 'OpenAI Responses',
    openai_responses_compaction: 'OpenAI Responses Compaction',
    openai_alpha_search: 'OpenAI Alpha Search',
    openai_audio: 'OpenAI Audio',
    openai_image: 'OpenAI Images',
    openai_realtime: 'OpenAI Realtime',
    rerank: 'Rerank',
    embedding: 'Embeddings',
    task: t('Task request'),
    mj_proxy: 'Midjourney',
  };
  return (
    <article
      className='active-request-card'
      data-status={request.status}
      aria-label={request.model_name || t('Unknown model')}
    >
      <div className='active-request-heading'>
        <strong title={request.model_name}>
          {request.model_name || t('Unknown model')}
        </strong>
        <span className='active-request-status'>
          {t(ACTIVE_STATUS_KEYS[request.status] || 'Processing request')}
        </span>
      </div>
      <div className='active-request-token' title={request.token_name}>
        <KeyRound size={13} aria-hidden='true' />
        <span>{request.token_name || t('Unnamed token')}</span>
      </div>
      <dl className='active-request-metrics'>
        <div>
          <dt>{t('Elapsed time')}</dt>
          <dd>{elapsed.toFixed(1)} s</dd>
        </div>
        <div>
          <dt>{t('First response')}</dt>
          <dd
            className={
              request.first_response_ms == null ? 'active-request-pending' : ''
            }
          >
            {request.first_response_ms == null
              ? t('Waiting for response')
              : (request.first_response_ms / 1000).toFixed(2) + ' s'}
          </dd>
        </div>
      </dl>
      <div className='active-request-footer'>
        <span>
          {protocolNames[request.request_type] || request.request_type || '—'} ·{' '}
          {request.is_stream
            ? t('Streaming request')
            : t('Non-streaming request')}
        </span>
        {request.retry_count > 0 && (
          <span>
            {t('Retry count: {{count}}', { count: request.retry_count })}
          </span>
        )}
      </div>
      {props.showRequestId && (
        <div className='active-request-id'>
          {request.request_id || request.id}
        </div>
      )}
    </article>
  );
}
