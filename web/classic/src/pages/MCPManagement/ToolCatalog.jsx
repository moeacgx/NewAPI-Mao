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

import React, { useId, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { categoryLabel } from './categories';

/**
 * 服务端工具目录契约；分类只是展示筛选，不作为授权通配符。
 * @typedef {{name: string, description: string, category: string, write?: boolean, min_role?: number, inputSchema?: Record<string, unknown>}} McpTool
 */

/** @param {{tools: McpTool[], selected?: string[], onChange?: (names: string[]) => void, disabled?: boolean, permissions?: boolean}} props */
export function ToolCatalog(props) {
  const { t } = useTranslation();
  const id = useId();
  const [search, setSearch] = useState('');
  const [category, setCategory] = useState('');
  const [access, setAccess] = useState('all');
  const selected = new Set(props.selected ?? []);
  const categories = useMemo(
    () => [...new Set(props.tools.map((tool) => tool.category))].sort(),
    [props.tools],
  );
  const effectiveCategory = categories.includes(category) ? category : '';
  const filtered = useMemo(() => {
    const needle = search.trim().toLowerCase();
    return props.tools.filter(
      (tool) =>
        (!effectiveCategory || tool.category === effectiveCategory) &&
        (access === 'all' || tool.write === (access === 'write')) &&
        `${tool.name} ${tool.description} ${tool.category} ${categoryLabel(tool.category, t)}`
          .toLowerCase()
          .includes(needle),
    );
  }, [props.tools, search, effectiveCategory, access, t]);
  const groups = useMemo(() => {
    /** @type {Map<string, McpTool[]>} */
    const result = new Map();
    for (const tool of filtered) {
      const items = result.get(tool.category) ?? [];
      items.push(tool);
      result.set(tool.category, items);
    }
    return [...result].sort(([a], [b]) => a.localeCompare(b));
  }, [filtered]);
  const reads = filtered.filter((tool) => tool.write === false);
  const writes = filtered.filter((tool) => tool.write);
  const selectedWrites = props.tools.filter(
    (tool) => tool.write && selected.has(tool.name),
  ).length;

  /** @param {McpTool[]} tools */
  function selectTools(tools) {
    props.onChange?.([
      ...new Set([...selected, ...tools.map((tool) => tool.name)]),
    ]);
  }

  return (
    <div className='mcp-catalog mcp-stack'>
      <div className='mcp-filters'>
        <div>
          <label htmlFor={`${id}-search`}>{t('Search tools')}</label>
          <input
            id={`${id}-search`}
            value={search}
            placeholder={t('Search by tool name, description or category')}
            disabled={props.disabled}
            onChange={(event) => setSearch(event.target.value)}
          />
        </div>
        <div>
          <label htmlFor={`${id}-category`}>{t('Tool category')}</label>
          <select
            id={`${id}-category`}
            value={effectiveCategory}
            disabled={props.disabled}
            onChange={(event) => setCategory(event.target.value)}
          >
            <option value=''>{t('All categories')}</option>
            {categories.map((value) => (
              <option key={value} value={value}>
                {categoryLabel(value, t)}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label htmlFor={`${id}-access`}>{t('Access')}</label>
          <select
            id={`${id}-access`}
            value={access}
            disabled={props.disabled}
            onChange={(event) => setAccess(event.target.value)}
          >
            <option value='all'>{t('All')}</option>
            <option value='read'>{t('Read-only tool')}</option>
            <option value='write'>{t('Write tool')}</option>
          </select>
        </div>
      </div>
      <p role='status' className='text-sm'>
        {t('{{total}} tools · {{read}} read-only · {{write}} write', {
          total: filtered.length,
          read: reads.length,
          write: writes.length,
        })}
      </p>
      {props.onChange && (
        <>
          <p className='text-sm mcp-muted'>
            {t(
              'Write tools can change data or trigger external actions. No tools are authorized by default.',
            )}
          </p>
          <div className='mcp-bulk-actions'>
            <button
              type='button'
              className='mcp-bulk-button'
              disabled={props.disabled || !reads.length}
              onClick={() => selectTools(reads)}
            >
              {t('Select visible read-only tools')}
            </button>
            <button
              type='button'
              className='mcp-bulk-button mcp-write-action'
              disabled={props.disabled || !writes.length}
              onClick={() => selectTools(writes)}
            >
              {t('Explicitly select visible write tools')}
            </button>
            <button
              type='button'
              className='mcp-bulk-button'
              disabled={
                props.disabled || !effectiveCategory || !filtered.length
              }
              onClick={() => selectTools(filtered)}
            >
              {t('Select all visible tools in this category (including write)')}
            </button>
            <button
              type='button'
              className='mcp-bulk-button'
              disabled={
                props.disabled ||
                !filtered.some((tool) => selected.has(tool.name))
              }
              onClick={() => {
                const visible = new Set(filtered.map((tool) => tool.name));
                props.onChange?.(
                  [...selected].filter((name) => !visible.has(name)),
                );
              }}
            >
              {t('Clear visible selection')}
            </button>
          </div>
          <p className='text-sm mcp-muted'>
            {t(
              'Bulk actions apply only to filtered tools. Hidden selections are kept. Choose a category to select all its visible tools.',
            )}
          </p>
          <p role='status' className='text-sm font-medium'>
            {t('{{selected}} selected in all categories · {{write}} write', {
              selected: selected.size,
              write: selectedWrites,
            })}
          </p>
        </>
      )}
      {!filtered.length && (
        <p className='rounded-lg border mcp-muted-surface p-4'>
          {t('No matching tools')}
        </p>
      )}
      <div className='mcp-catalog-list'>
        {groups.map(([name, tools]) => {
          const write = tools.filter((tool) => tool.write).length;
          return (
            <section key={name} aria-label={name} className='mcp-tool-group'>
              <div className='mcp-group-heading'>
                <h3>{categoryLabel(name, t)}</h3>
                <p className='text-sm mcp-muted'>
                  {t('{{total}} tools · {{read}} read-only · {{write}} write', {
                    total: tools.length,
                    read: tools.filter((tool) => tool.write === false).length,
                    write,
                  })}
                </p>
              </div>
              {tools.map((tool) =>
                props.onChange ? (
                  <div key={tool.name} className='mcp-checkbox'>
                    <input
                      type='checkbox'
                      id={`${id}-${tool.name}`}
                      checked={selected.has(tool.name)}
                      disabled={props.disabled}
                      onChange={(event) => {
                        if (event.target.checked) selectTools([tool]);
                        else
                          props.onChange?.(
                            [...selected].filter((name) => name !== tool.name),
                          );
                      }}
                    />
                    <div className='min-w-0'>
                      <label
                        htmlFor={`${id}-${tool.name}`}
                        className='mcp-tool-name'
                      >
                        {tool.name}
                        <span
                          className={
                            tool.write === undefined
                              ? 'mcp-unknown-badge'
                              : tool.write
                                ? 'mcp-write-badge'
                                : 'mcp-read-badge'
                          }
                        >
                          {tool.write === undefined
                            ? t('Permission details unavailable')
                            : tool.write
                              ? t('Write tool')
                              : t('Read-only tool')}
                        </span>
                      </label>
                      <p className='text-sm mcp-muted break-words'>
                        {tool.description}
                      </p>
                    </div>
                  </div>
                ) : (
                  <article key={tool.name} className='mcp-tool'>
                    <div className='mcp-toolbar'>
                      <strong className='mcp-break'>{tool.name}</strong>
                      <span
                        className={
                          tool.write === undefined
                            ? 'mcp-unknown-badge'
                            : tool.write
                              ? 'mcp-write-badge'
                              : 'mcp-read-badge'
                        }
                      >
                        {tool.write === undefined
                          ? t('Permission details unavailable')
                          : tool.write
                            ? t('Write tool')
                            : t('Read-only tool')}
                      </span>
                    </div>
                    <p>{tool.description}</p>
                    {!props.permissions && tool.min_role !== undefined && (
                      <p>
                        {t('Minimum role')}: {tool.min_role}
                      </p>
                    )}
                    {!props.permissions && tool.inputSchema && (
                      <details>
                        <summary>{t('Input schema')}</summary>
                        <pre className='mcp-code'>
                          {JSON.stringify(tool.inputSchema, null, 2)}
                        </pre>
                      </details>
                    )}
                  </article>
                ),
              )}
            </section>
          );
        })}
      </div>
    </div>
  );
}
