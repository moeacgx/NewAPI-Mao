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
function updateSingleMeta(selector, attribute, value, content) {
  const existing = Array.from(document.head.querySelectorAll(selector));
  const meta = existing.shift() || document.createElement('meta');
  existing.forEach((item) => item.remove());
  meta.setAttribute(attribute, value);
  meta.setAttribute('content', content);
  if (!meta.isConnected) document.head.appendChild(meta);
}

function removeMeta(selector) {
  document.head.querySelectorAll(selector).forEach((meta) => meta.remove());
}

export function applySiteMetadataFromStatus(status) {
  if (typeof document === 'undefined' || !status) return;

  if (typeof status.system_name === 'string' && status.system_name !== '') {
    document.title = status.system_name;
    const titles = Array.from(document.head.querySelectorAll('title'));
    const title = titles.shift() || document.createElement('title');
    titles.forEach((item) => item.remove());
    title.textContent = status.system_name;
    if (!title.isConnected) document.head.appendChild(title);

    updateSingleMeta('meta[name="title"]', 'name', 'title', status.system_name);
    updateSingleMeta(
      'meta[property="og:title"]',
      'property',
      'og:title',
      status.system_name,
    );
    updateSingleMeta(
      'meta[property="og:site_name"]',
      'property',
      'og:site_name',
      status.system_name,
    );
  }

  if (Object.prototype.hasOwnProperty.call(status, 'system_description')) {
    if (
      typeof status.system_description === 'string' &&
      status.system_description !== ''
    ) {
      updateSingleMeta(
        'meta[name="description"]',
        'name',
        'description',
        status.system_description,
      );
      updateSingleMeta(
        'meta[property="og:description"]',
        'property',
        'og:description',
        status.system_description,
      );
    } else {
      removeMeta('meta[name="description"], meta[property="og:description"]');
    }
  }
}
