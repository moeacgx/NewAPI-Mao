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
import {
  Button,
  Divider,
  Empty,
  Popover,
  Spin,
  Typography,
} from '@douyinfe/semi-ui';
import { Copy, Globe, ChevronDown, Server } from 'lucide-react';
import { API, copy, showError, showSuccess } from '../../../helpers';

const { Text } = Typography;

const TokenApiInfoPopover = ({ t }) => {
  const [visible, setVisible] = useState(false);
  const [loading, setLoading] = useState(false);
  const [apiInfo, setApiInfo] = useState([]);

  const loadApiInfo = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/status');
      const status = res?.data?.data || {};
      const items =
        status.api_info_enabled === false || !Array.isArray(status.api_info)
          ? []
          : status.api_info.filter(
              (item) => item && typeof item.url === 'string' && item.url,
            );
      const serverAddress =
        typeof status.server_address === 'string'
          ? status.server_address.trim()
          : '';
      setApiInfo(
        items.length
          ? items
          : [
              {
                url: serverAddress || window.location.origin,
                route: t(
                  serverAddress ? 'Default API address' : 'Current domain',
                ),
              },
            ],
      );
    } catch (error) {
      setApiInfo([]);
      showError(error?.message || t('加载失败'));
    } finally {
      setLoading(false);
    }
  };

  const handleVisibleChange = (nextVisible) => {
    setVisible(nextVisible);
    if (nextVisible) {
      loadApiInfo();
    }
  };

  const handleCopy = async (url) => {
    if (await copy(url)) {
      showSuccess(t('复制成功'));
    } else {
      showError(t('复制失败'));
    }
  };

  const content = (
    <div className='tokens-default-overlay tokens-api-popover'>
      <div className='mb-2 flex items-center gap-2 text-sm font-semibold'>
        <Server size={16} />
        {t('API信息')}
      </div>
      {loading ? (
        <div className='flex min-h-20 items-center justify-center'>
          <Spin size='small' />
        </div>
      ) : apiInfo.length > 0 ? (
        <div className='max-h-72 overflow-y-auto'>
          {apiInfo.map((item, index) => (
            <React.Fragment key={`${item.url}-${item.route || index}`}>
              {index > 0 && <Divider margin='12px' />}
              <div className='min-w-0 space-y-1'>
                <div className='flex items-center gap-2'>
                  <span
                    className='h-2 w-2 shrink-0 rounded-full'
                    style={{
                      backgroundColor:
                        item.color || 'var(--semi-color-primary)',
                    }}
                  />
                  <Text strong ellipsis={{ showTooltip: true }}>
                    {item.route || item.url}
                  </Text>
                </div>
                <div className='flex items-center gap-1'>
                  <Text
                    ellipsis={{ showTooltip: true }}
                    className='min-w-0 flex-1 font-mono text-xs'
                  >
                    {item.url}
                  </Text>
                  <Button
                    theme='borderless'
                    type='tertiary'
                    size='small'
                    icon={<Copy size={14} />}
                    aria-label={t('API地址')}
                    onClick={() => handleCopy(item.url)}
                  />
                </div>
                {item.description && (
                  <Text type='tertiary' size='small'>
                    {item.description}
                  </Text>
                )}
              </div>
            </React.Fragment>
          ))}
        </div>
      ) : (
        <Empty
          image={null}
          title={t('暂无API信息')}
          description={t('请联系管理员在系统设置中配置API信息')}
          className='py-2'
        />
      )}
    </div>
  );

  return (
    <Popover
      content={content}
      position='bottomRight'
      trigger='click'
      visible={visible}
      onVisibleChange={handleVisibleChange}
      showArrow
    >
      <Button
        type='tertiary'
        size='small'
        icon={<Globe size={15} />}
        theme='outline'
        className='tokens-api-info'
      >
        {t('API地址')} <ChevronDown size={13} style={{ marginLeft: 4 }} />
      </Button>
    </Popover>
  );
};

export default TokenApiInfoPopover;
