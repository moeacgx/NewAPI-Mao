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

import React, { useRef, useEffect } from 'react';
import { Form, Button } from '@douyinfe/semi-ui';

const TokensFilters = ({
  formInitValues,
  setFormApi,
  searchTokens,
  loading,
  searching,
  t,
}) => {
  // Handle form reset and immediate search
  const formApiRef = useRef(null);
  const searchRef = useRef(searchTokens);
  searchRef.current = searchTokens;
  const debounceRef = useRef(null);
  useEffect(() => () => clearTimeout(debounceRef.current), []);

  const handleReset = () => {
    if (!formApiRef.current) return;
    formApiRef.current.reset();
    setTimeout(() => {
      searchTokens();
    }, 100);
  };

  return (
    <Form
      initValues={formInitValues}
      getFormApi={(api) => {
        setFormApi(api);
        formApiRef.current = api;
      }}
      onSubmit={() => {
        clearTimeout(debounceRef.current);
        searchRef.current(1);
      }}
      onValueChange={() => {
        clearTimeout(debounceRef.current);
        debounceRef.current = setTimeout(() => searchRef.current(1), 300);
      }}
      allowEmpty={true}
      autoComplete='off'
      layout='horizontal'
      trigger='change'
      stopValidateWithError={false}
      className='tokens-filters-form'
    >
      <div className='tokens-filters'>
        <div className='tokens-filter-input'>
          <Form.Input
            field='searchKeyword'
            placeholder={t('Filter by name...')}
            aria-label={t('Filter by name...')}
            showClear
            pure
            size='small'
          />
        </div>

        <div className='tokens-filter-input'>
          <Form.Input
            field='searchToken'
            placeholder={t('Filter by API key...')}
            aria-label={t('Filter by API key...')}
            showClear
            pure
            size='small'
          />
        </div>

        <Form.Select
          field='searchStatus'
          pure
          size='small'
          aria-label={t('状态')}
          optionList={[
            [0, '状态'],
            [1, '已启用'],
            [2, '已禁用'],
            [3, '已过期'],
            [4, '已耗尽'],
          ].map(([value, label]) => ({ value, label: t(label) }))}
        />
        <div className='flex gap-2'>
          <Button
            type='tertiary'
            htmlType='submit'
            loading={loading || searching}
            className='flex-1 md:flex-initial md:w-auto'
            size='small'
          >
            {t('查询')}
          </Button>

          <Button
            type='tertiary'
            onClick={handleReset}
            className='flex-1 md:flex-initial md:w-auto'
            size='small'
          >
            {t('重置')}
          </Button>
        </div>
      </div>
    </Form>
  );
};

export default TokensFilters;
