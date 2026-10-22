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
import { Button, Modal } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { promptError, promptRequest } from './api';

export default function CategoryManager(props) {
  const { t } = useTranslation();
  const [categories, setCategories] = useState(props.categories || []);
  const [names, setNames] = useState({});
  const [name, setName] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  useEffect(() => {
    setCategories(props.categories || []);
  }, [props.categories]);
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
  return (
    <Modal
      visible
      title={t('Manage categories')}
      footer={null}
      onCancel={() => {
        if (!busy) props.onCancel();
      }}
      closable={!busy}
      closeOnEsc={!busy}
      maskClosable={false}
      width={560}
      style={{ maxWidth: 'calc(100vw - 24px)' }}
    >
      <div className='prompt-center-form'>
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
            </form>
          ))}
        </div>
      </div>
    </Modal>
  );
}
