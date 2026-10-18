package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishedPluginIndexPreservesVersions(t *testing.T) {
	root := t.TempDir()
	source := `export const meta = {apiVersion:1,key:"demo",name:"示例",version:"1.0.0",author:{name:"Maintainer"},models:["demo"],fetchMode:"per_task"};
export function buildSubmitRequest() { return {}; }
export function parseSubmitResponse() { return {}; }
export function buildQueryRequest() { return {}; }
export function parseTaskResult() { return {}; }
`
	file := filepath.Join(root, "published", "demo", "1.0.0", "plugin.js")
	require.NoError(t, os.MkdirAll(filepath.Dir(file), 0755))
	require.NoError(t, os.WriteFile(file, []byte(source), 0644))
	require.NoError(t, generateIndex(root, false))
	require.NoError(t, generateIndex(root, true))
	data, err := os.ReadFile(filepath.Join(root, "index.json"))
	require.NoError(t, err)
	var index pluginIndex
	require.NoError(t, common.Unmarshal(data, &index))
	require.Len(t, index.Plugins, 1)
	assert.Equal(t, "示例", index.Plugins[0].Name)
	assert.Equal(t, fmt.Sprintf("%x", sha256.Sum256([]byte(source))), index.Plugins[0].Versions[0].SHA256)
	assert.Equal(t, "published/demo/1.0.0/plugin.js", index.Plugins[0].Versions[0].Path)

	// 同版本改写和删除都会破坏用户从历史索引重装的能力，必须拒绝。
	require.NoError(t, os.WriteFile(file, []byte(source+"\n// changed"), 0644))
	assert.ErrorContains(t, generateIndex(root, false), "不可修改")
	require.NoError(t, os.WriteFile(file, []byte(source), 0644))
	newFile := filepath.Join(root, "published", "demo", "2.0.0", "plugin.js")
	require.NoError(t, os.MkdirAll(filepath.Dir(newFile), 0755))
	require.NoError(t, os.WriteFile(newFile, []byte(strings.Replace(source, "1.0.0", "2.0.0", 1)), 0644))
	assert.ErrorContains(t, generateIndex(root, true), "未同步")
	require.NoError(t, generateIndex(root, false))
	data, err = os.ReadFile(filepath.Join(root, "index.json"))
	require.NoError(t, err)
	require.NoError(t, common.Unmarshal(data, &index))
	assert.Equal(t, "1.0.0", index.Plugins[0].Latest)
	require.Len(t, index.Plugins[0].Versions, 2)
	require.NoError(t, os.Remove(file))
	require.Error(t, generateIndex(root, false))
}

func TestPublishedPluginIndexRejectsMismatchedIdentity(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "published", "demo", "1.0.0", "plugin.js")
	require.NoError(t, os.MkdirAll(filepath.Dir(file), 0755))
	source := `export const meta={apiVersion:1,key:"another",name:"Test",version:"1.0.0",author:{name:"Test"},models:["test"],fetchMode:"per_task"};
export function buildSubmitRequest(){} export function parseSubmitResponse(){} export function buildQueryRequest(){} export function parseTaskResult(){}`
	require.NoError(t, os.WriteFile(file, []byte(source), 0644))
	assert.ErrorContains(t, generateIndex(root, false), "目录不一致")
}
