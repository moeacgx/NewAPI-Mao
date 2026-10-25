package model

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
)

// MaxPromptDescriptionBytes 只保护 TEXT 列与网关技能目录分页，不作为面向用户的字数限制。
const MaxPromptDescriptionBytes = 16 * 1024

var ErrPromptTemplateNotFound = errors.New("技能不存在或已被删除")

var promptRepositoryCommit = regexp.MustCompile(`^[a-f0-9]{40}$`)
var promptRepositoryHash = regexp.MustCompile(`^[a-f0-9]{64}$`)

// PromptRepositorySource 记录公开 GitHub 导入来源；LocallyModified 只在读取时根据当前内容计算。
type PromptRepositorySource struct {
	URL             string `json:"url"`
	Ref             string `json:"ref"`
	Path            string `json:"path"`
	Commit          string `json:"commit"`
	ImportedAt      int64  `json:"imported_at"`
	ContentHash     string `json:"content_hash"`
	LocallyModified bool   `json:"locally_modified"`
}

func normalizePromptRepositorySource(source *PromptRepositorySource) error {
	source.URL, source.Ref, source.Path = strings.TrimSpace(source.URL), strings.TrimSpace(source.Ref), strings.Trim(strings.TrimSpace(source.Path), "/")
	source.Commit, source.ContentHash = strings.ToLower(source.Commit), strings.ToLower(source.ContentHash)
	parsed, err := url.Parse(source.URL)
	if err != nil || len(source.URL) > 2048 || parsed.Scheme != "https" || parsed.Host != "github.com" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || strings.ContainsAny(source.URL, "#\\\x00") {
		return errors.New("技能来源必须是不含认证、端口、查询参数或片段的 GitHub HTTPS 链接")
	}
	if source.Ref == "" || len(source.Ref) > 255 || !fs.ValidPath(source.Ref) || !utf8.ValidString(source.Ref) || strings.ContainsAny(source.Ref, "\\\x00") {
		return errors.New("技能来源的分支、标签或提交无效")
	}
	if source.Path != "" && !validSkillPath(source.Path) {
		return errors.New("技能来源目录无效")
	}
	if !promptRepositoryCommit.MatchString(source.Commit) || !promptRepositoryHash.MatchString(source.ContentHash) || source.ImportedAt <= 0 {
		return errors.New("技能来源的提交、导入时间或内容指纹无效")
	}
	source.LocallyModified = false
	return nil
}

// PromptTemplateContentHash 计算 GitHub 更新会替换的内容指纹。
// 指纹忽略随机技能 ID；单技能的标题和说明属于站点设置，更新时保留，因此也不计入。
func PromptTemplateContentHash(template *PromptTemplate) (string, error) {
	type hashedResource struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	type hashedSkill struct {
		Path        string           `json:"path"`
		Title       string           `json:"title"`
		Description string           `json:"description"`
		Content     string           `json:"content"`
		Resources   []hashedResource `json:"resources"`
	}
	payload := struct {
		Kind     string        `json:"kind"`
		LoadMode string        `json:"load_mode"`
		Content  string        `json:"content"`
		Skills   []hashedSkill `json:"skills"`
	}{Kind: template.Kind, LoadMode: template.LoadMode, Skills: make([]hashedSkill, 0, len(template.Skills))}
	if template.LoadMode != PromptLoadModeProgressive {
		payload.Content = template.Content
	}
	for _, skill := range template.Skills {
		item := hashedSkill{Path: skill.Path, Content: skill.Content, Resources: make([]hashedResource, 0, len(skill.Resources))}
		if template.Kind == "bundle" {
			item.Title, item.Description = skill.Title, skill.Description
		}
		for _, resource := range skill.Resources {
			item.Resources = append(item.Resources, hashedResource{Path: resource.Path, Content: resource.Content})
		}
		sort.Slice(item.Resources, func(i, j int) bool { return item.Resources[i].Path < item.Resources[j].Path })
		payload.Skills = append(payload.Skills, item)
	}
	encoded, err := common.Marshal(payload)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

// hydratePromptRepositorySource 只用于已装配完整内容的管理员视图，读取时重新判断站内是否改过来源内容。
func hydratePromptRepositorySource(template *PromptTemplate) {
	template.RepositorySource = nil
	if template.RepositorySourceJSON == "" {
		return
	}
	var source PromptRepositorySource
	if err := common.UnmarshalJsonStr(template.RepositorySourceJSON, &source); err != nil {
		common.SysError("技能 GitHub 来源记录无法解析，模板 ID " + strconv.Itoa(template.ID) + "：" + err.Error())
		return
	}
	hash, err := PromptTemplateContentHash(template)
	if err != nil {
		common.SysError("技能内容指纹计算失败，模板 ID " + strconv.Itoa(template.ID) + "：" + err.Error())
		return
	}
	source.LocallyModified = hash != source.ContentHash
	template.RepositorySource = &source
}
