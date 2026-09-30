package model

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// GroupTag 仅管理展示分类，不参与分组鉴权、路由或计费。
type GroupTag struct {
	Id          int      `json:"id"`
	Name        string   `json:"name" gorm:"size:64;not null"`
	Description string   `json:"description" gorm:"type:text"`
	IconsJSON   string   `json:"-" gorm:"column:icons;type:text"`
	SortOrder   int      `json:"sort_order"`
	Icons       []string `json:"icons" gorm:"-"`
	GroupIDs    []int    `json:"group_ids" gorm:"-"`
}

type GroupTagBinding struct {
	TagID   int `gorm:"primaryKey;autoIncrement:false"`
	GroupID int `gorm:"primaryKey;autoIncrement:false;index"`
}

var groupTagIconName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*(\.[A-Za-z][A-Za-z0-9]*)*$`)

func normalizeGroupTag(input GroupTag) (GroupTag, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	if input.Name == "" || utf8.RuneCountInString(input.Name) > 64 {
		return input, errors.New("标签名称不能为空且不能超过 64 个字符")
	}
	if utf8.RuneCountInString(input.Description) > 512 || input.SortOrder < -1000000 || input.SortOrder > 1000000 {
		return input, errors.New("标签说明或排序超出限制")
	}
	if len(input.Icons) > 6 || len(input.GroupIDs) > 10000 {
		return input, errors.New("标签最多配置 6 个 Logo 和 10000 个分组")
	}
	icons := make([]string, 0, len(input.Icons))
	seenIcons := make(map[string]bool)
	for _, icon := range input.Icons {
		icon = strings.TrimSpace(icon)
		if icon == "" || len(icon) > 2048 {
			return input, errors.New("Logo 不能为空且不能超过 2048 字节")
		}
		if !groupTagIconName.MatchString(icon) {
			parsed, err := url.Parse(icon)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil || strings.ContainsAny(icon, "\r\n\t ") {
				return input, errors.New("Logo 必须为内置图标名称或无凭据的 HTTP/HTTPS 图片地址")
			}
		}
		if !seenIcons[icon] {
			icons = append(icons, icon)
			seenIcons[icon] = true
		}
	}
	for _, id := range input.GroupIDs {
		if id <= 0 {
			return input, errors.New("标签只能绑定已保存的实体分组")
		}
	}
	input.Icons = icons
	input.GroupIDs = uniquePositiveGroupIDs(input.GroupIDs)
	sort.Ints(input.GroupIDs)
	encoded, err := common.Marshal(icons)
	input.IconsJSON = string(encoded)
	return input, err
}

// GetGroupTags 批量读取标签及现存分组关联。非 nil allowedIDs 将返回值裁剪到可用分组。
func GetGroupTags(allowedIDs []int) ([]GroupTag, error) {
	tags := make([]GroupTag, 0)
	if allowedIDs != nil && len(allowedIDs) == 0 {
		return tags, nil
	}
	query := DB.Model(&GroupTagBinding{}).
		Where("group_id IN (?)", DB.Model(&Group{}).Select("id"))
	if allowedIDs != nil {
		query = query.Where("group_tag_bindings.group_id IN ?", allowedIDs)
	}
	var bindings []GroupTagBinding
	if err := query.Select("group_tag_bindings.*").Order("group_tag_bindings.group_id ASC").Find(&bindings).Error; err != nil {
		return nil, err
	}
	idsByTag := make(map[int][]int)
	for _, binding := range bindings {
		idsByTag[binding.TagID] = append(idsByTag[binding.TagID], binding.GroupID)
	}
	tagQuery := DB.Order("sort_order ASC, id ASC")
	if allowedIDs != nil {
		ids := make([]int, 0, len(idsByTag))
		for id := range idsByTag {
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			return tags, nil
		}
		tagQuery = tagQuery.Where("id IN ?", ids)
	}
	if err := tagQuery.Find(&tags).Error; err != nil {
		return nil, err
	}
	for index := range tags {
		tag := &tags[index]
		tag.Icons = []string{}
		if err := common.UnmarshalJsonStr(tag.IconsJSON, &tag.Icons); err != nil {
			return nil, fmt.Errorf("读取标签 Logo 失败: %w", err)
		}
		tag.GroupIDs = idsByTag[tag.Id]
		if tag.GroupIDs == nil {
			tag.GroupIDs = []int{}
		}
	}
	return tags, nil
}

// SaveGroupTag 将标签和关联作为一个事务保存，旧分组配置接口无需传递分类字段。
func SaveGroupTag(input GroupTag) (*GroupTag, error) {
	if input.Id < 0 {
		return nil, errors.New("标签 ID 无效")
	}
	tag, err := normalizeGroupTag(input)
	if err != nil {
		return nil, err
	}
	optionWriteMutex.Lock()
	defer optionWriteMutex.Unlock()
	err = DB.Transaction(func(tx *gorm.DB) error {
		// 与分组删除保持一致：先锁实体分组，再写其展示关联。
		if len(tag.GroupIDs) > 0 {
			var groups []Group
			if err := lockForUpdate(tx).Where("id IN ?", tag.GroupIDs).Order("id ASC").Find(&groups).Error; err != nil {
				return err
			}
			if len(groups) != len(tag.GroupIDs) {
				return errors.New("绑定分组不存在，请刷新分组列表")
			}
			for _, group := range groups {
				if isVirtualAutoCode(group.Code) {
					return errors.New("自动选择不能绑定标签")
				}
			}
		}
		if tag.Id == 0 {
			if err := tx.Create(&tag).Error; err != nil {
				return err
			}
		} else {
			var existing GroupTag
			if err := lockForUpdate(tx).First(&existing, tag.Id).Error; err != nil {
				return err
			}
			if err := tx.Model(&existing).Updates(map[string]interface{}{
				"name": tag.Name, "description": tag.Description, "icons": tag.IconsJSON, "sort_order": tag.SortOrder,
			}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("tag_id = ?", tag.Id).Delete(&GroupTagBinding{}).Error; err != nil {
			return err
		}
		if len(tag.GroupIDs) == 0 {
			return nil
		}
		bindings := make([]GroupTagBinding, 0, len(tag.GroupIDs))
		for _, id := range tag.GroupIDs {
			bindings = append(bindings, GroupTagBinding{TagID: tag.Id, GroupID: id})
		}
		return tx.CreateInBatches(&bindings, 200).Error
	})
	return &tag, err
}

func DeleteGroupTag(id int) error {
	if id <= 0 {
		return errors.New("标签 ID 无效")
	}
	optionWriteMutex.Lock()
	defer optionWriteMutex.Unlock()
	return DB.Transaction(func(tx *gorm.DB) error {
		var tag GroupTag
		if err := lockForUpdate(tx).First(&tag, id).Error; err != nil {
			return err
		}
		if err := tx.Where("tag_id = ?", id).Delete(&GroupTagBinding{}).Error; err != nil {
			return err
		}
		return tx.Delete(&tag).Error
	})
}
