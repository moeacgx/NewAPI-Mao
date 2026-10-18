package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetUpstreamModelGuardConfig(c *gin.Context) {
	cfg, err := service.GetUpstreamModelGuardConfig(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "上游模型校验配置加载失败"})
		return
	}
	common.ApiSuccess(c, cfg)
}

func UpdateUpstreamModelGuardConfig(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 256*1024)
	var req service.UpstreamModelGuardConfigUpdate
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "上游模型校验配置参数无效"})
		return
	}
	cfg, err := service.SaveUpstreamModelGuardConfig(c.Request.Context(), req, c.GetInt("id"))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, model.ErrUpstreamModelGuardConfigConflict) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"success": false, "message": err.Error()})
		return
	}
	recordManageAudit(c, "extension.upstream_model_guard.update", map[string]interface{}{"config_version": cfg.ConfigVersion, "enabled": cfg.Enabled, "rule_count": len(cfg.Rules), "excluded_channel_count": len(cfg.ExcludedChannelIDs), "failure_threshold": cfg.FailureThreshold})
	common.ApiSuccess(c, cfg)
}

func ListUpstreamModelGuardChannels(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, sizeErr := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	keyword := strings.TrimSpace(c.Query("keyword"))
	if err != nil || sizeErr != nil || page < 1 || page > 1000000 || pageSize < 1 || pageSize > 100 || len(keyword) > 128 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "渠道搜索参数无效"})
		return
	}
	rows, total, err := service.ListUpstreamModelGuardChannels(c.Request.Context(), keyword, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "渠道列表加载失败"})
		return
	}
	common.ApiSuccess(c, gin.H{"items": rows, "total": total, "page": page, "page_size": pageSize})
}

func ListUpstreamModelGuardRecords(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, sizeErr := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if err != nil || sizeErr != nil || page < 1 || page > 1000000 || pageSize < 1 || pageSize > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "分页参数无效"})
		return
	}
	rows, total, err := model.ListUpstreamModelGuardRecords(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "上游模型校验记录加载失败"})
		return
	}
	common.ApiSuccess(c, gin.H{"items": rows, "total": total, "page": page, "page_size": pageSize})
}
