package extension

import (
	"archive/zip"
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpstreamModelGuardExternalInstallLifecycle(t *testing.T) {
	previous, previousVersion := DefaultManager, common.Version
	common.Version = "v1.0.0-rc.10.1.10.326"
	t.Setenv("EXTENSIONS_ROOT", t.TempDir())
	t.Cleanup(func() { DefaultManager, common.Version = previous, previousVersion })
	require.NoError(t, Init())
	_, installed := DefaultManager.Get(service.UpstreamModelGuardModuleID)
	assert.False(t, installed, "启动不能自动安装外置插件")
	assert.False(t, service.IsUpstreamModelGuardModuleEnabled())

	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	root := "../extensions/upstream-model-guard"
	for _, name := range []string{"manifest.json", "public/compat.html"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		require.NoError(t, err)
		entry, err := writer.Create(name)
		require.NoError(t, err)
		_, err = entry.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, fs.WalkDir(os.DirFS(root), "public/native", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return err
		}
		output, err := writer.Create(name)
		if err != nil {
			return err
		}
		_, err = output.Write(data)
		return err
	}))
	require.NoError(t, writer.Close())
	module, err := DefaultManager.InstallArchive(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	require.NoError(t, err)
	assert.False(t, module.Enabled)
	assert.False(t, service.IsUpstreamModelGuardModuleEnabled())
	_, err = DefaultManager.SetEnabled(module.ID, true)
	require.NoError(t, err)
	assert.True(t, service.IsUpstreamModelGuardModuleEnabled())
	_, err = DefaultManager.ResolveNotificationEvent(module.ID, "channel_disabled", common.RoleRootUser)
	require.NoError(t, err)
	_, err = DefaultManager.SetEnabled(module.ID, false)
	require.NoError(t, err)
	assert.False(t, service.IsUpstreamModelGuardModuleEnabled())
	_, err = DefaultManager.ResolveNotificationEvent(module.ID, "channel_disabled", common.RoleRootUser)
	require.Error(t, err)
	require.NoError(t, DefaultManager.Uninstall(module.ID))
	require.NoError(t, Init())
	_, installed = DefaultManager.Get(module.ID)
	assert.False(t, installed, "卸载后重启不能重新安装")
	assert.False(t, service.IsUpstreamModelGuardModuleEnabled())

	// 重新安装依然关闭，不能复用先前的启用状态。
	module, err = DefaultManager.InstallArchive(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	require.NoError(t, err)
	assert.False(t, module.Enabled)
}

func TestUpstreamModelGuardCapabilityRejectsWrongModuleOrRole(t *testing.T) {
	data, err := os.ReadFile("../extensions/upstream-model-guard/manifest.json")
	require.NoError(t, err)
	for _, scenario := range []struct{ name, id, role string }{
		{"普通管理员", "upstream-model-guard", "admin"},
		{"其他模块", "different-module", "root"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			var manifest Manifest
			require.NoError(t, common.Unmarshal(data, &manifest))
			manifest.ID = scenario.id
			manifest.Permissions.Roles = []string{scenario.role}
			require.ErrorContains(t, manifest.Validate(), "channel.upstream-model-guard requires")
		})
	}
}

func TestUpstreamModelGuardToleranceCapabilityRequiresGuardIdentity(t *testing.T) {
	data, err := os.ReadFile("../extensions/upstream-model-guard/manifest.json")
	require.NoError(t, err)
	var manifest Manifest
	require.NoError(t, common.Unmarshal(data, &manifest))
	manifest.Permissions.Capabilities = []string{CapabilityUINative, CapabilityNotificationEventsPublish, "channel.upstream-model-guard-tolerance"}
	require.NoError(t, manifest.Validate())
	manifest.ID = "unrelated-module"
	require.Error(t, manifest.Validate())
	manifest.ID = "upstream-model-guard"
	manifest.Permissions.Roles = []string{"admin"}
	require.Error(t, manifest.Validate())
}
