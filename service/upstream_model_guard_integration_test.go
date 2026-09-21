package service_test

import (
	"testing"

	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	"github.com/QuantumNous/new-api/service"
)

func TestUpstreamModelGuardOpenAIStreamDisablesChannelAndDispatchesTelegram(t *testing.T) {
	service.VerifyUpstreamModelGuardStreamIntegration(t, openai.OaiStreamHandler)
}

func TestUpstreamModelGuardHTTPUsesBodyModelOnly(t *testing.T) {
	t.Run("JSON", func(t *testing.T) {
		service.VerifyUpstreamModelGuardHTTPBodyIntegration(t, false, channel.DoRequest, openai.OaiResponsesHandler)
	})
	t.Run("SSE", func(t *testing.T) {
		service.VerifyUpstreamModelGuardHTTPBodyIntegration(t, true, channel.DoRequest, openai.OaiResponsesStreamHandler)
	})
}
