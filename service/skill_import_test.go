package service

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skillArchiveFixture(t *testing.T, names, contents []string, symlink bool) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for i, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		if symlink {
			header.SetMode(os.ModeSymlink | 0o777)
		}
		file, err := writer.CreateHeader(header)
		require.NoError(t, err)
		_, err = file.Write([]byte(contents[i]))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return buf.Bytes()
}

func TestSkillArchiveImportsOrderedSkillsAsUnpublishedDraft(t *testing.T) {
	data := skillArchiveFixture(t,
		[]string{"kit/z/SKILL.md", "kit/a/SKILL.md", "kit/scripts/run.sh", "kit/README.md"},
		[]string{"---\nname: 核验\ndescription: 检查结论\n---\n核验事实。", "---\nname: 提纲\n---\n先写提纲。", "echo private-script", "不应装载的说明"}, false)
	draft, err := ParseSkillArchive(data, "研究合集.zip")
	require.NoError(t, err)
	assert.Equal(t, "bundle", draft.Kind)
	assert.Equal(t, "研究合集", draft.Title)
	require.Len(t, draft.Skills, 2)
	assert.Equal(t, "提纲", draft.Skills[0].Title)
	assert.Equal(t, "先写提纲。", draft.Skills[0].Content)
	assert.Equal(t, "核验", draft.Skills[1].Title)
	assert.Equal(t, "## 技能：提纲\n先写提纲。\n\n## 技能：核验\n核验事实。", draft.Content)
	assert.False(t, draft.Enabled)
	assert.Empty(t, draft.GroupIDs)
	assert.Zero(t, draft.ID)
	assert.Zero(t, draft.CategoryID)
	assert.NotContains(t, draft.Content, "private-script")
	assert.NotContains(t, draft.Content, "不应装载")
}

func TestSkillArchiveSupportsSingleMarkdownAndStripsFrontMatter(t *testing.T) {
	data := skillArchiveFixture(t, []string{"assistant.md"}, []string{"\ufeff---\r\nname: 写作助理\r\ndescription: 帮助清晰表达\r\n---\r\n保持事实准确。"}, false)
	draft, err := ParseSkillArchive(data, "assistant.zip")
	require.NoError(t, err)
	assert.Equal(t, "prompt", draft.Kind)
	assert.Equal(t, "写作助理", draft.Title)
	assert.Equal(t, "帮助清晰表达", draft.Description)
	assert.Equal(t, "保持事实准确。", draft.Content)
	assert.Empty(t, draft.Skills)
}

func TestSkillArchiveReportsFailingFileAndBodySize(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
	}{
		{"空正文", "---\nname: 审查\n---\n \n", "提示词正文不能为空"},
		{"超限正文", strings.Repeat("字", 10923), "提示词正文为 32769 字节，超过 32768 字节上限"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := skillArchiveFixture(t, []string{"skills/a/SKILL.md", "skills/b/SKILL.md", "README.md", "scripts/run.sh"}, []string{"有效正文", tc.body, "普通说明", "exit 1"}, false)
			draft, err := ParseSkillArchive(data, "skills.zip")
			require.EqualError(t, err, "技能文件 \"skills/b/SKILL.md\"："+tc.want)
			assert.Nil(t, draft)
		})
	}
}

func TestSkillArchiveRejectsUnsafeOrUnboundedContent(t *testing.T) {
	for _, tc := range []struct {
		name     string
		paths    []string
		contents []string
		symlink  bool
	}{
		{name: "路径穿越", paths: []string{"../SKILL.md"}, contents: []string{"正文"}},
		{name: "绝对路径", paths: []string{"/SKILL.md"}, contents: []string{"正文"}},
		{name: "Windows路径", paths: []string{"C:\\SKILL.md"}, contents: []string{"正文"}},
		{name: "符号链接", paths: []string{"SKILL.md"}, contents: []string{"正文"}, symlink: true},
		{name: "解压上限包含非文本文件", paths: []string{"SKILL.md", "assets/image.bin"}, contents: []string{"正文", strings.Repeat("a", 512*1024)}},
		{name: "正文上限", paths: []string{"SKILL.md"}, contents: []string{strings.Repeat("a", 32769)}},
		{name: "无文本", paths: []string{"run.sh"}, contents: []string{"echo no"}},
		{name: "重名路径", paths: []string{"SKILL.md", "SKILL.md"}, contents: []string{"正文", "正文"}},
		{name: "无效UTF8", paths: []string{"SKILL.md"}, contents: []string{string([]byte{0xff, 0xfe})}},
		{name: "损坏元数据", paths: []string{"SKILL.md"}, contents: []string{"---\nname: [\n---\n正文"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := skillArchiveFixture(t, tc.paths, tc.contents, tc.symlink)
			_, err := ParseSkillArchive(data, "test.zip")
			require.Error(t, err)
		})
	}
	_, err := ParseSkillArchive([]byte("broken"), "test.zip")
	require.Error(t, err)
	_, err = ParseSkillArchive(make([]byte, 2*1024*1024+1), "large.zip")
	require.Error(t, err)
}

func TestSkillArchiveRejectsIncompleteBundlesAndCorruptContent(t *testing.T) {
	for _, test := range []struct {
		name    string
		count   int
		content string
		message string
	}{
		{"技能超限不能静默截断", 17, "正文", "1 至 16"},
		{"条目超限", 129, "正文", "128"},
		{"单项合法但合集总正文超限", 2, strings.Repeat("a", 16384), "32768"},
	} {
		t.Run(test.name, func(t *testing.T) {
			names, contents := make([]string, test.count), make([]string, test.count)
			for i := range names {
				names[i] = fmt.Sprintf("skill-%03d/SKILL.md", i)
				contents[i] = test.content
			}
			_, err := ParseSkillArchive(skillArchiveFixture(t, names, contents, false), "kit.zip")
			require.ErrorContains(t, err, test.message)
		})
	}
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	file, err := writer.CreateHeader(&zip.FileHeader{Name: "SKILL.md", Method: zip.Store})
	require.NoError(t, err)
	_, err = file.Write([]byte("unique-crc-skill-body"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	data := buf.Bytes()
	index := bytes.Index(data, []byte("unique-crc-skill-body"))
	require.GreaterOrEqual(t, index, 0)
	data[index] = 'X'
	_, err = ParseSkillArchive(data, "corrupt.zip")
	require.ErrorContains(t, err, "损坏")
}
