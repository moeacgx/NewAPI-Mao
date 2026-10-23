/*
Copyright (C) 2023-2026 QuantumNous

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
import { z } from 'zod'

function isSkillRelativePath(path: string): boolean {
  return (
    path.length > 0 &&
    new TextEncoder().encode(path).length <= 1024 &&
    !/[\\:]/.test(path) &&
    !path.includes('\0') &&
    path.split('/').every((part) => part && part !== '.' && part !== '..')
  )
}

export const promptTemplateSchema = z
  .object({
    title: z.string().trim().min(1).max(80),
    description: z.string().max(512),
    content: z.string(),
    kind: z.enum(['prompt', 'bundle']).optional(),
    load_mode: z.enum(['', 'eager', 'progressive']).optional(),
    category_id: z.number().int().min(0).optional(),
    skills: z
      .array(
        z.object({
          id: z.string().optional(),
          title: z.string(),
          description: z.string().max(512).optional(),
          path: z.string().optional(),
          content: z.string(),
          resources: z
            .array(z.object({ path: z.string(), content: z.string() }))
            .optional(),
        })
      )
      .optional(),
    enabled: z.boolean(),
    sort_order: z.number().int().min(-1000000).max(1000000),
    version: z.number().int().min(0),
    group_ids: z.array(z.number().int().positive()).max(1000),
  })
  .refine((value) => !value.enabled || value.group_ids.length > 0, {
    path: ['group_ids'],
    message: 'Select at least one allowed group',
  })
  .superRefine((value, ctx) => {
    const progressive = value.load_mode === 'progressive'
    if (progressive) {
      const bundle = value.kind === 'bundle'
      const skills = bundle
        ? (value.skills ?? [])
        : [
            {
              ...value.skills?.[0],
              title: value.title,
              description: value.description,
              content: value.content,
            },
          ]
      if (bundle && (skills.length < 2 || skills.length > 128)) {
        ctx.addIssue({
          code: 'custom',
          path: ['skills'],
          message: 'A progressive bundle must contain 2 to 128 skills',
        })
      }
      let totalBytes = 0
      let fileCount = skills.length
      const entryPaths = new Set<string>()
      for (const [index, skill] of skills.entries()) {
        const title = skill.title.trim()
        const skillPath = bundle ? ['skills', index] : []
        if (!title || [...title].length > 80) {
          ctx.addIssue({
            code: 'custom',
            path: [...skillPath, 'title'],
            message: 'Enter a title of 1 to 80 characters',
          })
        }
        if (value.enabled && !skill.description?.trim()) {
          ctx.addIssue({
            code: 'custom',
            path: [...skillPath, 'description'],
            message: 'Describe when to use this skill before publishing',
          })
        }
        const contentBytes = new TextEncoder().encode(skill.content).length
        totalBytes += contentBytes
        if (
          !skill.content.trim() ||
          skill.content.includes('\0') ||
          contentBytes > 8 * 1024 * 1024
        ) {
          ctx.addIssue({
            code: 'custom',
            path: [...skillPath, 'content'],
            message: 'Enter skill text up to 8 MiB per file',
          })
        }
        if (
          skill.path &&
          (!isSkillRelativePath(skill.path) || entryPaths.has(skill.path))
        ) {
          ctx.addIssue({
            code: 'custom',
            path: ['skills', index, 'path'],
            message: 'Use a relative path within the skill folder',
          })
        }
        if (skill.path) entryPaths.add(skill.path)
        const paths = new Set<string>()
        fileCount += skill.resources?.length ?? 0
        for (const [resourceIndex, resource] of (
          skill.resources ?? []
        ).entries()) {
          const path = ['skills', index, 'resources', resourceIndex]
          if (
            !isSkillRelativePath(resource.path) ||
            resource.path.toLowerCase() === 'skill.md' ||
            paths.has(resource.path)
          ) {
            ctx.addIssue({
              code: 'custom',
              path: [...path, 'path'],
              message: 'Use a unique relative path within the skill folder',
            })
          }
          paths.add(resource.path)
          const bytes = new TextEncoder().encode(resource.content).length
          totalBytes += bytes
          if (resource.content.includes('\0') || bytes > 8 * 1024 * 1024) {
            ctx.addIssue({
              code: 'custom',
              path: [...path, 'content'],
              message: 'Reference text must not exceed 8 MiB per file',
            })
          }
        }
      }
      if (totalBytes > 64 * 1024 * 1024) {
        ctx.addIssue({
          code: 'custom',
          path: bundle ? ['skills'] : ['content'],
          message: 'Skill package text must not exceed 64 MiB in total',
        })
      }
      if (fileCount > 4096) {
        ctx.addIssue({
          code: 'custom',
          path: bundle ? ['skills'] : ['content'],
          message: 'A skill package can contain at most 4096 text files',
        })
      }
      return
    }
    if (value.kind !== 'bundle') {
      if (
        !value.content.trim() ||
        new TextEncoder().encode(value.content).length > 32768
      ) {
        ctx.addIssue({
          code: 'custom',
          path: ['content'],
          message: 'Enter prompt text up to 32768 bytes',
        })
      }
      return
    }
    const skills = value.skills ?? []
    if (skills.length < 2 || skills.length > 16) {
      ctx.addIssue({
        code: 'custom',
        path: ['skills'],
        message: 'A skill bundle must contain 2 to 16 skills',
      })
    }
    const titles = new Set<string>()
    for (const [index, skill] of skills.entries()) {
      const title = skill.title.trim()
      if (!title || [...title].length > 80 || titles.has(title)) {
        ctx.addIssue({
          code: 'custom',
          path: ['skills', index, 'title'],
          message: 'Use a unique skill title of 1 to 80 characters',
        })
      }
      titles.add(title)
      if (
        !skill.content.trim() ||
        new TextEncoder().encode(skill.content).length > 32768
      ) {
        ctx.addIssue({
          code: 'custom',
          path: ['skills', index, 'content'],
          message: 'Enter prompt text up to 32768 bytes',
        })
      }
    }
    // 与服务端保持一致：标题和分隔符也占合集的装载字节数。
    const content = skills
      .map((skill) => `## 技能：${skill.title.trim()}\n${skill.content}`)
      .join('\n\n')
    if (new TextEncoder().encode(content).length > 32768) {
      ctx.addIssue({
        code: 'custom',
        path: ['skills'],
        message: 'Combined skill bundle text must not exceed 32768 bytes',
      })
    }
  })

export type PromptTemplateForm = z.infer<typeof promptTemplateSchema>
