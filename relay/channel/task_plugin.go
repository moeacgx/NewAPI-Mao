package channel

import (
	taskdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"net/http"
)

type TaskSubmitResponse struct {
	UpstreamTaskID string
	TaskData       []byte
	ClientResponse any
	Immediate      *relaycommon.TaskInfo
	PluginState    []byte
	// ActualTokenUsage 只用于本次同步请求的统计，不持久化或参与计费。
	ActualTokenUsage *dto.Usage `json:"-"`
}

type TaskArtifact struct {
	Key      string `json:"key"`
	Type     string `json:"type"`
	MimeType string `json:"mimeType,omitempty"`
}

type TaskArtifactClientRequest struct {
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
}

type TaskArtifactProvider interface {
	ListArtifacts(task *model.Task) ([]TaskArtifact, error)
}

type TaskContentRequest struct {
	URL            string
	Method         string
	Headers        map[string]string
	Body           []byte
	Credentialless bool
}

type TaskContentRequestProvider interface {
	BuildContentRequest(task *model.Task, artifactKey string, clientRequest TaskArtifactClientRequest) (*TaskContentRequest, error)
}

type TaskUsageFactsProvider interface {
	ExtractUsageFacts(c *gin.Context, info *relaycommon.RelayInfo) map[string]any
}

// TaskValidatedBillingProvider lets an adaptor reject invalid usage facts at
// the existing estimate point, after model mapping and before quota
// multiplication. Non-plugin task adaptors keep using EstimateBilling.
type TaskValidatedBillingProvider interface {
	EstimateBillingValidated(c *gin.Context, info *relaycommon.RelayInfo) (map[string]float64, error)
}

// TaskValidatedUsageFactsProvider is the tiered-billing counterpart to
// TaskValidatedBillingProvider.
type TaskValidatedUsageFactsProvider interface {
	ExtractUsageFactsValidated(c *gin.Context, info *relaycommon.RelayInfo) (map[string]any, error)
}

// PluginTaskParser 将纯解析与写客户端响应分开。
type PluginTaskParser interface {
	ParseResponse(*gin.Context, *http.Response, *relaycommon.RelayInfo) (*TaskSubmitResponse, *taskdto.TaskError)
}
