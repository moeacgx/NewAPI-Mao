package main

import (
	"embed"
	"io/fs"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFrontendBuildEmbedding 保护真实构建产物到二进制的完整资源合同。
func TestFrontendBuildEmbedding(t *testing.T) {
	for _, template := range []struct {
		name string
		root string
		data embed.FS
	}{
		{name: "Default", root: "web/dist", data: defaultBuildFS},
		{name: "Classic", root: "web/classic/dist", data: classicBuildFS},
	} {
		t.Run(template.name, func(t *testing.T) {
			require.DirExists(t, template.root, "请先生成真实前端构建产物")
			disk := os.DirFS(template.root)
			var diskFiles []string
			err := fs.WalkDir(disk, ".", func(name string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !entry.IsDir() {
					require.True(t, entry.Type().IsRegular(), "构建资源必须是普通文件：%s", name)
					diskFiles = append(diskFiles, name)
				}
				return nil
			})
			require.NoError(t, err)
			require.NotEmpty(t, diskFiles, "前端构建目录不能为空")
			require.Contains(t, diskFiles, "index.html")

			compiled, err := fs.Sub(template.data, template.root)
			require.NoError(t, err)
			var embeddedFiles []string
			err = fs.WalkDir(compiled, ".", func(name string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !entry.IsDir() {
					embeddedFiles = append(embeddedFiles, name)
				}
				return nil
			})
			require.NoError(t, err)
			assert.ElementsMatch(t, diskFiles, embeddedFiles, "磁盘与嵌入资源集合必须完全一致")

			for _, name := range diskFiles {
				want, err := fs.ReadFile(disk, name)
				require.NoError(t, err)
				got, err := fs.ReadFile(compiled, name)
				if assert.NoError(t, err, "构建资源必须嵌入：%s", name) {
					assert.Equal(t, want, got, "嵌入字节必须与磁盘一致：%s", name)
				}
			}
			t.Logf("磁盘 %d 文件，嵌入 %d 文件，已逐文件核对字节", len(diskFiles), len(embeddedFiles))
		})
	}
}
