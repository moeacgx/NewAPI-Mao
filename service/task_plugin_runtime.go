package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/plugins"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BatchTaskResult struct {
	relaycommon.TaskInfo
	Action                            string
	SubmitTime, StartTime, FinishTime int64
	Data                              any
}

var archivedTaskPlugins sync.Map

// InitTaskPlugins 归档内置源码，不覆盖已有版本、启停状态和原生渠道。
func InitTaskPlugins() error {
	for _, meta := range jsplugin.DefaultRegistry.Snapshot().Factory {
		source, err := plugins.Source(meta.Key)
		if err != nil {
			return err
		}
		hash := sha256.Sum256([]byte(source))
		row := model.TaskPlugin{Key: meta.Key, Version: meta.Version, APIVersion: meta.APIVersion, Source: source, SourceHash: hex.EncodeToString(hash[:]), SourceKind: "builtin", CreatedAt: time.Now().Unix()}
		if err := model.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
			return err
		}
		stored, err := model.GetTaskPluginVersion(meta.Key, meta.Version)
		if err != nil {
			return err
		}
		if stored.SourceHash != row.SourceHash {
			return fmt.Errorf("内置插件 %s 版本源码冲突", meta.Key)
		}
	}
	return nil
}

func TaskPluginsEnabled() (bool, error) {
	var option model.Option
	err := model.DB.Where(&model.Option{Key: setting.TaskPluginEnabledKey}).First(&option).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return option.Value == "true" && err == nil, err
}

// LoadPinnedTaskPlugin 不查询当前版本或启停状态，禁止静默回落到当前源码。
func LoadPinnedTaskPlugin(pin *model.TaskPluginSnapshot) (*jsplugin.LoadedPlugin, error) {
	if pin == nil || pin.SourceHash == "" || (pin.SourceKind != "builtin" && pin.SourceKind != "custom") {
		return nil, fmt.Errorf("插件缺少可信版本快照")
	}
	row, err := model.GetTaskPluginVersion(pin.Key, pin.Version)
	if err != nil {
		return nil, err
	}
	rowKind := row.SourceKind
	if rowKind == "" {
		rowKind = "builtin"
	}
	if rowKind != pin.SourceKind {
		return nil, fmt.Errorf("插件来源类型不匹配")
	}
	hash := sha256.Sum256([]byte(row.Source))
	if row.SourceHash != pin.SourceHash || hex.EncodeToString(hash[:]) != pin.SourceHash || row.APIVersion != pin.APIVersion {
		return nil, fmt.Errorf("插件历史源码校验失败")
	}
	if cached, ok := archivedTaskPlugins.Load(pin.SourceHash); ok {
		return cached.(*jsplugin.LoadedPlugin), nil
	}
	loaded, err := jsplugin.CompilePlugin(row.Source, jsplugin.Options{Key: pin.Key, Version: pin.Version, Log: func(string) {}})
	if err != nil {
		return nil, err
	}
	if loaded.Meta.Key != pin.Key || loaded.Meta.Version != pin.Version {
		return nil, fmt.Errorf("插件版本元数据不匹配")
	}
	actual, _ := archivedTaskPlugins.LoadOrStore(pin.SourceHash, loaded)
	return actual.(*jsplugin.LoadedPlugin), nil
}

func ActiveTaskPlugin(key string) (*jsplugin.LoadedPlugin, *model.TaskPluginSnapshot, error) {
	enabled, err := TaskPluginsEnabled()
	if err != nil {
		return nil, nil, err
	}
	if !enabled {
		return nil, nil, fmt.Errorf("官方任务插件已关闭")
	}
	row, err := model.GetTaskPluginVersion(key, "")
	if err != nil {
		return nil, nil, err
	}
	if !row.Active || !row.Enabled {
		return nil, nil, fmt.Errorf("任务插件未激活或已停用")
	}
	pin := &model.TaskPluginSnapshot{Key: row.Key, Version: row.Version, APIVersion: row.APIVersion, SourceHash: row.SourceHash, SourceKind: row.SourceKind}
	loaded, err := LoadPinnedTaskPlugin(pin)
	if err != nil {
		return nil, nil, err
	}
	pin.Name = loaded.Meta.Name
	return loaded, pin, nil
}
