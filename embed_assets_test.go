package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrontendBuildEmbedsUnderscoreAssets(t *testing.T) {
	for _, test := range []struct {
		name      string
		diskRoot  string
		embedRoot string
		read      func(string) ([]byte, error)
	}{
		{name: "default", diskRoot: "web/dist", embedRoot: "web/dist", read: defaultBuildFS.ReadFile},
		{name: "classic", diskRoot: "web/classic/dist", embedRoot: "web/classic/dist", read: classicBuildFS.ReadFile},
	} {
		t.Run(test.name, func(t *testing.T) {
			var underscoreAssets []string
			err := filepath.WalkDir(test.diskRoot, func(path string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() || !strings.HasPrefix(entry.Name(), "_") {
					return nil
				}
				underscoreAssets = append(underscoreAssets, path)
				return nil
			})
			require.NoError(t, err)
			if len(underscoreAssets) == 0 {
				t.Skip("frontend build contains no underscore-prefixed assets")
			}

			for _, diskPath := range underscoreAssets {
				diskContent, err := os.ReadFile(diskPath)
				require.NoError(t, err)
				relativePath, err := filepath.Rel(test.diskRoot, diskPath)
				require.NoError(t, err)
				embeddedContent, err := test.read(filepath.ToSlash(filepath.Join(test.embedRoot, relativePath)))
				require.NoError(t, err, "underscore asset %s was omitted from embed.FS", diskPath)
				assert.Equal(t, diskContent, embeddedContent)
			}
		})
	}
}
