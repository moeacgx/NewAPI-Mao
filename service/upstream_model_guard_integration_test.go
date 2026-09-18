package service_test

import (
	"testing"

	"github.com/QuantumNous/new-api/relay/channel/openai"
	"github.com/QuantumNous/new-api/service"
)

func TestUpstreamModelGuardOpenAIStreamDisablesChannelAndDispatchesTelegram(t *testing.T) {
	service.VerifyUpstreamModelGuardStreamIntegration(t, openai.OaiStreamHandler)
}
