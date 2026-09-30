package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetGroupTags(c *gin.Context) {
	tags, err := model.GetGroupTags(nil)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, tags)
}

func SaveGroupTag(c *gin.Context) {
	var tag model.GroupTag
	if err := common.DecodeJson(http.MaxBytesReader(c.Writer, c.Request.Body, 256*1024), &tag); err != nil {
		common.ApiErrorMsg(c, "标签请求格式错误或超过大小限制")
		return
	}
	tag.Id = 0
	if c.Request.Method == http.MethodPut {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			common.ApiErrorMsg(c, "标签 ID 无效")
			return
		}
		tag.Id = id
	}
	saved, err := model.SaveGroupTag(tag)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, saved)
}

func DeleteGroupTag(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "标签 ID 无效")
		return
	}
	if err := model.DeleteGroupTag(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
