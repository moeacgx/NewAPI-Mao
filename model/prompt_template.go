package model

import (
	"errors"
	"sort"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const MaxTokenPrompts = 8
const MaxPromptContentBytes = 32768
const MaxTokenPromptBytes = 65536

var ErrPromptVersionConflict = errors.New("提示词已被其他管理员更新，请刷新后重试")

type PromptTemplate struct {
	ID           int           `json:"id" gorm:"primaryKey"`
	Title        string        `json:"title" gorm:"size:320"`
	Description  string        `json:"description" gorm:"type:text"`
	Content      string        `json:"content" gorm:"type:text"`
	Kind         string        `json:"kind" gorm:"size:16"`
	LoadMode     string        `json:"load_mode" gorm:"size:16"`
	CategoryID   int           `json:"category_id" gorm:"index"`
	CategoryName string        `json:"category_name" gorm:"-"`
	SkillsJSON   string        `json:"-" gorm:"type:text"`
	Skills       []PromptSkill `json:"skills" gorm:"-"`
	SkillCount   int           `json:"skill_count" gorm:"-"`
	Enabled      bool          `json:"enabled"`
	SortOrder    int           `json:"sort_order"`
	Version      int           `json:"version"`
	// AllGroups 为 true 时任一启用的实际请求分组都可安装和装载，白名单关联被清空。
	// 迁移新增列对存量行为 NULL，查询只匹配 all_groups = true，NULL 与 false 都按白名单处理；
	// 不加布尔默认值标签，避免 MySQL/PostgreSQL 每次启动重复 ALTER。
	AllGroups    bool             `json:"all_groups"`
	GroupIDs     []int            `json:"group_ids" gorm:"-"`
	GroupDetails []GroupReference `json:"group_details" gorm:"-"`
	// 来源只在管理员接口返回；公共目录和令牌安装摘要使用独立的公开结构。
	RepositorySourceJSON string                  `json:"-" gorm:"type:text"`
	RepositorySource     *PromptRepositorySource `json:"repository_source,omitempty" gorm:"-"`
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

func hydratePromptTemplateGroups(tx *gorm.DB, templates []PromptTemplate, includeContent bool) error {
	if err := hydratePromptTemplateMetadata(tx, templates, includeContent); err != nil {
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
	return listPromptTemplates(enabledOnly, true)
}

// GetAdminPromptTemplate 读取包含正文、引用文件和来源状态的单个管理员模板。
func GetAdminPromptTemplate(id int) (*PromptTemplate, error) {
	var template PromptTemplate
	if err := DB.First(&template, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPromptTemplateNotFound
		}
		return nil, err
	}
	items := []PromptTemplate{template}
	if err := hydratePromptTemplateGroups(DB, items, true); err != nil {
		return nil, err
	}
	return &items[0], nil
}

// DeletePromptTemplate 按版本删除模板，并在同一事务中卸载所有令牌安装、白名单和渐进式文件。
// 返回被自动卸载的令牌数量。
func DeletePromptTemplate(id, version int) (int, error) {
	if id <= 0 || version <= 0 {
		return 0, errors.New("提示词 ID 或版本无效")
	}
	uninstalled := 0
	err := DB.Transaction(func(tx *gorm.DB) error {
		var stored PromptTemplate
		if err := lockForUpdate(tx).Select("id", "version").First(&stored, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPromptTemplateNotFound
			}
			return err
		}
		if stored.Version != version {
			return ErrPromptVersionConflict
		}
		bindings := tx.Where("template_id = ?", id).Delete(&TokenPromptBinding{})
		if bindings.Error != nil {
			return bindings.Error
		}
		uninstalled = int(bindings.RowsAffected)
		for _, table := range []interface{}{&PromptSkillChunk{}, &PromptSkillFile{}, &PromptSkillRecord{}, &PromptTemplateGroup{}} {
			if err := tx.Where("template_id = ?", id).Delete(table).Error; err != nil {
				return err
			}
		}
		deleted := tx.Where("id = ? AND version = ?", id, version).Delete(&PromptTemplate{})
		if deleted.Error != nil {
			return deleted.Error
		}
		if deleted.RowsAffected != 1 {
			return ErrPromptVersionConflict
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	// SQLite 可能复用最大主键；清理同 ID 的正文页，避免新模板命中旧缓存。
	progressiveSkillContents.removeTemplate(id)
	return uninstalled, nil
}

// ListPromptTemplateSummaries 不读取渐进式文件分块，供用户目录页面使用。
func ListPromptTemplateSummaries(enabledOnly bool) ([]PromptTemplate, error) {
	return listPromptTemplates(enabledOnly, false)
}

func listPromptTemplates(enabledOnly, includeContent bool) ([]PromptTemplate, error) {
	templates := make([]PromptTemplate, 0)
	query := DB.Order("sort_order, id")
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}
	if err := query.Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, hydratePromptTemplateGroups(DB, templates, includeContent)
}

func SavePromptTemplate(input PromptTemplate) (*PromptTemplate, error) {
	if err := NormalizePromptTemplateContent(&input); err != nil {
		return nil, err
	}
	if input.CategoryID < 0 {
		return nil, errors.New("技能分类无效")
	}
	if input.AllGroups {
		input.GroupIDs = []int{}
	}
	if (input.Enabled && !input.AllGroups && len(input.GroupIDs) == 0) || len(input.GroupIDs) > 1000 {
		return nil, errors.New("请选择不限分组，或 1 至 1000 个允许装载的分组")
	}
	if input.SortOrder < -1000000 || input.SortOrder > 1000000 {
		return nil, errors.New("提示词排序必须在 -1000000 至 1000000 之间")
	}
	groupIDs := uniquePositiveGroupIDs(input.GroupIDs)
	if len(groupIDs) != len(input.GroupIDs) {
		return nil, errors.New("白名单分组无效或重复")
	}
	sort.Ints(groupIDs)
	// 未提交来源的旧客户端编辑已有模板时保留数据库中的来源，不能借此清除记录。
	sourceProvided := input.RepositorySource != nil
	input.RepositorySourceJSON = ""
	if sourceProvided {
		source := *input.RepositorySource
		if err := normalizePromptRepositorySource(&source); err != nil {
			return nil, err
		}
		encoded, err := common.Marshal(source)
		if err != nil {
			return nil, err
		}
		input.RepositorySourceJSON = string(encoded)
	}
	// 统一按令牌、模板的顺序加锁，保证两个模板同时更新时不会突破安装总量上限。
	var affectedTokenIDs []int
	if input.ID > 0 {
		if err := DB.Model(&TokenPromptBinding{}).Where("template_id = ? AND enabled = ?", input.ID, true).Pluck("token_id", &affectedTokenIDs).Error; err != nil {
			return nil, err
		}
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		wasProgressive := false
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
			wasProgressive = stored.LoadMode == PromptLoadModeProgressive
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
					total := 0
					if input.LoadMode != PromptLoadModeProgressive {
						total = len(input.Content)
					}
					for _, other := range others {
						if other.LoadMode == PromptLoadModeProgressive {
							continue
						}
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
			storedContent := input.Content
			if input.LoadMode == PromptLoadModeProgressive {
				storedContent = ""
			}
			updates := map[string]interface{}{"title": input.Title, "description": input.Description, "content": storedContent, "kind": input.Kind, "load_mode": input.LoadMode, "category_id": input.CategoryID, "skills_json": input.SkillsJSON, "enabled": input.Enabled, "all_groups": input.AllGroups, "sort_order": input.SortOrder, "version": input.Version}
			if sourceProvided {
				updates["repository_source_json"] = input.RepositorySourceJSON
			} else {
				input.RepositorySourceJSON = stored.RepositorySourceJSON
			}
			result := tx.Model(&PromptTemplate{}).Where("id = ? AND version = ?", input.ID, stored.Version).Updates(updates)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return ErrPromptVersionConflict
			}
		} else {
			input.ID = 0
			input.Version = 1
			stored := input
			if stored.LoadMode == PromptLoadModeProgressive {
				stored.Content = ""
			}
			if err := tx.Create(&stored).Error; err != nil {
				return err
			}
			input.ID = stored.ID
		}
		if input.LoadMode == PromptLoadModeProgressive || wasProgressive {
			if err := persistProgressiveSkills(tx, &input); err != nil {
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
	if err := hydratePromptTemplateGroups(DB, items, true); err != nil {
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
	return getTokenPromptBindings(userID, tokenID, true)
}

// GetTokenPromptBindingSummaries 只装配安装目录，避免用户查询触发全包读取。
func GetTokenPromptBindingSummaries(userID, tokenID int) ([]TokenPromptInstallation, error) {
	return getTokenPromptBindings(userID, tokenID, false)
}

func getTokenPromptBindings(userID, tokenID int, includeContent bool) ([]TokenPromptInstallation, error) {
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
		if err := hydratePromptTemplateGroups(DB, templates, includeContent); err != nil {
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
		// 不限分组的模板只要求令牌当前至少有一个可用的启用分组。
		for _, template := range templates {
			if template.AllGroups && len(allowedGroupIDs) > 0 {
				eligible[template.ID] = true
			}
		}
		total := 0
		for _, input := range inputs {
			template := byID[input.TemplateID]
			if (!template.Enabled || !eligible[input.TemplateID]) && (!installed[input.TemplateID] || input.Enabled) {
				return errors.New("提示词已下架，或令牌可用分组不在装载白名单内")
			}
			if input.Enabled && template.LoadMode != PromptLoadModeProgressive {
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

// promptGroupAuthorization 是模板对当前实际请求分组生效的统一条件：白名单包含该分组，
// 或模板为不限分组且该分组在数据库中仍为启用状态。全文装载、渐进式目录与正文读取共用，
// 实时查库，不经缓存；白名单分支保持各调用方原有的分组状态检查。
func promptGroupAuthorization(tx *gorm.DB, templateIDColumn, allGroupsColumn string, groupID int) clause.Expr {
	activeGroup := tx.Model(&Group{}).Select("id").Where("id = ? AND status = ?", groupID, GroupStatusActive)
	whitelisted := tx.Table("prompt_template_groups g").Select("1").Where("g.template_id = "+templateIDColumn+" AND g.group_id = ?", groupID)
	return gorm.Expr("(("+allGroupsColumn+" = ? AND EXISTS (?)) OR EXISTS (?))", true, activeGroup, whitelisted)
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
		Joins("JOIN tokens t ON t.id = b.token_id AND t.deleted_at IS NULL").
		Where(promptGroupAuthorization(DB, "prompt_templates.id", "prompt_templates.all_groups", group.Id)).
		Where("b.token_id = ? AND b.enabled = ? AND prompt_templates.enabled = ? AND (prompt_templates.load_mode IS NULL OR prompt_templates.load_mode <> ?)", tokenID, true, true, PromptLoadModeProgressive).
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
