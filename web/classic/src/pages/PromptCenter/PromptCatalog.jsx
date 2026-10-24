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
import { Button } from '@douyinfe/semi-ui';
import {
  ArrowUpRight,
  Check,
  CheckCircle2,
  CircleSlash2,
  Layers3,
  PencilLine,
  ShieldCheck,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import SkillSummary from './SkillSummary';
import RepositoryUpdate from './RepositoryUpdate';

export default function PromptCatalog(props) {
  const { t } = useTranslation();
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
            >
              <div className='prompt-center-card-topline'>
                <h3 title={template.title}>{template.title}</h3>
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
              <div className='prompt-center-card-footer'>
                <div className='prompt-center-card-meta'>
                  <span className='prompt-center-meta-label'>
                    <ShieldCheck size={13} aria-hidden='true' />
                    {t('Allowed groups')}
                  </span>
                  <div className='prompt-center-group-chips'>
                    {template.group_details?.length ? (
                      template.group_details.map((group) => (
                        <span
                          className='prompt-center-group-chip'
                          key={group.id}
                        >
                          {group.name}
                        </span>
                      ))
                    ) : (
                      <span className='prompt-center-muted'>
                        {t('No eligible groups')}
                      </span>
                    )}
                  </div>
                </div>
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
