package controller

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// prepareTokenPromptBilling 将本次实际装载正文纳入估算，补足预扣后才允许发给上游。
func prepareTokenPromptBilling(c *gin.Context, info *relaycommon.RelayInfo) *types.NewAPIError {
	if !service.TokenPromptManaged(c) {
		return nil
	}
	meta := info.Request.GetTokenCountMeta()
	tokens, err := service.EstimateRequestToken(c, meta, info)
	if err != nil {
		return types.NewError(err, types.ErrorCodeCountTokenFailed, types.ErrOptionWithSkipRetry())
	}
	// 管理员关闭常规计数时，装载请求仍需保留可用于缺失 usage 回退的文本估算。
	if !constant.CountToken {
		tokens = service.CountTextToken(meta.CombineText, info.OriginModelName)
	}
	info.SetEstimatePromptTokens(tokens)
	if info.BillingRequestInput != nil {
		storage, readErr := common.GetBodyStorage(c)
		if readErr != nil {
			return types.NewError(readErr, types.ErrorCodeReadRequestBodyFailed, types.ErrOptionWithSkipRetry())
		}
		body, readErr := storage.Bytes()
		if readErr != nil {
			return types.NewError(readErr, types.ErrorCodeReadRequestBodyFailed, types.ErrOptionWithSkipRetry())
		}
		input := *info.BillingRequestInput
		input.Body = append([]byte(nil), body...)
		info.BillingRequestInput = &input
	}
	// 阶梯表达式沿用首次请求的表达式版本，不因管理员并发修改而更换结算规则。
	if snapshot := info.TieredBillingSnapshot; snapshot != nil {
		input := billingexpr.RequestInput{}
		if info.BillingRequestInput != nil {
			input = *info.BillingRequestInput
		}
		raw, trace, runErr := billingexpr.RunExprWithRequest(snapshot.ExprString, billingexpr.TokenParams{
			P: float64(tokens), C: float64(snapshot.EstimatedCompletionTokens), Len: float64(tokens),
		}, input)
		if runErr != nil {
			return types.NewError(runErr, types.ErrorCodeModelPriceError, types.ErrOptionWithSkipRetry())
		}
		snapshot.EstimatedPromptTokens = tokens
		snapshot.EstimatedQuotaBeforeGroup = raw / 1_000_000 * snapshot.QuotaPerUnit
		snapshot.GroupRatio = info.PriceData.GroupRatioInfo.GroupRatio
		snapshot.EstimatedTier = trace.MatchedTier
		quota, convertErr := billingexpr.QuotaRoundStrict(snapshot.EstimatedQuotaBeforeGroup * snapshot.GroupRatio)
		if convertErr != nil {
			return types.NewError(convertErr, types.ErrorCodeModelPriceError, types.ErrOptionWithSkipRetry())
		}
		snapshot.EstimatedQuotaAfterGroup = quota
		info.PriceData.QuotaToPreConsume = quota
		if snapshot.GroupRatio == 0 {
			return nil
		}
		info.PriceData.FreeModel = false
		return reserveTokenPromptQuota(c, info, quota)
	}
	price, err := helper.ModelPriceHelper(c, info, tokens, meta)
	if err != nil {
		return types.NewErrorWithStatusCode(err, types.ErrorCodeModelPriceError, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	if price.FreeModel {
		return nil
	}
	return reserveTokenPromptQuota(c, info, price.QuotaToPreConsume)
}

func reserveTokenPromptQuota(c *gin.Context, info *relaycommon.RelayInfo, quota int) *types.NewAPIError {
	if info.Billing == nil {
		return service.PreConsumeBilling(c, quota, info)
	}
	reserver, ok := info.Billing.(interface{ ReserveRequestQuota(int) error })
	if !ok {
		return types.NewError(fmt.Errorf("计费会话不支持安全追加预扣"), types.ErrorCodeUpdateDataError, types.ErrOptionWithSkipRetry())
	}
	if err := reserver.ReserveRequestQuota(quota); err != nil {
		if errors.Is(err, service.ErrInsufficientWalletQuota) {
			return types.NewErrorWithStatusCode(err, types.ErrorCodeInsufficientUserQuota, http.StatusForbidden, types.ErrOptionWithSkipRetry())
		}
		return types.NewError(fmt.Errorf("提示词装载预扣费失败: %w", err), types.ErrorCodeUpdateDataError, types.ErrOptionWithSkipRetry())
	}
	info.FinalPreConsumedQuota = info.Billing.GetPreConsumedQuota()
	return nil
}
