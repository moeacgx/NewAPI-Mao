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

import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Empty,
  Popconfirm,
  Space,
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../helpers';
import useGroupTags from '../../../hooks/common/useGroupTags';
import { GROUP_TAGS_UPDATED_EVENT } from '../../../helpers/groupTags';
import { GROUP_DETAILS_UPDATED_EVENT } from '../../../helpers/groupDetails';
import GroupTagLogos from '../../../components/common/group-tags/GroupTagLogos';
import GroupTagEditor from './components/GroupTagEditor';

export default function GroupTagSettings() {
  const { t } = useTranslation();
  const { tags, loading, error, reload } = useGroupTags();
  const [groups, setGroups] = useState([]);
  const [editing, setEditing] = useState(undefined);
  const [groupsLoading, setGroupsLoading] = useState(false);
  const [deleting, setDeleting] = useState(null);
  const [groupsError, setGroupsError] = useState(false);
  const requestVersion = useRef(0);

  const loadGroups = useCallback(async () => {
    const version = ++requestVersion.current;
    try {
      const response = await API.get('/api/group/details', {
        disableDuplicate: true,
      });
      if (version !== requestVersion.current) return false;
      if (!response.data.success) throw new Error(response.data.message);
      setGroups(response.data.data || []);
      setGroupsError(false);
      return true;
    } catch {
      if (version === requestVersion.current) setGroupsError(true);
      return false;
    }
  }, []);

  const openEditor = async (tag) => {
    setGroupsLoading(true);
    try {
      if (await loadGroups()) setEditing(tag);
    } catch (error) {
      showError(error.message || t('加载分组失败'));
    } finally {
      setGroupsLoading(false);
    }
  };

  useEffect(() => {
    loadGroups();
    window.addEventListener(GROUP_DETAILS_UPDATED_EVENT, loadGroups);
    return () => {
      requestVersion.current += 1;
      window.removeEventListener(GROUP_DETAILS_UPDATED_EVENT, loadGroups);
    };
  }, [loadGroups]);

  const notifySaved = () => {
    setEditing(undefined);
    window.dispatchEvent(new Event(GROUP_TAGS_UPDATED_EVENT));
  };

  const deleteTag = async (tag) => {
    setDeleting(tag.id);
    try {
      const response = await API.delete(`/api/group/tags/${tag.id}`);
      if (!response.data.success) throw new Error(response.data.message);
      showSuccess(t('Group tag deleted'));
      window.dispatchEvent(new Event(GROUP_TAGS_UPDATED_EVENT));
    } catch (error) {
      showError(error.message || t('操作失败'));
    } finally {
      setDeleting(null);
    }
  };
  const groupNames = new Map(
    groups.map((group) => [group.id, group.name || group.code]),
  );

  return (
    <Card className='mt-4' title={t('Group tags')}>
      <div className='group-tag-actions'>
        <Typography.Text type='tertiary'>
          {t(
            'Organize groups into tags, then filter by tag when creating tokens.',
          )}
        </Typography.Text>
        <Button
          theme='solid'
          disabled={deleting !== null}
          onClick={() => openEditor(null)}
          loading={groupsLoading}
        >
          {t('New group tag')}
        </Button>
      </div>
      {error && (
        <Banner
          type='danger'
          description={t('Failed to load group tags')}
          closeIcon={null}
          extra={<Button onClick={reload}>{t('重试')}</Button>}
        />
      )}
      {groupsError && (
        <Banner
          type='danger'
          description={t('加载分组失败')}
          closeIcon={null}
          extra={<Button onClick={loadGroups}>{t('重试')}</Button>}
        />
      )}
      <Spin spinning={loading}>
        {!error && tags.length === 0 && (
          <Empty description={t('No group tags yet')} />
        )}
        <div className='grid grid-cols-1 gap-3 lg:grid-cols-2'>
          {tags.map((tag) => (
            <Card
              key={tag.id}
              className='group-tag-admin-card'
              title={
                <div className='group-tag-admin-heading'>
                  <GroupTagLogos icons={tag.icons} />
                  <span className='group-tag-name'>{tag.name}</span>
                </div>
              }
            >
              <p className='group-tag-description'>{tag.description}</p>
              <div className='my-3 flex flex-wrap gap-2'>
                {tag.group_ids.map(
                  (id) =>
                    groupNames.has(id) && (
                      <Tag key={id}>{groupNames.get(id)}</Tag>
                    ),
                )}
                {tag.group_ids.length === 0 && (
                  <Typography.Text type='tertiary'>
                    {t('No bound groups')}
                  </Typography.Text>
                )}
              </div>
              <Space className='group-tag-admin-footer' wrap>
                <Button
                  disabled={groupsLoading || deleting !== null}
                  onClick={() => openEditor(tag)}
                >
                  {t('编辑')}
                </Button>
                <Popconfirm
                  title={t(
                    'Delete this tag? Groups and tokens will be preserved.',
                  )}
                  onConfirm={() => deleteTag(tag)}
                >
                  <Button
                    type='danger'
                    loading={deleting === tag.id}
                    disabled={deleting !== null}
                  >
                    {t('删除')}
                  </Button>
                </Popconfirm>
              </Space>
            </Card>
          ))}
        </div>
      </Spin>
      {editing !== undefined && (
        <GroupTagEditor
          tag={editing}
          groups={groups}
          onSaved={notifySaved}
          onCancel={() => setEditing(undefined)}
        />
      )}
    </Card>
  );
}
