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

// Copyright (C) 2023-2026 QuantumNous；移植自 Default upstream/main@789c97019，保留上游归属。
import React from 'react';
import {
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
} from 'lucide-react';
function getPageNumbers(currentPage, totalPages) {
  const maxVisiblePages = 4;
  const rangeWithDots = [];

  if (totalPages <= maxVisiblePages) {
    for (let i = 1; i <= totalPages; i++) {
      rangeWithDots.push(i);
    }
  } else {
    rangeWithDots.push(1);

    if (currentPage <= 2) {
      rangeWithDots.push(2);
      rangeWithDots.push('...', totalPages);
    } else if (currentPage >= totalPages - 1) {
      rangeWithDots.push('...');
      rangeWithDots.push(totalPages - 1, totalPages);
    } else {
      rangeWithDots.push('...');
      rangeWithDots.push(currentPage);
      rangeWithDots.push('...', totalPages);
    }
  }

  return rangeWithDots;
}
export default function TokensPagination(props) {
  const pages = Math.max(1, Math.ceil(props.tokenCount / props.pageSize));
  return (
    <div className='tokens-pagination'>
      <span className='tokens-pagination-label'>{props.t('Total:')}</span>
      <span>{props.tokenCount.toLocaleString()}</span>
      <select
        aria-label={props.t('Rows per page')}
        value={props.pageSize}
        onChange={(e) => props.handlePageSizeChange(Number(e.target.value))}
      >
        {[10, 20, 30, 40, 50, 100].map((size) => (
          <option key={size} value={size}>
            {size}
          </option>
        ))}
      </select>
      <button
        aria-label={props.t('Go to first page')}
        disabled={props.activePage <= 1}
        onClick={() => props.handlePageChange(1)}
      >
        <ChevronsLeft size={16} />
      </button>
      <button
        aria-label={props.t('Go to previous page')}
        disabled={props.activePage <= 1}
        onClick={() => props.handlePageChange(props.activePage - 1)}
      >
        <ChevronLeft size={16} />
      </button>
      {getPageNumbers(props.activePage, pages).map((page, i) =>
        page === '...' ? (
          <span key={'gap' + i}>...</span>
        ) : (
          <button
            key={page}
            aria-current={page === props.activePage ? 'page' : undefined}
            aria-label={props.t('Go to page {{page}}', { page })}
            onClick={() => props.handlePageChange(page)}
          >
            {page}
          </button>
        ),
      )}
      <button
        aria-label={props.t('Go to next page')}
        disabled={props.activePage >= pages}
        onClick={() => props.handlePageChange(props.activePage + 1)}
      >
        <ChevronRight size={16} />
      </button>
      <button
        aria-label={props.t('Go to last page')}
        disabled={props.activePage >= pages}
        onClick={() => props.handlePageChange(pages)}
      >
        <ChevronsRight size={16} />
      </button>
    </div>
  );
}
