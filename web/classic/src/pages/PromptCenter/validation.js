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

export function validatePromptTemplateDraft(draft) {
  const progressive = draft.load_mode === 'progressive';
  if (
    !draft.title.trim() ||
    Array.from(draft.title.trim()).length > 80 ||
    (draft.enabled && !draft.group_ids.length)
  )
    return 'Enter a title and at least one allowed group before publishing';
  if (progressive && draft.enabled && !draft.description.trim())
    return 'Add a description explaining when to use this skill before publishing';

  const skills =
    draft.kind === 'bundle'
      ? draft.skills
      : [{ ...draft.skills[0], title: draft.title, content: draft.content }];
  const titles = skills.map((skill) => skill.title.trim());
  if (
    draft.kind === 'bundle' &&
    (skills.length < 2 ||
      skills.length > (progressive ? 128 : 16) ||
      new Set(titles).size !== titles.length)
  )
    return progressive
      ? 'Use 2 to 128 skills with unique titles'
      : 'Use 2 to 16 unique skill titles and keep the combined text within 32768 bytes';

  let packageBytes = 0;
  const mainPaths = new Set();
  for (const skill of skills) {
    if (
      !skill.title.trim() ||
      Array.from(skill.title.trim()).length > 80 ||
      !skill.content?.trim()
    )
      return 'Enter a title and nonempty text for every skill';
    if (
      progressive &&
      draft.kind === 'bundle' &&
      draft.enabled &&
      !skill.description?.trim()
    )
      return 'Add a description explaining when to use every skill before publishing';
    if (progressive && skill.path) {
      if (mainPaths.has(skill.path))
        return 'Use unique relative file paths without parent directory references';
      mainPaths.add(skill.path);
    }

    const files = [
      { path: skill.path || 'SKILL.md', content: skill.content },
      ...(progressive ? skill.resources || [] : []),
    ];
    const paths = new Set();
    for (const [fileIndex, file] of files.entries()) {
      if (progressive) {
        const segments = file.path.split('/');
        if (
          !file.path ||
          /[\\:\u0000]/.test(file.path) ||
          segments.some(
            (segment) => !segment || segment === '.' || segment === '..',
          ) ||
          (fileIndex > 0 &&
            (file.path.toLowerCase() === 'skill.md' || paths.has(file.path)))
        )
          return 'Use unique relative file paths without parent directory references';
        if (fileIndex > 0) paths.add(file.path);
      }
      const bytes = new TextEncoder().encode(file.content || '').length;
      if (bytes > (progressive ? 8 * 1024 * 1024 : 32768))
        return progressive
          ? 'Each skill file must not exceed 8 MiB'
          : 'Prompt text must not exceed 32768 bytes';
      packageBytes += bytes;
    }
  }
  if (progressive && packageBytes > 64 * 1024 * 1024)
    return 'The total skill text must not exceed 64 MiB';
  if (!progressive && draft.kind === 'bundle') {
    const combined = skills
      .map((skill) => `## 技能：${skill.title.trim()}\n${skill.content}`)
      .join('\n\n');
    if (new TextEncoder().encode(combined).length > 32768)
      return 'Use 2 to 16 unique skill titles and keep the combined text within 32768 bytes';
  }
  return '';
}
