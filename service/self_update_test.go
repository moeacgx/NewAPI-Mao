package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelfUpdateAssetNamesForLinux(t *testing.T) {
	tests := []struct {
		name         string
		goarch       string
		wantAsset    string
		wantChecksum string
	}{
		{
			name:         "amd64",
			goarch:       "amd64",
			wantAsset:    "new-api-v1.0.0-rc.10",
			wantChecksum: "checksums-linux.txt",
		},
		{
			name:         "arm64",
			goarch:       "arm64",
			wantAsset:    "new-api-arm64-v1.0.0-rc.10",
			wantChecksum: "checksums-linux.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAsset, gotChecksum, err := selfUpdateAssetNames("linux", tt.goarch, "v1.0.0-rc.10")
			if err != nil {
				t.Fatalf("selfUpdateAssetNames returned error: %v", err)
			}
			if gotAsset != tt.wantAsset {
				t.Fatalf("asset = %q, want %q", gotAsset, tt.wantAsset)
			}
			if gotChecksum != tt.wantChecksum {
				t.Fatalf("checksum = %q, want %q", gotChecksum, tt.wantChecksum)
			}
		})
	}
}

func TestSelfUpdateAssetNamesRejectsUnsupportedArch(t *testing.T) {
	_, _, err := selfUpdateAssetNames("linux", "386", "v1.0.0-rc.10")
	if err == nil {
		t.Fatal("expected unsupported architecture error")
	}
}

func TestChecksumForAssetParsesSha256Manifest(t *testing.T) {
	manifest := "" +
		"56186c6e6b7493f9cc54b2297c65a363568fe6328e5d1ebe1db6b2e855a10089  new-api-v1.0.0-rc.10\n" +
		"b72df122b0f09ae9b172d6cae36bbc1ee2380aa7660c325c1a974bbb9ef1e436 *new-api-arm64-v1.0.0-rc.10\n"

	got, err := checksumForAsset(manifest, "new-api-arm64-v1.0.0-rc.10")
	if err != nil {
		t.Fatalf("checksumForAsset returned error: %v", err)
	}
	if got != "b72df122b0f09ae9b172d6cae36bbc1ee2380aa7660c325c1a974bbb9ef1e436" {
		t.Fatalf("checksum = %q", got)
	}
}

func TestSelfUpdateRepoDefaultsToNewAPIMao(t *testing.T) {
	t.Setenv("SELF_UPDATE_REPO", "")
	t.Setenv("SELF_UPDATE_GITHUB_REPO", "")

	if got := selfUpdateRepo(); got != "moeacgx/NewAPI-Mao" {
		t.Fatalf("default repo = %q, want moeacgx/NewAPI-Mao", got)
	}
}
func TestValidateSelfUpdateRepo(t *testing.T) {
	if err := validateSelfUpdateRepo("moeacgx/NewAPI-Mao"); err != nil {
		t.Fatalf("expected repo to be valid: %v", err)
	}
	if err := validateSelfUpdateRepo("https://github.com/moeacgx/NewAPI-Mao"); err == nil {
		t.Fatal("expected full URL repo to be rejected")
	}
}

func TestReleaseAssetDownloadURLPrefersAPIAssetURL(t *testing.T) {
	asset := GitHubReleaseAsset{
		Name:               "new-api-v1.0.0-rc.10",
		URL:                "https://api.github.com/repos/moeacgx/NewAPI-Mao/releases/assets/123",
		BrowserDownloadURL: "https://github.com/moeacgx/NewAPI-Mao/releases/download/v1.0.0-rc.10/new-api-v1.0.0-rc.10",
	}

	if got := releaseAssetDownloadURL(asset); got != asset.URL {
		t.Fatalf("download URL = %q, want %q", got, asset.URL)
	}
}

func TestValidateGitHubDownloadURLAcceptsReleaseAssetAPIURL(t *testing.T) {
	err := validateGitHubDownloadURL("moeacgx/NewAPI-Mao", "https://api.github.com/repos/moeacgx/NewAPI-Mao/releases/assets/123")
	if err != nil {
		t.Fatalf("expected GitHub asset API URL to be valid: %v", err)
	}
}

func TestSelfUpdateConfiguredRepositoryAssetValidation(t *testing.T) {
	tests := []struct {
		name      string
		primary   string
		fallback  string
		wantRepo  string
		assetURL  string
		wantError bool
	}{
		{
			name:    "旧主配置接受规范 API 资产地址",
			primary: "moeacgx/maolaonewapi", fallback: "other/project",
			wantRepo: "moeacgx/NewAPI-Mao",
			assetURL: "https://api.github.com/repos/moeacgx/NewAPI-Mao/releases/assets/123",
		},
		{
			name:    "旧备用配置接受规范浏览器下载地址",
			primary: " ", fallback: " moeacgx/maolaonewapi ",
			wantRepo: "moeacgx/NewAPI-Mao",
			assetURL: "https://github.com/moeacgx/NewAPI-Mao/releases/download/v1/new-api-v1",
		},
		{
			name:     "小写新配置接受规范 API 资产地址",
			primary:  "moeacgx/newapi-mao",
			wantRepo: "moeacgx/NewAPI-Mao",
			assetURL: "https://api.github.com/repos/moeacgx/NewAPI-Mao/releases/assets/123",
		},
		{
			name:    "主配置优先且其他仓库仍能下载自身资产",
			primary: "other/project", fallback: "moeacgx/maolaonewapi",
			wantRepo: "other/project",
			assetURL: "https://api.github.com/repos/other/project/releases/assets/123",
		},
		{
			name:    "其他仓库不接受本项目规范地址",
			primary: "other/project", fallback: "moeacgx/maolaonewapi",
			wantRepo: "other/project",
			assetURL: "https://api.github.com/repos/moeacgx/NewAPI-Mao/releases/assets/123", wantError: true,
		},
		{
			name:    "旧配置拒绝同作者其他仓库",
			primary: "moeacgx/maolaonewapi", wantRepo: "moeacgx/NewAPI-Mao",
			assetURL: "https://api.github.com/repos/moeacgx/other/releases/assets/123", wantError: true,
		},
		{
			name:    "旧配置拒绝相似前缀仓库",
			primary: "moeacgx/maolaonewapi", wantRepo: "moeacgx/NewAPI-Mao",
			assetURL: "https://github.com/moeacgx/NewAPI-Mao-untrusted/releases/download/v1/new-api-v1", wantError: true,
		},
		{
			name:    "同名旧仓库的其他作者不被映射",
			primary: "other/maolaonewapi", wantRepo: "other/maolaonewapi",
			assetURL: "https://api.github.com/repos/moeacgx/NewAPI-Mao/releases/assets/123", wantError: true,
		},
		{
			name:    "别名不绕过 HTTPS 要求",
			primary: "moeacgx/maolaonewapi", wantRepo: "moeacgx/NewAPI-Mao",
			assetURL: "http://api.github.com/repos/moeacgx/NewAPI-Mao/releases/assets/123", wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SELF_UPDATE_REPO", tt.primary)
			t.Setenv("SELF_UPDATE_GITHUB_REPO", tt.fallback)
			repo := selfUpdateRepo()
			require.NoError(t, validateSelfUpdateRepo(repo))
			assert.Equal(t, tt.wantRepo, repo)
			err := validateGitHubDownloadURL(repo, tt.assetURL)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
