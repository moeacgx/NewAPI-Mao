package model

import (
	"errors"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"
)

type PromptCategory struct {
	ID      int    `json:"id" gorm:"primaryKey"`
	Name    string `json:"name" gorm:"size:160"`
	NameKey string `json:"-" gorm:"size:160;uniqueIndex"`
	Version int    `json:"version"`
}

func ListPromptCategories() ([]PromptCategory, error) {
	items := make([]PromptCategory, 0)
	err := DB.Order("id").Find(&items).Error
	return items, err
}

func SavePromptCategory(input PromptCategory) (*PromptCategory, error) {
	input.Name = strings.TrimSpace(input.Name)
	if !utf8.ValidString(input.Name) || utf8.RuneCountInString(input.Name) < 1 || utf8.RuneCountInString(input.Name) > 40 {
		return nil, errors.New("技能分类名称必须为 1 至 40 个字符")
	}
	input.NameKey = strings.ToLower(input.Name)
	err := DB.Transaction(func(tx *gorm.DB) error {
		var stored PromptCategory
		if input.ID > 0 {
			if err := lockForUpdate(tx).First(&stored, input.ID).Error; err != nil {
				return err
			}
			if input.Version != stored.Version {
				return ErrPromptVersionConflict
			}
		}
		var duplicate int64
		if err := tx.Model(&PromptCategory{}).Where("name_key = ? AND id <> ?", input.NameKey, input.ID).Count(&duplicate).Error; err != nil {
			return err
		}
		if duplicate > 0 {
			return errors.New("技能分类名称已存在")
		}
		if input.ID <= 0 {
			input.ID = 0
			input.Version = 1
			return tx.Create(&input).Error
		}
		input.Version = stored.Version + 1
		result := tx.Model(&PromptCategory{}).Where("id = ? AND version = ?", input.ID, stored.Version).Updates(map[string]interface{}{"name": input.Name, "name_key": input.NameKey, "version": input.Version})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrPromptVersionConflict
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &input, nil
}
