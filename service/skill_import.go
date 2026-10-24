package service

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/model"
	"gopkg.in/yaml.v3"
)

const MaxSkillArchiveBytes = 16 * 1024 * 1024
const maxSkillExpandedBytes = model.MaxSkillPackageBytes
const maxSkillArchiveEntries = model.MaxSkillResourceFiles

// ParseSkillArchive 只在内存读取技能与引用文本，不落盘，也不运行包内脚本。
func ParseSkillArchive(data []byte, filename string) (*model.PromptTemplate, error) {
	if len(data) == 0 || len(data) > MaxSkillArchiveBytes {
		return nil, errors.New("技能压缩包不能超过 16 MiB")
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, errors.New("无法读取 ZIP 技能压缩包")
	}
	if len(reader.File) > maxSkillArchiveEntries {
		return nil, errors.New("技能压缩包不能超过 4096 个条目")
	}
	var markdown, skills []string
	files := make(map[string]*zip.File)
	seen := make(map[string]bool, len(reader.File))
	var expanded uint64
	for _, file := range reader.File {
		name := strings.TrimSuffix(file.Name, "/")
		if !fs.ValidPath(name) || strings.ContainsAny(name, "\\:\x00") || !utf8.ValidString(name) || len(name) > 1024 || seen[name] {
			return nil, errors.New("技能压缩包包含无效或重复路径")
		}
		seen[name] = true
		if file.Mode()&fs.ModeSymlink != 0 || (!file.FileInfo().IsDir() && !file.Mode().IsRegular()) {
			return nil, errors.New("技能压缩包不能包含符号链接或特殊文件")
		}
		if file.UncompressedSize64 > maxSkillExpandedBytes || expanded > maxSkillExpandedBytes-file.UncompressedSize64 {
			return nil, errors.New("技能压缩包解压内容不能超过 64 MiB")
		}
		expanded += file.UncompressedSize64
		if file.FileInfo().IsDir() || strings.HasPrefix(name, "__MACOSX/") {
			continue
		}
		files[name] = file
		if strings.EqualFold(path.Ext(name), ".md") {
			markdown = append(markdown, name)
			if strings.EqualFold(path.Base(name), "SKILL.md") {
				skills = append(skills, name)
			}
		}
	}
	if len(skills) > 0 {
		markdown = skills
	}
	if len(markdown) == 0 || len(markdown) > model.MaxProgressiveSkills {
		return nil, errors.New("技能压缩包需要包含 1 至 128 个技能 Markdown 文件")
	}
	sort.Strings(markdown)
	textFiles := make(map[string][]byte)
	var readTotal int
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		file := files[name]
		if !supportedSkillTextFile(name) {
			continue
		}
		owner := skillFileOwner(name, markdown)
		if owner < 0 {
			continue
		}
		if file.UncompressedSize64 > model.MaxSkillFileBytes {
			return nil, fmt.Errorf("技能文件 %q 超过 8 MiB 文件上限", name)
		}
		stream, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("无法读取技能文件 %q", name)
		}
		content, readErr := io.ReadAll(io.LimitReader(stream, model.MaxSkillFileBytes+1))
		closeErr := stream.Close()
		if readErr != nil || closeErr != nil || len(content) > model.MaxSkillFileBytes {
			return nil, fmt.Errorf("技能文件 %q 损坏或超出大小限制", name)
		}
		readTotal += len(content)
		if readTotal > maxSkillExpandedBytes {
			return nil, errors.New("技能包文本合计不能超过 64 MiB")
		}
		if !utf8.Valid(content) || bytes.ContainsRune(content, 0) {
			return nil, fmt.Errorf("技能文件 %q 必须为 UTF-8 文本", name)
		}
		textFiles[name] = content
	}
	return buildProgressiveSkillDraft(strings.TrimSuffix(path.Base(strings.ReplaceAll(filename, "\\", "/")), path.Ext(filename)), markdown, textFiles)
}

// skillFileOwner 使用最深的技能根目录，防止父技能顺便读取嵌套技能正文。
func skillFileOwner(filename string, mains []string) int {
	owner, depth := -1, -1
	for i, main := range mains {
		if filename == main {
			return i
		}
		root := path.Dir(main)
		if root != "." && !strings.HasPrefix(filename, root+"/") {
			continue
		}
		if len(root) > depth {
			owner, depth = i, len(root)
		}
	}
	return owner
}

func supportedSkillTextFile(filename string) bool {
	switch strings.ToLower(path.Ext(filename)) {
	case ".md", ".txt", ".json", ".yaml", ".yml", ".csv", ".tsv", ".xml", ".rst", ".toml", ".ini", ".cfg", ".html", ".htm", ".tex":
		return true
	}
	return false
}

func buildProgressiveSkillDraft(title string, mains []string, files map[string][]byte) (*model.PromptTemplate, error) {
	items := make([]model.PromptSkill, 0, len(mains))
	mainSet := make(map[string]bool, len(mains))
	for _, filename := range mains {
		mainSet[filename] = true
	}
	names := make([]string, 0, len(files))
	for filename := range files {
		names = append(names, filename)
	}
	sort.Strings(names)
	for index, filename := range mains {
		item, _, err := parseSkillMarkdown(files[filename], filename)
		if err != nil {
			return nil, fmt.Errorf("技能文件 %q：%w", filename, err)
		}
		root := path.Dir(filename)
		for _, name := range names {
			if mainSet[name] || skillFileOwner(name, mains) != index {
				continue
			}
			relative := name
			if root != "." {
				relative = strings.TrimPrefix(name, root+"/")
			}
			item.Resources = append(item.Resources, model.PromptSkillResource{Path: relative, Content: string(files[name])})
		}
		items = append(items, item)
	}
	if title == "" {
		title = "技能合集"
	}
	draft := &model.PromptTemplate{Title: title, Kind: "bundle", LoadMode: model.PromptLoadModeProgressive, Skills: items, GroupIDs: []int{}}
	if len(items) == 1 {
		draft.Kind, draft.Title, draft.Content, draft.Description = "prompt", items[0].Title, items[0].Content, items[0].Description
	}
	if err := model.NormalizePromptTemplateContent(draft); err != nil {
		return nil, err
	}
	return draft, nil
}

func parseSkillMarkdown(data []byte, filename string) (model.PromptSkill, string, error) {
	var item model.PromptSkill
	if !utf8.Valid(data) || bytes.ContainsRune(data, 0) {
		return item, "", errors.New("技能 Markdown 必须为 UTF-8 文本")
	}
	text := strings.TrimPrefix(strings.ReplaceAll(string(data), "\r\n", "\n"), "\ufeff")
	metadata := struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}{}
	if strings.HasPrefix(text, "---\n") {
		end := strings.Index(text[4:], "\n---\n")
		if end < 0 {
			return item, "", errors.New("技能 Markdown 元数据缺少结束标记")
		}
		if err := yaml.Unmarshal([]byte(text[4:4+end]), &metadata); err != nil {
			return item, "", errors.New("技能 Markdown 元数据格式无效")
		}
		text = text[4+end+5:]
	}
	item.Title = strings.TrimSpace(metadata.Name)
	if item.Title == "" {
		item.Title = strings.TrimSuffix(path.Base(filename), path.Ext(filename))
		if strings.EqualFold(item.Title, "SKILL") {
			item.Title = path.Base(path.Dir(filename))
			if item.Title == "." {
				item.Title = "技能"
			}
		}
		if heading, _, _ := strings.Cut(text, "\n"); strings.HasPrefix(heading, "# ") {
			item.Title = strings.TrimSpace(strings.TrimPrefix(heading, "# "))
		}
	}
	item.Content, item.Description, item.Path = strings.TrimSpace(text), strings.TrimSpace(metadata.Description), filename
	// 用途说明按原文保存，不做截断。显示名称受站内 80 字符列宽限制，超长时完整名称保留在按需读取的主文件中；
	// 元数据不能替代原本为空的正文，合并后的文本继续接受文件和包总量校验。
	if titleRunes := []rune(item.Title); len(titleRunes) > 80 {
		item.Title = string(titleRunes[:79]) + "…"
		if item.Content != "" {
			item.Content = "完整技能名称：" + string(titleRunes) + "\n\n" + item.Content
		}
	}
	check := &model.PromptTemplate{Title: item.Title, Description: item.Description, Content: item.Content, Kind: "prompt", LoadMode: model.PromptLoadModeProgressive, Skills: []model.PromptSkill{item}}
	if err := model.NormalizePromptTemplateContent(check); err != nil {
		return item, "", err
	}
	item = check.Skills[0]
	return item, item.Description, nil
}
