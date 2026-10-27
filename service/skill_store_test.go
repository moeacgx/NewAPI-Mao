package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type skillStoreTransport func(*http.Request) (*http.Response, error)

func (transport skillStoreTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestSkillStoreBundledCatalogImportsIndependentDrafts(t *testing.T) {
	items, err := ListSkillStore(context.Background(), false)
	require.NoError(t, err)
	require.NotEmpty(t, items)
	for _, item := range items {
		draft, err := ImportStoreSkill(context.Background(), item.ID)
		require.NoError(t, err, item.ID)
		assert.Equal(t, item.Kind, draft.Kind)
		assert.Equal(t, item.Title, draft.Title)
		assert.False(t, draft.Enabled)
		assert.Zero(t, draft.CategoryID)
		assert.Empty(t, draft.GroupIDs)
		assert.Equal(t, item.SkillCount, draft.SkillCount)
		assert.Equal(t, model.PromptLoadModeProgressive, draft.LoadMode)
		require.NotEmpty(t, draft.Skills)
		assert.NotEmpty(t, draft.Skills[0].Content)
		// 修改导入副本不会修改仓库中的原件。
		draft.Skills[0].Content = "站点自定义"
		next, err := ImportStoreSkill(context.Background(), item.ID)
		require.NoError(t, err)
		assert.NotEqual(t, draft.Skills[0].Content, next.Skills[0].Content)
	}
}

func TestSkillStoreBundledCatalogPublishesGeneralDevelopmentSafetySkill(t *testing.T) {
	items, err := ListSkillStore(context.Background(), false)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "bundled:general-development-safety", items[0].ID)
	assert.Equal(t, "通用编程开发安全助手", items[0].Title)
	assert.Equal(t, "prompt", items[0].Kind)
	assert.Equal(t, "开发编程", items[0].Category)
	assert.Equal(t, 1, items[0].SkillCount)
	draft, err := ImportStoreSkill(context.Background(), items[0].ID)
	require.NoError(t, err)
	require.Len(t, draft.Skills, 1)
	assert.Contains(t, draft.Skills[0].Content, "当前任务明确要求执行它")
	assert.Contains(t, draft.Skills[0].Content, "至少三轮独立核验")

	for _, oldID := range []string{"bundled:code-review", "bundled:writing-kit", "bundled:research-kit"} {
		_, err := ImportStoreSkill(context.Background(), oldID)
		require.Error(t, err, "已移除的商店条目不可再导入：%s", oldID)
	}
}

func TestSkillStoreOnlineUsesFixedOriginAndChecksFileHash(t *testing.T) {
	content := "---\nname: 在线技能\n---\n保持原意。"
	digest := sha256.Sum256([]byte(content))
	manifest := map[string]interface{}{"version": 1, "items": []map[string]interface{}{{
		"id": "online-example", "title": "在线技能", "description": "在线目录示例", "kind": "prompt", "category": "写作", "version": "1.0.0",
		"files": []map[string]string{{"path": "packages/example/SKILL.md", "sha256": hex.EncodeToString(digest[:])}},
	}}}
	manifestBytes, err := common.Marshal(manifest)
	require.NoError(t, err)
	servedContent := content
	client := &http.Client{Transport: skillStoreTransport(func(request *http.Request) (*http.Response, error) {
		assert.Equal(t, "https", request.URL.Scheme)
		assert.Equal(t, "raw.githubusercontent.com", request.URL.Host)
		assert.True(t, strings.HasPrefix(request.URL.Path, "/moeacgx/NewAPI-Mao/custom-main/skill-store/"))
		body := servedContent
		if strings.HasSuffix(request.URL.Path, "/index.json") {
			body = string(manifestBytes)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	items, err := loadSkillStoreCatalog(context.Background(), "online", client)
	require.NoError(t, err)
	require.Len(t, items, 1)
	draft, err := importStoreSkill(context.Background(), "online:online-example", client)
	require.NoError(t, err)
	assert.Equal(t, "保持原意。", draft.Content)
	servedContent = "被替换的正文"
	_, err = importStoreSkill(context.Background(), "online:online-example", client)
	require.ErrorContains(t, err, "校验")
}

func TestSkillStoreOnlineKeepsReferencedFilesWithinTheirSkill(t *testing.T) {
	files := map[string]string{
		"packages/demo/review/SKILL.md":              "---\nname: 代码审查\ndescription: 审查代码时使用\n---\n读取 references/rules.json。",
		"packages/demo/review/references/rules.json": `{"rule":"可维护性"}`,
		"packages/demo/writing/SKILL.md":             "---\nname: 写作\ndescription: 写作时使用\n---\n参考 examples/style.txt。",
		"packages/demo/writing/examples/style.txt":   "使用自然的语言。",
	}
	record := skillStoreRecord{ID: "with-resources", Title: "研发合集", Description: "独立技能与引用资料", Kind: "bundle", Version: "1"}
	for filename, body := range files {
		digest := sha256.Sum256([]byte(body))
		record.Files = append(record.Files, skillStoreFile{Path: filename, SHA256: hex.EncodeToString(digest[:])})
	}
	manifest, err := common.Marshal(struct {
		Version int                `json:"version"`
		Items   []skillStoreRecord `json:"items"`
	}{Version: 1, Items: []skillStoreRecord{record}})
	require.NoError(t, err)
	client := &http.Client{Transport: skillStoreTransport(func(request *http.Request) (*http.Response, error) {
		filename := strings.TrimPrefix(request.URL.Path, "/moeacgx/NewAPI-Mao/custom-main/skill-store/")
		body, exists := files[filename]
		if filename == "index.json" {
			body, exists = string(manifest), true
		}
		require.True(t, exists, "不应请求清单之外的文件：%s", filename)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	draft, err := importStoreSkill(context.Background(), "online:with-resources", client)
	require.NoError(t, err)
	assert.Equal(t, model.PromptLoadModeProgressive, draft.LoadMode)
	assert.Empty(t, draft.Content)
	require.Len(t, draft.Skills, 2)
	assert.Equal(t, "审查代码时使用", draft.Skills[0].Description)
	assert.Equal(t, []model.PromptSkillResource{{Path: "references/rules.json", Content: `{"rule":"可维护性"}`}}, draft.Skills[0].Resources)
	assert.Equal(t, []model.PromptSkillResource{{Path: "examples/style.txt", Content: "使用自然的语言。"}}, draft.Skills[1].Resources)
	files["packages/demo/review/references/rules.json"] = "被替换的引用资料"
	_, err = importStoreSkill(context.Background(), "online:with-resources", client)
	require.ErrorContains(t, err, "校验", "引用资料也必须通过商店清单的完整性校验")
}

func TestSkillStoreRejectsInvalidManifestAndUnknownIDs(t *testing.T) {
	for _, payload := range []string{
		`{"version":2,"items":[]}`,
		`{"version":1,"items":[{"id":"x","title":"x","kind":"prompt","version":"1","files":[{"path":"../SKILL.md","sha256":"aaaa"}]}]}`,
		`{"version":1,"items":[{"id":"x","title":"x","kind":"bundle","version":"1","files":[]}]}`,
	} {
		client := &http.Client{Transport: skillStoreTransport(func(request *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(payload)), Header: make(http.Header)}, nil
		})}
		_, err := loadSkillStoreCatalog(context.Background(), "online", client)
		require.Error(t, err)
	}
	for _, id := range []string{"bundled:missing", "https://evil.example/skill", "unknown:x"} {
		_, err := ImportStoreSkill(context.Background(), id)
		require.Error(t, err)
	}
}

func TestSkillStoreRejectsOversizedResponsesAndRedirects(t *testing.T) {
	for _, limit := range []int64{64 * 1024, 128 * 1024} {
		client := &http.Client{Transport: skillStoreTransport(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(strings.Repeat("x", int(limit)+1))), Header: make(http.Header)}, nil
		})}
		_, err := readSkillStoreFile(context.Background(), "online", "index.json", limit, client)
		require.ErrorContains(t, err, "大小限制")
	}
	previous := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previous })
	requests := 0
	http.DefaultTransport = skillStoreTransport(func(request *http.Request) (*http.Response, error) {
		requests++
		assert.Equal(t, "raw.githubusercontent.com", request.URL.Host)
		return &http.Response{StatusCode: http.StatusFound, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{"Location": []string{"https://127.0.0.1/private"}}}, nil
	})
	_, err := readSkillStoreFile(context.Background(), "online", "index.json", 128*1024, nil)
	require.Error(t, err)
	assert.Equal(t, 1, requests, "不能跟随远端响应跳转到任意主机")
}
