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
import { Button, Card, Tag, Typography } from '@douyinfe/semi-ui';
import { Gift } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { getCurrencyConfig, renderQuota } from '../../helpers';
import {
  benefitClaimReasonLabel,
  formatDisplayAmount,
  isBenefitActivityClaimable,
} from './benefitLabels';

const { Title, Text } = Typography;

// Fixed mode: every remaining share is worth exactly fixed_quota. Random
// mode: shares vary between min_quota and max_quota, so a range is shown
// instead of a single number. Both are real per-share backend fields, not a
// client-side average.
const SharePriceValue = ({ activity }) => {
  if (activity.amount_mode === 'random') {
    return (
      <>
        {renderQuota(activity.min_quota)} ~ {renderQuota(activity.max_quota)}
      </>
    );
  }
  return <>{renderQuota(activity.fixed_quota)}</>;
};

export default function ClaimableActivityCard(props) {
  const { t } = useTranslation();
  const { activity, onClaim, claiming } = props;
  const claimable = isBenefitActivityClaimable(activity);

  return (
    <Card
      className='!rounded-lg border border-[var(--semi-color-border)] bg-[var(--semi-color-bg-0)] shadow-sm'
      bodyStyle={{ padding: 16 }}
    >
      <div className='flex flex-wrap items-start justify-between gap-3'>
        <div className='min-w-0'>
          <Title heading={6} className='!mb-1 truncate'>
            {activity.name}
          </Title>
          <Text type='tertiary' size='small' className='truncate'>
            {activity.group_name_snapshot}
          </Text>
        </div>
        {activity.has_claimed ? (
          <Tag color='green'>{t('Already claimed')}</Tag>
        ) : claimable ? (
          <Button
            theme='solid'
            type='primary'
            icon={<Gift size={14} />}
            loading={claiming}
            onClick={() => onClaim(activity.id)}
          >
            {t('Claim')}
          </Button>
        ) : (
          <Tag color='grey'>
            {benefitClaimReasonLabel(t, activity.eligibility_reason)}
          </Tag>
        )}
      </div>
      <div className='mt-3 rounded-md bg-[var(--semi-color-fill-0)] px-3 py-2 text-sm'>
        <span className='text-[var(--semi-color-text-2)]'>
          {t('Claim requirement')}：
        </span>{' '}
        {Number(activity.claim_paid_threshold || 0) > 0
          ? t('Historical top-up of at least {{amount}}', {
              amount: formatDisplayAmount(
                t,
                activity.claim_paid_threshold,
                getCurrencyConfig(),
              ),
            })
          : t('No historical top-up requirement')}
      </div>

      <div className='mt-3 grid grid-cols-2 gap-3 border-t border-[var(--semi-color-border)] pt-3 text-sm sm:grid-cols-3'>
        <div>
          <Text type='tertiary' size='small'>
            {t('Shares remaining')}
          </Text>
          <div className='font-semibold'>
            {activity.remaining_count || 0} / {activity.total_count || 0}
          </div>
        </div>
        <div>
          <Text type='tertiary' size='small'>
            {t('Amount per share')}
          </Text>
          <div className='font-semibold'>
            <SharePriceValue activity={activity} />
          </div>
        </div>
        <div>
          <Text type='tertiary' size='small'>
            {t('Personal validity')}
          </Text>
          <div className='font-semibold'>
            {t('{{hours}}h', {
              hours: Number(activity.personal_valid_hours || 0),
            })}
          </div>
        </div>
      </div>
    </Card>
  );
}
