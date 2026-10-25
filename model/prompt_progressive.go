package model

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	PromptLoadModeEager       = "eager"
	PromptLoadModeProgressive = "progressive"
	MaxProgressiveSkills      = 128
	MaxSkillFileBytes         = 8 * 1024 * 1024
	MaxSkillPackageBytes      = 64 * 1024 * 1024
	MaxSkillReadBytes         = 24 * 1024
	skillChunkBytes           = 48 * 1024
	MaxSkillResourceFiles     = 4096
)

// PromptSkillRecord 只存目录元数据，发现技能时不读取正文。
type PromptSkillRecord struct {
	SkillID     string `gorm:"primaryKey;size:64"`
	TemplateID  int    `gorm:"index"`
	Title       string `gorm:"size:320"`
	Description string `gorm:"type:text"`
	Path        string `gorm:"type:text"`
	SortOrder   int
}

type PromptSkillFile struct {
	TemplateID int    `gorm:"primaryKey;autoIncrement:false"`
	SkillID    string `gorm:"primaryKey;size:64"`
	FileID     string `gorm:"primaryKey;size:64"`
	Path       string `gorm:"type:text"`
	TotalBytes int
}

// PromptSkillChunk 的 UTF-8 字节块小于 MySQL TEXT 上限，不依赖数据库专属大字段。
type PromptSkillChunk struct {
	TemplateID int    `gorm:"primaryKey;autoIncrement:false"`
	SkillID    string `gorm:"primaryKey;size:64"`
	FileID     string `gorm:"primaryKey;size:64"`
	ChunkIndex int    `gorm:"primaryKey;autoIncrement:false"`
	ByteOffset int
	Content    string `gorm:"type:text"`
}

type TokenProgressiveSkill struct {
	TemplateID   int    `json:"template_id"`
	SkillID      string `json:"skill_id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	PackageTitle string `json:"package_title"`
	Version      int    `json:"version"`
}

type SkillFilePage struct {
	Content    string   `json:"content"`
	Offset     int      `json:"offset"`
	NextOffset int      `json:"next_offset"`
	TotalBytes int      `json:"total_bytes"`
	EOF        bool     `json:"eof"`
	Resources  []string `json:"resources"`
}

func validSkillPath(value string) bool {
	return value != "" && value != "." && len(value) <= 1024 && fs.ValidPath(value) && utf8.ValidString(value) && !strings.ContainsAny(value, "\\:\x00")
}

func normalizeProgressiveSkills(template *PromptTemplate) error {
	if template.Kind == "" {
		template.Kind = "prompt"
	}
	switch template.Kind {
	case "prompt":
		var skill PromptSkill
		if len(template.Skills) > 1 {
			return errors.New("单技能只能包含一个技能")
		}
		if len(template.Skills) == 1 {
			skill = template.Skills[0]
		}
		skill.Title, skill.Description, skill.Content = template.Title, template.Description, template.Content
		template.Skills = []PromptSkill{skill}
	case "bundle":
		if len(template.Skills) < 2 || len(template.Skills) > MaxProgressiveSkills {
			return errors.New("渐进式技能合集必须包含 2 至 128 个技能")
		}
		template.Content = ""
	default:
		return errors.New("技能类型无效")
	}
	ids, paths := make(map[string]bool), make(map[string]bool)
	total, fileCount := 0, 0
	for i := range template.Skills {
		skill := &template.Skills[i]
		if skill.ID == "" {
			skill.ID = common.GetUUID()
		}
		if len(skill.ID) > 64 || ids[skill.ID] {
			return errors.New("技能身份无效或重复")
		}
		for _, char := range skill.ID {
			if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' && char != '_' {
				return errors.New("技能身份无效")
			}
		}
		ids[skill.ID] = true
		skill.Title, skill.Description = strings.TrimSpace(skill.Title), strings.TrimSpace(skill.Description)
		if !utf8.ValidString(skill.Title) || strings.ContainsRune(skill.Title, 0) || utf8.RuneCountInString(skill.Title) < 1 || utf8.RuneCountInString(skill.Title) > 80 {
			return errors.New("技能标题必须为 1 至 80 个字符的有效文本")
		}
		if !utf8.ValidString(skill.Description) || strings.ContainsRune(skill.Description, 0) || len(skill.Description) > MaxPromptDescriptionBytes {
			return fmt.Errorf("技能「%s」用途说明必须为有效文本，且不超过 %d KiB", skill.Title, MaxPromptDescriptionBytes/1024)
		}
		if template.Enabled && skill.Description == "" {
			return fmt.Errorf("技能「%s」发布前需要填写用途说明", skill.Title)
		}
		if skill.Path == "" {
			skill.Path = skill.ID + "/SKILL.md"
		}
		if !validSkillPath(skill.Path) || paths[skill.Path] {
			return errors.New("技能主文件路径无效或重复")
		}
		paths[skill.Path] = true
		if !utf8.ValidString(skill.Content) || strings.ContainsRune(skill.Content, 0) || strings.TrimSpace(skill.Content) == "" {
			return fmt.Errorf("技能「%s」正文必须为非空 UTF-8 文本", skill.Title)
		}
		if len(skill.Content) > MaxSkillFileBytes {
			return fmt.Errorf("技能「%s」正文超过 8 MiB 文件上限", skill.Title)
		}
		total += len(skill.Content)
		fileCount++
		resources := make(map[string]bool)
		for _, resource := range skill.Resources {
			if !validSkillPath(resource.Path) || strings.EqualFold(resource.Path, "SKILL.md") || resources[resource.Path] {
				return errors.New("技能引用资料路径无效、重复或覆盖主文件")
			}
			if !utf8.ValidString(resource.Content) || strings.ContainsRune(resource.Content, 0) || len(resource.Content) > MaxSkillFileBytes {
				return fmt.Errorf("技能引用资料 %q 必须为不超过 8 MiB 的 UTF-8 文本", resource.Path)
			}
			resources[resource.Path] = true
			total += len(resource.Content)
			fileCount++
			if total > MaxSkillPackageBytes || fileCount > MaxSkillResourceFiles {
				return errors.New("技能包文本总量不能超过 64 MiB 或 4096 个文件")
			}
		}
		if total > MaxSkillPackageBytes || fileCount > MaxSkillResourceFiles {
			return errors.New("技能包文本总量不能超过 64 MiB 或 4096 个文件")
		}
	}
	if template.Kind == "prompt" {
		template.Title, template.Description, template.Content = template.Skills[0].Title, template.Skills[0].Description, template.Skills[0].Content
	}
	template.SkillsJSON, template.SkillCount = "", len(template.Skills)
	return nil
}

func skillFileID(path string) string {
	digest := sha256.Sum256([]byte(path))
	return hex.EncodeToString(digest[:])
}

func persistProgressiveSkills(tx *gorm.DB, template *PromptTemplate) error {
	var foreign int64
	ids := make([]string, 0, len(template.Skills))
	if template.LoadMode == PromptLoadModeProgressive {
		for _, skill := range template.Skills {
			ids = append(ids, skill.ID)
		}
		if err := tx.Model(&PromptSkillRecord{}).Where("skill_id IN ? AND template_id <> ?", ids, template.ID).Count(&foreign).Error; err != nil {
			return err
		}
		if foreign > 0 {
			return errors.New("不能使用其他技能包的技能身份")
		}
	}
	for _, table := range []interface{}{&PromptSkillChunk{}, &PromptSkillFile{}, &PromptSkillRecord{}} {
		if err := tx.Where("template_id = ?", template.ID).Delete(table).Error; err != nil {
			return err
		}
	}
	if template.LoadMode != PromptLoadModeProgressive {
		return nil
	}
	for order, skill := range template.Skills {
		record := PromptSkillRecord{SkillID: skill.ID, TemplateID: template.ID, Title: skill.Title, Description: skill.Description, Path: skill.Path, SortOrder: order}
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		files := append([]PromptSkillResource{{Path: "SKILL.md", Content: skill.Content}}, skill.Resources...)
		for _, file := range files {
			fileID := skillFileID(file.Path)
			metadata := PromptSkillFile{TemplateID: template.ID, SkillID: skill.ID, FileID: fileID, Path: file.Path, TotalBytes: len(file.Content)}
			if err := tx.Create(&metadata).Error; err != nil {
				return err
			}
			for offset, index := 0, 0; offset < len(file.Content); index++ {
				end := min(offset+skillChunkBytes, len(file.Content))
				for end < len(file.Content) && !utf8.RuneStart(file.Content[end]) {
					end--
				}
				chunk := PromptSkillChunk{TemplateID: template.ID, SkillID: skill.ID, FileID: fileID, ChunkIndex: index, ByteOffset: offset, Content: file.Content[offset:end]}
				if err := tx.Create(&chunk).Error; err != nil {
					return err
				}
				offset = end
			}
		}
	}
	return nil
}

func hydrateProgressiveSkills(tx *gorm.DB, templates []PromptTemplate, includeContent bool) error {
	ids := make([]int, 0)
	for _, template := range templates {
		if template.LoadMode == PromptLoadModeProgressive && template.ID > 0 {
			ids = append(ids, template.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	var records []PromptSkillRecord
	if err := tx.Where("template_id IN ?", ids).Order("template_id, sort_order, skill_id").Find(&records).Error; err != nil {
		return err
	}
	if !includeContent {
		byTemplate := make(map[int][]PromptSkill)
		for _, record := range records {
			byTemplate[record.TemplateID] = append(byTemplate[record.TemplateID], PromptSkill{ID: record.SkillID, Title: record.Title, Description: record.Description, Path: record.Path})
		}
		for i := range templates {
			if templates[i].LoadMode != PromptLoadModeProgressive {
				continue
			}
			templates[i].Skills = byTemplate[templates[i].ID]
			templates[i].SkillCount = len(templates[i].Skills)
		}
		return nil
	}
	var files []PromptSkillFile
	if err := tx.Where("template_id IN ?", ids).Order("path").Find(&files).Error; err != nil {
		return err
	}
	var chunks []PromptSkillChunk
	if err := tx.Where("template_id IN ?", ids).Order("skill_id, file_id, chunk_index").Find(&chunks).Error; err != nil {
		return err
	}
	contents := make(map[string]*strings.Builder)
	for _, chunk := range chunks {
		key := chunk.SkillID + "/" + chunk.FileID
		if contents[key] == nil {
			contents[key] = &strings.Builder{}
		}
		if chunk.ByteOffset != contents[key].Len() || len(chunk.Content) == 0 || len(chunk.Content) > skillChunkBytes || !utf8.ValidString(chunk.Content) {
			return errors.New("技能文件分块不完整")
		}
		contents[key].WriteString(chunk.Content)
	}
	resources, bodies := make(map[string][]PromptSkillResource), make(map[string]string)
	for _, file := range files {
		body := ""
		if value := contents[file.SkillID+"/"+file.FileID]; value != nil {
			body = value.String()
		}
		if len(body) != file.TotalBytes {
			return errors.New("技能文件分块不完整")
		}
		if file.Path == "SKILL.md" {
			bodies[file.SkillID] = body
		} else {
			resources[file.SkillID] = append(resources[file.SkillID], PromptSkillResource{Path: file.Path, Content: body})
		}
	}
	byTemplate := make(map[int][]PromptSkill)
	for _, record := range records {
		skill := PromptSkill{ID: record.SkillID, Title: record.Title, Description: record.Description, Path: record.Path, Content: bodies[record.SkillID], Resources: resources[record.SkillID]}
		byTemplate[record.TemplateID] = append(byTemplate[record.TemplateID], skill)
	}
	for i := range templates {
		if templates[i].LoadMode != PromptLoadModeProgressive || templates[i].ID <= 0 {
			continue
		}
		templates[i].Skills = byTemplate[templates[i].ID]
		if templates[i].Kind == "prompt" && len(templates[i].Skills) == 1 {
			templates[i].Content = templates[i].Skills[0].Content
		}
	}
	return nil
}

func GetActiveTokenSkills(tokenID int, groupCode string) ([]TokenProgressiveSkill, error) {
	result := make([]TokenProgressiveSkill, 0)
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
	if group.Status != GroupStatusActive {
		return result, nil
	}
	err = DB.Table("prompt_skill_records s").Select("s.template_id, s.skill_id, s.title, s.description, p.title AS package_title, p.version").
		Joins("JOIN prompt_templates p ON p.id = s.template_id").
		Joins("JOIN token_prompt_bindings b ON b.template_id = p.id").
		Joins("JOIN prompt_template_groups g ON g.template_id = p.id").
		Joins("JOIN tokens t ON t.id = b.token_id AND t.deleted_at IS NULL").
		Where("g.group_id IN (?)", DB.Model(&Group{}).Select("id").Where("status = ?", GroupStatusActive)).
		Where("b.token_id = ? AND b.enabled = ? AND p.enabled = ? AND p.load_mode = ? AND g.group_id = ? AND t.status = ? AND (t.expired_time = -1 OR t.expired_time >= ?)", tokenID, true, true, PromptLoadModeProgressive, group.Id, common.TokenStatusEnabled, common.GetTimestamp()).
		Order("b.sort_order, p.id, s.sort_order, s.skill_id").Scan(&result).Error
	return result, err
}

// ReadActiveTokenSkillFile 只按当前实际分组读取已安装版本；每次都重新检查撤销与版本。
func ReadActiveTokenSkillFile(tokenID int, groupCode string, templateID int, skillID string, version int, path string, offset int, limit int) (*SkillFilePage, error) {
	if path == "" {
		path = "SKILL.md"
	}
	if !validSkillPath(path) || tokenID <= 0 || templateID <= 0 || skillID == "" || version <= 0 || offset < 0 || limit <= 0 || limit > MaxSkillReadBytes {
		return nil, errors.New("技能读取参数无效")
	}
	group, err := GetGroupByCodeOrAlias(groupCode)
	if err != nil || groupCode == "auto" || groupCode == "" || group.Status != GroupStatusActive {
		return nil, errors.New("当前分组无权读取技能")
	}
	database := DB
	pageKey := progressiveSkillCacheKey{TemplateID: templateID, Version: version, SkillID: skillID, Path: path, Offset: offset}
	directoryKey := progressiveSkillCacheKey{TemplateID: templateID, Version: version, SkillID: skillID}
	var loadedPage *SkillFilePage
	loadedDirectory := false
	page := &SkillFilePage{Resources: []string{}}
	err = database.Transaction(func(tx *gorm.DB) error {
		var token Token
		if err := lockForShare(tx).Select("id", "status", "expired_time").First(&token, tokenID).Error; err != nil || token.Status != common.TokenStatusEnabled || (token.ExpiredTime != -1 && token.ExpiredTime < common.GetTimestamp()) {
			return errors.New("令牌已停用、过期或删除")
		}
		var template PromptTemplate
		if err := lockForShare(tx).Select("id", "enabled", "load_mode", "version").First(&template, templateID).Error; err != nil {
			return errors.New("技能包不存在")
		}
		if !template.Enabled || template.LoadMode != PromptLoadModeProgressive || template.Version != version {
			return errors.New("技能已下架或版本已更新，请重新发现技能目录")
		}
		var authorized int64
		if err := tx.Table("token_prompt_bindings b").Joins("JOIN prompt_template_groups g ON g.template_id = b.template_id").Where("g.group_id IN (?)", tx.Model(&Group{}).Select("id").Where("status = ?", GroupStatusActive)).Where("b.token_id = ? AND b.template_id = ? AND b.enabled = ? AND g.group_id = ?", tokenID, templateID, true, group.Id).Count(&authorized).Error; err != nil {
			return err
		}
		if authorized != 1 {
			return errors.New("当前令牌或分组无权读取技能")
		}
		cached, found := progressiveSkillContents.get(database, pageKey)
		if !found {
			var file PromptSkillFile
			fileErr := tx.Table("prompt_skill_files f").Select("f.*").
				Joins("JOIN prompt_skill_records s ON s.template_id = f.template_id AND s.skill_id = f.skill_id").
				Where("f.template_id = ? AND f.skill_id = ? AND f.file_id = ?", templateID, skillID, skillFileID(path)).First(&file).Error
			if fileErr != nil || file.Path != path {
				var records int64
				if err := tx.Model(&PromptSkillRecord{}).Where("template_id = ? AND skill_id = ?", templateID, skillID).Count(&records).Error; err != nil {
					return err
				}
				if records != 1 {
					return errors.New("技能不存在于当前安装包")
				}
				return errors.New("技能文件不存在")
			}
			if file.TotalBytes < 0 || file.TotalBytes > MaxSkillFileBytes {
				return errors.New("技能文件分块不完整")
			}
			if offset > file.TotalBytes {
				return errors.New("技能读取偏移超过文件长度")
			}
			// 以固定最大页读取，正式生成以实际页长复核时仍可复用同一正文缓存。
			end := min(offset+MaxSkillReadBytes, file.TotalBytes)
			var chunks []PromptSkillChunk
			if err := tx.Where("template_id = ? AND skill_id = ? AND file_id = ? AND byte_offset < ? AND byte_offset >= ?", templateID, skillID, file.FileID, end, max(0, offset-skillChunkBytes)).Order("chunk_index").Find(&chunks).Error; err != nil {
				return err
			}
			content, start := "", offset
			if len(chunks) > 0 {
				start = chunks[0].ByteOffset
				var builder strings.Builder
				for _, chunk := range chunks {
					if chunk.ByteOffset < 0 || chunk.ByteOffset != start+builder.Len() || len(chunk.Content) == 0 || len(chunk.Content) > skillChunkBytes || !utf8.ValidString(chunk.Content) {
						return errors.New("技能文件分块不完整")
					}
					builder.WriteString(chunk.Content)
				}
				content = builder.String()
			}
			if start > offset || offset-start > len(content) || end-start > len(content) || len(content) > file.TotalBytes-start {
				return errors.New("技能文件分块不完整")
			}
			if offset-start < len(content) && !utf8.RuneStart(content[offset-start]) {
				return errors.New("技能读取偏移不在 UTF-8 字符边界")
			}
			for end > offset && end-start < len(content) && !utf8.RuneStart(content[end-start]) {
				end--
			}
			cached = SkillFilePage{Content: content[offset-start : end-start], Offset: offset, NextOffset: end, TotalBytes: file.TotalBytes, EOF: end == file.TotalBytes}
			loadedPage = &cached
		}
		*page = cached
		page.Resources = []string{}
		end := min(offset+limit, page.NextOffset)
		for end > offset && end-offset < len(page.Content) && !utf8.RuneStart(page.Content[end-offset]) {
			end--
		}
		if end == offset && offset < page.TotalBytes {
			return errors.New("技能读取长度不足一个 UTF-8 字符")
		}
		page.Content, page.NextOffset, page.EOF = page.Content[:end-offset], end, end == page.TotalBytes
		if path == "SKILL.md" {
			if directory, found := progressiveSkillContents.get(database, directoryKey); found {
				page.Resources = directory.Resources
			} else {
				if err := tx.Model(&PromptSkillFile{}).Where("template_id = ? AND skill_id = ? AND path <> ?", templateID, skillID, "SKILL.md").Pluck("path", &page.Resources).Error; err != nil {
					return err
				}
				sort.Strings(page.Resources)
				loadedDirectory = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// 事务成功后才发布缓存；版本化键允许旧读取与后续更新安全交错完成。
	if loadedPage != nil {
		progressiveSkillContents.put(database, pageKey, *loadedPage)
	}
	if loadedDirectory {
		progressiveSkillContents.put(database, directoryKey, SkillFilePage{Resources: page.Resources})
	}
	return page, nil
}
