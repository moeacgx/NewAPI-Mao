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
import { useTranslation } from 'react-i18next';
import {
  ArrowLeftRight,
  Box,
  ChevronDown,
  Clock3,
  Hash,
  KeyRound,
  Layers,
  Radio,
  RotateCw,
  Zap,
} from 'lucide-react';
import { resolveModelProvider } from '../../helpers/modelProvider';
import { getLobeHubIcon } from '../../helpers/render';
import wanIcon from '../../assets/wan.png';

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
  const [expanded, setExpanded] = useState(false);
  const detailsId = useId();
  const modelName = request.model_name || t('Unknown model');
  const provider = resolveModelProvider(request.model_name);
  const statusLabel = t(
    ACTIVE_STATUS_KEYS[request.status] || 'Processing request',
  );
  const elapsedLabel = `${elapsed.toFixed(1)} s`;
  const firstResponseLabel =
    request.first_response_ms == null
      ? t('Waiting for response')
      : `${(request.first_response_ms / 1000).toFixed(2)} s`;
  const providerMark = (
    <span
      className='active-request-provider'
      data-provider={provider?.name || 'unknown'}
      aria-hidden='true'
    >
      {provider?.name === 'Wan' ? (
        <img src={wanIcon} alt='' />
      ) : provider ? (
        getLobeHubIcon(provider.icon, props.compact ? 18 : 26)
      ) : (
        <Box size={props.compact ? 18 : 24} />
      )}
    </span>
  );
  const status = (
    <span className='active-request-status' title={statusLabel}>
      <span className='active-request-state-mark' aria-hidden='true'>
        {request.status === 'receiving' ? (
          <>
            <i />
            <i />
            <i />
          </>
        ) : request.status === 'processing' || request.status === 'retrying' ? (
          <RotateCw size={11} />
        ) : (
          <i />
        )}
      </span>
      <span>{statusLabel}</span>
    </span>
  );
  const token = (
    <div className='active-request-token' title={request.token_name}>
      <KeyRound size={12} aria-hidden='true' />
      <span>{request.token_name || t('Unnamed token')}</span>
    </div>
  );
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
      className={`active-request-card${props.compact ? ' active-request-card-compact' : ''}${props.wide ? ' active-request-card-wide' : ''}`}
      data-status={request.status}
      aria-label={modelName}
    >
      {props.compact ? (
        <button
          type='button'
          className='active-request-compact-row'
          aria-label={`${modelName}: ${statusLabel}, ${t('Elapsed time')} ${elapsedLabel}`}
          aria-expanded={expanded}
          aria-controls={expanded ? detailsId : undefined}
          onClick={() => setExpanded(!expanded)}
        >
          {providerMark}
          <strong title={modelName}>{modelName}</strong>
          {status}
          <span className='active-request-row-elapsed'>{elapsedLabel}</span>
          <ChevronDown size={13} aria-hidden='true' />
        </button>
      ) : (
        <div className='active-request-heading'>
          {providerMark}
          <div className='active-request-identity'>
            <strong title={modelName}>{modelName}</strong>
            {token}
          </div>
          {status}
        </div>
      )}
      {(!props.compact || expanded) && (
        <div className='active-request-body' id={detailsId}>
          {props.compact && (
            <div className='active-request-expanded-identity'>
              <strong>{modelName}</strong>
              {token}
            </div>
          )}
          <dl className='active-request-metrics'>
            <div>
              <dt>
                <Clock3 size={13} aria-hidden='true' />
                {t('Elapsed time')}
              </dt>
              <dd aria-label={elapsedLabel}>
                {elapsed.toFixed(1)}
                <span className='active-request-unit'> s</span>
              </dd>
            </div>
            <div>
              <dt>
                <Zap size={13} aria-hidden='true' />
                {t('First response')}
              </dt>
              <dd
                aria-label={firstResponseLabel}
                className={
                  request.first_response_ms == null
                    ? 'active-request-pending'
                    : ''
                }
              >
                {request.first_response_ms == null ? (
                  t('Waiting for response')
                ) : (
                  <>
                    {(request.first_response_ms / 1000).toFixed(2)}
                    <span className='active-request-unit'> s</span>
                  </>
                )}
              </dd>
            </div>
          </dl>
          <div className='active-request-footer'>
            <span className='active-request-protocol'>
              <ArrowLeftRight size={12} aria-hidden='true' />
              <span>
                {protocolNames[request.request_type] ||
                  request.request_type ||
                  '—'}
              </span>
            </span>
            <span
              className='active-request-mode'
              data-stream={request.is_stream}
            >
              {request.is_stream ? (
                <Radio size={12} aria-hidden='true' />
              ) : (
                <Layers size={12} aria-hidden='true' />
              )}
              {request.is_stream
                ? t('Streaming request')
                : t('Non-streaming request')}
            </span>
            {request.retry_count > 0 && (
              <span className='active-request-retry'>
                {t('Retry count: {{count}}', { count: request.retry_count })}
              </span>
            )}
          </div>
          {(props.showRequestId || props.compact) && (
            <div className='active-request-id'>
              <Hash size={12} aria-hidden='true' />
              <code title={request.request_id || request.id}>
                {request.request_id || request.id}
              </code>
            </div>
          )}
        </div>
      )}
    </article>
  );
}
