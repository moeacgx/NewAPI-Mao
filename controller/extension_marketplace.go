package controller

import (
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/extension"
	"github.com/gin-gonic/gin"
)

const ExtensionMarketplaceCatalogURL = "https://raw.githubusercontent.com/moeacgx/maolaonewapi-extensions/main/catalog.json"
const ExtensionMarketplaceRepositoryURL = "https://github.com/moeacgx/maolaonewapi-extensions"

var extensionMarketplaceIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
var extensionMarketplaceVersionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]{0,127}$`)

func GetExtensionMarketplace(c *gin.Context) {
	if !requireExtensionRoot(c) {
		return
	}
	c.Header("Cache-Control", "no-store")
	common.ApiSuccess(c, gin.H{
		"catalog_url": ExtensionMarketplaceCatalogURL, "repository_url": ExtensionMarketplaceRepositoryURL,
		"host_version": common.Version, "max_archive_bytes": extension.MaxInstallArchiveBytes,
	})
}

// 只接受 multipart 内完整的固定来源元数据；服务器不请求目录或归档 URL。
func extensionMarketplaceUploadExpectation(c *gin.Context) (*extension.ArchiveExpectation, error) {
	fields := []string{"archiveSha256", "expectedId", "expectedVersion", "catalogUrl", "archivePath"}
	values := make(map[string]string, len(fields))
	provided := 0
	for _, field := range fields {
		if len(c.Request.MultipartForm.File[field]) > 0 {
			return nil, errors.New("marketplace metadata must use text fields")
		}
		items, ok := c.Request.MultipartForm.Value[field]
		if !ok {
			continue
		}
		provided++
		if len(items) != 1 || items[0] == "" {
			return nil, errors.New("marketplace metadata must contain exactly one non-empty value per field")
		}
		values[field] = items[0]
	}
	if provided == 0 {
		return nil, nil
	}
	if provided != len(fields) {
		return nil, errors.New("marketplace metadata fields must be supplied together")
	}
	hash, err := hex.DecodeString(values["archiveSha256"])
	if err != nil || len(hash) != 32 {
		return nil, errors.New("marketplace archive checksum is invalid")
	}
	id, version := values["expectedId"], values["expectedVersion"]
	if !extensionMarketplaceIDPattern.MatchString(id) || !extensionMarketplaceVersionPattern.MatchString(version) || strings.Contains(version, "..") {
		return nil, errors.New("marketplace module identity or version is invalid")
	}
	if values["catalogUrl"] != ExtensionMarketplaceCatalogURL {
		return nil, errors.New("marketplace catalog source is not supported")
	}
	if values["archivePath"] != fmt.Sprintf("published/%s/%s/%s-%s.zip", id, version, id, version) {
		return nil, errors.New("marketplace archive path does not match expected identity and version")
	}
	return &extension.ArchiveExpectation{ArchiveSHA256: values["archiveSha256"], ExpectedID: id, ExpectedVersion: version}, nil
}
