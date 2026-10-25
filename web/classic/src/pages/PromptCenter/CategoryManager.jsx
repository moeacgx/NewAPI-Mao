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

import React, { useEffect, useRef, useState } from 'react';
import { Button, Modal } from '@douyinfe/semi-ui';
import { Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { showSuccess } from '../../helpers/utils';
import { deletePromptCategory, promptError, promptRequest } from './api';

export default function CategoryManager(props) {
  const { t } = useTranslation();
  const [categories, setCategories] = useState(props.categories || []);
  const [names, setNames] = useState({});
  const [name, setName] = useState('');
  const [busy, setBusy] = useState(false);
  const busyRef = useRef(false);
  const [error, setError] = useState('');
  const [deleting, setDeleting] = useState(null);
  const [deleteError, setDeleteError] = useState('');
  const confirmRef = useRef(null);
  const returnFocusRef = useRef(null);
  useEffect(() => {
    setCategories(props.categories || []);
  }, [props.categories]);
  useEffect(() => {
    if (deleting) {
      confirmRef.current?.focus();
      return;
    }
    returnFocusRef.current?.focus();
    returnFocusRef.current = null;
  }, [deleting]);
  const closeDelete = () => {
    if (busyRef.current) return;
    setDeleting(null);
    setDeleteError('');
  };
  const remove = async () => {
    if (busyRef.current) return;
    busyRef.current = true;
    setBusy(true);
    setDeleteError('');
    try {
      const result = await deletePromptCategory(deleting.id, deleting.version);
      const next = categories.filter((item) => item.id !== deleting.id);
      setCategories(next);
      setNames((current) => {
        const rest = { ...current };
        delete rest[deleting.id];
        return rest;
      });
      setDeleting(null);
      showSuccess(
        t(
          'Deleted category {{name}}. {{count}} skills moved to Uncategorized',
          {
            name: deleting.name,
            count: result?.moved_templates ?? 0,
          },
        ),
      );
      props.onChange(next);
    } catch (reason) {
      setDeleteError(promptError(reason, t));
    } finally {
      busyRef.current = false;
      setBusy(false);
    }
  };
  const save = async (category) => {
    const value = (
      category ? (names[category.id] ?? category.name) : name
    ).trim();
    if (!value || Array.from(value).length > 40) {
      setError(t('Category names must contain 1 to 40 characters'));
      return;
    }
    setBusy(true);
    setError('');
    try {
      const item = await promptRequest(
        category ? 'put' : 'post',
        `/api/prompt-center/admin/categories/${category?.id || ''}`,
        { name: value, ...(category ? { version: category.version } : {}) },
      );
      const next = category
        ? categories.map((current) => (current.id === item.id ? item : current))
        : [...categories, item];
      setCategories(next);
      if (!category) setName('');
      props.onChange(next);
    } catch (reason) {
      setError(promptError(reason, t));
    } finally {
      setBusy(false);
    }
  };
  // 删除确认作为同一弹窗的一步展示：Semi 弹窗标题 id 固定，叠加弹窗的可访问名称会变成外层标题
  return (
    <Modal
      visible
      title={deleting ? t('Delete category?') : t('Manage categories')}
      footer={
        deleting ? (
          <div className='prompt-center-row'>
            <Button disabled={busy} onClick={closeDelete}>
              {t('Cancel')}
            </Button>
            <Button
              type='danger'
              theme='solid'
              loading={busy}
              disabled={busy}
              onClick={remove}
            >
              {t('Delete')}
            </Button>
          </div>
        ) : null
      }
      onCancel={() => {
        if (deleting) closeDelete();
        else if (!busy) props.onCancel();
      }}
      closable={!busy}
      closeOnEsc={!busy}
      maskClosable={false}
      width={560}
      style={{ maxWidth: 'calc(100vw - 24px)' }}
    >
      {deleting && (
        <div ref={confirmRef} tabIndex={-1} className='prompt-category-delete'>
          <p>
            {t(
              'Delete category “{{name}}”? Its skills will become Uncategorized. This action cannot be undone.',
              { name: deleting.name },
            )}
          </p>
          {deleteError && (
            <p role='alert' className='prompt-template-delete-error'>
              {deleteError}
            </p>
          )}
        </div>
      )}
      {/* 确认期间仅隐藏列表，取消后焦点可回到原删除按钮 */}
      <div className='prompt-center-form' hidden={Boolean(deleting)}>
        {error && (
          <p className='prompt-center-error' role='alert'>
            {error}
          </p>
        )}
        <p className='prompt-center-muted'>
          {t('Categories organize skills and do not grant group permissions.')}
        </p>
        <form
          className='prompt-category-row'
          onSubmit={(event) => {
            event.preventDefault();
            save();
          }}
        >
          <input
            aria-label={t('New category name')}
            placeholder={t('New category name')}
            maxLength={40}
            required
            value={name}
            disabled={busy}
            onChange={(event) => setName(event.target.value)}
          />
          <Button htmlType='submit' disabled={busy}>
            {t('Add category')}
          </Button>
        </form>
        <div className='prompt-category-list'>
          {categories.map((category) => (
            <form
              key={category.id}
              className='prompt-category-row'
              onSubmit={(event) => {
                event.preventDefault();
                save(category);
              }}
            >
              <input
                aria-label={t('Rename {{name}}', { name: category.name })}
                required
                maxLength={40}
                disabled={busy}
                value={names[category.id] ?? category.name}
                onChange={(event) =>
                  setNames((current) => ({
                    ...current,
                    [category.id]: event.target.value,
                  }))
                }
              />
              <Button
                htmlType='submit'
                disabled={busy}
                aria-label={t('Save {{name}}', { name: category.name })}
              >
                {t('Save')}
              </Button>
              <Button
                className='prompt-center-delete-button'
                type='danger'
                theme='borderless'
                icon={<Trash2 size={14} aria-hidden='true' />}
                disabled={busy}
                aria-label={t('Delete category {{name}}', {
                  name: category.name,
                })}
                onClick={(event) => {
                  returnFocusRef.current = event.currentTarget;
                  setDeleteError('');
                  setDeleting(category);
                }}
              >
                {t('Delete')}
              </Button>
            </form>
          ))}
        </div>
      </div>
    </Modal>
  );
}
