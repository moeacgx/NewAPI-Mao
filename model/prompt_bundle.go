package model

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type PromptSkill struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// NormalizePromptTemplateContent 校验可导入草稿并生成合集正文，不查询数据库或要求发布分组。
func NormalizePromptTemplateContent(template *PromptTemplate) error {
	if template == nil {
		return errors.New("技能内容不能为空")
	}
	template.Title = strings.TrimSpace(template.Title)
	if !utf8.ValidString(template.Title) || !utf8.ValidString(template.Description) || utf8.RuneCountInString(template.Title) < 1 || utf8.RuneCountInString(template.Title) > 80 || utf8.RuneCountInString(template.Description) > 512 {
		return errors.New("提示词标题或说明长度不合法")
	}
	return normalizePromptTemplateContent(template)
}

// normalizePromptTemplateContent 使保存、安装和运行时采用同一份合集正文，拒绝未知类型。
func normalizePromptTemplateContent(template *PromptTemplate) error {
	if template.Kind == "" {
		template.Kind = "prompt"
	}
	switch template.Kind {
	case "prompt":
		template.Skills = []PromptSkill{}
		template.SkillsJSON = ""
		template.SkillCount = 1
	case "bundle":
		if template.Skills == nil && template.SkillsJSON != "" {
			if err := common.UnmarshalJsonStr(template.SkillsJSON, &template.Skills); err != nil {
				return errors.New("技能合集明细格式错误")
			}
		}
		if len(template.Skills) < 2 || len(template.Skills) > 16 {
			return errors.New("技能合集必须包含 2 至 16 个技能")
		}
		seen := make(map[string]bool, len(template.Skills))
		parts := make([]string, 0, len(template.Skills))
		for i := range template.Skills {
			skill := &template.Skills[i]
			skill.Title = strings.TrimSpace(skill.Title)
			if !utf8.ValidString(skill.Title) || utf8.RuneCountInString(skill.Title) < 1 || utf8.RuneCountInString(skill.Title) > 80 || seen[skill.Title] {
				return errors.New("合集内技能标题必须为 1 至 80 个字符且不能重复")
			}
			if !utf8.ValidString(skill.Content) {
				return fmt.Errorf("合集内技能「%s」正文必须为有效 UTF-8 文本", skill.Title)
			}
			if strings.TrimSpace(skill.Content) == "" {
				return fmt.Errorf("合集内技能「%s」正文不能为空", skill.Title)
			}
			if len(skill.Content) > MaxPromptContentBytes {
				return fmt.Errorf("合集内技能「%s」正文为 %d 字节，超过 %d 字节上限", skill.Title, len(skill.Content), MaxPromptContentBytes)
			}
			seen[skill.Title] = true
			parts = append(parts, "## 技能："+skill.Title+"\n"+skill.Content)
		}
		template.Content = strings.Join(parts, "\n\n")
		if len(template.Content) > MaxPromptContentBytes {
			return fmt.Errorf("技能合集拼接正文为 %d 字节，超过 %d 字节上限（包含技能标题和分隔符），请减少技能或拆分合集", len(template.Content), MaxPromptContentBytes)
		}
		encoded, err := common.Marshal(template.Skills)
		if err != nil {
			return err
		}
		template.SkillsJSON = string(encoded)
		template.SkillCount = len(template.Skills)
	default:
		return errors.New("技能类型无效")
	}
	if !utf8.ValidString(template.Content) {
		return errors.New("提示词正文必须为有效 UTF-8 文本")
	}
	if strings.TrimSpace(template.Content) == "" {
		return errors.New("提示词正文不能为空")
	}
	if len(template.Content) > MaxPromptContentBytes {
		return fmt.Errorf("提示词正文为 %d 字节，超过 %d 字节上限", len(template.Content), MaxPromptContentBytes)
	}
	return nil
}

func hydratePromptTemplateMetadata(tx *gorm.DB, templates []PromptTemplate) error {
	categoryIDs := make([]int, 0, len(templates))
	for i := range templates {
		if err := normalizePromptTemplateContent(&templates[i]); err != nil {
			return err
		}
		if templates[i].CategoryID > 0 {
			categoryIDs = append(categoryIDs, templates[i].CategoryID)
		}
		templates[i].CategoryName = ""
	}
	if len(categoryIDs) == 0 {
		return nil
	}
	var categories []PromptCategory
	if err := tx.Where("id IN ?", categoryIDs).Find(&categories).Error; err != nil {
		return err
	}
	names := make(map[int]string, len(categories))
	for _, category := range categories {
		names[category.ID] = category.Name
	}
	for i := range templates {
		templates[i].CategoryName = names[templates[i].CategoryID]
	}
	return nil
}
