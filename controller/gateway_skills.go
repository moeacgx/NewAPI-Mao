package controller

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// gatewaySkillWriter 隔离私有模型响应；任何响应头、工具调用或预览文本都不发送给客户端。
type gatewaySkillWriter struct {
	gin.ResponseWriter
	header   http.Header
	body     bytes.Buffer
	status   int
	written  bool
	overflow bool
}

func (w *gatewaySkillWriter) Header() http.Header { return w.header }
func (w *gatewaySkillWriter) WriteHeader(status int) {
	if !w.written {
		w.status = status
	}
}
func (w *gatewaySkillWriter) WriteHeaderNow() { w.written = true }
func (w *gatewaySkillWriter) Write(body []byte) (int, error) {
	w.written = true
	if w.body.Len()+len(body) > 8*1024*1024 {
		w.overflow = true
		return 0, errors.New("技能读取阶段模型响应超过 8 MiB")
	}
	return w.body.Write(body)
}
func (w *gatewaySkillWriter) WriteString(body string) (int, error) { return w.Write([]byte(body)) }
func (w *gatewaySkillWriter) Status() int                          { return w.status }
func (w *gatewaySkillWriter) Size() int {
	if !w.written {
		return -1
	}
	return w.body.Len()
}
func (w *gatewaySkillWriter) Written() bool { return w.written }
func (w *gatewaySkillWriter) Flush()        { w.WriteHeaderNow() }
func (w *gatewaySkillWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, errors.New("技能读取阶段不支持连接升级")
}
func (w *gatewaySkillWriter) Pusher() http.Pusher { return nil }

// discoverGatewaySkills 先在固定渠道完成按需读取，再恢复原始工具、输出格式和流式生成。
// 每个模型轮次使用独立计费会话，已结算的轮次不会因后续失败而退款。
func discoverGatewaySkills(c *gin.Context, info *relaycommon.RelayInfo, format types.RelayFormat, admission **service.GatewaySkillAdmission, originalRequestBytes int64) (result *relaycommon.RelayInfo, apiError *types.NewAPIError) {
	session, err := service.NewGatewaySkillSession(c, format)
	if err != nil {
		return info, gatewaySkillError(err)
	}
	if session == nil {
		return info, nil
	}
	if *admission == nil {
		var admissionErr *types.NewAPIError
		*admission, admissionErr = service.TryAcquireGatewaySkillRequest(originalRequestBytes)
		if admissionErr != nil {
			return info, admissionErr
		}
	}
	clientContext := c.Request.Context()
	discoveryContext, cancelDiscovery := context.WithTimeout(clientContext, service.GatewaySkillDiscoveryTimeout())
	c.Request = c.Request.WithContext(discoveryContext)
	defer func() {
		deadlineExpired := errors.Is(discoveryContext.Err(), context.DeadlineExceeded)
		// 保留期间替换的 Body、长度和头部，只恢复客户端原有取消语义。
		c.Request = c.Request.WithContext(clientContext)
		cancelDiscovery()
		if clientErr := clientContext.Err(); clientErr != nil {
			// 子期限先到而父请求随后取消时，也必须保留父原因，避免误记渠道故障。
			apiError = gatewaySkillError(clientErr)
		} else if deadlineExpired {
			apiError = types.NewErrorWithStatusCode(fmt.Errorf("云端技能读取阶段超时: %w", context.DeadlineExceeded),
				types.ErrorCode("gateway_skills_timeout"), http.StatusGatewayTimeout, types.ErrOptionWithSkipRetry())
		}
	}()
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return info, gatewaySkillError(err)
	}
	body, err := storage.Bytes()
	if err != nil {
		return info, gatewaySkillError(err)
	}
	// BodyStorage 提供不可变正文；后续重建只替换 storage，无需额外复制整段历史。
	originalBody := body
	catalogue, err := session.Catalogue()
	if err != nil {
		return info, gatewaySkillError(err)
	}
	privateRequest, err := service.NewGatewaySkillRequest(originalBody, format, info.OriginModelName, catalogue)
	if err != nil {
		return info, gatewaySkillError(err)
	}
	privateBody := privateRequest.Bytes()
	clientURL := c.Request.URL
	clientHeader := c.Request.Header
	clientStream := info.Request.IsStream(c.Request)
	clientRequestPath := info.RequestURLPath
	clientDisablePing := info.DisablePing
	defer func() {
		c.Request.URL = clientURL
		c.Request.Header = clientHeader
		info.IsStream = clientStream
		info.DisablePing = clientDisablePing
		info.RequestURLPath = clientRequestPath
		info.RequestHeaders = map[string]string{}
		for key := range clientHeader {
			info.RequestHeaders[key] = clientHeader.Get(key)
		}
		common.SetContextKey(c, constant.ContextKeyIsStream, clientStream)
		service.BindChannelMetricRelayInfo(c, info)
	}()
	privateURL := *clientURL
	if format == types.RelayFormatGemini {
		privateURL.Path = strings.TrimSuffix(privateURL.Path, ":streamGenerateContent") + ":generateContent"
		if !strings.HasSuffix(clientURL.Path, ":streamGenerateContent") {
			privateURL.Path = clientURL.Path
		}
		query := privateURL.Query()
		query.Del("alt")
		privateURL.RawQuery = query.Encode()
	}
	privateHeader := clientHeader.Clone()
	privateHeader.Set("Accept", "application/json")
	c.Request.URL = &privateURL
	c.Request.Header = privateHeader
	clientTools := service.GatewaySkillClientTools(originalBody, format)
	clientToolCorrected := false
	for round := 1; round <= service.GatewaySkillMaxRounds; round++ {
		if err := c.Request.Context().Err(); err != nil {
			return info, gatewaySkillError(err)
		}
		if err := resetRelayRequestBody(c, privateBody); err != nil {
			return info, gatewaySkillError(err)
		}
		filtered, filterErr := service.ApplySensitiveFilterToRequestBody(c, format)
		if filterErr != nil {
			return info, gatewaySkillError(filterErr)
		}
		if filtered.Blocked {
			return info, service.NewSensitiveFilterAPIError(nil)
		}
		request, parseErr := helper.GetAndValidateRequest(c, format)
		if parseErr != nil {
			return info, gatewaySkillError(parseErr)
		}
		info.Request = request
		info.IsStream = false
		info.DisablePing = true
		info.ForcePreConsume = true
		info.RequestURLPath = privateURL.String()
		if strings.HasPrefix(info.RequestURLPath, "/pg/") {
			info.RequestURLPath = "/v1" + strings.TrimPrefix(info.RequestURLPath, "/pg")
		}
		info.RequestHeaders = map[string]string{}
		for key := range privateHeader {
			info.RequestHeaders[key] = privateHeader.Get(key)
		}
		common.SetContextKey(c, constant.ContextKeyIsStream, false)
		session.Audit(c, "discovery", round)
		if billingErr := prepareTokenPromptBilling(c, info); billingErr != nil {
			return info, billingErr
		}
		if billingErr := service.PrepareTieredBillingForSelectedGroup(c, info); billingErr != nil {
			return info, billingErr
		}
		service.BeginChannelMetricGatewaySkillAttempt(c, info, common.GetContextKeyInt(c, constant.ContextKeyChannelId),
			common.GetContextKeyString(c, constant.ContextKeyChannelName), common.GetContextKeyInt(c, constant.ContextKeyChannelType))
		captured := &gatewaySkillWriter{ResponseWriter: c.Writer, header: make(http.Header), status: http.StatusOK}
		clientWriter := c.Writer
		var apiErr *types.NewAPIError
		func() {
			streamHeaders := c.GetBool("event_stream_headers_set")
			c.Set("event_stream_headers_set", false)
			c.Writer = captured
			defer func() {
				c.Writer = clientWriter
				c.Set("event_stream_headers_set", streamHeaders)
			}()
			switch format {
			case types.RelayFormatClaude:
				apiErr = relay.ClaudeHelper(c, info)
			case types.RelayFormatGemini:
				apiErr = geminiRelayHandler(c, info)
			default:
				apiErr = relayHandler(c, info)
			}
		}()
		if apiErr == nil && captured.overflow {
			apiErr = gatewaySkillError(errors.New("技能读取阶段模型响应超过 8 MiB"))
		}
		if apiErr == nil && format == types.RelayFormatOpenAI && info.StreamStatus != nil &&
			(!info.StreamStatus.IsNormalEnd() || info.StreamStatus.HasErrors()) {
			apiErr = gatewaySkillError(errors.New("技能读取阶段流式响应未完整结束"))
		}
		if apiErr != nil {
			if info.Billing != nil && !info.Billing.NeedsRefund() {
				fresh, freshErr := nextGatewaySkillRelayInfo(c, info, format)
				if freshErr != nil {
					return info, freshErr
				}
				info = fresh
			}
			return info, apiErr
		}
		service.FinishChannelMetricAttempt(c, info, nil, false, "")
		if info.Billing != nil && info.Billing.NeedsRefund() {
			return info, gatewaySkillError(errors.New("技能读取阶段未完成用量结算"))
		}
		fresh, freshErr := nextGatewaySkillRelayInfo(c, info, format)
		if freshErr != nil {
			return info, freshErr
		}
		info = fresh
		var turn *service.GatewaySkillTurn
		var turnParseErr error
		if format == types.RelayFormatOpenAI && strings.HasPrefix(strings.ToLower(captured.Header().Get("Content-Type")), "text/event-stream") {
			// 上游可能忽略 stream=false；私有 SSE 必须完整聚合后再执行授权读取。
			turn, turnParseErr = service.ParseGatewaySkillChatStream(captured.body.Bytes(), clientTools)
		} else {
			turn, turnParseErr = service.ParseGatewaySkillTurn(captured.body.Bytes(), format, clientTools)
		}
		if turnParseErr != nil {
			return info, gatewaySkillError(turnParseErr)
		}
		if len(turn.Calls) == 0 && len(turn.ClientCalls) == 0 {
			break
		}
		results, executeErr := session.Execute(c, turn.Calls)
		if executeErr != nil {
			return info, gatewaySkillError(executeErr)
		}
		if len(turn.ClientCalls) > 0 {
			// 先处理同轮授权读取；客户端动作只在从原请求重建的正式生成阶段交回 Agent。
			if session.HasReadFiles() {
				break
			}
			if clientToolCorrected || round == service.GatewaySkillMaxRounds {
				if turn.UnrecognizedClientCalls && format == types.RelayFormatOpenAIResponses {
					// 纠偏后仍返回原请求未声明的调用时，结束可选发现阶段并恢复原请求。
					// 不执行私有调用，也不让它阻断原本可用的正式请求。
					session.MarkDiscoveryFallback()
					break
				}
				return info, gatewaySkillError(errors.New("模型在技能读取完成前连续请求客户端工具，未读取技能正文"))
			}
			err = privateRequest.Refocus()
			if err != nil {
				return info, gatewaySkillError(err)
			}
			privateBody = privateRequest.Bytes()
			clientToolCorrected = true
			continue
		}
		if round < service.GatewaySkillMaxRounds {
			err = privateRequest.Continue(turn, results)
			if err != nil {
				return info, gatewaySkillError(err)
			}
			privateBody = privateRequest.Bytes()
		}
	}
	c.Request.URL, c.Request.Header = clientURL, clientHeader
	finalBody, err := session.FinalBody(c, originalBody)
	if err != nil {
		return info, gatewaySkillError(err)
	}
	if err := resetRelayRequestBody(c, finalBody); err != nil {
		return info, gatewaySkillError(err)
	}
	filtered, err := service.ApplySensitiveFilterToRequestBody(c, format)
	if err != nil {
		return info, gatewaySkillError(err)
	}
	if filtered.Blocked {
		return info, service.NewSensitiveFilterAPIError(nil)
	}
	request, err := helper.GetAndValidateRequest(c, format)
	if err != nil {
		return info, gatewaySkillError(err)
	}
	info.Request = request
	info.IsStream = clientStream
	info.DisablePing = false
	info.RequestURLPath = clientURL.String()
	if strings.HasPrefix(info.RequestURLPath, "/pg/") {
		info.RequestURLPath = "/v1" + strings.TrimPrefix(info.RequestURLPath, "/pg")
	}
	info.RequestHeaders = map[string]string{}
	for key := range clientHeader {
		info.RequestHeaders[key] = clientHeader.Get(key)
	}
	session.Audit(c, "generation", 0)
	return info, nil
}

func gatewaySkillError(err error) *types.NewAPIError {
	var apiErr *types.NewAPIError
	if errors.As(err, &apiErr) {
		return types.NewError(err, apiErr.GetErrorCode(), types.ErrOptionWithSkipRetry())
	}
	return types.NewErrorWithStatusCode(err, types.ErrorCode("gateway_skills_failed"), http.StatusBadGateway, types.ErrOptionWithSkipRetry())
}

func nextGatewaySkillRelayInfo(c *gin.Context, previous *relaycommon.RelayInfo, format types.RelayFormat) (*relaycommon.RelayInfo, *types.NewAPIError) {
	fresh, err := relaycommon.GenRelayInfo(c, format, previous.Request, nil)
	if err != nil {
		return nil, gatewaySkillError(err)
	}
	serial := c.GetInt("gateway_skill_billing_serial") + 1
	c.Set("gateway_skill_billing_serial", serial)
	fresh.RequestId = fmt.Sprintf("%s.skills.%d", common.GetContextKeyString(c, common.RequestIdKey), serial)
	if common.GetContextKeyString(c, common.RequestIdKey) == "" {
		fresh.RequestId = fmt.Sprintf("%s.skills.%d", previous.RequestId, serial)
	}
	fresh.StartTime = time.Now()
	fresh.FirstResponseTime = fresh.StartTime.Add(-time.Second)
	fresh.ForcePreConsume = true
	fresh.OriginalRequestURLPath = previous.OriginalRequestURLPath
	fresh.RetryIndex = previous.RetryIndex
	fresh.PriceData = previous.PriceData
	if previous.TieredBillingSnapshot != nil {
		snapshot := *previous.TieredBillingSnapshot
		fresh.TieredBillingSnapshot = &snapshot
	}
	if previous.BillingRequestInput != nil {
		input := *previous.BillingRequestInput
		fresh.BillingRequestInput = &input
	}
	fresh.OnResponseData = previous.OnResponseData
	fresh.OnUpstreamModelEvidence = previous.OnUpstreamModelEvidence
	service.BindChannelMetricRelayInfo(c, fresh)
	return fresh, nil
}
