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
	"sync"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// 仓库导入总期限需低于正式站 Cloudflare 100 秒的源站响应上限；原始文件按有限并发下载。
const repositoryImportTimeout = 90 * time.Second
const repositoryDownloadConcurrency = 8

var githubRepositoryName = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
var githubCommitSHA = regexp.MustCompile(`^[a-fA-F0-9]{40}$`)

// ImportRepositorySkills 只匿名读取公开 GitHub 技能文本，返回未发布草稿，不执行仓库内容。
func ImportRepositorySkills(ctx context.Context, repositoryURL string) (*model.PromptTemplate, error) {
	transport := &http.Transport{DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 15 * time.Second}
	defer transport.CloseIdleConnections()
	return importRepositorySkills(ctx, repositoryURL, &http.Client{Transport: transport})
}

// ErrRepositoryLocalChanges 表示站内副本与上次导入内容不同，需要管理员确认覆盖。
var ErrRepositoryLocalChanges = errors.New("该技能在站内修改过正文或引用文件，确认覆盖后才能从 GitHub 更新")

// UpdatePromptTemplateFromRepository 重新读取记录的 GitHub 来源并替换技能正文与引用文件。
// 模板 ID、令牌安装、分类、白名单、上下架、排序和站点顶层标题/说明保持不变；返回值表示内容是否发生变化。
func UpdatePromptTemplateFromRepository(ctx context.Context, id, version int, overwriteLocal bool) (*model.PromptTemplate, bool, error) {
	transport := &http.Transport{DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 15 * time.Second}
	defer transport.CloseIdleConnections()
	return updatePromptTemplateFromRepository(ctx, id, version, overwriteLocal, &http.Client{Transport: transport})
}

func updatePromptTemplateFromRepository(ctx context.Context, id, version int, overwriteLocal bool, client *http.Client) (*model.PromptTemplate, bool, error) {
	current, err := model.GetAdminPromptTemplate(id)
	if err != nil {
		return nil, false, err
	}
	if current.Version != version {
		return nil, false, model.ErrPromptVersionConflict
	}
	source := current.RepositorySource
	if source == nil {
		return nil, false, errors.New("该技能没有记录 GitHub 来源，无法一键更新；请重新从 GitHub 导入")
	}
	if source.LocallyModified && !overwriteLocal {
		return nil, false, ErrRepositoryLocalChanges
	}
	draft, err := importRepositorySkills(ctx, source.URL, client)
	if err != nil {
		return nil, false, fmt.Errorf("从 GitHub 拉取更新失败，站内技能保持不变：%w", err)
	}
	if !source.LocallyModified && draft.RepositorySource.ContentHash == source.ContentHash {
		return current, false, nil
	}
	updated := *current
	updated.Kind, updated.LoadMode, updated.Content, updated.Skills = draft.Kind, draft.LoadMode, draft.Content, draft.Skills
	// 相同主文件路径沿用原技能 ID，正在进行的目录发现和统计仍指向同一技能。
	skillIDs := make(map[string]string, len(current.Skills))
	for _, skill := range current.Skills {
		skillIDs[skill.Path] = skill.ID
	}
	for i := range updated.Skills {
		if skillID, ok := skillIDs[updated.Skills[i].Path]; ok {
			updated.Skills[i].ID = skillID
		}
	}
	updated.RepositorySource = draft.RepositorySource
	saved, err := model.SavePromptTemplate(updated)
	if err != nil {
		return nil, false, err
	}
	return saved, true, nil
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
	ctx, cancel := context.WithTimeout(ctx, repositoryImportTimeout)
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
	// refName 记录用户链接里的分支、标签或提交，ref 是本次读取使用的具体对象。
	refName, ref, selectedPath, blob := metadata.DefaultBranch, metadata.DefaultBranch, "", false
	if len(parts) > 2 {
		blob = parts[2] == "blob"
		rest := strings.Join(parts[3:], "/")
		if githubCommitSHA.MatchString(parts[3]) {
			refName, ref = strings.ToLower(parts[3]), parts[3]
			selectedPath = strings.Join(parts[4:], "/")
		} else {
			refName, ref, selectedPath, err = reader.resolveReference(apiBase, rest)
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
	type repositoryTextFile struct {
		path string
		url  string
	}
	downloads := make([]repositoryTextFile, 0)
	var declared int64
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
		if len(downloads) >= model.MaxSkillResourceFiles || declared+entry.Size > maxSkillExpandedBytes {
			return nil, errors.New("技能包不能超过 64 MiB 或 4096 个文本文件")
		}
		declared += entry.Size
		segments := strings.Split(entry.Path, "/")
		for i := range segments {
			segments[i] = url.PathEscape(segments[i])
		}
		downloads = append(downloads, repositoryTextFile{path: entry.Path, url: "https://raw.githubusercontent.com/" + owner + "/" + repository + "/" + commit.SHA + "/" + strings.Join(segments, "/")})
	}
	// 固定提交下的文件互不依赖，按有限并发下载；任一失败立即取消其余请求，并按路径顺序报告首个错误。
	downloadCtx, cancelDownloads := context.WithCancel(ctx)
	defer cancelDownloads()
	downloadReader := skillRepositoryReader{ctx: downloadCtx, client: reader.client}
	bodies := make([][]byte, len(downloads))
	failures := make([]error, len(downloads))
	jobs := make(chan int)
	var workers sync.WaitGroup
	for worker := 0; worker < min(repositoryDownloadConcurrency, len(downloads)); worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range jobs {
				body, err := downloadReader.read(downloads[index].url, model.MaxSkillFileBytes)
				if err == nil {
					body, err = normalizeImportedSkillText(body)
				}
				if err != nil {
					failures[index] = err
					cancelDownloads()
					continue
				}
				bodies[index] = body
			}
		}()
	}
	for index := range downloads {
		if downloadCtx.Err() != nil {
			break
		}
		jobs <- index
	}
	close(jobs)
	workers.Wait()
	if err := ctx.Err(); err != nil {
		downloaded := 0
		for _, body := range bodies {
			if body != nil {
				downloaded++
			}
		}
		return nil, reader.contextError(fmt.Sprintf("已读取 %d / %d 个文本文件", downloaded, len(downloads)))
	}
	textFiles := make(map[string][]byte, len(downloads))
	total := 0
	for index, download := range downloads {
		if failures[index] != nil && !errors.Is(failures[index], context.Canceled) {
			return nil, fmt.Errorf("读取技能文件 %q 失败：%w", download.path, failures[index])
		}
	}
	for index, download := range downloads {
		if bodies[index] == nil {
			return nil, fmt.Errorf("读取技能文件 %q 失败：下载未完成", download.path)
		}
		total += len(bodies[index])
		if total > maxSkillExpandedBytes {
			return nil, errors.New("技能包正文合计不能超过 64 MiB")
		}
		textFiles[download.path] = bodies[index]
	}
	draft, err := buildProgressiveSkillDraft(repository, files, textFiles)
	if err != nil {
		return nil, err
	}
	hash, err := model.PromptTemplateContentHash(draft)
	if err != nil {
		return nil, err
	}
	draft.RepositorySource = &model.PromptRepositorySource{URL: strings.TrimSpace(repositoryURL), Ref: refName, Path: selectedPath,
		Commit: strings.ToLower(commit.SHA), ImportedAt: time.Now().Unix(), ContentHash: hash}
	return draft, nil
}

type skillRepositoryReader struct {
	ctx    context.Context
	client *http.Client
}

// resolveReference 按最长完整引用识别带斜杠的分支和标签，避免把引用的一部分当成目录。
// 返回引用名称、解引用后的提交 SHA 和引用之后的目录。
func (reader skillRepositoryReader) resolveReference(apiBase, rest string) (string, string, string, error) {
	var selectedName, selectedNamespace, selectedSHA, selectedType string
	ambiguous := false
	first := strings.SplitN(rest, "/", 2)[0]
	for _, namespace := range []string{"heads", "tags"} {
		body, err := reader.read(apiBase+"/git/matching-refs/"+namespace+"/"+url.PathEscape(first), 256*1024)
		if err != nil {
			return "", "", "", err
		}
		var matches []struct {
			Ref    string `json:"ref"`
			Object struct {
				SHA  string `json:"sha"`
				Type string `json:"type"`
			} `json:"object"`
		}
		if err := common.Unmarshal(body, &matches); err != nil || len(matches) > 1024 {
			return "", "", "", errors.New("GitHub 分支或标签列表无效或过大，请使用完整 commit SHA 链接")
		}
		prefix := "refs/" + namespace + "/"
		for _, match := range matches {
			name := strings.TrimPrefix(match.Ref, prefix)
			if !strings.HasPrefix(match.Ref, prefix) || !fs.ValidPath(name) || strings.ContainsAny(name, "\\\x00") || !utf8.ValidString(name) || !githubCommitSHA.MatchString(match.Object.SHA) || (match.Object.Type != "commit" && match.Object.Type != "tag") {
				return "", "", "", errors.New("GitHub 分支或标签引用无效")
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
		return "", "", "", errors.New("链接没有匹配的 GitHub 分支或标签，请检查路径或使用完整 commit SHA")
	}
	if ambiguous {
		return "", "", "", errors.New("链接同时匹配同名分支和标签，请使用完整 commit SHA 链接")
	}
	// 附注标签指向标签对象；只沿官方 API 的固定 SHA 解引用，最多五层。
	for depth := 0; selectedType == "tag"; depth++ {
		if depth >= 5 {
			return "", "", "", errors.New("GitHub 标签引用嵌套过深，请使用完整 commit SHA 链接")
		}
		body, err := reader.read(apiBase+"/git/tags/"+selectedSHA, 64*1024)
		if err != nil {
			return "", "", "", err
		}
		var tag struct {
			Object struct {
				SHA  string `json:"sha"`
				Type string `json:"type"`
			} `json:"object"`
		}
		if err := common.Unmarshal(body, &tag); err != nil || !githubCommitSHA.MatchString(tag.Object.SHA) || (tag.Object.Type != "commit" && tag.Object.Type != "tag") {
			return "", "", "", errors.New("GitHub 标签没有指向有效提交")
		}
		selectedSHA, selectedType = tag.Object.SHA, tag.Object.Type
	}
	return selectedName, selectedSHA, strings.TrimPrefix(strings.TrimPrefix(rest, selectedName), "/"), nil
}

// contextError 区分导入期限到达与客户端取消，避免把超时误报为可以直接重试的网络错误。
func (reader skillRepositoryReader) contextError(progress string) error {
	if !errors.Is(reader.ctx.Err(), context.DeadlineExceeded) {
		return reader.ctx.Err()
	}
	if progress != "" {
		progress = "（" + progress + "）"
	}
	return fmt.Errorf("GitHub 仓库导入超过 %d 秒时限%s；仓库文件较多时请改用 tree 目录链接只导入需要的技能", int(repositoryImportTimeout/time.Second), progress)
}

func (reader skillRepositoryReader) read(endpoint string, maxBytes int64) ([]byte, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || (parsed.Host != "api.github.com" && parsed.Host != "raw.githubusercontent.com") || parsed.User != nil {
		return nil, errors.New("技能仓库请求地址无效")
	}
	// 大仓库会发出数百个请求；连接中断和 GitHub 网关错误各自最多重试两次，其他状态直接失败。
	for attempt := 0; ; attempt++ {
		body, retryable, err := reader.readOnce(endpoint, maxBytes)
		if err == nil || !retryable || attempt >= 2 {
			return body, err
		}
		select {
		case <-reader.ctx.Done():
			return nil, reader.contextError("")
		case <-time.After(time.Duration(attempt+1) * 500 * time.Millisecond):
		}
	}
}

func (reader skillRepositoryReader) readOnce(endpoint string, maxBytes int64) ([]byte, bool, error) {
	request, err := http.NewRequestWithContext(reader.ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, false, errors.New("无法创建技能仓库请求")
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "NewAPI-Mao-Skill-Import")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	response, err := reader.client.Do(request)
	if err != nil {
		if reader.ctx.Err() != nil {
			return nil, false, reader.contextError("")
		}
		return nil, true, errors.New("无法连接 GitHub 技能仓库，请检查服务器网络后重试")
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return nil, true, fmt.Errorf("GitHub 暂时不可用（HTTP %d），请稍后重试", response.StatusCode)
	default:
		return nil, false, errors.New("GitHub 仓库不可公开读取、请求受限或链接不存在")
	}
	if response.ContentLength > maxBytes {
		return nil, false, errors.New("技能仓库响应超过大小限制")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil && reader.ctx.Err() != nil {
		return nil, false, reader.contextError("")
	}
	if err != nil {
		return nil, true, errors.New("GitHub 技能仓库响应中断")
	}
	if int64(len(body)) > maxBytes {
		return nil, false, errors.New("技能仓库响应超过大小限制")
	}
	return body, false, nil
}
