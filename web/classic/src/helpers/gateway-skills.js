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

const HEADER_NAME = 'x-new-api-gateway-skills';

const decodeBase64UrlUtf8 = (value) => {
  if (
    typeof value !== 'string' ||
    !value ||
    value.length > 6 * 1024 ||
    value.length % 4 === 1 ||
    !/^[A-Za-z0-9_-]+$/.test(value)
  ) {
    return null;
  }

  try {
    const base64 = value.replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64.padEnd(Math.ceil(base64.length / 4) * 4, '=');
    const binary = atob(padded);
    const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
    return new TextDecoder('utf-8', { fatal: true }).decode(bytes);
  } catch {
    return null;
  }
};

const headerValue = (headers) => {
  if (!headers) return null;
  if (Array.isArray(headers[HEADER_NAME])) return headers[HEADER_NAME][0];
  if (typeof headers.get === 'function') return headers.get(HEADER_NAME);
  return headers[HEADER_NAME] || headers['X-New-API-Gateway-Skills'] || null;
};

const isSafeSkill = (skill) =>
  skill &&
  Number.isSafeInteger(skill.template_id) &&
  skill.template_id > 0 &&
  typeof skill.skill_id === 'string' &&
  skill.skill_id.length > 0 &&
  skill.skill_id.length <= 256 &&
  Number.isSafeInteger(skill.version) &&
  skill.version > 0 &&
  typeof skill.title === 'string' &&
  skill.title.trim().length > 0 &&
  skill.title.length <= 512;

export const parseGatewaySkillsHeader = (headers) => {
  const decoded = decodeBase64UrlUtf8(headerValue(headers));
  if (!decoded) return null;

  try {
    const parsed = JSON.parse(decoded);
    if (
      !parsed ||
      parsed.version !== 1 ||
      !Number.isSafeInteger(parsed.total) ||
      parsed.total < 0 ||
      parsed.total > 1000000 ||
      !Array.isArray(parsed.skills) ||
      parsed.skills.length > 256 ||
      parsed.skills.length === 0 ||
      parsed.total < parsed.skills.length ||
      typeof parsed.truncated !== 'boolean' ||
      (!parsed.truncated && parsed.total !== parsed.skills.length) ||
      parsed.skills.some((skill) => !isSafeSkill(skill))
    ) {
      return null;
    }

    return {
      version: 1,
      total: parsed.total,
      truncated: parsed.truncated,
      skills: parsed.skills.map((skill) => ({
        template_id: skill.template_id,
        skill_id: skill.skill_id,
        version: skill.version,
        title: skill.title,
      })),
    };
  } catch {
    return null;
  }
};
