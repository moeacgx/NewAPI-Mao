package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"gorm.io/gorm"
)

// GetTaskPluginMarketplaceSources reads the database on every call so all nodes
// observe updates immediately. A missing option uses the built-in defaults;
// an explicitly persisted empty array disables all marketplace sources.
func GetTaskPluginMarketplaceSources() ([]setting.TaskPluginMarketplaceSource, error) {
	var option Option
	err := DB.Where(&Option{Key: setting.TaskPluginMarketplaceSourcesKey}).First(&option).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return append([]setting.TaskPluginMarketplaceSource(nil), setting.DefaultTaskPluginMarketplaceSources...), nil
	}
	if err != nil {
		return nil, err
	}
	var sources []setting.TaskPluginMarketplaceSource
	if err := common.UnmarshalJsonStr(option.Value, &sources); err != nil {
		return nil, err
	}
	if sources == nil {
		sources = []setting.TaskPluginMarketplaceSource{}
	}
	if err := setting.ValidateTaskPluginMarketplaceSources(sources); err != nil {
		return nil, err
	}
	return sources, nil
}

// SaveTaskPluginMarketplaceSources validates and atomically persists the source
// list. The empty list is meaningful and is stored explicitly.
func SaveTaskPluginMarketplaceSources(sources []setting.TaskPluginMarketplaceSource) error {
	if err := setting.ValidateTaskPluginMarketplaceSources(sources); err != nil {
		return err
	}
	payload, err := common.Marshal(sources)
	if err != nil {
		return err
	}
	if err := UpdateOption(setting.TaskPluginMarketplaceSourcesKey, string(payload)); err != nil {
		return fmt.Errorf("save marketplace sources: %w", err)
	}
	return nil
}
