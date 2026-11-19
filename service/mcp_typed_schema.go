package service

import (
	"reflect"
	"strings"
)

// mcpTypedBodySchema 复用固定管理 DTO 的字段契约，拒绝未声明字段。
// 非指针且没有 omitempty 的字段必须提供，避免完整保存接口静默清空配置。
// 业务范围、枚举和字段间关系仍由真实服务校验。
func mcpTypedBodySchema(value any) map[string]any {
	return mcpSchemaForType(reflect.TypeOf(value))
}

func mcpSchemaForType(typ reflect.Type) map[string]any {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	switch typ.Kind() {
	case reflect.Struct:
		fields := map[string]any{}
		required := []string{}
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if field.PkgPath != "" {
				continue
			}
			tag := field.Tag.Get("json")
			name := strings.Split(tag, ",")[0]
			if name == "-" {
				continue
			}
			if field.Anonymous && name == "" {
				embedded := mcpSchemaForType(field.Type)
				for key, schema := range embedded["properties"].(map[string]any) {
					fields[key] = schema
				}
				required = append(required, embedded["required"].([]string)...)
				continue
			}
			if name == "" {
				name = field.Name
			}
			schema := mcpSchemaForType(field.Type)
			if (name == "id" || strings.HasSuffix(name, "_id")) && schema["type"] == "integer" {
				schema["minimum"], schema["maximum"] = 0, 2147483647
			}
			if (name == "ids" || strings.HasSuffix(name, "_ids")) && schema["type"] == "array" {
				item := schema["items"].(map[string]any)
				if item["type"] == "integer" {
					item["minimum"], item["maximum"] = 1, 2147483647
				}
			}
			fields[name] = schema
			if field.Type.Kind() != reflect.Pointer && !strings.Contains(tag, ",omitempty") {
				required = append(required, name)
			}
		}
		return map[string]any{"type": "object", "properties": fields, "required": required, "additionalProperties": false}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": mcpSchemaForType(typ.Elem()), "maxItems": 10000}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": mcpSchemaForType(typ.Elem()), "maxProperties": 10000}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.String:
		return map[string]any{"type": "string", "maxLength": 32768}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return map[string]any{"type": "integer", "minimum": -2147483648, "maximum": 2147483647}
	case reflect.Int64:
		return map[string]any{"type": "integer", "minimum": int64(-9007199254740991), "maximum": int64(9007199254740991)}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return map[string]any{"type": "integer", "minimum": 0, "maximum": int64(4294967295)}
	case reflect.Uint64:
		return map[string]any{"type": "integer", "minimum": 0, "maximum": int64(9007199254740991)}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number", "minimum": -1e15, "maximum": 1e15}
	default:
		// 未知类型不能退化为任意输入。目录测试会暴露不可支持的 DTO。
		return map[string]any{"type": "null"}
	}
}
