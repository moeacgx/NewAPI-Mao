/*
Copyright (C) 2023-2026 QuantumNous

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

import React, { useState, useEffect } from 'react';
import {
  Table,
  Button,
  Empty,
  SideSheet,
  Tag,
  Typography,
  Card,
  Space,
  Spin,
} from '@douyinfe/semi-ui';
import { IconEyeOpened } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, timestamp2string } from '../../helpers';
import {
  SupplierConversationDialog,
  appendSupplierMessage,
  applySupplierUnread,
  supplierDetailColumnStyle,
  supplierDetailFactsStyle,
  supplierSheetBodyStyle,
  useSupplierSheetWidth,
} from './SupplierConversation';

const { Text, Paragraph } = Typography;

const statusColors = {
  pending: 'amber',
  contacted: 'blue',
  cooperating: 'green',
  rejected: 'red',
};

const statusLabels = {
  pending: 'Pending',
  contacted: 'Contacted',
  cooperating: 'Cooperating',
  rejected: 'Rejected',
};

const contactTypeLabels = {
  wechat: 'WeChat',
  qq: 'QQ',
  telegram: 'Telegram',
  email: 'Email',
  phone: 'Phone',
  other: 'Other',
};

const platformLabels = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
  gemini: 'Gemini',
  grok: 'Grok',
  deepseek: 'DeepSeek',
  other: 'Other',
};

const MySupplierSubmissions = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [applications, setApplications] = useState([]);
  const [pagination, setPagination] = useState({
    currentPage: 1,
    pageSize: 10,
    total: 0,
  });
  const [selectedApp, setSelectedApp] = useState(null);
  const [detailVisible, setDetailVisible] = useState(false);
  const [conversationOpen, setConversationOpen] = useState(false);
  const sheetWidth = useSupplierSheetWidth(600);

  const loadApplications = async (page = 1) => {
    setLoading(true);
    try {
      const res = await API.get('/api/supplier-applications', {
        params: { p: page, page_size: pagination.pageSize },
      });
      const data = res.data;
      if (data.success) {
        setApplications(data.data.items || []);
        setPagination({
          currentPage: data.data.page || 1,
          pageSize: data.data.page_size || 10,
          total: data.data.total || 0,
        });
      } else {
        showError(data.message || t('Could not load submissions.'));
      }
    } catch (error) {
      showError(t('Could not load submissions.'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadApplications();
  }, []);

  const handlePageChange = (page) => {
    loadApplications(page);
  };

  const conversationFor = (record, lockSheet) => (
    <SupplierConversationDialog
      applicationId={record.id}
      siteName={record.site_name}
      messages={record.messages}
      canSend
      mode='personal'
      unreadCount={record.unread_count}
      onOpenChange={lockSheet ? setConversationOpen : undefined}
      onRead={(unread) => {
        setSelectedApp((current) =>
          applySupplierUnread(current, record.id, unread),
        );
        setApplications((current) =>
          current.map((item) => applySupplierUnread(item, record.id, unread)),
        );
      }}
      onSent={(message) => {
        setSelectedApp((current) =>
          appendSupplierMessage(current, record.id, message),
        );
        setApplications((current) =>
          current.map((item) =>
            appendSupplierMessage(item, record.id, message),
          ),
        );
      }}
    />
  );

  const viewDetail = (record) => {
    setSelectedApp(record);
    setDetailVisible(true);
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: t('Site name'),
      dataIndex: 'site_name',
      render: (text, record) => (
        <span className='inline-flex items-center gap-2'>
          <Text strong>{text}</Text>
          {conversationFor(record, false)}
        </span>
      ),
    },
    {
      title: t('Status'),
      dataIndex: 'status',
      width: 120,
      render: (status) => (
        <Tag color={statusColors[status] || 'grey'}>
          {t(statusLabels[status] || status)}
        </Tag>
      ),
    },
    {
      title: t('Submitted at'),
      dataIndex: 'created_at',
      width: 180,
      render: (time) => timestamp2string(time),
    },
    {
      title: t('Actions'),
      width: 120,
      render: (text, record) => (
        <Button
          icon={<IconEyeOpened />}
          size='small'
          onClick={() => viewDetail(record)}
        >
          {t('View details')}
        </Button>
      ),
    },
  ];

  return (
    <div className='p-4'>
      <Spin spinning={loading}>
        {applications.length === 0 && !loading ? (
          <Empty
            description={t('No submissions yet')}
            style={{ padding: '60px 0' }}
          />
        ) : (
          <Table
            columns={columns}
            dataSource={applications}
            rowKey='id'
            pagination={{
              currentPage: pagination.currentPage,
              pageSize: pagination.pageSize,
              total: pagination.total,
              onPageChange: handlePageChange,
            }}
          />
        )}
      </Spin>

      <SideSheet
        title={
          selectedApp ? (
            <span className='inline-flex max-w-full items-center gap-2'>
              <span style={{ overflowWrap: 'anywhere' }}>
                {`#${selectedApp.id} ${selectedApp.site_name}`}
              </span>
              {conversationFor(selectedApp, true)}
            </span>
          ) : (
            t('View details')
          )
        }
        visible={detailVisible}
        onCancel={() => setDetailVisible(false)}
        closeOnEsc={!conversationOpen}
        width={sheetWidth}
        bodyStyle={supplierSheetBodyStyle}
      >
        {selectedApp && (
          <div data-supplier-detail style={supplierDetailColumnStyle}>
            <div
              data-supplier-detail-facts
              className='space-y-4'
              style={supplierDetailFactsStyle}
            >
              <Card>
                <div className='mb-3'>
                  <Text strong>{t('Status')}: </Text>
                  <Tag color={statusColors[selectedApp.status] || 'grey'}>
                    {t(statusLabels[selectedApp.status] || selectedApp.status)}
                  </Tag>
                </div>
                <div className='mb-3'>
                  <Text strong>{t('Submitted at')}: </Text>
                  <Text>{timestamp2string(selectedApp.created_at)}</Text>
                </div>
              </Card>

              <Card title={t('Site information')}>
                <div className='mb-3'>
                  <Text type='tertiary'>{t('Site name')}</Text>
                  <Paragraph>{selectedApp.site_name}</Paragraph>
                </div>
                {selectedApp.site_url && (
                  <div className='mb-3'>
                    <Text type='tertiary'>{t('Site URL')}</Text>
                    <Paragraph copyable>{selectedApp.site_url}</Paragraph>
                  </div>
                )}
                <div className='mb-3'>
                  <Text type='tertiary'>{t('Contact type')}</Text>
                  <Paragraph>
                    {t(
                      contactTypeLabels[selectedApp.contact_type] ||
                        selectedApp.contact_type,
                    )}
                  </Paragraph>
                </div>
                <div className='mb-3'>
                  <Text type='tertiary'>{t('Contact')}</Text>
                  <Paragraph copyable>{selectedApp.contact}</Paragraph>
                </div>
                {selectedApp.remark && (
                  <div className='mb-3'>
                    <Text type='tertiary'>{t('Remarks')}</Text>
                    <Paragraph style={{ whiteSpace: 'pre-wrap' }}>
                      {selectedApp.remark}
                    </Paragraph>
                  </div>
                )}
              </Card>

              <Card title={t('Group quotes')}>
                {selectedApp.groups && selectedApp.groups.length > 0 ? (
                  <Space vertical spacing='medium' style={{ width: '100%' }}>
                    {selectedApp.groups.map((group, index) => (
                      <Card key={group.id || index} bodyStyle={{ padding: 12 }}>
                        <div className='mb-2'>
                          <Text strong>{group.group_name}</Text>
                        </div>
                        <div className='mb-2'>
                          <Tag color='blue'>
                            {t(
                              platformLabels[group.platform] || group.platform,
                            )}
                          </Tag>
                          <Tag color='cyan'>×{group.ratio}</Tag>
                        </div>
                        {group.models && group.models.length > 0 && (
                          <div className='mb-2'>
                            <Text type='tertiary' size='small'>
                              {t('Main models')}:
                            </Text>
                            <div className='mt-1 flex flex-wrap gap-1'>
                              {group.models.map((model, idx) => (
                                <Tag
                                  key={idx}
                                  size='small'
                                  style={{
                                    maxWidth: '100%',
                                    wordBreak: 'break-word',
                                  }}
                                >
                                  {model}
                                </Tag>
                              ))}
                            </div>
                          </div>
                        )}
                        {group.description && (
                          <div>
                            <Text type='tertiary' size='small'>
                              {t('Notes')}:
                            </Text>
                            <Paragraph
                              size='small'
                              style={{ whiteSpace: 'pre-wrap', marginTop: 4 }}
                            >
                              {group.description}
                            </Paragraph>
                          </div>
                        )}
                      </Card>
                    ))}
                  </Space>
                ) : (
                  <Empty description={t('No group quotes')} />
                )}
              </Card>
            </div>
          </div>
        )}
      </SideSheet>
    </div>
  );
};

export default MySupplierSubmissions;
