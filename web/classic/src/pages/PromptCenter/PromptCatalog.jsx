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
import {
  ArrowUpRight,
  Check,
  CheckCircle2,
  ChevronDown,
  CircleSlash2,
  Layers3,
  PencilLine,
  ShieldCheck,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import SkillSummary from './SkillSummary';
import RepositoryUpdate from './RepositoryUpdate';
import TemplateDelete from './TemplateDelete';

export default function PromptCatalog(props) {
  const { t } = useTranslation();
  const prefix = useId();
  const [expanded, setExpanded] = useState(new Set());
  const toggle = (id) =>
    setExpanded((current) => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  return (
    <section className='prompt-center-section' aria-label={t('Prompt catalog')}>
      <div className='prompt-center-section-heading'>
        <div className='prompt-center-section-icon' aria-hidden='true'>
          <Layers3 size={18} />
        </div>
        <h2>{t('Prompt catalog')}</h2>
        <span className='prompt-center-count'>{props.templates.length}</span>
      </div>
      <div className='prompt-center-grid'>
        {!props.templates.length && (
          <div className='prompt-center-empty-state' role='status'>
            <Layers3 size={24} aria-hidden='true' />
            <span>{t('No prompts available')}</span>
          </div>
        )}
        {props.templates.map((template) => {
          const installed = props.bindings.some(
            (binding) => binding.template.id === template.id,
          );
          return (
            <article
              className='prompt-center-card prompt-center-catalog-card'
              key={template.id}
              onClick={(event) => {
                // 卡片正文可展开；安装、编辑、删除及合集明细保留各自操作。
                if (
                  event.target.closest(
                    'button, a, input, select, textarea, summary, details, [role="dialog"]',
                  )
                )
                  return;
                toggle(template.id);
              }}
            >
              <div className='prompt-center-card-topline'>
                <h3 title={template.title}>
                  <button
                    type='button'
                    className='prompt-center-card-title'
                    aria-expanded={expanded.has(template.id)}
                    aria-controls={`${prefix}-groups-${template.id}`}
                    onClick={() => toggle(template.id)}
                  >
                    <span>{template.title}</span>
                    <ChevronDown size={16} aria-hidden='true' />
                  </button>
                </h3>
                <span
                  className={`prompt-center-status ${template.enabled ? 'is-published' : 'is-unpublished'}`}
                >
                  {template.enabled ? (
                    <CheckCircle2 size={12} aria-hidden='true' />
                  ) : (
                    <CircleSlash2 size={12} aria-hidden='true' />
                  )}
                  {template.enabled ? t('Published') : t('Unpublished')}
                </span>
              </div>
              <SkillSummary template={template} />
              <p className='prompt-center-card-description'>
                {template.description || t('Prompt')}
              </p>
              {props.onEdit &&
                props.onRepositoryUpdated &&
                template.repository_source && (
                  <RepositoryUpdate
                    template={template}
                    onUpdated={props.onRepositoryUpdated}
                  />
                )}
              {expanded.has(template.id) && (
                <div
                  className='prompt-center-card-group-details'
                  id={`${prefix}-groups-${template.id}`}
                >
                  <span className='prompt-center-meta-label'>
                    <ShieldCheck size={13} aria-hidden='true' />
                    {t('Allowed groups')}
                  </span>
                  <div className='prompt-center-group-chips'>
                    {template.all_groups && (
                      <span className='prompt-center-group-chip'>
                        {t('All groups (no restriction)')}
                      </span>
                    )}
                    {!template.all_groups &&
                      template.group_details?.map((group) => (
                        <span
                          className='prompt-center-group-chip'
                          key={group.id}
                        >
                          {group.name}
                        </span>
                      ))}
                    {!template.all_groups &&
                      !template.group_details?.length && (
                        <span className='prompt-center-muted'>
                          {t('No eligible groups')}
                        </span>
                      )}
                  </div>
                </div>
              )}
              <div className='prompt-center-card-footer'>
                <div className='prompt-center-card-actions'>
                  <Button
                    className='prompt-center-install-button'
                    icon={
                      installed ? (
                        <Check size={14} />
                      ) : (
                        <ArrowUpRight size={14} />
                      )
                    }
                    disabled={
                      !props.canInstall ||
                      installed ||
                      !template.enabled ||
                      props.busy ||
                      props.bindings.length >= 8
                    }
                    onClick={() => props.onInstall(template)}
                    aria-label={t('Install {{title}}', {
                      title: template.title,
                    })}
                  >
                    {installed ? t('Installed') : t('Install to token')}
                  </Button>
                  {props.onEdit && (
                    <div className='prompt-center-card-admin-actions'>
                      <Button
                        className='prompt-center-edit-button'
                        theme='borderless'
                        icon={<PencilLine size={14} />}
                        onClick={() => props.onEdit(template)}
                        aria-label={t('Edit {{title}}', {
                          title: template.title,
                        })}
                      >
                        {t('Edit template')}
                      </Button>
                      {props.onDeleted && (
                        <TemplateDelete
                          template={template}
                          onDeleted={props.onDeleted}
                        />
                      )}
                    </div>
                  )}
                </div>
              </div>
            </article>
          );
        })}
      </div>
    </section>
  );
}
