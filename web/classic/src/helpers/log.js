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

export function getLogOther(otherStr) {
  if (otherStr === undefined || otherStr === null || otherStr === '') {
    return {};
  }
  if (typeof otherStr === 'object') {
    return otherStr;
  }
  try {
    return JSON.parse(otherStr);
  } catch (e) {
    console.error(`Failed to parse record.other: "${otherStr}".`, e);
    return null;
  }
}

export function isWebSocketLog(other) {
  return other?.ws === true;
}

/**
 * 从日志 other 解析技能使用情况，按以下优先级至多返回一种：
 * 私有发现轮次 { kind: 'discovery', round, readSkills } →
 * 已使用技能 { kind: 'used', count, alwaysLoaded, onDemand }（始终装载 + 最终生成按需读取，按模板与技能去重）→
 * 最终生成未读取技能 { kind: 'unused', catalogueCount }。
 * 字段缺失或类型不符时视为无数据并返回 null；名称仅用于展示。
 */
export function getLogSkillUsage(other) {
  const gatewaySkills = other?.gateway_skills;
  const stage = gatewaySkills?.stage;
  const files = Array.isArray(gatewaySkills?.files) ? gatewaySkills.files : [];
  const readSkills = [];
  const seenSkillKeys = new Set();
  for (const file of files) {
    const templateId = file?.template_id;
    const skillId = typeof file?.skill_id === 'string' ? file.skill_id : '';
    if (typeof templateId !== 'number' || !skillId) continue;
    const skillKey = `${templateId}/${skillId}`;
    if (seenSkillKeys.has(skillKey)) continue;
    seenSkillKeys.add(skillKey);
    const title = typeof file.title === 'string' ? file.title.trim() : '';
    readSkills.push(title || skillId);
  }

  if (stage === 'discovery') {
    const round = gatewaySkills.round;
    if (typeof round !== 'number' || round < 1) return null;
    return { kind: 'discovery', round, readSkills };
  }

  const templates = Array.isArray(other?.prompt_templates)
    ? other.prompt_templates
    : [];
  const alwaysLoaded = [];
  const seenTemplateIds = new Set();
  for (const template of templates) {
    const templateId = template?.id;
    if (typeof templateId !== 'number' || seenTemplateIds.has(templateId)) {
      continue;
    }
    seenTemplateIds.add(templateId);
    const title =
      typeof template.title === 'string' ? template.title.trim() : '';
    alwaysLoaded.push(title || `#${templateId}`);
  }

  const onDemand = stage === 'generation' ? readSkills : [];
  const count = alwaysLoaded.length + onDemand.length;
  if (count > 0) return { kind: 'used', count, alwaysLoaded, onDemand };

  if (stage !== 'generation') return null;
  const catalogueCount = gatewaySkills.catalogue_count;
  if (typeof catalogueCount !== 'number' || catalogueCount < 0) return null;
  return { kind: 'unused', catalogueCount };
}

/** 技能使用情况的展示文案 { label, lines }，分组角标提示与展开详情共用。 */
export function describeLogSkillUsage(usage, t) {
  if (usage.kind === 'discovery') {
    const lines = [t('私有技能发现轮次，单独计费')];
    if (usage.readSkills.length > 0) {
      lines.push(`${t('已读技能')}：${usage.readSkills.join(', ')}`);
    }
    return {
      label: t('技能发现 · 第 {{round}} 轮', { round: usage.round }),
      lines,
    };
  }

  if (usage.kind === 'unused') {
    return {
      label: t('未用技能'),
      lines: [
        t('可用 {{count}} 个技能，本次未读取', {
          count: usage.catalogueCount,
        }),
      ],
    };
  }

  const lines = [];
  if (usage.alwaysLoaded.length > 0) {
    lines.push(`${t('始终装载')}：${usage.alwaysLoaded.join(', ')}`);
  }
  if (usage.onDemand.length > 0) {
    lines.push(`${t('按需读取')}：${usage.onDemand.join(', ')}`);
  }
  return { label: t('技能 {{count}}', { count: usage.count }), lines };
}
