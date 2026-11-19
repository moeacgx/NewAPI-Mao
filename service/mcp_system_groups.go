package service

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// 系统配置保留业务 code，额外提供当前显示名称；一次响应只加载一次名称映射。
func mcpSystemGroupNames(raw json.RawMessage, names *map[string]string, depth int) (json.RawMessage, error) {
	if depth > 32 {
		return nil, errors.New("group configuration nesting too deep")
	}
	switch common.GetJsonType(raw) {
	case "object":
		var object map[string]json.RawMessage
		if err := common.Unmarshal(raw, &object); err != nil {
			return nil, err
		}
		for key, value := range object {
			outputName := ""
			codes := []string{}
			if strings.HasSuffix(key, "group_codes") {
				outputName = strings.TrimSuffix(key, "codes") + "names"
				if err := common.Unmarshal(value, &codes); err != nil {
					return nil, err
				}
			} else if key == "using_group" {
				outputName = "using_group_name"
				var code string
				if err := common.Unmarshal(value, &code); err != nil {
					return nil, err
				}
				if code != "" {
					codes = append(codes, code)
				}
			}
			if outputName != "" {
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
				display := make([]string, 0, len(codes))
				for _, code := range codes {
					display = append(display, model.FormatGroupDisplayNames(code, *names))
				}
				var encoded []byte
				var err error
				if key == "using_group" {
					text := ""
					if len(display) > 0 {
						text = display[0]
					}
					encoded, err = common.Marshal(text)
				} else {
					encoded, err = common.Marshal(display)
				}
				if err != nil {
					return nil, err
				}
				object[outputName] = encoded
			}
			child, err := mcpSystemGroupNames(value, names, depth+1)
			if err != nil {
				return nil, err
			}
			object[key] = child
		}
		return common.Marshal(object)
	case "array":
		var values []json.RawMessage
		if err := common.Unmarshal(raw, &values); err != nil {
			return nil, err
		}
		for i, value := range values {
			child, err := mcpSystemGroupNames(value, names, depth+1)
			if err != nil {
				return nil, err
			}
			values[i] = child
		}
		return common.Marshal(values)
	default:
		return raw, nil
	}
}
