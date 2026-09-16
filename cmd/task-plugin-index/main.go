// task-plugin-index 为经维护者审核的本地源码生成兼容官方格式的插件源索引。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
)

type indexVersion struct {
	Version       string `json:"version"`
	Path          string `json:"path"`
	SHA256        string `json:"sha256"`
	Kind          string `json:"kind"`
	MinAPIVersion int    `json:"minApiVersion"`
}

type indexPlugin struct {
	Key         string                 `json:"key"`
	Name        string                 `json:"name"`
	Description jsplugin.LocalizedText `json:"description,omitempty"`
	Models      []string               `json:"models"`
	Latest      string                 `json:"latest"`
	Versions    []indexVersion         `json:"versions"`
}

type pluginIndex struct {
	IndexVersion int           `json:"indexVersion"`
	Name         string        `json:"name"`
	Plugins      []indexPlugin `json:"plugins"`
}

func main() {
	root := flag.String("root", "", "独立插件仓库目录（必填）")
	check := flag.Bool("check", false, "仅验证已提交索引")
	flag.Parse()
	if strings.TrimSpace(*root) == "" {
		fmt.Fprintln(os.Stderr, "请通过 -root 指定独立插件仓库目录")
		os.Exit(2)
	}
	if err := generateIndex(*root, *check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generateIndex(root string, check bool) error {
	indexPath := filepath.Join(root, "index.json")
	oldData, err := os.ReadFile(indexPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	old := pluginIndex{}
	if len(oldData) > 0 {
		if err := common.Unmarshal(oldData, &old); err != nil {
			return fmt.Errorf("现有索引无效: %w", err)
		}
		if old.IndexVersion != 1 {
			return fmt.Errorf("不支持现有索引版本")
		}
	}
	previous := make(map[string]indexPlugin)
	for _, plugin := range old.Plugins {
		previous[plugin.Key] = plugin
	}
	entries, err := os.ReadDir(filepath.Join(root, "published"))
	if err != nil {
		return err
	}
	index := pluginIndex{IndexVersion: 1, Name: "MaoLao Maintained", Plugins: []indexPlugin{}}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		key := entry.Name()
		if !jsplugin.ValidPluginKey(key) {
			return fmt.Errorf("插件目录 key 无效: %s", key)
		}
		versions, err := os.ReadDir(filepath.Join(root, "published", key))
		if err != nil {
			return err
		}
		plugin := indexPlugin{Key: key, Versions: []indexVersion{}}
		metadata := make(map[string]jsplugin.Meta)
		for _, version := range versions {
			if !version.IsDir() {
				continue
			}
			rel := "published/" + key + "/" + version.Name() + "/plugin.js"
			file := filepath.Join(root, filepath.FromSlash(rel))
			info, err := os.Lstat(file)
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() || info.Size() > 1<<20 {
				return fmt.Errorf("源码必须是 1 MiB 以内普通文件: %s", rel)
			}
			source, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			if !utf8.Valid(source) {
				return fmt.Errorf("源码必须为 UTF-8: %s", rel)
			}
			loaded, err := jsplugin.CompilePlugin(string(source), jsplugin.Options{Log: func(string) {}})
			if err != nil {
				return fmt.Errorf("插件编译校验失败: %s", rel)
			}
			if loaded.Meta.Key != key || loaded.Meta.Version != version.Name() {
				return fmt.Errorf("源码 key/version 与目录不一致: %s", rel)
			}
			hash := fmt.Sprintf("%x", sha256.Sum256(source))
			metadata[version.Name()] = loaded.Meta
			plugin.Versions = append(plugin.Versions, indexVersion{Version: version.Name(), Path: rel, SHA256: hash, Kind: "task", MinAPIVersion: loaded.Meta.APIVersion})
		}
		if len(plugin.Versions) == 0 {
			return fmt.Errorf("插件目录没有可发布版本: %s", key)
		}
		// 保留已有 latest，由维护者通过索引明确选择升级版本，禁止按字符串排序猜测 semver。
		plugin.Latest = previous[key].Latest
		if plugin.Latest == "" && len(plugin.Versions) == 1 {
			plugin.Latest = plugin.Versions[0].Version
		}
		meta, exists := metadata[plugin.Latest]
		if !exists {
			return fmt.Errorf("请在索引中为 %s 指定现有版本 latest", key)
		}
		plugin.Name, plugin.Description, plugin.Models = meta.Name, meta.Description, meta.Models
		for _, existing := range previous[key].Versions {
			found := false
			for _, current := range plugin.Versions {
				if existing.Version != current.Version {
					continue
				}
				found = true
				if !strings.EqualFold(existing.SHA256, current.SHA256) || existing.Path != current.Path {
					return fmt.Errorf("已发布版本不可修改: %s@%s", key, current.Version)
				}
			}
			if !found {
				return fmt.Errorf("已发布版本不可删除: %s@%s", key, existing.Version)
			}
		}
		delete(previous, key)
		index.Plugins = append(index.Plugins, plugin)
	}
	if len(previous) > 0 {
		return fmt.Errorf("已发布插件不可移除")
	}
	sort.Slice(index.Plugins, func(i, j int) bool { return index.Plugins[i].Key < index.Plugins[j].Key })
	data, err := common.Marshal(index)
	if err != nil {
		return err
	}
	// 使用宿主 JSON 包生成数据；缩进仅用于维护者审阅索引。
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, data, "", "  "); err != nil {
		return err
	}
	formatted.WriteByte('\n')
	if check {
		if !bytes.Equal(oldData, formatted.Bytes()) {
			return fmt.Errorf("索引未同步，请运行 go run ./cmd/task-plugin-index")
		}
		fmt.Printf("插件源索引验证通过：%d 个插件\n", len(index.Plugins))
		return nil
	}
	return os.WriteFile(indexPath, formatted.Bytes(), 0644)
}
