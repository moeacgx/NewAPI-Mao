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
import { Folder, GitBranch, GitCommitHorizontal, Github } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export default function RepositorySource(props) {
  const { t } = useTranslation();
  const source = props.source;
  if (!source) return null;
  let repositoryURL;
  try {
    const parsed = new URL(source.url);
    if (
      parsed.protocol === 'https:' &&
      parsed.hostname === 'github.com' &&
      !parsed.port &&
      !parsed.username &&
      !parsed.password &&
      !parsed.search &&
      !parsed.hash
    )
      repositoryURL = parsed.href;
  } catch {
    // 来源异常时仅显示转义后的文本，不生成可执行链接。
  }
  return (
    <div className='prompt-repository-source'>
      <div className='prompt-repository-source-heading'>
        <Github size={14} aria-hidden='true' />
        {repositoryURL ? (
          <a
            href={repositoryURL}
            target='_blank'
            rel='noopener noreferrer'
            aria-label={t('GitHub source')}
          >
            {t('GitHub source')}
          </a>
        ) : (
          <span>{t('GitHub source')}</span>
        )}
        {source.locally_modified && (
          <span className='prompt-repository-modified'>
            {t('Local skill changes')}
          </span>
        )}
      </div>
      <div className='prompt-repository-url'>{source.url}</div>
      <div className='prompt-repository-details'>
        <span title={t('Repository reference')}>
          <GitBranch size={12} aria-hidden='true' />
          {source.ref}
        </span>
        <span title={t('Repository path')}>
          <Folder size={12} aria-hidden='true' />
          {source.path || t('Repository root')}
        </span>
        <span title={t('Repository commit')}>
          <GitCommitHorizontal size={12} aria-hidden='true' />
          <code>{source.commit?.slice(0, 8)}</code>
        </span>
      </div>
    </div>
  );
}
