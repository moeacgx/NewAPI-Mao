package controller

import (
	"bufio"
	"bytes"
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
func discoverGatewaySkills(c *gin.Context, info *relaycommon.RelayInfo, format types.RelayFormat) (*relaycommon.RelayInfo, *types.NewAPIError) {
	session, err := service.NewGatewaySkillSession(c, format)
	if err != nil {
		return info, gatewaySkillError(err)
	}
	if session == nil {
		return info, nil
	}
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return info, gatewaySkillError(err)
	}
	body, err := storage.Bytes()
	if err != nil {
		return info, gatewaySkillError(err)
	}
	originalBody := append([]byte(nil), body...)
	catalogue, err := session.Catalogue()
	if err != nil {
		return info, gatewaySkillError(err)
	}
	privateBody, err := service.PrepareGatewaySkillRequest(originalBody, format, info.OriginModelName, catalogue)
	if err != nil {
		return info, gatewaySkillError(err)
	}
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
			c.Writer = captured
			defer func() { c.Writer = clientWriter }()
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
		turn, parseErr := service.ParseGatewaySkillTurn(captured.body.Bytes(), format, clientTools)
		if parseErr != nil {
			return info, gatewaySkillError(parseErr)
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
				return info, gatewaySkillError(errors.New("模型在技能读取完成前连续请求客户端工具，未读取技能正文"))
			}
			privateBody, err = service.RefocusGatewaySkillRequest(privateBody, format)
			if err != nil {
				return info, gatewaySkillError(err)
			}
			clientToolCorrected = true
			continue
		}
		if round < service.GatewaySkillMaxRounds {
			privateBody, err = service.ContinueGatewaySkillRequest(privateBody, format, turn, results)
			if err != nil {
				return info, gatewaySkillError(err)
			}
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
