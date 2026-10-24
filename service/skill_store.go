package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	skillstore "github.com/QuantumNous/new-api/skill-store"
)

const skillStoreBaseURL = "https://raw.githubusercontent.com/moeacgx/NewAPI-Mao/custom-main/skill-store/"

type skillStoreFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type skillStoreRecord struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Kind        string           `json:"kind"`
	Category    string           `json:"category"`
	Version     string           `json:"version"`
	Files       []skillStoreFile `json:"files"`
}

type SkillStoreSummary struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
	Category    string `json:"category"`
	Version     string `json:"version"`
	Source      string `json:"source"`
	SkillCount  int    `json:"skill_count"`
}

func ListSkillStore(ctx context.Context, online bool) ([]SkillStoreSummary, error) {
	source := "bundled"
	if online {
		source = "online"
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	records, err := loadSkillStoreCatalog(ctx, source, nil)
	if err != nil {
		return nil, err
	}
	items := make([]SkillStoreSummary, 0, len(records))
	for _, record := range records {
		items = append(items, SkillStoreSummary{ID: source + ":" + record.ID, Title: record.Title, Description: record.Description,
			Kind: record.Kind, Category: record.Category, Version: record.Version, Source: source, SkillCount: len(skillStoreMainFiles(record))})
	}
	return items, nil
}

func ImportStoreSkill(ctx context.Context, id string) (*model.PromptTemplate, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return importStoreSkill(ctx, id, nil)
}

func importStoreSkill(ctx context.Context, id string, client *http.Client) (*model.PromptTemplate, error) {
	source, recordID, found := strings.Cut(id, ":")
	if !found || (source != "bundled" && source != "online") || !validSkillStoreID(recordID) {
		return nil, errors.New("技能商店条目无效")
	}
	records, err := loadSkillStoreCatalog(ctx, source, client)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		if record.ID != recordID {
			continue
		}
		files := make(map[string][]byte, len(record.Files))
		total := 0
		for _, file := range record.Files {
			data, err := readSkillStoreFile(ctx, source, file.Path, model.MaxSkillFileBytes, client)
			if err != nil {
				return nil, err
			}
			digest := sha256.Sum256(data)
			if hex.EncodeToString(digest[:]) != file.SHA256 {
				return nil, errors.New("技能商店文件校验失败，请刷新目录后重试")
			}
			if !utf8.Valid(data) || strings.ContainsRune(string(data), 0) {
				return nil, errors.New("技能商店文件必须为 UTF-8 文本")
			}
			total += len(data)
			if total > model.MaxSkillPackageBytes {
				return nil, errors.New("技能包正文合计不能超过 64 MiB")
			}
			files[file.Path] = data
		}
		draft, err := buildProgressiveSkillDraft(record.Title, skillStoreMainFiles(record), files)
		if err != nil {
			return nil, err
		}
		draft.Title, draft.Description = record.Title, record.Description
		if draft.Kind == "prompt" {
			draft.Skills[0].Title, draft.Skills[0].Description = record.Title, record.Description
		}
		return draft, nil
	}
	return nil, errors.New("技能商店条目不存在，请刷新商店")
}

func loadSkillStoreCatalog(ctx context.Context, source string, client *http.Client) ([]skillStoreRecord, error) {
	data, err := readSkillStoreFile(ctx, source, "index.json", 128*1024, client)
	if err != nil {
		return nil, err
	}
	var catalog struct {
		Version int                `json:"version"`
		Items   []skillStoreRecord `json:"items"`
	}
	if err := common.Unmarshal(data, &catalog); err != nil || catalog.Version != 1 || len(catalog.Items) > 64 {
		return nil, errors.New("技能商店目录格式无效")
	}
	seen := make(map[string]bool, len(catalog.Items))
	for _, record := range catalog.Items {
		if !validSkillStoreID(record.ID) || seen[record.ID] || !utf8.ValidString(record.Title+record.Description+record.Category+record.Version) ||
			strings.TrimSpace(record.Title) == "" || utf8.RuneCountInString(record.Title) > 80 || len(record.Description) > model.MaxPromptDescriptionBytes ||
			utf8.RuneCountInString(record.Category) > 40 || record.Version == "" || len(record.Version) > 40 ||
			(record.Kind != "prompt" && record.Kind != "bundle") ||
			len(record.Files) > model.MaxSkillResourceFiles || (record.Kind == "prompt" && len(skillStoreMainFiles(record)) != 1) || (record.Kind == "bundle" && (len(skillStoreMainFiles(record)) < 2 || len(skillStoreMainFiles(record)) > model.MaxProgressiveSkills)) {
			return nil, errors.New("技能商店条目格式无效")
		}
		seen[record.ID] = true
		files := make(map[string]bool, len(record.Files))
		for _, file := range record.Files {
			checksum, err := hex.DecodeString(file.SHA256)
			if !fs.ValidPath(file.Path) || !strings.HasPrefix(file.Path, "packages/") || strings.ContainsAny(file.Path, "\\:?#%\x00") ||
				!supportedSkillTextFile(file.Path) || files[file.Path] || len(file.Path) > 240 || err != nil || len(checksum) != 32 || file.SHA256 != strings.ToLower(file.SHA256) {
				return nil, errors.New("技能商店文件路径或校验和无效")
			}
			files[file.Path] = true
		}
	}
	return catalog.Items, nil
}

func validSkillStoreID(id string) bool {
	if len(id) == 0 || len(id) > 80 {
		return false
	}
	for _, char := range id {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
			return false
		}
	}
	return true
}

func readSkillStoreFile(ctx context.Context, source, filename string, limit int64, client *http.Client) ([]byte, error) {
	if source == "bundled" {
		data, err := skillstore.Files.ReadFile(filename)
		if err != nil || int64(len(data)) > limit {
			return nil, errors.New("内置技能商店文件不存在或过大")
		}
		return data, nil
	}
	if source != "online" {
		return nil, errors.New("技能商店来源无效")
	}
	// 仅访问固定公开仓库，不使用用户地址，也不跟随重定向到其他主机。
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, skillStoreBaseURL+filename, nil)
	if err != nil {
		return nil, errors.New("无法构造技能商店请求")
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.New("在线技能商店暂不可用，请使用内置商店或稍后重试")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("在线技能商店文件不可用，请确认仓库目录已发布")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, errors.New("技能商店文件读取失败或超过大小限制")
	}
	return data, nil
}

// skillStoreMainFiles 优先识别技能入口，缺少标准入口时兼容 Markdown 单文件。
func skillStoreMainFiles(record skillStoreRecord) []string {
	mains, markdown := []string{}, []string{}
	for _, file := range record.Files {
		if strings.EqualFold(path.Ext(file.Path), ".md") {
			markdown = append(markdown, file.Path)
		}
		if strings.EqualFold(path.Base(file.Path), "SKILL.md") {
			mains = append(mains, file.Path)
		}
	}
	if len(mains) == 0 {
		mains = markdown
	}
	sort.Strings(mains)
	return mains
}
