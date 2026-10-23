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
export interface PromptGroup {
  id: number
  name: string
}

export interface SkillResource {
  path: string
  content: string
}

export interface PromptSkill {
  id?: string
  title: string
  description?: string
  path?: string
  content?: string
  resources?: SkillResource[]
}

export interface PromptTemplate {
  id: number
  title: string
  description: string
  content?: string
  enabled: boolean
  sort_order: number
  version: number
  group_ids: number[]
  group_details: PromptGroup[]
  kind?: 'prompt' | 'bundle'
  load_mode?: '' | 'eager' | 'progressive'
  category_id?: number
  category_name?: string
  skills?: PromptSkill[]
  skill_count?: number
}

export interface PromptCategory {
  id: number
  name: string
  version: number
}

export interface SkillStoreEntry {
  id: string
  title: string
  description: string
  kind: 'prompt' | 'bundle'
  category: string
  skill_count: number
  source: 'bundled' | 'online'
  version: string
}

export interface TokenPromptBinding {
  template: PromptTemplate
  enabled: boolean
  sort_order: number
}

export interface PromptToken {
  id: number
  name: string
  prompt_count?: number
  enabled_prompt_count?: number
}

export type PromptTemplateInput = Omit<
  PromptTemplate,
  'id' | 'group_details' | 'content' | 'category_name' | 'skill_count'
> & { content: string }

export interface PromptResponse<T> {
  success: boolean
  message?: string
  data?: T
}
