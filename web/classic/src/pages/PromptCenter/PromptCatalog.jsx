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
import { useTranslation } from 'react-i18next';

export default function PromptCatalog(props) {
  const { t } = useTranslation();
  return (
    <section className='classic-console-panel' aria-label={t('Prompt catalog')}>
      <div className='classic-console-panel-header px-4 py-3'>
        <h2>{t('Prompt catalog')}</h2>
      </div>
      <div className='classic-console-panel-content prompt-center-grid'>
        {!props.templates.length && (
          <p role='status'>{t('No prompts available')}</p>
        )}
        {props.templates.map((template) => {
          const installed = props.bindings.some(
            (binding) => binding.template.id === template.id,
          );
          return (
            <article className='prompt-center-card' key={template.id}>
              <div className='prompt-center-row'>
                <h3>{template.title}</h3>
                {!template.enabled && <span>{t('Unpublished')}</span>}
              </div>
              <p>{template.description}</p>
              <p className='prompt-center-muted'>
                {t('Allowed groups')}:{' '}
                {template.group_details
                  ?.map((group) => group.name)
                  .join(', ') || t('No eligible groups')}
              </p>
              <details>
                <summary>{t('View prompt')}</summary>
                <pre className='prompt-center-content'>{template.content}</pre>
              </details>
              <div className='prompt-center-row'>
                <Button
                  disabled={
                    !props.canInstall ||
                    installed ||
                    !template.enabled ||
                    props.busy ||
                    props.bindings.length >= 8
                  }
                  onClick={() => props.onInstall(template)}
                  aria-label={t('Install {{title}}', { title: template.title })}
                >
                  {installed ? t('Installed') : t('Install')}
                </Button>
                {props.onEdit && (
                  <Button
                    theme='borderless'
                    onClick={() => props.onEdit(template)}
                    aria-label={t('Edit {{title}}', { title: template.title })}
                  >
                    {t('Edit template')}
                  </Button>
                )}
              </div>
            </article>
          );
        })}
      </div>
    </section>
  );
}
