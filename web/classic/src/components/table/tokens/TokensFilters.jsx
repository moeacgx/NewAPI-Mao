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

import React, { useRef, useEffect, useState } from 'react';
import { Form, Button, Popover } from '@douyinfe/semi-ui';
import { CirclePlus, X } from 'lucide-react';
export default function TokensFilters({
  formInitValues,
  setFormApi,
  searchTokens,
  t,
}) {
  const formApiRef = useRef(null),
    searchRef = useRef(searchTokens),
    debounceRef = useRef(null);
  const [statusOpen, setStatusOpen] = useState(false);
  searchRef.current = searchTokens;
  useEffect(() => () => clearTimeout(debounceRef.current), []);
  return (
    <Form
      initValues={formInitValues}
      getFormApi={(api) => {
        formApiRef.current = api;
        setFormApi(api);
      }}
      onSubmit={() => {
        clearTimeout(debounceRef.current);
        searchRef.current(1);
      }}
      onValueChange={() => {
        clearTimeout(debounceRef.current);
        debounceRef.current = setTimeout(() => searchRef.current(1), 300);
      }}
      autoComplete='off'
      layout='horizontal'
      className='tokens-filters-form'
    >
      {({ values }) => (
        <div className='tokens-filters'>
          <div className='tokens-filter-input'>
            <Form.Input
              field='searchKeyword'
              placeholder={t('Filter by name...')}
              aria-label={t('Filter by name...')}
              pure
              size='small'
            />
          </div>
          <div className='tokens-filter-input'>
            <Form.Input
              field='searchToken'
              placeholder={t('Filter by API key...')}
              aria-label={t('Filter by API key...')}
              pure
              size='small'
            />
          </div>
          <Popover
            trigger='click'
            position='bottomLeft'
            visible={statusOpen}
            onVisibleChange={setStatusOpen}
            content={
              <div
                className='tokens-default-overlay tokens-status-options'
                role='radiogroup'
                aria-label={t('状态')}
              >
                {[
                  [0, '全部'],
                  [1, '已启用'],
                  [2, '已禁用'],
                  [3, '已过期'],
                  [4, '已耗尽'],
                ].map(([value, label]) => (
                  <label key={value}>
                    <input
                      type='radio'
                      name='token-status-filter'
                      checked={Number(values.searchStatus || 0) === value}
                      onChange={() => {
                        formApiRef.current.setValue('searchStatus', value);
                        setStatusOpen(false);
                      }}
                    />
                    {t(label)}
                  </label>
                ))}
              </div>
            }
          >
            <Button
              className='tokens-status-filter'
              data-active={!!values.searchStatus}
              theme='outline'
              type='tertiary'
              size='small'
              icon={<CirclePlus size={16} />}
            >
              <span>{t('状态')}</span>
              {!!values.searchStatus && (
                <span className='tokens-filter-value'>
                  {t(
                    { 1: '已启用', 2: '已禁用', 3: '已过期', 4: '已耗尽' }[
                      values.searchStatus
                    ],
                  )}
                </span>
              )}
            </Button>
          </Popover>
          {values.searchKeyword || values.searchToken || values.searchStatus ? (
            <Button
              theme='borderless'
              type='tertiary'
              size='small'
              icon={<X size={14} />}
              onClick={() => {
                formApiRef.current.reset();
                formApiRef.current.setValue('searchStatus', 0);
              }}
            >
              {t('重置')}
            </Button>
          ) : null}
        </div>
      )}
    </Form>
  );
}
