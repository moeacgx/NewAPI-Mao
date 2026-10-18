package deepseek

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/claude"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/reasoning"
	"github.com/gin-gonic/gin"
)

type Adaptor struct {
}

func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, req *dto.ClaudeRequest) (any, error) {
	adaptor := claude.Adaptor{}
	convertedRequest, err := adaptor.ConvertClaudeRequest(c, info, req)
	if err != nil {
		return nil, err
	}
	claudeRequest, ok := convertedRequest.(*dto.ClaudeRequest)
	if !ok {
		return convertedRequest, nil
	}
	if err := applyDeepSeekV4ClaudeThinkingSuffix(info, claudeRequest); err != nil {
		return nil, err
	}
	return claudeRequest, nil
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
	// DeepSeek runs Claude Code and Codex searches server-side and bills them
	// through its Search API, whatever model name the client sent.
	info.WebSearchBillingKey = SearchToolPriceKey
	if info.RelayMode == constant.RelayModeResponses {
		info.VendorToolUsage = responsesToolUsage
	}
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	fimBaseUrl := info.ChannelBaseUrl
	switch info.RelayFormat {
	case types.RelayFormatClaude:
		return fmt.Sprintf("%s/anthropic/v1/messages", info.ChannelBaseUrl), nil
	default:
		if !strings.HasSuffix(info.ChannelBaseUrl, "/beta") {
			fimBaseUrl += "/beta"
		}
		switch info.RelayMode {
		case constant.RelayModeCompletions:
			return fmt.Sprintf("%s/completions", fimBaseUrl), nil
		case constant.RelayModeResponses:
			return fmt.Sprintf("%s/responses", info.ChannelBaseUrl), nil
		case constant.RelayModeToolsSearch:
			return fmt.Sprintf("%s/v1/tools/search", info.ChannelBaseUrl), nil
		case constant.RelayModeToolsFetch:
			return fmt.Sprintf("%s/v1/tools/fetch", info.ChannelBaseUrl), nil
		default:
			return fmt.Sprintf("%s/v1/chat/completions", info.ChannelBaseUrl), nil
		}
	}
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	req.Set("Authorization", "Bearer "+info.ApiKey)
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	if err := applyDeepSeekV4OpenAIThinkingSuffix(info, request); err != nil {
		return nil, err
	}
	applyDeepSeekModelModifier(info, request)

	return request, nil
}

func applyDeepSeekV4OpenAIThinkingSuffix(info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) error {
	modelName := request.Model
	if info != nil && info.ChannelMeta != nil && info.UpstreamModelName != "" {
		modelName = info.UpstreamModelName
	}
	baseModel, thinkingType, effort, ok := reasoning.ParseDeepSeekV4ThinkingSuffix(modelName)
	if !ok {
		return nil
	}
	thinking, err := common.Marshal(map[string]string{
		"type": thinkingType,
	})
	if err != nil {
		return fmt.Errorf("error marshalling thinking: %w", err)
	}
	request.Model = baseModel
	request.THINKING = thinking
	request.ReasoningEffort = effort
	if info != nil {
		if info.ChannelMeta != nil {
			info.UpstreamModelName = baseModel
		}
		info.SetReasoningEffort(effort)
	}
	return nil
}

func applyDeepSeekV4ClaudeThinkingSuffix(info *relaycommon.RelayInfo, request *dto.ClaudeRequest) error {
	modelName := request.Model
	if info != nil && info.ChannelMeta != nil && info.UpstreamModelName != "" {
		modelName = info.UpstreamModelName
	}
	baseModel, thinkingType, effort, ok := reasoning.ParseDeepSeekV4ThinkingSuffix(modelName)
	if !ok {
		applyDeepSeekClaudeModelModifier(info, request)
		return nil
	}
	request.Model = baseModel
	request.Thinking = &dto.Thinking{Type: thinkingType}
	// Only effort is set or removed; other output_config members stay.
	outputConfig := make(map[string]any)
	if len(request.OutputConfig) > 0 && common.GetJsonType(request.OutputConfig) == "object" {
		if err := common.Unmarshal(request.OutputConfig, &outputConfig); err != nil {
			return fmt.Errorf("invalid output_config: %w", err)
		}
	}
	delete(outputConfig, "effort")
	if effort != "" {
		outputConfig["effort"] = effort
	}
	request.OutputConfig = nil
	if len(outputConfig) > 0 {
		encoded, err := common.Marshal(outputConfig)
		if err != nil {
			return fmt.Errorf("error marshalling output_config: %w", err)
		}
		request.OutputConfig = encoded
	}
	if info != nil {
		if info.ChannelMeta != nil {
			info.UpstreamModelName = baseModel
		}
		info.SetReasoningEffort(effort)
	}
	return nil
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, nil
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(_ *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	applyDeepSeekV4ResponsesThinkingSuffix(info, &request)
	applyDeepSeekResponsesModelModifier(info, &request)
	return request, nil
}

// applyDeepSeekModelModifier handles the provider-specific @ controls without
// changing the shared model-modifier pipeline. These controls are accepted on
// DeepSeek V4.1 model aliases and are stripped before the upstream request.
func applyDeepSeekModelModifier(info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) {
	base, thinking, effort, temperature, ok := parseDeepSeekModelModifier(request.Model)
	if !ok {
		return
	}
	request.Model = base
	request.Temperature = temperature
	request.THINKING, _ = common.Marshal(map[string]string{"type": thinking})
	request.ReasoningEffort = effort
	request.Reasoning = nil
	if info != nil {
		info.SetReasoningEffort(deepSeekLogEffort(thinking, effort))
		if info.ChannelMeta != nil {
			info.UpstreamModelName = base
		}
	}
}

func applyDeepSeekClaudeModelModifier(info *relaycommon.RelayInfo, request *dto.ClaudeRequest) {
	base, thinking, effort, temperature, ok := parseDeepSeekModelModifier(request.Model)
	if !ok {
		return
	}
	request.Model = base
	request.Thinking = &dto.Thinking{Type: thinking}
	request.Temperature = temperature
	if effort == "" {
		request.OutputConfig = nil
	} else {
		request.OutputConfig, _ = common.Marshal(map[string]string{"effort": effort})
	}
	if info != nil {
		info.SetReasoningEffort(deepSeekLogEffort(thinking, effort))
		if info.ChannelMeta != nil {
			info.UpstreamModelName = base
		}
	}
}

func applyDeepSeekResponsesModelModifier(info *relaycommon.RelayInfo, request *dto.OpenAIResponsesRequest) {
	base, thinking, effort, _, ok := parseDeepSeekModelModifier(request.Model)
	if !ok {
		return
	}
	request.Model = base
	if request.Reasoning == nil {
		request.Reasoning = &dto.Reasoning{}
	}
	if thinking == "disabled" {
		effort = "none"
	}
	request.Reasoning.Effort = effort
	if info != nil {
		info.SetReasoningEffort(deepSeekLogEffort(thinking, effort))
		if info.ChannelMeta != nil {
			info.UpstreamModelName = base
		}
	}
}

func parseDeepSeekModelModifier(model string) (base, thinking, effort string, temperature *float64, ok bool) {
	parts := strings.Split(model, "@")
	if len(parts) < 2 || !strings.HasPrefix(parts[0], "deepseek-") {
		return model, "", "", nil, false
	}
	base = parts[0]
	// A provider suffix and an explicit @ modifier may be combined. The @
	// control wins, but the legacy tail must still be removed from the model ID.
	if suffixBase, _, _, suffixOK := reasoning.ParseDeepSeekV4ThinkingSuffix(base); suffixOK {
		base = suffixBase
	}
	thinking = "enabled"
	for _, part := range parts[1:] {
		key, value, found := strings.Cut(strings.ToLower(strings.TrimSpace(part)), ":")
		if !found {
			continue
		}
		switch key {
		case "thinking":
			switch value {
			case "off", "none", "0":
				thinking, effort = "disabled", ""
			case "on", "adaptive":
				thinking, effort = "enabled", ""
			default:
				if budget, err := strconv.Atoi(value); err == nil && budget > 0 {
					thinking, effort = "enabled", "medium"
				}
			}
		case "effort":
			if value == "none" {
				thinking, effort = "disabled", ""
			} else if slicesContains([]string{"low", "medium", "high", "xhigh", "max"}, value) {
				thinking, effort = "enabled", value
			}
		case "temperature":
			if parsed, err := strconv.ParseFloat(value, 64); err == nil {
				temperature = &parsed
			}
		}
	}
	return base, thinking, effort, temperature, true
}

func slicesContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func deepSeekLogEffort(thinking, effort string) string {
	if thinking == "disabled" {
		return "none"
	}
	if effort == "" || effort == "medium" || effort == "xhigh" {
		return "high"
	}
	return effort
}

func applyDeepSeekV4ResponsesThinkingSuffix(info *relaycommon.RelayInfo, request *dto.OpenAIResponsesRequest) {
	modelName := request.Model
	if info != nil && info.ChannelMeta != nil && info.UpstreamModelName != "" {
		modelName = info.UpstreamModelName
	}
	baseModel, thinkingType, effort, ok := reasoning.ParseDeepSeekV4ThinkingSuffix(modelName)
	if !ok {
		if info != nil && request.Reasoning != nil {
			info.SetReasoningEffort(request.Reasoning.Effort)
		}
		return
	}
	if thinkingType == "disabled" {
		effort = "none"
	}
	request.Model = baseModel
	if request.Reasoning == nil && effort != "" {
		request.Reasoning = &dto.Reasoning{}
	}
	if request.Reasoning != nil {
		request.Reasoning.Effort = effort
	}
	if info != nil {
		if info.ChannelMeta != nil {
			info.UpstreamModelName = baseModel
		}
		info.SetReasoningEffort(effort)
	}
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	switch info.RelayFormat {
	case types.RelayFormatClaude:
		adaptor := claude.Adaptor{}
		return adaptor.DoResponse(c, resp, info)
	default:
		adaptor := openai.Adaptor{}
		return adaptor.DoResponse(c, resp, info)
	}
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
