package service

import (
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
)

// 节点限定结构位置，避免把同名字段在任意深度都当成安全元数据。
type mcpOperationProjection struct {
	fields     map[string]*mcpOperationProjection
	items      *mcpOperationProjection
	dictionary *mcpOperationProjection
	scalar     bool
	url        bool
	configured bool
	logText    bool
}

func mcpOperationFields(names string) *mcpOperationProjection {
	fields := map[string]*mcpOperationProjection{}
	for _, name := range strings.Fields(names) {
		fields[name] = &mcpOperationProjection{scalar: true}
	}
	return &mcpOperationProjection{fields: fields}
}

func mcpOperationList(item *mcpOperationProjection) *mcpOperationProjection {
	return &mcpOperationProjection{items: item}
}

// projectMCPFullOperationResponse 的 bool 仅表示接管，业务失败应由核心提前处理。
func projectMCPFullOperationResponse(tool MCPTool, raw []byte) ([]byte, bool, error) {
	shape := mcpFullOperationResponseShape(tool.Name)
	if shape == nil {
		return nil, false, nil
	}
	if len(raw) > mcpResponseLimit(tool) {
		return nil, true, errors.New("操作响应超过大小限制")
	}
	envelope := mcpOperationFields("success")
	envelope.fields["data"] = shape
	projected, err := projectMCPOperationNode(raw, envelope, 0)
	return projected, true, err
}

func mcpFullOperationResponseShape(name string) *mcpOperationProjection {
	scalar := &mcpOperationProjection{scalar: true}
	bot := mcpOperationFields("id name username type enabled last_test_at created_at updated_at token_configured")
	task := mcpOperationFields("id name event_type bot_id template enabled created_at updated_at bot_name event_name last_triggered_at")
	target := mcpOperationFields("id task_id chat_id mention_user_id mention_name enabled created_at updated_at")
	task.fields["targets"] = mcpOperationList(target)
	filter := mcpOperationFields("status_codes prefix_dedup_seconds")
	filter.fields["error_keywords"] = mcpOperationList(scalar)
	task.fields["filter_config"] = filter
	event := mcpOperationFields("value label description default_template owner available")
	event.fields["variables"] = mcpOperationList(scalar)
	marketSource := mcpOperationFields("name")
	marketSource.fields["index_url"] = &mcpOperationProjection{scalar: true, url: true}
	switch name {
	case "newapi_extension_notification_publish":
		return mcpOperationFields("status event_type delivery_count")
	case "newapi_plugin_source_get":
		return mcpOperationFields("key version api_version source_hash source_kind source")
	case "newapi_notification_bots":
		return mcpOperationList(bot)
	case "newapi_notification_bot_create", "newapi_notification_bot_update":
		return mcpOperationFields("id type enabled last_test_at created_at updated_at token_configured")
	case "newapi_notification_tasks_edit_list":
		return mcpOperationList(task)
	case "newapi_notification_event_definitions":
		return mcpOperationList(event)
	case "newapi_notification_preview":
		return mcpOperationFields("html text_length")
	case "newapi_notification_task_create":
		return scalar
	case "newapi_notification_bot_disable", "newapi_notification_bot_test", "newapi_notification_task_update", "newapi_notification_task_disable", "newapi_plugin_activate", "newapi_plugin_status_set":
		return mcpOperationFields("")
	case "newapi_plugin_runtime", "newapi_plugin_runtime_set":
		return mcpOperationFields("enabled channel_type builtin_only billing resource_access production_ready")
	case "newapi_plugin_marketplace_sources", "newapi_plugin_marketplace_sources_set":
		return mcpOperationList(marketSource)
	case "newapi_plugin_options":
		item := mcpOperationFields("key name version channel_type")
		item.fields["models"] = mcpOperationList(scalar)
		return mcpOperationList(item)
	case "newapi_plugin_version_delete":
		result := mcpOperationFields("DeletedActive")
		result.fields["Promoted"] = mcpOperationFields("id key version enabled active")
		return result
	case "newapi_plugin_upload", "newapi_plugin_upload_put":
		// 上传只返回固定版本收据；源码、备注和声明可能含输入中的认证文本，不能写入幂等表。
		return mcpOperationFields("id key version api_version source_hash enabled active source_kind channel_count in_flight_count")
	case "newapi_plugins_inventory", "newapi_plugin_get", "newapi_plugin_versions":
		plugin := mcpOperationFields("id key version api_version source_hash enabled active source_kind channel_count in_flight_count remark created_at")
		provenance := mcpOperationFields("name path")
		provenance.fields["index_url"] = &mcpOperationProjection{scalar: true, url: true}
		plugin.fields["marketplace"] = provenance
		meta := mcpOperationFields("key name version apiVersion sortPriority fetchMode")
		meta.fields["website"] = &mcpOperationProjection{scalar: true, url: true}
		meta.fields["description"] = &mcpOperationProjection{scalar: true, dictionary: scalar}
		meta.fields["author"] = mcpOperationFields("name")
		for _, field := range strings.Fields("models requiredCapabilities submitResponseTypes channelTypes") {
			meta.fields[field] = mcpOperationList(scalar)
		}
		meta.fields["routes"] = mcpOperationList(mcpOperationFields("method path type action decode render taskIdParam retainResult"))
		protocol := mcpOperationFields("name")
		protocol.fields["models"] = mcpOperationList(scalar)
		protocol.fields["supports"] = mcpOperationList(scalar)
		meta.fields["protocols"] = mcpOperationList(protocol)
		usageField := mcpOperationFields("type unit")
		usageField.fields["enum"] = mcpOperationList(scalar)
		for _, field := range strings.Fields("unitLabel description") {
			usageField.fields[field] = &mcpOperationProjection{scalar: true, dictionary: scalar}
		}
		meta.fields["usageSchema"] = &mcpOperationProjection{dictionary: usageField}
		plugin.fields["meta"] = meta
		if name == "newapi_plugins_inventory" || name == "newapi_plugin_versions" {
			return mcpOperationList(plugin)
		}
		return plugin
	case "newapi_extensions_inventory", "newapi_extensions_refresh", "newapi_extension_uninstall", "newapi_extension_enabled_set", "newapi_extension_upload":
		module := mcpOperationFields("id name version description author enabled asset_revision")
		if name != "newapi_extensions_inventory" {
			// 安装与状态变更回执只保留稳定身份和状态，不持久化原模块正文或通知模板。
			module = mcpOperationFields("id version enabled asset_revision")
			if name == "newapi_extension_enabled_set" {
				return module
			}
			result := mcpOperationFields("")
			if name == "newapi_extension_upload" {
				result.fields["module"] = module
			} else {
				result.fields["modules"] = mcpOperationList(module)
			}
			return result
		}
		module.fields["host"] = mcpOperationFields("min max")
		module.fields["runtime"] = mcpOperationFields("type health_path")
		permissions := mcpOperationFields("")
		permissions.fields["roles"] = mcpOperationList(scalar)
		permissions.fields["capabilities"] = mcpOperationList(scalar)
		module.fields["permissions"] = permissions
		ui := mcpOperationFields("")
		ui.fields["nav"] = mcpOperationList(mcpOperationFields("title page icon section order"))
		ui.fields["pages"] = mcpOperationList(mcpOperationFields("key path title"))
		module.fields["ui"] = ui
		notifications := mcpOperationFields("")
		contribution := mcpOperationFields("id label description default_template")
		contribution.fields["variables"] = mcpOperationList(mcpOperationFields("name label type required"))
		notifications.fields["events"] = mcpOperationList(contribution)
		module.fields["notifications"] = notifications
		// 错误原文可能含磁盘路径，仅输出是否存在错误。
		module.fields["error"] = &mcpOperationProjection{configured: true}
		list := mcpOperationFields("")
		list.fields["modules"] = mcpOperationList(module)
		return list
	case "newapi_extension_marketplace":
		market := mcpOperationFields("host_version max_archive_bytes")
		for _, field := range strings.Fields("catalog_url repository_url") {
			market.fields[field] = &mcpOperationProjection{scalar: true, url: true}
		}
		return market
	case "newapi_extension_rate_config_set":
		return mcpOperationFields("side tier adjustment_type adjustment_value")
	case "newapi_extension_rate_config", "newapi_extension_rate_preview":
		rate := mcpOperationFields("side tier adjustment_type adjustment_value raw_rate adjusted_rate source order_id nick_name")
		rate.fields["rate_api_url"] = &mcpOperationProjection{scalar: true, url: true}
		return rate
	case "newapi_deployment_settings":
		return mcpOperationFields("provider enabled configured can_connect")
	case "newapi_deployment_create", "newapi_deployment_update", "newapi_deployment_delete":
		return mcpOperationFields("deployment_id status")
	case "newapi_deployment_name_set":
		return mcpOperationFields("id name status")
	case "newapi_deployment_name_check":
		return mcpOperationFields("available name")
	case "newapi_deployment_test_connection", "newapi_deployment_settings_test_connection":
		return mcpOperationFields("hardware_count total_available")
	case "newapi_deployment_logs":
		return &mcpOperationProjection{logText: true}
	case "newapi_deployment_price_estimate":
		price := mcpOperationFields("estimated_cost currency estimation_valid")
		price.fields["price_breakdown"] = mcpOperationFields("compute_cost network_cost storage_cost total_cost hourly_rate")
		return price
	case "newapi_deployment_hardware_types":
		result := mcpOperationFields("total total_available")
		result.fields["hardware_types"] = mcpOperationList(mcpOperationFields("id name description gpu_type gpu_memory max_gpus cpu memory storage hourly_rate available brand_name available_count"))
		return result
	case "newapi_deployment_locations":
		result := mcpOperationFields("total")
		result.fields["locations"] = mcpOperationList(mcpOperationFields("id name iso2 region country latitude longitude available description"))
		return result
	case "newapi_deployment_replicas":
		result := mcpOperationFields("")
		result.fields["replicas"] = mcpOperationList(mcpOperationFields("location_id location_name hardware_id hardware_name available_count max_gpus"))
		return result
	case "newapi_deployment_container_get", "newapi_deployment_containers":
		container := mcpOperationFields("deployment_id container_id device_id status hardware brand_name created_at uptime_percent gpus_per_container")
		container.fields["public_url"] = &mcpOperationProjection{scalar: true, url: true}
		container.fields["events"] = mcpOperationList(mcpOperationFields("time"))
		if name == "newapi_deployment_container_get" {
			return container
		}
		result := mcpOperationFields("total")
		result.fields["containers"] = mcpOperationList(container)
		return result
	case "newapi_deployments_list", "newapi_deployments_search", "newapi_deployment_get", "newapi_deployment_extend":
		deployment := mcpOperationFields("id deployment_name container_name status type time_remaining time_remaining_minutes hardware_info hardware_name brand_name hardware_quantity completed_percent compute_minutes_served compute_minutes_remaining created_at updated_at model_name model_version instance_count description provider hardware_id amount_paid gpus_per_container total_gpus total_containers")
		deployment.fields["resource_config"] = mcpOperationFields("cpu memory gpu")
		deployment.fields["locations"] = mcpOperationList(mcpOperationFields("id iso2 name"))
		config := mcpOperationFields("traffic_port")
		config.fields["image_url"] = &mcpOperationProjection{scalar: true, url: true}
		config.fields["entrypoint"] = mcpOperationList(scalar)
		config.fields["env_variables"] = &mcpOperationProjection{configured: true}
		deployment.fields["container_config"] = config
		if name == "newapi_deployment_get" || name == "newapi_deployment_extend" {
			return deployment
		}
		result := mcpOperationFields("total page page_size")
		result.fields["items"] = mcpOperationList(deployment)
		result.fields["status_counts"] = mcpOperationFields("all running completed failed destroyed")
		result.fields["status_counts"].fields["deployment requested"] = scalar
		result.fields["status_counts"].fields["termination requested"] = scalar
		return result
	}
	return nil
}

func projectMCPOperationNode(raw json.RawMessage, shape *mcpOperationProjection, depth int) (json.RawMessage, error) {
	if depth > 32 {
		return nil, errors.New("操作响应嵌套过深")
	}
	typ := common.GetJsonType(raw)
	if shape.logText {
		if typ != "string" {
			return nil, errors.New("部署日志响应必须是文本")
		}
		var text string
		if err := common.Unmarshal(raw, &text); err != nil {
			return nil, err
		}
		return projectMCPOperationLogText(text)
	}
	if typ == "null" || len(raw) == 0 {
		return json.RawMessage(`null`), nil
	}
	if shape.configured {
		return common.Marshal(string(raw) != `""` && string(raw) != `{}` && string(raw) != `[]`)
	}
	switch typ {
	case "object":
		if shape.fields == nil && shape.dictionary == nil {
			return nil, errors.New("操作响应对象结构不符合合同")
		}
		var object map[string]json.RawMessage
		if err := common.Unmarshal(raw, &object); err != nil {
			return nil, err
		}
		out := map[string]json.RawMessage{}
		for key, value := range object {
			child := shape.fields[key]
			if child == nil {
				// 本地化与用量字段字典不是任意配置出口，凭据形状的动态键仍需剔除。
				switch strings.ToLower(strings.ReplaceAll(key, "-", "_")) {
				case "key", "token", "api_key", "access_token", "refresh_token", "secret", "password", "authorization", "cookie", "credentials":
					continue
				}
				child = shape.dictionary
			}
			if child == nil {
				continue
			}
			projected, err := projectMCPOperationNode(value, child, depth+1)
			if err != nil {
				return nil, err
			}
			if child.configured {
				key += "_configured"
			}
			out[key] = projected
		}
		return common.Marshal(out)
	case "array":
		if shape.items == nil {
			return nil, errors.New("操作响应数组结构不符合合同")
		}
		var items []json.RawMessage
		if err := common.Unmarshal(raw, &items); err != nil {
			return nil, err
		}
		out := make([]json.RawMessage, 0, len(items))
		for _, item := range items {
			projected, err := projectMCPOperationNode(item, shape.items, depth+1)
			if err != nil {
				return nil, err
			}
			out = append(out, projected)
		}
		return common.Marshal(out)
	default:
		if !shape.scalar {
			return nil, errors.New("操作响应标量结构不符合合同")
		}
		if shape.url && typ == "string" {
			var text string
			if err := common.Unmarshal(raw, &text); err != nil {
				return nil, err
			}
			parsed, err := url.Parse(text)
			if err != nil {
				return common.Marshal("[redacted]")
			}
			// URL query 和 userinfo 可携带任意命名的凭据，整体隐藏并明确告知不能直接回写。
			if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
				return common.Marshal("[redacted]")
			}
		}
		return raw, nil
	}
}

// 日志没有可靠字段结构，敏感行连同后续非空段落整体隐藏，覆盖多行 headers/credentials。
// 未标记的秘密无法可靠自动识别，调用方必须具有独立的 Root 日志授权。
var mcpOperationLogSensitive = regexp.MustCompile(`(?i)(credentials?|headers?|authorization|cookies?|bearer|basic\s|api[._ -]?key|tokens?|secrets?|password|passwd|private[ _-]key|https?://[^\s]*[?@#])`)

func mcpOperationLogJSONSensitive(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" || !strings.ContainsAny(text[:1], "[{\"") {
		return false
	}
	var value any
	if common.Unmarshal([]byte(text), &value) != nil {
		return false
	}
	// 先还原 JSON 转义键和值，避免 headers/api_key 被 Unicode 转义后绕过识别。
	normalized, err := common.Marshal(value)
	return err == nil && mcpOperationLogSensitive.Match(normalized)
}

func projectMCPOperationLogText(text string) ([]byte, error) {
	const limit = 16 * 1024
	if mcpOperationLogJSONSensitive(text) {
		return common.Marshal(map[string]any{"text": "[redacted]", "redacted": true, "truncated": false})
	}
	var out strings.Builder
	redacted, suppressed, truncated := false, false, false
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			suppressed = false
		} else if suppressed {
			continue
		} else if mcpOperationLogSensitive.MatchString(line) || mcpOperationLogJSONSensitive(line) {
			line = "[redacted]"
			redacted, suppressed = true, true
		}
		if out.Len() > 0 {
			out.WriteByte('\n')
		}
		out.WriteString(line)
		if out.Len() > limit {
			truncated = true
			break
		}
	}
	result := out.String()
	if len(result) > limit {
		result = result[:limit]
		for !utf8.ValidString(result) {
			result = result[:len(result)-1]
		}
	}
	return common.Marshal(map[string]any{"text": result, "redacted": redacted, "truncated": truncated})
}
