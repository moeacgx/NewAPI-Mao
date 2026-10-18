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

export const INSUFFICIENT_BALANCE_KEYWORDS = ['预扣费额度失败', '余额不足'];
export const INSUFFICIENT_BALANCE_PREFIX_DEDUP_SECONDS = 300;

const parsePrefixDedupSeconds = (value) => {
  const seconds = typeof value === 'number' ? value : Number(value);
  if (!Number.isFinite(seconds) || seconds <= 0) return 0;
  return Math.trunc(seconds);
};

// 只发送渠道禁用事件支持的筛选字段，并按后端匹配语义清理重复或空关键词。
export const normalizeNotificationFilterConfig = (config) => {
  if (!config) return undefined;

  const statusCodes = String(config.status_codes || '').trim();
  const errorKeywords = [];
  const seenKeywords = new Set();
  for (const value of Array.isArray(config.error_keywords)
    ? config.error_keywords
    : []) {
    const keyword = String(value || '').trim();
    const identity = keyword.toLowerCase();
    if (!keyword || seenKeywords.has(identity)) continue;
    seenKeywords.add(identity);
    errorKeywords.push(keyword);
  }
  const prefixDedupSeconds = parsePrefixDedupSeconds(
    config.prefix_dedup_seconds,
  );

  if (!statusCodes && errorKeywords.length === 0 && prefixDedupSeconds <= 0) {
    return undefined;
  }

  return {
    ...(statusCodes ? { status_codes: statusCodes } : {}),
    ...(errorKeywords.length > 0 ? { error_keywords: errorKeywords } : {}),
    ...(prefixDedupSeconds > 0
      ? { prefix_dedup_seconds: prefixDedupSeconds }
      : {}),
  };
};

export const applyInsufficientBalanceDedupPreset = (config) => {
  const keywords = [...INSUFFICIENT_BALANCE_KEYWORDS];
  const seen = new Set(keywords.map((keyword) => keyword.toLowerCase()));
  for (const value of Array.isArray(config?.error_keywords)
    ? config.error_keywords
    : []) {
    const keyword = String(value || '').trim();
    const identity = keyword.toLowerCase();
    if (!keyword || seen.has(identity)) continue;
    seen.add(identity);
    keywords.push(keyword);
  }
  const statusCodes = String(config?.status_codes || '').trim();
  return {
    ...(statusCodes ? { status_codes: statusCodes } : {}),
    error_keywords: keywords,
    prefix_dedup_seconds: INSUFFICIENT_BALANCE_PREFIX_DEDUP_SECONDS,
  };
};
