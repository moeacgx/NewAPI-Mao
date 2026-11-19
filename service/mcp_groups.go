package service

import (
	"encoding/json"
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// 模型元数据只提供兼容 code 列表时，一次批量解析当前名称；已水合的渠道引用不查库。
func enrichMCPModelGroupNames(raw json.RawMessage, names *map[string]string, depth int) ([]byte, error) {
	if depth > 32 {
		return nil, errors.New("response nesting too deep")
	}
	switch common.GetJsonType(raw) {
	case "object":
		var object map[string]json.RawMessage
		if err := common.Unmarshal(raw, &object); err != nil {
			return nil, err
		}
		if groups, exists := object["enable_groups"]; exists {
			if _, hydrated := object["enable_group_names"]; !hydrated {
				var codes []string
				if err := common.Unmarshal(groups, &codes); err != nil {
					return nil, err
				}
				display := make([]string, 0, len(codes))
				if len(codes) > 0 && *names == nil {
					if model.DB == nil {
						return nil, errors.New("group names unavailable")
					}
					resolved, err := model.GetGroupDisplayNameMap()
					if err != nil {
						return nil, errors.New("group names unavailable")
					}
					*names = resolved
				}
				for _, code := range codes {
					display = append(display, model.FormatGroupDisplayNames(code, *names))
				}
				encoded, err := common.Marshal(display)
				if err != nil {
					return nil, err
				}
				object["enable_group_names"] = encoded
			}
		}
		for key, child := range object {
			updated, err := enrichMCPModelGroupNames(child, names, depth+1)
			if err != nil {
				return nil, err
			}
			object[key] = updated
		}
		return common.Marshal(object)
	case "array":
		var array []json.RawMessage
		if err := common.Unmarshal(raw, &array); err != nil {
			return nil, err
		}
		for index, child := range array {
			updated, err := enrichMCPModelGroupNames(child, names, depth+1)
			if err != nil {
				return nil, err
			}
			array[index] = updated
		}
		return common.Marshal(array)
	default:
		return raw, nil
	}
}
