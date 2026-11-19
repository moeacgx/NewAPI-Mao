package service

import (
	"bytes"
	"encoding/base64"
	"errors"
	"mime/multipart"
	"regexp"

	"github.com/QuantumNous/new-api/common"
)

const MCPMaxUploadBytes = 8 * 1024 * 1024

var mcpUploadFilename = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

type mcpUploadInput struct {
	Filename        string `json:"filename"`
	ContentBase64   string `json:"content_base64"`
	ArchiveSHA256   string `json:"archiveSha256,omitempty"`
	ExpectedID      string `json:"expectedId,omitempty"`
	ExpectedVersion string `json:"expectedVersion,omitempty"`
	CatalogURL      string `json:"catalogUrl,omitempty"`
	ArchivePath     string `json:"archivePath,omitempty"`
}

func mcpUploadSchema() map[string]any {
	return map[string]any{
		"type": "object", "additionalProperties": false,
		"required": []string{"filename", "content_base64"},
		"properties": map[string]any{
			"filename":       map[string]any{"type": "string", "minLength": 1, "maxLength": 128, "pattern": mcpUploadFilename.String()},
			"content_base64": map[string]any{"type": "string", "minLength": 4, "maxLength": MCPMaxUploadBytes - 4096, "description": "文件完整内容的标准 Base64 编码；不接受服务器文件路径或下载 URL。"},
		},
	}
}

// mcpUploadBody 仅构建目录声明的单个文件字段，保留宿主文件类型、解包与权限校验。
func mcpUploadBody(field string, raw []byte) ([]byte, string, error) {
	var input mcpUploadInput
	if common.Unmarshal(raw, &input) != nil || !mcpUploadFilename.MatchString(input.Filename) {
		return nil, "", errors.New("invalid upload filename")
	}
	fields := []struct{ name, value string }{
		{"archiveSha256", input.ArchiveSHA256}, {"expectedId", input.ExpectedID},
		{"expectedVersion", input.ExpectedVersion}, {"catalogUrl", input.CatalogURL},
		{"archivePath", input.ArchivePath},
	}
	provided := 0
	for _, item := range fields {
		if item.value != "" {
			provided++
		}
	}
	if provided != 0 && provided != len(fields) {
		return nil, "", errors.New("marketplace upload metadata must be complete")
	}
	content, err := base64.StdEncoding.Strict().DecodeString(input.ContentBase64)
	if err != nil || len(content) == 0 || len(content) > 6*1024*1024 {
		return nil, "", errors.New("invalid or oversized base64 upload")
	}
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	part, err := writer.CreateFormFile(field, input.Filename)
	if err != nil {
		return nil, "", err
	}
	if _, err = part.Write(content); err != nil {
		return nil, "", err
	}
	if provided != 0 {
		for _, item := range fields {
			if err = writer.WriteField(item.name, item.value); err != nil {
				return nil, "", err
			}
		}
	}
	if err = writer.Close(); err != nil {
		return nil, "", err
	}
	return buffer.Bytes(), writer.FormDataContentType(), nil
}
