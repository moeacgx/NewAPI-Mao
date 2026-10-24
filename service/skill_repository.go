package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

var githubRepositoryName = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
var githubCommitSHA = regexp.MustCompile(`^[a-fA-F0-9]{40}$`)

// ImportRepositorySkills 只匿名读取公开 GitHub 技能文本，返回未发布草稿，不执行仓库内容。
func ImportRepositorySkills(ctx context.Context, repositoryURL string) (*model.PromptTemplate, error) {
	transport := &http.Transport{DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 15 * time.Second}
	defer transport.CloseIdleConnections()
	return importRepositorySkills(ctx, repositoryURL, &http.Client{Transport: transport})
}

func importRepositorySkills(ctx context.Context, repositoryURL string, client *http.Client) (*model.PromptTemplate, error) {
	parsed, err := url.Parse(strings.TrimSpace(repositoryURL))
	if err != nil || parsed.Scheme != "https" || parsed.Host != "github.com" || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || strings.Contains(repositoryURL, "#") {
		return nil, errors.New("请输入不含认证、端口、查询参数或片段的公开 GitHub HTTPS 仓库链接")
	}
	if parsed.RawPath != "" || !utf8.ValidString(parsed.Path) || strings.ContainsAny(parsed.Path, "\\\x00") {
		return nil, errors.New("GitHub 仓库路径无效")
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if !fs.ValidPath(strings.Trim(parsed.Path, "/")) {
		return nil, errors.New("GitHub 仓库路径无效")
	}
	if len(parts) < 2 || !githubRepositoryName.MatchString(parts[0]) || !githubRepositoryName.MatchString(parts[1]) || parts[0] == "." || parts[0] == ".." || parts[1] == "." || parts[1] == ".." {
		return nil, errors.New("GitHub 仓库链接需要包含所有者和仓库名称")
	}
	owner, repository := parts[0], strings.TrimSuffix(parts[1], ".git")
	if repository == "" || repository == "." || repository == ".." || (len(parts) > 2 && (len(parts) < 4 || (parts[2] != "tree" && parts[2] != "blob"))) {
		return nil, errors.New("仅支持仓库、tree 目录或 blob SKILL.md 链接")
	}
	if len(parts) > 2 && parts[2] == "blob" && (len(parts) < 5 || !strings.EqualFold(path.Base(parsed.Path), "SKILL.md")) {
		return nil, errors.New("文件链接必须指向 SKILL.md")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	// 不继承 Cookie，也不跟随 GitHub 返回的任意重定向地址。
	isolatedClient := *client
	isolatedClient.Jar = nil
	isolatedClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	reader := skillRepositoryReader{ctx: ctx, client: &isolatedClient}
	apiBase := "https://api.github.com/repos/" + owner + "/" + repository
	metadataBytes, err := reader.read(apiBase, 64*1024)
	if err != nil {
		return nil, err
	}
	var metadata struct {
		Private       bool   `json:"private"`
		Visibility    string `json:"visibility"`
		DefaultBranch string `json:"default_branch"`
	}
	if err := common.Unmarshal(metadataBytes, &metadata); err != nil || metadata.Private || (metadata.Visibility != "" && metadata.Visibility != "public") || !fs.ValidPath(metadata.DefaultBranch) || strings.ContainsAny(metadata.DefaultBranch, "\\\x00") || !utf8.ValidString(metadata.DefaultBranch) {
		return nil, errors.New("仅支持可公开读取的 GitHub 仓库")
	}
	ref, selectedPath, blob := metadata.DefaultBranch, "", false
	if len(parts) > 2 {
		blob = parts[2] == "blob"
		rest := strings.Join(parts[3:], "/")
		if githubCommitSHA.MatchString(parts[3]) {
			ref = parts[3]
			selectedPath = strings.Join(parts[4:], "/")
		} else {
			ref, selectedPath, err = reader.resolveReference(apiBase, rest)
			if err != nil {
				return nil, err
			}
		}
		if !fs.ValidPath(ref) || (selectedPath != "" && !fs.ValidPath(selectedPath)) || (blob && !strings.EqualFold(path.Base(selectedPath), "SKILL.md")) {
			return nil, errors.New("技能目录或文件路径无效")
		}
	}
	commitBytes, err := reader.read(apiBase+"/commits/"+url.PathEscape(ref), 256*1024)
	if err != nil {
		return nil, err
	}
	var commit struct {
		SHA    string `json:"sha"`
		Commit struct {
			Tree struct {
				SHA string `json:"sha"`
			} `json:"tree"`
		} `json:"commit"`
	}
	if err := common.Unmarshal(commitBytes, &commit); err != nil || !githubCommitSHA.MatchString(commit.SHA) || !githubCommitSHA.MatchString(commit.Commit.Tree.SHA) {
		return nil, errors.New("无法确定 GitHub 仓库的固定提交")
	}
	treeBytes, err := reader.read(apiBase+"/git/trees/"+commit.Commit.Tree.SHA+"?recursive=1", 2*1024*1024)
	if err != nil {
		return nil, err
	}
	var tree struct {
		Truncated bool `json:"truncated"`
		Tree      []struct {
			Path string `json:"path"`
			Mode string `json:"mode"`
			Type string `json:"type"`
			Size int64  `json:"size"`
		} `json:"tree"`
	}
	if err := common.Unmarshal(treeBytes, &tree); err != nil || tree.Truncated || len(tree.Tree) > 4096 {
		return nil, errors.New("仓库文件树不完整或超过 4096 项，请使用更小的技能仓库")
	}
	files := make([]string, 0, model.MaxProgressiveSkills)
	allMains := make([]string, 0)
	seen := make(map[string]bool)
	for _, entry := range tree.Tree {
		if !fs.ValidPath(entry.Path) || strings.ContainsAny(entry.Path, "\\:\x00") || !utf8.ValidString(entry.Path) || len(entry.Path) > 1024 || seen[entry.Path] {
			return nil, errors.New("仓库存在无效或重复的文件路径")
		}
		seen[entry.Path] = true
		if strings.EqualFold(path.Base(entry.Path), "SKILL.md") {
			allMains = append(allMains, entry.Path)
		}
		selected := selectedPath == "" || entry.Path == selectedPath || (!blob && strings.HasPrefix(entry.Path, selectedPath+"/"))
		if !selected {
			continue
		}
		if entry.Mode == "160000" || entry.Type == "commit" {
			return nil, errors.New("技能目录不能包含 Git 子模块")
		}
		if !strings.EqualFold(path.Base(entry.Path), "SKILL.md") {
			continue
		}
		if entry.Type != "blob" || (entry.Mode != "100644" && entry.Mode != "100755") {
			return nil, errors.New("技能入口必须为普通文件，不能使用符号链接")
		}
		if entry.Size < 0 || entry.Size > model.MaxSkillFileBytes {
			return nil, fmt.Errorf("技能文件 %q 的大小无效或超过 8 MiB", entry.Path)
		}
		files = append(files, entry.Path)
	}
	if len(files) == 0 || len(files) > model.MaxProgressiveSkills {
		return nil, errors.New("技能仓库必须包含 1 至 128 个 SKILL.md 文件")
	}
	sort.Strings(files)
	sort.Strings(allMains)
	selectedMains := make(map[string]bool, len(files))
	for _, filename := range files {
		selectedMains[filename] = true
	}
	textFiles := make(map[string][]byte)
	total := 0
	for _, entry := range tree.Tree {
		if !supportedSkillTextFile(entry.Path) {
			continue
		}
		skillOwner := skillFileOwner(entry.Path, allMains)
		if skillOwner < 0 || !selectedMains[allMains[skillOwner]] {
			continue
		}
		if entry.Type != "blob" || (entry.Mode != "100644" && entry.Mode != "100755") {
			return nil, fmt.Errorf("技能文件 %q 必须为普通文件，不能使用符号链接", entry.Path)
		}
		if entry.Size < 0 || entry.Size > model.MaxSkillFileBytes {
			return nil, fmt.Errorf("技能文件 %q 的大小无效或超过 8 MiB", entry.Path)
		}
		if len(textFiles) >= model.MaxSkillResourceFiles || int64(total)+entry.Size > maxSkillExpandedBytes {
			return nil, errors.New("技能包不能超过 64 MiB 或 4096 个文本文件")
		}
		segments := strings.Split(entry.Path, "/")
		for i := range segments {
			segments[i] = url.PathEscape(segments[i])
		}
		body, err := reader.read("https://raw.githubusercontent.com/"+owner+"/"+repository+"/"+commit.SHA+"/"+strings.Join(segments, "/"), model.MaxSkillFileBytes)
		if err != nil {
			return nil, fmt.Errorf("读取技能文件 %q 失败：%w", entry.Path, err)
		}
		total += len(body)
		if total > maxSkillExpandedBytes {
			return nil, errors.New("技能包正文合计不能超过 64 MiB")
		}
		if !utf8.Valid(body) || strings.ContainsRune(string(body), 0) {
			return nil, fmt.Errorf("技能文件 %q 必须为 UTF-8 文本", entry.Path)
		}
		textFiles[entry.Path] = body
	}
	return buildProgressiveSkillDraft(repository, files, textFiles)
}

type skillRepositoryReader struct {
	ctx    context.Context
	client *http.Client
}

// resolveReference 按最长完整引用识别带斜杠的分支和标签，避免把引用的一部分当成目录。
func (reader skillRepositoryReader) resolveReference(apiBase, rest string) (string, string, error) {
	var selectedName, selectedNamespace, selectedSHA, selectedType string
	ambiguous := false
	first := strings.SplitN(rest, "/", 2)[0]
	for _, namespace := range []string{"heads", "tags"} {
		body, err := reader.read(apiBase+"/git/matching-refs/"+namespace+"/"+url.PathEscape(first), 256*1024)
		if err != nil {
			return "", "", err
		}
		var matches []struct {
			Ref    string `json:"ref"`
			Object struct {
				SHA  string `json:"sha"`
				Type string `json:"type"`
			} `json:"object"`
		}
		if err := common.Unmarshal(body, &matches); err != nil || len(matches) > 1024 {
			return "", "", errors.New("GitHub 分支或标签列表无效或过大，请使用完整 commit SHA 链接")
		}
		prefix := "refs/" + namespace + "/"
		for _, match := range matches {
			name := strings.TrimPrefix(match.Ref, prefix)
			if !strings.HasPrefix(match.Ref, prefix) || !fs.ValidPath(name) || strings.ContainsAny(name, "\\\x00") || !utf8.ValidString(name) || !githubCommitSHA.MatchString(match.Object.SHA) || (match.Object.Type != "commit" && match.Object.Type != "tag") {
				return "", "", errors.New("GitHub 分支或标签引用无效")
			}
			if rest != name && !strings.HasPrefix(rest, name+"/") {
				continue
			}
			if len(name) < len(selectedName) {
				continue
			}
			if name == selectedName && namespace != selectedNamespace {
				ambiguous = true
				continue
			}
			if len(name) > len(selectedName) {
				ambiguous = false
			}
			selectedName, selectedNamespace, selectedSHA, selectedType = name, namespace, match.Object.SHA, match.Object.Type
		}
	}
	if selectedName == "" {
		return "", "", errors.New("链接没有匹配的 GitHub 分支或标签，请检查路径或使用完整 commit SHA")
	}
	if ambiguous {
		return "", "", errors.New("链接同时匹配同名分支和标签，请使用完整 commit SHA 链接")
	}
	// 附注标签指向标签对象；只沿官方 API 的固定 SHA 解引用，最多五层。
	for depth := 0; selectedType == "tag"; depth++ {
		if depth >= 5 {
			return "", "", errors.New("GitHub 标签引用嵌套过深，请使用完整 commit SHA 链接")
		}
		body, err := reader.read(apiBase+"/git/tags/"+selectedSHA, 64*1024)
		if err != nil {
			return "", "", err
		}
		var tag struct {
			Object struct {
				SHA  string `json:"sha"`
				Type string `json:"type"`
			} `json:"object"`
		}
		if err := common.Unmarshal(body, &tag); err != nil || !githubCommitSHA.MatchString(tag.Object.SHA) || (tag.Object.Type != "commit" && tag.Object.Type != "tag") {
			return "", "", errors.New("GitHub 标签没有指向有效提交")
		}
		selectedSHA, selectedType = tag.Object.SHA, tag.Object.Type
	}
	return selectedSHA, strings.TrimPrefix(strings.TrimPrefix(rest, selectedName), "/"), nil
}

func (reader skillRepositoryReader) read(endpoint string, maxBytes int64) ([]byte, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || (parsed.Host != "api.github.com" && parsed.Host != "raw.githubusercontent.com") || parsed.User != nil {
		return nil, errors.New("技能仓库请求地址无效")
	}
	request, err := http.NewRequestWithContext(reader.ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.New("无法创建技能仓库请求")
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "NewAPI-Mao-Skill-Import")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	response, err := reader.client.Do(request)
	if err != nil {
		return nil, errors.New("无法读取 GitHub 技能仓库，请稍后重试")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("GitHub 仓库不可公开读取、请求受限或链接不存在")
	}
	if response.ContentLength > maxBytes {
		return nil, errors.New("技能仓库响应超过大小限制")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil || int64(len(body)) > maxBytes {
		return nil, errors.New("技能仓库响应损坏或超过大小限制")
	}
	return body, nil
}
