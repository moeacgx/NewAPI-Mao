package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// largeRepositoryClient 返回一个含 count 个引用文件的单技能仓库；原始文件请求交给 raw 处理。
func largeRepositoryClient(t *testing.T, count int, raw func(*http.Request) (*http.Response, error)) *http.Client {
	t.Helper()
	const commitSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const treeSHA = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	entries := []string{`{"path":"skills/big/SKILL.md","mode":"100644","type":"blob","size":40}`}
	for i := 0; i < count; i++ {
		entries = append(entries, fmt.Sprintf(`{"path":"skills/big/references/%03d.md","mode":"100644","type":"blob","size":10}`, i))
	}
	tree := `{"tree":[` + strings.Join(entries, ",") + `]}`
	return &http.Client{Transport: skillRepositoryTransport(func(request *http.Request) (*http.Response, error) {
		var body string
		switch request.URL.Path {
		case "/repos/acme/skills":
			body = `{"private":false,"visibility":"public","default_branch":"main"}`
		case "/repos/acme/skills/commits/main":
			body = `{"sha":"` + commitSHA + `","commit":{"tree":{"sha":"` + treeSHA + `"}}}`
		case "/repos/acme/skills/git/trees/" + treeSHA:
			body = tree
		default:
			require.Equal(t, "raw.githubusercontent.com", request.URL.Host)
			return raw(request)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
}

func rawRepositoryBody(request *http.Request) (*http.Response, error) {
	body := "引用正文"
	if strings.HasSuffix(request.URL.Path, "/SKILL.md") {
		body = "---\nname: big\ndescription: 大型技能\n---\n主正文"
	}
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
}

func TestRepositoryImportDownloadsFilesConcurrentlyWithBound(t *testing.T) {
	// 前 8 个原始文件请求全部到达后才放行：串行下载会一直卡在第一个请求，直到期限到达。
	var mu sync.Mutex
	inFlight, maxInFlight := 0, 0
	release := make(chan struct{})
	client := largeRepositoryClient(t, 40, func(request *http.Request) (*http.Response, error) {
		mu.Lock()
		inFlight++
		maxInFlight = max(maxInFlight, inFlight)
		if inFlight == repositoryDownloadConcurrency {
			select {
			case <-release:
			default:
				close(release)
			}
		}
		mu.Unlock()
		defer func() {
			mu.Lock()
			inFlight--
			mu.Unlock()
		}()
		select {
		case <-release:
		case <-request.Context().Done():
			return nil, request.Context().Err()
		}
		return rawRepositoryBody(request)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	draft, err := importRepositorySkills(ctx, "https://github.com/acme/skills", client)
	require.NoError(t, err)
	require.Len(t, draft.Skills, 1)
	assert.Len(t, draft.Skills[0].Resources, 40)
	assert.Equal(t, "references/000.md", draft.Skills[0].Resources[0].Path, "并发下载不改变引用文件顺序")
	assert.Equal(t, repositoryDownloadConcurrency, maxInFlight, "同时下载数量必须达到且不超过上限")
}

func TestRepositoryImportReportsDeadlineInsteadOfRetryHint(t *testing.T) {
	client := largeRepositoryClient(t, 3, func(request *http.Request) (*http.Response, error) {
		<-request.Context().Done()
		return nil, request.Context().Err()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := importRepositorySkills(ctx, "https://github.com/acme/skills", client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "超过")
	assert.Contains(t, err.Error(), "tree 目录链接")
	assert.NotContains(t, err.Error(), "请稍后重试")
}

func TestSkillImportsEscapeNULBytesAndRejectInvalidUTF8(t *testing.T) {
	// 空字节截断 payload 示例中的真实 NUL 无法写入 PostgreSQL TEXT，导入时转为可读的 \x00。
	payload := "filename=\"shell.php\x00.jpg\""
	want := `filename="shell.php\x00.jpg"`
	main := "---\nname: big\ndescription: 大型技能\n---\n主正文\x00结尾"

	client := largeRepositoryClient(t, 1, func(request *http.Request) (*http.Response, error) {
		body := payload
		if strings.HasSuffix(request.URL.Path, "/SKILL.md") {
			body = main
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	draft, err := importRepositorySkills(context.Background(), "https://github.com/acme/skills", client)
	require.NoError(t, err)
	assert.Equal(t, `主正文\x00结尾`, draft.Skills[0].Content)
	assert.Equal(t, want, draft.Skills[0].Resources[0].Content)

	archive, err := ParseSkillArchive(skillArchiveFixture(t, []string{"big/SKILL.md", "big/references/payload.md"}, []string{main, payload}, false), "big.zip")
	require.NoError(t, err)
	assert.Equal(t, `主正文\x00结尾`, archive.Skills[0].Content)
	assert.Equal(t, want, archive.Skills[0].Resources[0].Content)

	invalid := largeRepositoryClient(t, 1, func(request *http.Request) (*http.Response, error) {
		body := "\xff\xfe"
		if strings.HasSuffix(request.URL.Path, "/SKILL.md") {
			body = main
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	_, err = importRepositorySkills(context.Background(), "https://github.com/acme/skills", invalid)
	require.ErrorContains(t, err, `"skills/big/references/000.md"`)
	require.ErrorContains(t, err, "不是 UTF-8")
}

func TestRepositoryImportRetriesTransientFailuresOnly(t *testing.T) {
	for _, test := range []struct {
		name     string
		failures []func() (*http.Response, error)
		attempts int
		err      string
	}{
		{"连接错误和 502 重试后成功", []func() (*http.Response, error){
			func() (*http.Response, error) { return nil, errors.New("connection reset by peer") },
			func() (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusBadGateway, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("bad gateway"))}, nil
			},
		}, 3, ""},
		{"404 不重试并报告具体文件", []func() (*http.Response, error){
			func() (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusNotFound, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("missing"))}, nil
			},
		}, 1, `"skills/big/references/000.md"`},
		{"持续 503 在重试上限后失败", []func() (*http.Response, error){
			func() (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("busy"))}, nil
			},
			func() (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("busy"))}, nil
			},
			func() (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("busy"))}, nil
			},
		}, 3, `"skills/big/references/000.md"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var mu sync.Mutex
			attempts := 0
			client := largeRepositoryClient(t, 1, func(request *http.Request) (*http.Response, error) {
				if !strings.HasSuffix(request.URL.Path, "/000.md") {
					return rawRepositoryBody(request)
				}
				mu.Lock()
				attempt := attempts
				attempts++
				mu.Unlock()
				if attempt < len(test.failures) {
					return test.failures[attempt]()
				}
				return rawRepositoryBody(request)
			})
			_, err := importRepositorySkills(context.Background(), "https://github.com/acme/skills", client)
			if test.err == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.err)
			}
			assert.Equal(t, test.attempts, attempts)
		})
	}
}
