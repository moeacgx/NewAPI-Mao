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

import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal, RadioGroup, Radio, Checkbox, Space } from '@douyinfe/semi-ui';
import { downloadTextAsFile, renderQuota } from '../../../../helpers';
import { buildRedemptionExport } from '../redemptionExport';

export default function RedemptionExportModal({ data, onClose }) {
  const { t } = useTranslation();
  const [format, setFormat] = useState('txt');
  const [includeName, setIncludeName] = useState(false);
  const [includeQuota, setIncludeQuota] = useState(false);
  return (
    <Modal
      okButtonProps={{ 'aria-label': t('下载') }}
      cancelButtonProps={{ 'aria-label': t('取消') }}
      visible
      title={t('导出兑换码')}
      onCancel={onClose}
      okText={t('下载')}
      cancelText={t('取消')}
      onOk={() => {
        const result = buildRedemptionExport({
          ...data,
          quota: renderQuota(data.quota, 6),
          format,
          includeName,
          includeQuota,
          labels: { code: t('兑换码'), name: t('名称'), quota: t('额度') },
        });
        downloadTextAsFile(result.text, result.filename);
        onClose();
      }}
    >
      <Space vertical align='start'>
        <p>{t('兑换码创建成功，是否下载兑换码？')}</p>
        <RadioGroup
          aria-label={t('文件格式')}
          value={format}
          onChange={(event) => setFormat(event.target.value)}
        >
          <Radio value='txt'>TXT</Radio>
          <Radio value='md'>Markdown</Radio>
        </RadioGroup>
        <Checkbox
          checked={includeName}
          onChange={(event) => setIncludeName(event.target.checked)}
        >
          {t('包含名称')}
        </Checkbox>
        <Checkbox
          checked={includeQuota}
          onChange={(event) => setIncludeQuota(event.target.checked)}
        >
          {t('包含额度')}
        </Checkbox>
      </Space>
    </Modal>
  );
}
