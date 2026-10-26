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
import { Sparkles } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import TokenPromptInstallationModal from '../../../pages/PromptCenter/TokenPromptInstallationModal';
import '../../../pages/PromptCenter/style.css';

export default function TokenPromptBadge(props) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [updated, setUpdated] = useState(null);
  useEffect(() => {
    setUpdated(null);
  }, [props.token.prompt_count, props.token.enabled_prompt_count]);
  const count = updated?.count ?? props.token.prompt_count ?? 0;
  const enabled = updated?.enabled ?? props.token.enabled_prompt_count ?? 0;
  if (!count && !open) return null;
  return (
    <>
      {count > 0 && (
        <button
          type='button'
          className='token-prompt-badge'
          data-active={enabled > 0}
          onClick={() => setOpen(true)}
          aria-label={t(
            '{{name}}: {{count}} installed prompts, {{enabled}} enabled',
            { name: props.token.name, count, enabled },
          )}
        >
          <Sparkles size={12} aria-hidden='true' />
          {t('Skills')} · {count}
        </button>
      )}
      {open && (
        <TokenPromptInstallationModal
          token={props.token}
          showCenterLink
          onClose={() => setOpen(false)}
          onChange={(items) =>
            setUpdated({
              count: items.length,
              enabled: items.filter(
                (item) => item.enabled && item.template.enabled,
              ).length,
            })
          }
        />
      )}
    </>
  );
}
