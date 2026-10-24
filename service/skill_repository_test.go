package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type skillRepositoryTransport func(*http.Request) (*http.Response, error)

func (transport skillRepositoryTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func repositoryFixtureClient(t *testing.T, treeBody string, raw map[string]string, requests *[]string) *http.Client {
	t.Helper()
	const commitSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const treeSHA = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	return &http.Client{Transport: skillRepositoryTransport(func(request *http.Request) (*http.Response, error) {
		assert.Equal(t, "https", request.URL.Scheme)
		assert.Empty(t, request.Header.Get("Authorization"))
		assert.Empty(t, request.Header.Get("Cookie"))
		deadline, ok := request.Context().Deadline()
		require.True(t, ok)
		assert.False(t, deadline.IsZero())
		*requests = append(*requests, request.URL.String())
		var body string
		switch request.URL.Host {
		case "api.github.com":
			switch request.URL.Path {
			case "/repos/acme/skills":
				body = `{"private":false,"visibility":"public","default_branch":"main"}`
			case "/repos/acme/skills/commits/main":
				body = `{"sha":"` + commitSHA + `","commit":{"tree":{"sha":"` + treeSHA + `"}}}`
			case "/repos/acme/skills/commits/" + commitSHA:
				body = `{"sha":"` + commitSHA + `","commit":{"tree":{"sha":"` + treeSHA + `"}}}`
			case "/repos/acme/skills/git/matching-refs/heads/main":
				body = `[{"ref":"refs/heads/main","object":{"type":"commit","sha":"` + commitSHA + `"}}]`
			case "/repos/acme/skills/git/matching-refs/tags/main":
				body = `[]`
			case "/repos/acme/skills/git/trees/" + treeSHA:
				assert.Equal(t, "1", request.URL.Query().Get("recursive"))
				body = treeBody
			default:
				t.Fatalf("不应请求此 API 路径：%s", request.URL.Path)
			}
		case "raw.githubusercontent.com":
			prefix := "/acme/skills/" + commitSHA + "/"
			require.True(t, strings.HasPrefix(request.URL.Path, prefix))
			var exists bool
			body, exists = raw[strings.TrimPrefix(request.URL.Path, prefix)]
			require.True(t, exists, request.URL.Path)
		default:
			t.Fatalf("不应访问其他主机：%s", request.URL.Host)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
}

func TestRepositoryImportPinsCommitAndFiltersDirectoryOrFile(t *testing.T) {
	tree := `{"tree":[{"path":"skills/z/SKILL.md","mode":"100644","type":"blob","size":40},{"path":"skills/a/SKILL.md","mode":"100644","type":"blob","size":40},{"path":"README.md","mode":"100644","type":"blob","size":10}]}`
	raw := map[string]string{"skills/a/SKILL.md": "---\nname: 提纲\n---\n先列提纲。", "skills/z/SKILL.md": "---\nname: 核验\n---\n检查结论。"}
	for _, test := range []struct {
		link  string
		count int
	}{
		{"https://github.com/acme/skills", 2},
		{"https://github.com/acme/skills/tree/main/skills/a", 1},
		{"https://github.com/acme/skills/blob/main/skills/z/SKILL.md", 1},
	} {
		t.Run(test.link, func(t *testing.T) {
			var requests []string
			draft, err := importRepositorySkills(context.Background(), test.link, repositoryFixtureClient(t, tree, raw, &requests))
			require.NoError(t, err)
			assert.False(t, draft.Enabled)
			assert.Zero(t, draft.ID)
			assert.Zero(t, draft.CategoryID)
			assert.NotNil(t, draft.GroupIDs)
			assert.Empty(t, draft.GroupIDs)
			assert.Equal(t, test.count, draft.SkillCount)
			assert.Equal(t, model.PromptLoadModeProgressive, draft.LoadMode)
			encoded, encodeErr := common.Marshal(draft)
			require.NoError(t, encodeErr)
			var metadata struct {
				Source struct {
					URL    string `json:"url"`
					Ref    string `json:"ref"`
					Commit string `json:"commit"`
					Hash   string `json:"content_hash"`
				} `json:"repository_source"`
			}
			require.NoError(t, common.Unmarshal(encoded, &metadata))
			assert.Equal(t, test.link, metadata.Source.URL)
			assert.Equal(t, "main", metadata.Source.Ref)
			assert.Equal(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", metadata.Source.Commit)
			assert.Len(t, metadata.Source.Hash, 64)
			expectedRequests := 3 + test.count
			if strings.Contains(test.link, "/tree/") || strings.Contains(test.link, "/blob/") {
				expectedRequests += 2
			}
			assert.Len(t, requests, expectedRequests)
			if test.count == 2 {
				assert.Equal(t, "bundle", draft.Kind)
				assert.Equal(t, "提纲", draft.Skills[0].Title)
				assert.Equal(t, "核验", draft.Skills[1].Title)
			} else {
				assert.Equal(t, "prompt", draft.Kind)
				require.Len(t, draft.Skills, 1)
				assert.NotEmpty(t, draft.Skills[0].ID)
			}
		})
	}
}

func TestRepositoryImportRejectsUnsafeURLsBeforeNetwork(t *testing.T) {
	client := &http.Client{Transport: skillRepositoryTransport(func(*http.Request) (*http.Response, error) {
		t.Fatal("无效链接不能访问网络")
		return nil, nil
	})}
	for _, link := range []string{
		"http://github.com/acme/skills", "https://example.com/acme/skills", "https://github.com.evil/acme/skills",
		"https://user:secret@github.com/acme/skills", "https://github.com:443/acme/skills", "https://github.com/acme/skills?token=secret",
		"https://github.com/acme/skills#readme", "https://github.com/acme/skills?", "https://github.com/acme/skills#",
		"https://github.com/acme/skills/blob/main/readme.md", "https://github.com/acme/skills/tree/main/../other",
		"https://github.com/acme/skills/issues", "https://github.com/acme/skills/tree", "https://github.com/acme",
	} {
		_, err := importRepositorySkills(context.Background(), link, client)
		require.Error(t, err, link)
	}
}

func TestRepositoryImportReportsFailingFileAndBodySize(t *testing.T) {
	tree := `{"tree":[{"path":"skills/a/SKILL.md","mode":"100644","type":"blob"},{"path":"skills/b/SKILL.md","mode":"100644","type":"blob"},{"path":"README.md","mode":"100644","type":"blob"},{"path":"scripts/run.sh","mode":"100644","type":"blob"}]}`
	for _, tc := range []struct {
		name, body, want string
	}{
		{"空正文", "---\nname: 审查\n---\n \n", "技能「审查」正文必须为非空 UTF-8 文本"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests []string
			client := repositoryFixtureClient(t, tree, map[string]string{"skills/a/SKILL.md": "有效正文", "skills/b/SKILL.md": tc.body}, &requests)
			draft, err := importRepositorySkills(context.Background(), "https://github.com/acme/skills", client)
			require.EqualError(t, err, "技能文件 \"skills/b/SKILL.md\"："+tc.want)
			assert.Nil(t, draft)
			assert.Len(t, requests, 5)
		})
	}
}

func TestRepositoryImportRejectsUnsafeTreesAndBodyLimits(t *testing.T) {
	for _, test := range []struct{ name, tree, body, message string }{
		{"树被截断", `{"truncated":true,"tree":[]}`, "", "不完整"},
		{"没有技能", `{"tree":[]}`, "", "1 至 128"},
		{"符号链接", `{"tree":[{"path":"SKILL.md","mode":"120000","type":"blob"}]}`, "", "符号链接"},
		{"子模块", `{"tree":[{"path":"module","mode":"160000","type":"commit"}]}`, "", "子模块"},
		{"大小声明超限", `{"tree":[{"path":"SKILL.md","mode":"100644","type":"blob","size":8388609}]}`, "", "8 MiB"},
		{"实际文件超限", `{"tree":[{"path":"SKILL.md","mode":"100644","type":"blob","size":1}]}`, strings.Repeat("a", model.MaxSkillFileBytes+1), "大小限制"},
		{"路径穿越", `{"tree":[{"path":"../SKILL.md","mode":"100644","type":"blob"}]}`, "", "无效"},
		{"重复路径", `{"tree":[{"path":"SKILL.md","mode":"100644","type":"blob"},{"path":"SKILL.md","mode":"100644","type":"blob"}]}`, "", "重复"},
		{"文件树响应超限", strings.Repeat("x", 2*1024*1024+1), "", "大小限制"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var requests []string
			_, err := importRepositorySkills(context.Background(), "https://github.com/acme/skills", repositoryFixtureClient(t, test.tree, map[string]string{"SKILL.md": test.body}, &requests))
			require.ErrorContains(t, err, test.message)
		})
	}
	for _, count := range []int{129, 4097} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			entries := make([]map[string]interface{}, count)
			for i := range entries {
				entries[i] = map[string]interface{}{"path": fmt.Sprintf("skill-%d/SKILL.md", i), "mode": "100644", "type": "blob"}
			}
			body, err := common.Marshal(map[string]interface{}{"tree": entries})
			require.NoError(t, err)
			var requests []string
			_, err = importRepositorySkills(context.Background(), "https://github.com/acme/skills", repositoryFixtureClient(t, string(body), nil, &requests))
			require.Error(t, err)
			assert.Len(t, requests, 3)
		})
	}
}

func TestRepositoryImportPreservesLargeFilesAndScopedResources(t *testing.T) {
	tree := `{"tree":[{"path":"skills/review/SKILL.md","mode":"100644","type":"blob"},{"path":"skills/review/references/policy.md","mode":"100644","type":"blob"},{"path":"skills/review/data/example.json","mode":"100644","type":"blob"},{"path":"skills/review/scripts/run.py","mode":"100644","type":"blob"},{"path":"skills/review/nested/SKILL.md","mode":"100644","type":"blob"},{"path":"skills/review/nested/references/child.txt","mode":"100644","type":"blob"},{"path":"README.md","mode":"100644","type":"blob"}]}`
	body := strings.Repeat("审查步骤。", 6000)
	raw := map[string]string{
		"skills/review/SKILL.md":                    "---\nname: 审查\ndescription: 检查代码质量\n---\n" + body,
		"skills/review/references/policy.md":        "规范",
		"skills/review/data/example.json":           `{"ok":true}`,
		"skills/review/nested/SKILL.md":             "---\nname: 嵌套技能\ndescription: 处理独立任务\n---\n子正文",
		"skills/review/nested/references/child.txt": "子资料",
	}
	for _, link := range []string{"https://github.com/acme/skills", "https://github.com/acme/skills/blob/main/skills/review/SKILL.md"} {
		t.Run(link, func(t *testing.T) {
			var requests []string
			draft, err := importRepositorySkills(context.Background(), link, repositoryFixtureClient(t, tree, raw, &requests))
			require.NoError(t, err)
			assert.Equal(t, model.PromptLoadModeProgressive, draft.LoadMode)
			assert.Equal(t, body, draft.Skills[0].Content)
			assert.Equal(t, "检查代码质量", draft.Skills[0].Description)
			assert.Equal(t, []model.PromptSkillResource{{Path: "data/example.json", Content: `{"ok":true}`}, {Path: "references/policy.md", Content: "规范"}}, draft.Skills[0].Resources)
			if strings.Contains(link, "/blob/") {
				assert.Len(t, draft.Skills, 1)
				assert.NotContains(t, strings.Join(requests, "\n"), "/nested/")
			} else {
				require.Len(t, draft.Skills, 2)
				assert.Equal(t, []model.PromptSkillResource{{Path: "references/child.txt", Content: "子资料"}}, draft.Skills[1].Resources)
				assert.Empty(t, draft.Content)
			}
			assert.NotContains(t, strings.Join(requests, "\n"), "run.py")
			assert.NotContains(t, strings.Join(requests, "\n"), "/README.md")
		})
	}
}

func TestRepositoryImportEnforcesAggregateDownloadBudget(t *testing.T) {
	content := strings.Repeat("a", model.MaxSkillFileBytes)
	entries := make([]map[string]interface{}, 9)
	raw := make(map[string]string)
	for i := range entries {
		filename := fmt.Sprintf("skill-%d/SKILL.md", i)
		entries[i] = map[string]interface{}{"path": filename, "mode": "100644", "type": "blob", "size": len(content)}
		raw[filename] = content
	}
	encoded, err := common.Marshal(map[string]interface{}{"tree": entries})
	require.NoError(t, err)
	var requests []string
	_, err = importRepositorySkills(context.Background(), "https://github.com/acme/skills", repositoryFixtureClient(t, string(encoded), raw, &requests))
	require.ErrorContains(t, err, "64 MiB")
	assert.Len(t, requests, 11, "第九个文件下载前应按声明大小拒绝超限包")
}

func TestRepositoryImportRejectsPrivateAndRedirects(t *testing.T) {
	for _, test := range []struct {
		status   int
		body     string
		redirect bool
	}{
		{200, `{"private":true,"default_branch":"main"}`, false},
		{200, `{"private":false,"visibility":"internal","default_branch":"main"}`, false},
		{302, "", true},
		{403, "private-response-must-not-leak", false},
		{200, strings.Repeat("x", 64*1024+1), false},
	} {
		calls := 0
		client := &http.Client{Transport: skillRepositoryTransport(func(request *http.Request) (*http.Response, error) {
			calls++
			response := &http.Response{StatusCode: test.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(test.body))}
			if test.redirect {
				response.Header.Set("Location", "https://other.example/secret")
			}
			return response, nil
		})}
		_, err := importRepositorySkills(context.Background(), "https://github.com/acme/skills", client)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "private-response")
		assert.Equal(t, 1, calls)
	}
}

func TestRepositoryImportSupportsSlashDefaultBranchWithoutFollowingAPILinks(t *testing.T) {
	const sha = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	var requests []string
	client := &http.Client{Transport: skillRepositoryTransport(func(request *http.Request) (*http.Response, error) {
		requests = append(requests, request.URL.String())
		var body string
		switch len(requests) {
		case 1:
			body = `{"private":false,"default_branch":"release/main","url":"http://127.0.0.1/private"}`
		case 2:
			assert.Equal(t, "/repos/acme/skills/git/matching-refs/heads/release", request.URL.Path)
			body = `[{"ref":"refs/heads/release/main","object":{"type":"commit","sha":"` + sha + `"}}]`
		case 3:
			assert.Equal(t, "/repos/acme/skills/git/matching-refs/tags/release", request.URL.Path)
			body = `[]`
		case 4:
			assert.Equal(t, "/repos/acme/skills/commits/"+sha, request.URL.Path)
			body = `{"sha":"` + sha + `","commit":{"tree":{"sha":"` + sha + `","url":"http://127.0.0.1/private"}}}`
		case 5:
			body = `{"tree":[{"path":"folder/SKILL.md","mode":"100644","type":"blob","url":"http://127.0.0.1/private"}]}`
		case 6:
			assert.Equal(t, "raw.githubusercontent.com", request.URL.Host)
			assert.Equal(t, "/acme/skills/"+sha+"/folder/SKILL.md", request.URL.Path)
			body = "# 助理\n保持准确"
		default:
			t.Fatal("不应再访问网络")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	draft, err := importRepositorySkills(context.Background(), "https://github.com/acme/skills/tree/release/main/folder", client)
	require.NoError(t, err)
	assert.Equal(t, "助理", draft.Title)
	assert.Len(t, requests, 6)
}

func TestRepositoryImportResolvesLongestBranchOrTagWithoutAmbiguousFallback(t *testing.T) {
	const commitSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const otherSHA = "cccccccccccccccccccccccccccccccccccccccc"
	for _, test := range []struct {
		name, link, heads, tags, expectedError string
		annotated                              bool
	}{
		{
			name:  "非默认分支含斜杠并优先最长分支",
			link:  "tree/feature/topic/folder",
			heads: `[{"ref":"refs/heads/feature","object":{"type":"commit","sha":"` + otherSHA + `"}},{"ref":"refs/heads/feature/topic","object":{"type":"commit","sha":"` + commitSHA + `"}}]`,
			tags:  `[]`,
		},
		{
			name:  "含斜杠标签文件链接",
			link:  "blob/feature/topic/folder/SKILL.md",
			heads: `[{"ref":"refs/heads/feature","object":{"type":"commit","sha":"` + otherSHA + `"}}]`,
			tags:  `[{"ref":"refs/tags/feature/topic","object":{"type":"commit","sha":"` + commitSHA + `"}}]`,
		},
		{
			name:      "附注标签按固定SHA解析",
			link:      "tree/feature/topic/folder",
			heads:     `[]`,
			tags:      `[{"ref":"refs/tags/feature/topic","object":{"type":"tag","sha":"` + otherSHA + `"}}]`,
			annotated: true,
		},
		{
			name:          "同名分支和标签拒绝歧义",
			link:          "tree/feature/topic/folder",
			heads:         `[{"ref":"refs/heads/feature/topic","object":{"type":"commit","sha":"` + commitSHA + `"}}]`,
			tags:          `[{"ref":"refs/tags/feature/topic","object":{"type":"commit","sha":"` + otherSHA + `"}}]`,
			expectedError: "同名分支和标签",
		},
		{
			name:  "短引用同名不妨碍更长标签",
			link:  "tree/feature/topic/folder",
			heads: `[{"ref":"refs/heads/feature","object":{"type":"commit","sha":"` + otherSHA + `"}}]`,
			tags:  `[{"ref":"refs/tags/feature","object":{"type":"commit","sha":"` + otherSHA + `"}},{"ref":"refs/tags/feature/topic","object":{"type":"commit","sha":"` + commitSHA + `"}}]`,
		},
		{
			name:  "前缀名称不能部分匹配",
			link:  "tree/feature/topic/folder",
			heads: `[{"ref":"refs/heads/feature/top","object":{"type":"commit","sha":"` + commitSHA + `"}}]`,
			tags:  `[]`, expectedError: "没有匹配",
		},
		{
			name:  "引用响应超过上限",
			link:  "tree/feature/topic/folder",
			heads: strings.Repeat("x", 256*1024+1), tags: `[]`, expectedError: "大小限制",
		},
		{
			name: "完整提交绕过引用猜测",
			link: "tree/" + commitSHA + "/folder",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var requests []string
			base := repositoryFixtureClient(t, `{"tree":[{"path":"folder/SKILL.md","mode":"100644","type":"blob"}]}`, map[string]string{"folder/SKILL.md": "# 助理\n保持准确"}, &requests)
			annotatedReads, commitReads := 0, 0
			client := &http.Client{Transport: skillRepositoryTransport(func(request *http.Request) (*http.Response, error) {
				assert.Equal(t, "https", request.URL.Scheme)
				assert.Empty(t, request.Header.Get("Authorization"))
				assert.Empty(t, request.Header.Get("Cookie"))
				if strings.Contains(request.URL.Path, "/commits/") {
					commitReads++
					assert.Equal(t, "/repos/acme/skills/commits/"+commitSHA, request.URL.Path)
				}
				var body string
				switch request.URL.Path {
				case "/repos/acme/skills/git/matching-refs/heads/feature":
					body = test.heads
				case "/repos/acme/skills/git/matching-refs/tags/feature":
					body = test.tags
				case "/repos/acme/skills/git/tags/" + otherSHA:
					annotatedReads++
					body = `{"object":{"type":"commit","sha":"` + commitSHA + `"},"url":"http://127.0.0.1/ignored"}`
				default:
					return base.Transport.RoundTrip(request)
				}
				assert.Equal(t, "api.github.com", request.URL.Host)
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
			})}
			draft, err := importRepositorySkills(context.Background(), "https://github.com/acme/skills/"+test.link, client)
			if test.expectedError != "" {
				require.ErrorContains(t, err, test.expectedError)
				assert.Zero(t, commitReads)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "助理", draft.Title)
			assert.Equal(t, "# 助理\n保持准确", draft.Content)
			assert.Equal(t, 1, commitReads)
			if test.annotated {
				assert.Equal(t, 1, annotatedReads)
			}
		})
	}
}
