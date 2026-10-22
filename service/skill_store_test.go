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
		assert.NotEmpty(t, draft.Content)
		// 修改导入副本不会修改仓库中的原件。
		draft.Content = "站点自定义"
		next, err := ImportStoreSkill(context.Background(), item.ID)
		require.NoError(t, err)
		assert.NotEqual(t, draft.Content, next.Content)
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
