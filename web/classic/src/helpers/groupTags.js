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

// 标签只筛选候选项，调用者始终保留完整分组状态和已有选择。
export function filterGroupsByTag(groups, tags, selectedTag) {
  if (selectedTag === 'all') return groups;
  if (selectedTag === 'untagged') {
    const taggedIDs = new Set(tags.flatMap((tag) => tag.group_ids));
    return groups.filter((group) => !taggedIDs.has(group.id));
  }
  const tag = tags.find((item) => String(item.id) === selectedTag);
  if (!tag) return groups;
  const ids = new Set(tag.group_ids);
  return groups.filter((group) => ids.has(group.id));
}

export const GROUP_TAGS_UPDATED_EVENT = 'group-tags-updated';
