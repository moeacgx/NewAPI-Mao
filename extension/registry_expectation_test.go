package extension

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagerInstallArchiveWithExpectationPreservesExistingOnMismatch(t *testing.T) {
	for _, test := range []struct{ name, hash, id, version string }{
		{name: "字节摘要", hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{name: "清单身份", id: "another-module"},
		{name: "清单版本", version: "9.9.9"},
	} {
		t.Run(test.name, func(t *testing.T) {
			manager := NewManager(t.TempDir())
			archive := buildModuleArchiveWithManifest(t, Manifest{ID: "market-demo", Name: "已安装模块", Version: "1.0.0", Runtime: Runtime{BaseURL: "https://example.com"}})
			_, err := manager.InstallArchive(bytes.NewReader(archive), int64(len(archive)))
			require.NoError(t, err)
			_, err = manager.SetEnabled("market-demo", true)
			require.NoError(t, err)
			manifestPath := filepath.Join(manager.RootDir(), "market-demo", "manifest.json")
			beforeManifest, err := os.ReadFile(manifestPath)
			require.NoError(t, err)
			beforeState, err := os.ReadFile(manager.statePath())
			require.NoError(t, err)
			update := buildModuleArchiveWithManifest(t, Manifest{ID: "market-demo", Name: "新版模块", Version: "2.0.0", Runtime: Runtime{BaseURL: "https://example.com"}})
			expectation := ArchiveExpectation{ArchiveSHA256: fmt.Sprintf("%x", sha256.Sum256(update)), ExpectedID: "market-demo", ExpectedVersion: "2.0.0"}
			if test.hash != "" {
				expectation.ArchiveSHA256 = test.hash
			}
			if test.id != "" {
				expectation.ExpectedID = test.id
			}
			if test.version != "" {
				expectation.ExpectedVersion = test.version
			}
			_, err = manager.InstallArchiveWithExpectation(bytes.NewReader(update), int64(len(update)), expectation)
			require.Error(t, err)
			afterManifest, err := os.ReadFile(manifestPath)
			require.NoError(t, err)
			afterState, err := os.ReadFile(manager.statePath())
			require.NoError(t, err)
			assert.Equal(t, beforeManifest, afterManifest)
			assert.Equal(t, beforeState, afterState)
			installed, ok := manager.Get("market-demo")
			require.True(t, ok)
			assert.True(t, installed.Enabled)
			assert.Equal(t, "1.0.0", installed.Version)
			_, foundUnexpected := manager.Get("another-module")
			assert.False(t, foundUnexpected)
		})
	}
}

func TestManagerInstallArchiveWithExpectationInstallsDisabledAndPreservesEnabledUpdate(t *testing.T) {
	manager := NewManager(t.TempDir())
	for _, version := range []string{"1.0.0", "2.0.0"} {
		archive := buildModuleArchiveWithManifest(t, Manifest{ID: "market-demo", Name: "市场模块", Version: version, Runtime: Runtime{BaseURL: "https://example.com"}})
		module, err := manager.InstallArchiveWithExpectation(bytes.NewReader(archive), int64(len(archive)), ArchiveExpectation{ArchiveSHA256: fmt.Sprintf("%x", sha256.Sum256(archive)), ExpectedID: "market-demo", ExpectedVersion: version})
		require.NoError(t, err)
		assert.Equal(t, version, module.Version)
		assert.Equal(t, version == "2.0.0", module.Enabled)
		if version == "1.0.0" {
			_, err = manager.SetEnabled(module.ID, true)
			require.NoError(t, err)
		}
	}
}
