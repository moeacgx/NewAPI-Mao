package model

import (
	"errors"
	"sort"

	"gorm.io/gorm"
)

const MaxTokenPrompts = 8
const MaxPromptContentBytes = 32768
const MaxTokenPromptBytes = 65536

var ErrPromptVersionConflict = errors.New("提示词已被其他管理员更新，请刷新后重试")

type PromptTemplate struct {
	ID           int              `json:"id" gorm:"primaryKey"`
	Title        string           `json:"title" gorm:"size:320"`
	Description  string           `json:"description" gorm:"type:text"`
	Content      string           `json:"content" gorm:"type:text"`
	Kind         string           `json:"kind" gorm:"size:16"`
	CategoryID   int              `json:"category_id" gorm:"index"`
	CategoryName string           `json:"category_name" gorm:"-"`
	SkillsJSON   string           `json:"-" gorm:"type:text"`
	Skills       []PromptSkill    `json:"skills" gorm:"-"`
	SkillCount   int              `json:"skill_count" gorm:"-"`
	Enabled      bool             `json:"enabled"`
	SortOrder    int              `json:"sort_order"`
	Version      int              `json:"version"`
	GroupIDs     []int            `json:"group_ids" gorm:"-"`
	GroupDetails []GroupReference `json:"group_details" gorm:"-"`
}

type PromptTemplateGroup struct {
	TemplateID int `gorm:"primaryKey;autoIncrement:false"`
	GroupID    int `gorm:"primaryKey;autoIncrement:false;index"`
}

type TokenPromptBinding struct {
	TokenID    int `gorm:"primaryKey;autoIncrement:false"`
	TemplateID int `gorm:"primaryKey;autoIncrement:false;index"`
	Enabled    bool
	SortOrder  int
}

type TokenPromptBindingInput struct {
	TemplateID int  `json:"template_id"`
	Enabled    bool `json:"enabled"`
	SortOrder  int  `json:"sort_order"`
}

type TokenPromptInstallation struct {
	Template  PromptTemplate `json:"template"`
	Enabled   bool           `json:"enabled"`
	SortOrder int            `json:"sort_order"`
}

type TokenPromptContent struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Content string `json:"-"`
	Version int    `json:"version"`
}

func hydratePromptTemplateGroups(tx *gorm.DB, templates []PromptTemplate) error {
	if err := hydratePromptTemplateMetadata(tx, templates); err != nil {
		return err
	}
	ids := make([]int, 0, len(templates))
	for i := range templates {
		ids = append(ids, templates[i].ID)
		templates[i].GroupIDs = []int{}
		templates[i].GroupDetails = []GroupReference{}
	}
	if len(ids) == 0 {
		return nil
	}
	var bindings []PromptTemplateGroup
	if err := tx.Where("template_id IN ?", ids).Order("group_id").Find(&bindings).Error; err != nil {
		return err
	}
	groupIDs := make([]int, 0, len(bindings))
	for _, binding := range bindings {
		groupIDs = append(groupIDs, binding.GroupID)
	}
	var groups []Group
	if len(groupIDs) > 0 {
		if err := tx.Where("id IN ?", groupIDs).Find(&groups).Error; err != nil {
			return err
		}
	}
	groupMap := make(map[int]*Group, len(groups))
	for i := range groups {
		groupMap[groups[i].Id] = &groups[i]
	}
	indexes := make(map[int]int, len(templates))
	for i := range templates {
		indexes[templates[i].ID] = i
	}
	for _, binding := range bindings {
		index, ok := indexes[binding.TemplateID]
		if !ok {
			continue
		}
		templates[index].GroupIDs = append(templates[index].GroupIDs, binding.GroupID)
		if group := groupMap[binding.GroupID]; group != nil {
			templates[index].GroupDetails = append(templates[index].GroupDetails, newGroupReference(group))
		}
	}
	return nil
}

func ListPromptTemplates(enabledOnly bool) ([]PromptTemplate, error) {
	templates := make([]PromptTemplate, 0)
	query := DB.Order("sort_order, id")
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}
	if err := query.Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, hydratePromptTemplateGroups(DB, templates)
}

func SavePromptTemplate(input PromptTemplate) (*PromptTemplate, error) {
	if err := NormalizePromptTemplateContent(&input); err != nil {
		return nil, err
	}
	if input.CategoryID < 0 {
		return nil, errors.New("技能分类无效")
	}
	if (input.Enabled && len(input.GroupIDs) == 0) || len(input.GroupIDs) > 1000 {
		return nil, errors.New("请选择 1 至 1000 个允许装载的分组")
	}
	if input.SortOrder < -1000000 || input.SortOrder > 1000000 {
		return nil, errors.New("提示词排序必须在 -1000000 至 1000000 之间")
	}
	groupIDs := uniquePositiveGroupIDs(input.GroupIDs)
	if len(groupIDs) != len(input.GroupIDs) {
		return nil, errors.New("白名单分组无效或重复")
	}
	sort.Ints(groupIDs)
	// 统一按令牌、模板的顺序加锁，保证两个模板同时更新时不会突破安装总量上限。
	var affectedTokenIDs []int
	if input.ID > 0 {
		if err := DB.Model(&TokenPromptBinding{}).Where("template_id = ? AND enabled = ?", input.ID, true).Pluck("token_id", &affectedTokenIDs).Error; err != nil {
			return nil, err
		}
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if input.CategoryID > 0 {
			var category PromptCategory
			if err := tx.First(&category, input.CategoryID).Error; err != nil {
				return errors.New("技能分类不存在")
			}
		}
		lockedTokenIDs := make(map[int]bool, len(affectedTokenIDs))
		if len(affectedTokenIDs) > 0 {
			var tokens []Token
			if err := lockForUpdate(tx).Select("id").Where("id IN ?", affectedTokenIDs).Order("id").Find(&tokens).Error; err != nil {
				return err
			}
			for _, token := range tokens {
				lockedTokenIDs[token.Id] = true
			}
		}
		var groupCount int64
		if err := tx.Model(&Group{}).Where("id IN ?", groupIDs).Count(&groupCount).Error; err != nil {
			return err
		}
		if int(groupCount) != len(groupIDs) {
			return errors.New("白名单包含不存在的分组")
		}
		if input.ID > 0 {
			var stored PromptTemplate
			if err := lockForUpdate(tx).First(&stored, input.ID).Error; err != nil {
				return err
			}
			if input.Version != stored.Version {
				return ErrPromptVersionConflict
			}
			// 更新正文或重新上架时，也保护已有令牌的总正文边界。
			if input.Enabled {
				var installed []TokenPromptBinding
				if err := lockForUpdate(tx.Model(&TokenPromptBinding{})).Joins("JOIN tokens t ON t.id = token_prompt_bindings.token_id AND t.deleted_at IS NULL").Where("template_id = ? AND token_prompt_bindings.enabled = ?", input.ID, true).Find(&installed).Error; err != nil {
					return err
				}
				for _, binding := range installed {
					if !lockedTokenIDs[binding.TokenID] {
						return ErrPromptVersionConflict
					}
					var others []PromptTemplate
					if err := lockForUpdate(tx.Table("prompt_templates")).Select("prompt_templates.*").Joins("JOIN token_prompt_bindings b ON b.template_id = prompt_templates.id").Where("b.token_id = ? AND b.enabled = ? AND prompt_templates.enabled = ? AND prompt_templates.id <> ?", binding.TokenID, true, true, input.ID).Find(&others).Error; err != nil {
						return err
					}
					total := len(input.Content)
					for _, other := range others {
						if err := normalizePromptTemplateContent(&other); err != nil {
							return err
						}
						total += len(other.Content)
					}
					if total > MaxTokenPromptBytes {
						return errors.New("更新后已安装令牌的启用提示词正文超过 65536 字节")
					}
				}
			}
			input.Version++
			result := tx.Model(&PromptTemplate{}).Where("id = ? AND version = ?", input.ID, stored.Version).Updates(map[string]interface{}{"title": input.Title, "description": input.Description, "content": input.Content, "kind": input.Kind, "category_id": input.CategoryID, "skills_json": input.SkillsJSON, "enabled": input.Enabled, "sort_order": input.SortOrder, "version": input.Version})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return ErrPromptVersionConflict
			}
		} else {
			input.ID = 0
			input.Version = 1
			if err := tx.Create(&input).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("template_id = ?", input.ID).Delete(&PromptTemplateGroup{}).Error; err != nil {
			return err
		}
		for _, groupID := range groupIDs {
			if err := tx.Create(&PromptTemplateGroup{TemplateID: input.ID, GroupID: groupID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	items := []PromptTemplate{input}
	if err := hydratePromptTemplateGroups(DB, items); err != nil {
		return nil, err
	}
	return &items[0], nil
}

func GetTokenForPromptInstallation(userID, tokenID int) (*Token, error) {
	if userID <= 0 || tokenID <= 0 {
		return nil, errors.New("令牌身份无效")
	}
	var token Token
	if err := DB.Select("id", "user_id", "group", "group_mode", "auto_groups").Where("id = ? AND user_id = ?", tokenID, userID).First(&token).Error; err != nil {
		return nil, err
	}
	if err := HydrateTokenGroupBindings(DB, []*Token{&token}); err != nil {
		return nil, err
	}
	return &token, nil
}

func GetTokenPromptBindings(userID, tokenID int) ([]TokenPromptInstallation, error) {
	var token Token
	if userID <= 0 || tokenID <= 0 {
		return nil, errors.New("令牌身份无效")
	}
	if err := DB.Select("id").Where("id = ? AND user_id = ?", tokenID, userID).First(&token).Error; err != nil {
		return nil, err
	}
	var bindings []TokenPromptBinding
	if err := DB.Where("token_id = ?", tokenID).Order("sort_order, template_id").Find(&bindings).Error; err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(bindings))
	for _, binding := range bindings {
		ids = append(ids, binding.TemplateID)
	}
	templates := make([]PromptTemplate, 0)
	if len(ids) > 0 {
		if err := DB.Where("id IN ?", ids).Find(&templates).Error; err != nil {
			return nil, err
		}
		if err := hydratePromptTemplateGroups(DB, templates); err != nil {
			return nil, err
		}
	}
	byID := make(map[int]PromptTemplate, len(templates))
	for _, template := range templates {
		byID[template.ID] = template
	}
	result := make([]TokenPromptInstallation, 0, len(bindings))
	for _, binding := range bindings {
		if template, ok := byID[binding.TemplateID]; ok {
			result = append(result, TokenPromptInstallation{Template: template, Enabled: binding.Enabled, SortOrder: binding.SortOrder})
		}
	}
	return result, nil
}

func ReplaceTokenPromptBindings(userID, tokenID int, inputs []TokenPromptBindingInput, allowedGroupIDs []int) error {
	if userID <= 0 || tokenID <= 0 {
		return errors.New("令牌身份无效")
	}
	if len(inputs) > MaxTokenPrompts {
		return errors.New("每个令牌最多安装 8 个提示词")
	}
	ids := make([]int, 0, len(inputs))
	seen := make(map[int]bool, len(inputs))
	for _, input := range inputs {
		if input.TemplateID <= 0 || seen[input.TemplateID] {
			return errors.New("提示词 ID 无效或重复")
		}
		seen[input.TemplateID] = true
		ids = append(ids, input.TemplateID)
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var token Token
		if err := lockForUpdate(tx).Select("id").Where("id = ? AND user_id = ?", tokenID, userID).First(&token).Error; err != nil {
			return err
		}
		var existing []TokenPromptBinding
		if err := tx.Where("token_id = ?", tokenID).Find(&existing).Error; err != nil {
			return err
		}
		installed := make(map[int]bool, len(existing))
		for _, binding := range existing {
			installed[binding.TemplateID] = true
		}
		templates := make([]PromptTemplate, 0, len(ids))
		if len(ids) > 0 {
			if err := lockForUpdate(tx).Where("id IN ?", ids).Order("id").Find(&templates).Error; err != nil {
				return err
			}
		}
		if len(templates) != len(ids) {
			return errors.New("提示词不存在")
		}
		byID := make(map[int]PromptTemplate, len(templates))
		for _, template := range templates {
			byID[template.ID] = template
		}
		var allowed []PromptTemplateGroup
		if len(ids) > 0 && len(allowedGroupIDs) > 0 {
			if err := tx.Where("template_id IN ? AND group_id IN ?", ids, allowedGroupIDs).Find(&allowed).Error; err != nil {
				return err
			}
		}
		eligible := make(map[int]bool, len(allowed))
		for _, group := range allowed {
			eligible[group.TemplateID] = true
		}
		total := 0
		for _, input := range inputs {
			template := byID[input.TemplateID]
			if (!template.Enabled || !eligible[input.TemplateID]) && (!installed[input.TemplateID] || input.Enabled) {
				return errors.New("提示词已下架，或令牌可用分组不在装载白名单内")
			}
			if input.Enabled {
				if err := normalizePromptTemplateContent(&template); err != nil {
					return err
				}
				total += len(template.Content)
			}
		}
		if total > MaxTokenPromptBytes {
			return errors.New("启用提示词正文合计不能超过 65536 字节")
		}
		if err := tx.Where("token_id = ?", tokenID).Delete(&TokenPromptBinding{}).Error; err != nil {
			return err
		}
		for _, input := range inputs {
			binding := TokenPromptBinding{TokenID: tokenID, TemplateID: input.TemplateID, Enabled: input.Enabled, SortOrder: input.SortOrder}
			if err := tx.Create(&binding).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func GetActiveTokenPrompts(tokenID int, groupCode string) ([]TokenPromptContent, error) {
	result := make([]TokenPromptContent, 0)
	if tokenID <= 0 || groupCode == "" || groupCode == "auto" {
		return result, nil
	}
	group, err := GetGroupByCodeOrAlias(groupCode)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	var templates []PromptTemplate
	err = DB.Table("prompt_templates").Select("prompt_templates.*").
		Joins("JOIN token_prompt_bindings b ON b.template_id = prompt_templates.id").
		Joins("JOIN prompt_template_groups g ON g.template_id = prompt_templates.id").
		Joins("JOIN tokens t ON t.id = b.token_id AND t.deleted_at IS NULL").
		Where("b.token_id = ? AND b.enabled = ? AND prompt_templates.enabled = ? AND g.group_id = ?", tokenID, true, true, group.Id).
		Order("b.sort_order, prompt_templates.id").Find(&templates).Error
	if err != nil {
		return nil, err
	}
	total := 0
	for _, template := range templates {
		if err := normalizePromptTemplateContent(&template); err != nil {
			return nil, err
		}
		total += len(template.Content)
		if len(result) >= MaxTokenPrompts || total > MaxTokenPromptBytes || len(template.Content) > MaxPromptContentBytes {
			return nil, errors.New("令牌提示词超出装载上限")
		}
		result = append(result, TokenPromptContent{ID: template.ID, Title: template.Title, Content: template.Content, Version: template.Version})
	}
	return result, nil
}

func HydrateTokenPromptCounts(tokens []*Token) error {
	ids := make([]int, 0, len(tokens))
	for _, token := range tokens {
		if token != nil {
			token.PromptCount = 0
			token.EnabledPromptCount = 0
			ids = append(ids, token.Id)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	var rows []struct {
		TokenID         int
		Enabled         bool
		TemplateEnabled bool
	}
	if err := DB.Table("token_prompt_bindings b").Select("b.token_id, b.enabled, p.enabled AS template_enabled").Joins("JOIN prompt_templates p ON p.id = b.template_id").Where("b.token_id IN ?", ids).Scan(&rows).Error; err != nil {
		return err
	}
	counts := make(map[int][2]int, len(tokens))
	for _, row := range rows {
		count := counts[row.TokenID]
		count[0]++
		if row.Enabled && row.TemplateEnabled {
			count[1]++
		}
		counts[row.TokenID] = count
	}
	for _, token := range tokens {
		if token != nil {
			count := counts[token.Id]
			token.PromptCount, token.EnabledPromptCount = count[0], count[1]
		}
	}
	return nil
}
