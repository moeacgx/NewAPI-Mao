package controller

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func ImportSkillArchive(c *gin.Context) {
	// 流式解析 multipart，避免上传内容落入临时文件。
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.MaxSkillArchiveBytes+64*1024)
	reader, err := c.Request.MultipartReader()
	if err != nil {
		common.ApiErrorMsg(c, "请上传 ZIP 技能压缩包")
		return
	}
	var data []byte
	var filename string
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			common.ApiErrorMsg(c, "压缩包上传失败或超过大小限制")
			return
		}
		if data != nil || part.FormName() != "file" || part.FileName() == "" {
			part.Close()
			common.ApiErrorMsg(c, "每次只能上传一个 file 压缩包")
			return
		}
		filename = part.FileName()
		data, err = io.ReadAll(io.LimitReader(part, service.MaxSkillArchiveBytes+1))
		part.Close()
		if err != nil || len(data) > service.MaxSkillArchiveBytes {
			common.ApiErrorMsg(c, "压缩包上传失败或超过 16 MiB 限制")
			return
		}
	}
	draft, err := service.ParseSkillArchive(data, filename)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, draft)
}

func GetSkillStore(c *gin.Context) {
	source := c.Query("source")
	if source != "" && source != "bundled" && source != "online" {
		common.ApiErrorMsg(c, "技能商店来源无效")
		return
	}
	items, err := service.ListSkillStore(c.Request.Context(), source == "online")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, items)
}

func ImportStoreSkill(c *gin.Context) {
	draft, err := service.ImportStoreSkill(c.Request.Context(), c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, draft)
}

func ImportRepositorySkills(c *gin.Context) {
	var input struct {
		URL string `json:"url"`
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 4097))
	if err != nil || len(body) > 4096 || common.Unmarshal(body, &input) != nil {
		common.ApiErrorMsg(c, "仓库链接请求格式错误或超过大小限制")
		return
	}
	draft, err := service.ImportRepositorySkills(c.Request.Context(), input.URL)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, draft)
}

func UpdatePromptTemplateRepository(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "提示词 ID 无效")
		return
	}
	var input struct {
		Version        int  `json:"version"`
		OverwriteLocal bool `json:"overwrite_local"`
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1025))
	if err != nil || len(body) > 1024 || common.Unmarshal(body, &input) != nil || input.Version <= 0 {
		common.ApiErrorMsg(c, "技能更新请求格式错误，缺少当前版本")
		return
	}
	template, updated, err := service.UpdatePromptTemplateFromRepository(c.Request.Context(), id, input.Version, input.OverwriteLocal)
	if errors.Is(err, service.ErrRepositoryLocalChanges) {
		c.JSON(http.StatusConflict, gin.H{"success": false, "code": "repository_local_changes", "message": err.Error()})
		return
	}
	if errors.Is(err, model.ErrPromptVersionConflict) {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"template": template, "updated": updated})
}
