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
  Input,
  Select,
  Empty,
  SideSheet,
  Tag,
  Typography,
  Card,
  Form,
  TextArea,
  Space,
  Spin,
  Banner,
} from '@douyinfe/semi-ui';
import { IconSearch, IconEyeOpened } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess, timestamp2string } from '../../helpers';
import SupplierConversation, { useSupplierSheetWidth } from './SupplierConversation';
import SupplierMessageCount from './SupplierMessageCount';

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

const SupplierApplicationManagementTable = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [applications, setApplications] = useState([]);
  const [pagination, setPagination] = useState({ currentPage: 1, pageSize: 20, total: 0 });
  const [keyword, setKeyword] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [userIdFilter, setUserIdFilter] = useState('');
  const [selectedApp, setSelectedApp] = useState(null);
  const [detailVisible, setDetailVisible] = useState(false);
  const [saving, setSaving] = useState(false);
  const sheetWidth = useSupplierSheetWidth(700);
  const [hasWritePermission, setHasWritePermission] = useState(false);

  // Check permissions
  useEffect(() => {
    const user = JSON.parse(localStorage.getItem('user') || '{}');
    const hasWrite =
      user.role >= 10 &&
      (user.role === 100 ||
        user.permissions?.admin_permissions?.supplier_applications?.write === true);
    setHasWritePermission(hasWrite);
  }, []);

  const loadApplications = async (page = 1) => {
    setLoading(true);
    try {
      const params = {
        p: page,
        page_size: pagination.pageSize,
      };
      if (keyword) params.keyword = keyword;
      if (statusFilter) params.status = statusFilter;
      if (userIdFilter) params.user_id = userIdFilter;

      const res = await API.get('/api/supplier-application-admin', { params });
      const data = res.data;
      if (data.success) {
        setApplications(data.data.items || []);
        setPagination({
          currentPage: data.data.page || 1,
          pageSize: data.data.page_size || 20,
          total: data.data.total || 0,
        });
      } else {
        showError(data.message || t('Could not load applications.'));
      }
    } catch (error) {
      if (error.response?.status === 403) {
        showError(t('You do not have permission to view supplier applications.'));
      } else {
        showError(t('Could not load applications.'));
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadApplications();
  }, []);

  const handleSearch = () => {
    loadApplications(1);
  };

  const handlePageChange = (page) => {
    loadApplications(page);
  };

  const viewDetail = async (record) => {
    try {
      const res = await API.get(`/api/supplier-application-admin/${record.id}`);
      const data = res.data;
      if (data.success) {
        setSelectedApp(data.data);
        setDetailVisible(true);
      } else {
        showError(data.message || t('Could not load the application.'));
      }
    } catch (error) {
      showError(t('Could not load the application.'));
    }
  };

  const handleSave = async (values) => {
    if (!selectedApp || !hasWritePermission) return;

    setSaving(true);
    try {
      const payload = {
        status: values.status,
        admin_note: values.admin_note?.trim() || '',
        version: selectedApp.version,
      };

      const res = await API.put(`/api/supplier-application-admin/${selectedApp.id}`, payload);
      const data = res.data;
      if (data.success) {
        showSuccess(t('Application updated.'));
        setSelectedApp(data.data);
        loadApplications(pagination.currentPage);
      } else {
        if (data.message?.includes('changed')) {
          showError(t('Supplier application changed, please refresh'));
          // Reload the detail
          const detailRes = await API.get(`/api/supplier-application-admin/${selectedApp.id}`);
          if (detailRes.data.success) {
            setSelectedApp(detailRes.data.data);
          }
        } else {
          showError(data.message || t('Could not update the application.'));
        }
      }
    } catch (error) {
      if (error.response?.status === 409) {
        showError(t('Supplier application changed, please refresh'));
      } else {
        showError(t('Could not update the application.'));
      }
    } finally {
      setSaving(false);
    }
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
          <SupplierMessageCount count={record.message_count} />
        </span>
      ),
    },
    {
      title: t('Contact'),
      dataIndex: 'contact',
      width: 150,
    },
    {
      title: t('User'),
      dataIndex: 'username',
      width: 120,
      render: (text, record) => (
        <div>
          <div>{text}</div>
          <Text type='tertiary' size='small'>
            #{record.user_id}
          </Text>
        </div>
      ),
    },
    {
      title: t('Status'),
      dataIndex: 'status',
      width: 120,
      render: (status) => (
        <Tag color={statusColors[status] || 'grey'}>{t(statusLabels[status] || status)}</Tag>
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
      <div className='mb-4 flex gap-2 flex-wrap'>
        <Input
          placeholder={t('Site, contact, or group name')}
          value={keyword}
          onChange={setKeyword}
          onEnterPress={handleSearch}
          style={{ width: 300 }}
          aria-label={t('Search applications')}
        />
        <Select
          placeholder={t('Status')}
          value={statusFilter}
          onChange={setStatusFilter}
          style={{ width: 150 }}
          allowClear
          optionList={[
            { value: 'pending', label: t('Pending') },
            { value: 'contacted', label: t('Contacted') },
            { value: 'cooperating', label: t('Cooperating') },
            { value: 'rejected', label: t('Rejected') },
          ]}
        />
        <Input
          placeholder={t('User ID')}
          value={userIdFilter}
          onChange={setUserIdFilter}
          onEnterPress={handleSearch}
          style={{ width: 120 }}
        />
        <Button
          icon={<IconSearch />}
          onClick={handleSearch}
          theme='solid'
          type='primary'
        >
          {t('Search')}
        </Button>
      </div>

      <Spin spinning={loading}>
        {applications.length === 0 && !loading ? (
          <Empty
            description={t('No applications found')}
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
          selectedApp ? `#${selectedApp.id} ${selectedApp.site_name}` : t('View details')
        }
        visible={detailVisible}
        onCancel={() => setDetailVisible(false)}
        width={sheetWidth}
        bodyStyle={{ overflowX: 'hidden' }}
      >
        {selectedApp && (
          <div className='space-y-4'>
            <Card title={t('Processing')}>
              <Form
                initValues={{
                  status: selectedApp.status,
                  admin_note: selectedApp.admin_note || '',
                }}
                onSubmit={handleSave}
                disabled={!hasWritePermission || saving}
              >
                <Form.Select
                  field='status'
                  label={t('Status')}
                  optionList={[
                    { value: 'pending', label: t('Pending') },
                    { value: 'contacted', label: t('Contacted') },
                    { value: 'cooperating', label: t('Cooperating') },
                    { value: 'rejected', label: t('Rejected') },
                  ]}
                  rules={[{ required: true }]}
                />
                <Form.TextArea
                  field='admin_note'
                  label={t('Internal note')}
                  placeholder={t('Internal notes (not shown to users)')}
                  rows={4}
                  maxLength={2000}
                  showClear
                />
                {hasWritePermission && (
                  <Button
                    htmlType='submit'
                    type='primary'
                    theme='solid'
                    loading={saving}
                    block
                  >
                    {t('Save')}
                  </Button>
                )}
              </Form>
            </Card>

            <Card>
              <div className='mb-3'>
                <Text strong>{t('Status')}: </Text>
                <Tag color={statusColors[selectedApp.status] || 'grey'}>
                  {t(statusLabels[selectedApp.status] || selectedApp.status)}
                </Tag>
              </div>
              <div className='mb-3'>
                <Text strong>{t('User')}: </Text>
                <Text>{selectedApp.username} (#{selectedApp.user_id})</Text>
              </div>
              <div className='mb-3'>
                <Text strong>{t('Submitted at')}: </Text>
                <Text>{timestamp2string(selectedApp.created_at)}</Text>
              </div>
              {selectedApp.handled_by > 0 && (
                <>
                  <div className='mb-3'>
                    <Text strong>{t('Handled by')}: </Text>
                    <Text>{selectedApp.handled_by_name || `#${selectedApp.handled_by}`}</Text>
                  </div>
                  <div className='mb-3'>
                    <Text strong>{t('Handled at')}: </Text>
                    <Text>{timestamp2string(selectedApp.handled_at)}</Text>
                  </div>
                </>
              )}
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
                  {t(contactTypeLabels[selectedApp.contact_type] || selectedApp.contact_type)}
                </Paragraph>
              </div>
              <div className='mb-3'>
                <Text type='tertiary'>{t('Contact')}</Text>
                <Paragraph copyable>{selectedApp.contact}</Paragraph>
              </div>
              {selectedApp.remark && (
                <div className='mb-3'>
                  <Text type='tertiary'>{t('Remarks')}</Text>
                  <Paragraph style={{ whiteSpace: 'pre-wrap' }}>{selectedApp.remark}</Paragraph>
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
                        <Tag color='blue'>{t(platformLabels[group.platform] || group.platform)}</Tag>
                        <Tag color='cyan'>×{group.ratio}</Tag>
                      </div>
                      {group.models && group.models.length > 0 && (
                        <div className='mb-2'>
                          <Text type='tertiary' size='small'>
                            {t('Main models')}:
                          </Text>
                          <div className='mt-1 flex flex-wrap gap-1'>
                            {group.models.map((model, idx) => (
                              <Tag key={idx} size='small' style={{ maxWidth: '100%', wordBreak: 'break-word', overflowWrap: 'anywhere' }}>
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
                          <Paragraph size='small' style={{ whiteSpace: 'pre-wrap', marginTop: 4 }}>
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

            <Card>
              <SupplierConversation
                applicationId={selectedApp.id}
                messages={selectedApp.messages}
                canSend={hasWritePermission}
                mode='admin'
                onSent={(message) => {
                  const append = (application) => {
                    if (!application || application.id !== selectedApp.id) return application;
                    const next = [...(application.messages || [])];
                    if (next.some((item) => item.id === message.id)) return application;
                    next.push(message);
                    return {
                      ...application,
                      messages: next,
                      message_count: (application.message_count || 0) + 1,
                    };
                  };
                  setSelectedApp((current) => append(current));
                  setApplications((current) => current.map((item) => append(item)));
                }}
              />
            </Card>

            {!hasWritePermission && (
              <Banner
                type='info'
                description={t('You do not have permission to edit this application.')}
              />
            )}
          </div>
        )}
      </SideSheet>
    </div>
  );
};

export default SupplierApplicationManagementTable;
