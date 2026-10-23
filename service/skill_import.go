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

const MaxSkillArchiveBytes = 2 * 1024 * 1024
const maxSkillExpandedBytes = 512 * 1024
const maxSkillArchiveEntries = 128

// ParseSkillArchive 只在内存读取技能文本，不解压到磁盘，不运行包内脚本或附件。
func ParseSkillArchive(data []byte, filename string) (*model.PromptTemplate, error) {
	if len(data) == 0 || len(data) > MaxSkillArchiveBytes {
		return nil, errors.New("技能压缩包不能超过 2 MiB")
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, errors.New("无法读取 ZIP 技能压缩包")
	}
	if len(reader.File) > maxSkillArchiveEntries {
		return nil, errors.New("技能压缩包不能超过 128 个条目")
	}
	var markdown, skills []*zip.File
	seen := make(map[string]bool, len(reader.File))
	var expanded uint64
	for _, file := range reader.File {
		name := strings.TrimSuffix(file.Name, "/")
		if !fs.ValidPath(name) || strings.ContainsAny(name, "\\:\x00") || !utf8.ValidString(name) || seen[name] {
			return nil, errors.New("技能压缩包包含无效或重复路径")
		}
		seen[name] = true
		if file.Mode()&fs.ModeSymlink != 0 || (!file.FileInfo().IsDir() && !file.Mode().IsRegular()) {
			return nil, errors.New("技能压缩包不能包含符号链接或特殊文件")
		}
		if file.UncompressedSize64 > maxSkillExpandedBytes || expanded > maxSkillExpandedBytes-file.UncompressedSize64 {
			return nil, errors.New("技能压缩包解压内容不能超过 512 KiB")
		}
		expanded += file.UncompressedSize64
		if file.FileInfo().IsDir() || strings.HasPrefix(name, "__MACOSX/") {
			continue
		}
		if strings.EqualFold(path.Ext(name), ".md") {
			markdown = append(markdown, file)
			if strings.EqualFold(path.Base(name), "SKILL.md") {
				skills = append(skills, file)
			}
		}
	}
	if len(skills) > 0 {
		markdown = skills
	}
	if len(markdown) == 0 || len(markdown) > 16 {
		return nil, errors.New("技能压缩包需要包含 1 至 16 个技能 Markdown 文件")
	}
	sort.Slice(markdown, func(i, j int) bool { return markdown[i].Name < markdown[j].Name })
	items := make([]model.PromptSkill, 0, len(markdown))
	var description string
	var readTotal int
	for _, file := range markdown {
		stream, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("无法读取技能文件 %q", file.Name)
		}
		content, readErr := io.ReadAll(io.LimitReader(stream, maxSkillExpandedBytes+1))
		closeErr := stream.Close()
		if readErr != nil || closeErr != nil || len(content) > maxSkillExpandedBytes {
			return nil, fmt.Errorf("技能文件 %q 损坏或超出大小限制", file.Name)
		}
		readTotal += len(content)
		if readTotal > maxSkillExpandedBytes {
			return nil, errors.New("技能压缩包解压内容不能超过 512 KiB")
		}
		item, info, err := parseSkillMarkdown(content, file.Name)
		if err != nil {
			return nil, fmt.Errorf("技能文件 %q：%w", file.Name, err)
		}
		items = append(items, item)
		description = info
	}
	title := strings.TrimSuffix(path.Base(strings.ReplaceAll(filename, "\\", "/")), path.Ext(filename))
	if title == "" {
		title = "技能合集"
	}
	draft := &model.PromptTemplate{Title: title, Kind: "bundle", Skills: items, GroupIDs: []int{}}
	if len(items) == 1 {
		draft.Kind, draft.Title, draft.Content, draft.Description = "prompt", items[0].Title, items[0].Content, description
		draft.Skills = nil
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
	item.Content = strings.TrimSpace(text)
	// 复用保存时的正文约束，让导入错误在打开编辑器前给出。
	check := &model.PromptTemplate{Title: item.Title, Description: strings.TrimSpace(metadata.Description), Content: item.Content, Kind: "prompt"}
	if err := model.NormalizePromptTemplateContent(check); err != nil {
		return item, "", err
	}
	item.Title = check.Title
	return item, check.Description, nil
}
