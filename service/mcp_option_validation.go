package service

import (
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// 在调用持久化选项接口前拒绝无效计费输入，避免先写数据库再更新内存失败。
func validateMCPOptionValue(key, value string) error {
	switch key {
	case "QuotaPerUnit", "Price", "StripeUnitPrice", "BepusdtUnitPrice", "WaffoUnitPrice", "WaffoPancakeUnitPrice", "USDExchangeRate":
		number, err := strconv.ParseFloat(value, 64)
		if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number <= 0 || number > float64(common.MaxQuota) {
			return errors.New("monetary option must be finite, positive and within quota bounds")
		}
	case "QuotaForNewUser", "QuotaForInviter", "QuotaForInvitee", "QuotaRemindThreshold":
		number, err := strconv.ParseInt(value, 10, 32)
		if err != nil || number < 0 {
			return errors.New("quota option must be a nonnegative int32")
		}
	case "ModelRatio", "ModelPrice", "CompletionRatio", "CacheRatio", "CreateCacheRatio", "ImageRatio", "AudioRatio", "AudioCompletionRatio", "GroupRatio", "group_ratio_setting.group_ratio", "TopupGroupRatio":
		var ratios map[string]json.RawMessage
		if common.UnmarshalJsonStr(value, &ratios) != nil || ratios == nil {
			return errors.New("pricing option must be a JSON object of numeric values")
		}
		for name, raw := range ratios {
			var ratio float64
			if common.GetJsonType(raw) != "number" || common.Unmarshal(raw, &ratio) != nil || strings.TrimSpace(name) == "" || math.IsNaN(ratio) || math.IsInf(ratio, 0) || ratio < 0 || ratio > float64(common.MaxQuota) {
				return errors.New("pricing values must be finite, nonnegative and within quota bounds")
			}
		}
	case "GroupGroupRatio", "group_ratio_setting.group_group_ratio":
		var ratios map[string]json.RawMessage
		if common.UnmarshalJsonStr(value, &ratios) != nil || ratios == nil {
			return errors.New("group pricing option must be a nested numeric object")
		}
		for name, rawGroup := range ratios {
			var group map[string]json.RawMessage
			if strings.TrimSpace(name) == "" || common.Unmarshal(rawGroup, &group) != nil || group == nil {
				return errors.New("invalid group pricing object")
			}
			for target, raw := range group {
				var ratio float64
				if strings.TrimSpace(target) == "" || common.GetJsonType(raw) != "number" || common.Unmarshal(raw, &ratio) != nil || math.IsNaN(ratio) || math.IsInf(ratio, 0) || ratio < 0 || ratio > float64(common.MaxQuota) {
					return errors.New("invalid group pricing value")
				}
			}
		}
	}
	return nil
}
