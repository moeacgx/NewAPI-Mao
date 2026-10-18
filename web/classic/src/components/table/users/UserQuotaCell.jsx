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
import { Button, Popover, Progress, Typography } from '@douyinfe/semi-ui';
import { getCurrencyConfig, renderQuota } from '../../../helpers/render';
import { formatQuotaDetail } from './quotaDetails';

export default function UserQuotaCell({ record }) {
  const { t } = useTranslation();
  const used = Number(record.used_quota) || 0;
  const remain = Number(record.quota) || 0;
  const total = used + remain;
  const percent =
    total > 0 ? Math.min(100, Math.max(0, (remain / total) * 100)) : 0;
  const config = getCurrencyConfig();
  const quotaPerUnit = localStorage.getItem('quota_per_unit');
  const content = (
    <div className='text-xs p-2'>
      {[
        [t('已用额度'), used],
        [t('剩余额度'), remain],
        [t('总额度'), total],
      ].map(([label, value]) => (
        <div key={label}>
          <Typography.Paragraph
            copyable={{
              content: formatQuotaDetail(
                value,
                { ...config, type: 'USD' },
                quotaPerUnit,
              ),
            }}
          >
            {label}:{' '}
            {formatQuotaDetail(value, { ...config, type: 'USD' }, quotaPerUnit)}
          </Typography.Paragraph>
          <Typography.Paragraph copyable={{ content: String(value) }}>
            {label} (Tokens): {String(value)}
          </Typography.Paragraph>
        </div>
      ))}
    </div>
  );
  return (
    <Popover content={content} position='top' trigger='click'>
      <Button theme='borderless' aria-label={t('额度详情')}>
        <span className='flex flex-col items-end'>
          <span>
            {renderQuota(remain)} / {renderQuota(total)}
          </span>
          <Progress
            percent={percent}
            aria-label={t('剩余额度')}
            style={{ width: '100%' }}
          />
        </span>
      </Button>
    </Popover>
  );
}
