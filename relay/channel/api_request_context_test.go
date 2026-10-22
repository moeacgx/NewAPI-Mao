package channel

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	rootconstant "github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelayRequestPathsCancelUpstreamHTTPTransport(t *testing.T) {
	service.InitHttpClient()

	tests := []struct {
		name   string
		invoke func(*gin.Context, *relaycommon.RelayInfo, string) (*http.Response, error)
	}{
		{
			name: "api",
			invoke: func(c *gin.Context, info *relaycommon.RelayInfo, target string) (*http.Response, error) {
				return DoApiRequest(&downstreamRequestIDTestAdaptor{url: target}, c, info, strings.NewReader(`{}`))
			},
		},
		{
			name: "form",
			invoke: func(c *gin.Context, info *relaycommon.RelayInfo, target string) (*http.Response, error) {
				return DoFormRequest(&downstreamRequestIDTestAdaptor{url: target}, c, info, strings.NewReader("field=value"))
			},
		},
		{
			name: "task",
			invoke: func(c *gin.Context, info *relaycommon.RelayInfo, target string) (*http.Response, error) {
				return DoTaskApiRequest(&stubTaskAdaptor{baseURL: target}, c, info, strings.NewReader(`{}`))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			upstreamStarted := make(chan struct{})
			upstreamCanceled := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
				close(upstreamStarted)
				<-r.Context().Done()
				close(upstreamCanceled)
			}))
			defer server.Close()

			requestContext, cancel := context.WithCancel(context.Background())
			defer cancel()
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{}`)).WithContext(requestContext)
			info := &relaycommon.RelayInfo{
				RelayMode:   relayconstant.RelayModeChatCompletions,
				RelayFormat: types.RelayFormatOpenAI,
				DisablePing: true,
				ChannelMeta: &relaycommon.ChannelMeta{},
			}

			type requestResult struct {
				resp *http.Response
				err  error
			}
			result := make(chan requestResult, 1)
			go func() {
				resp, err := test.invoke(c, info, server.URL)
				result <- requestResult{resp: resp, err: err}
			}()

			select {
			case <-upstreamStarted:
			case <-time.After(2 * time.Second):
				t.Fatal("upstream request did not start")
			}

			var outcome requestResult
			select {
			case outcome = <-result:
				require.NoError(t, outcome.err)
				require.NotNil(t, outcome.resp)
			case <-time.After(2 * time.Second):
				t.Fatal("relay request did not receive upstream headers")
			}
			cancel()

			select {
			case <-upstreamCanceled:
			case <-time.After(2 * time.Second):
				t.Fatal("upstream request context was not canceled")
			}

			require.NoError(t, outcome.resp.Body.Close())
		})
	}
}

func TestClaudeResponsesRequestKeepsUpstreamAliveUntilDrainBodyCloses(t *testing.T) {
	service.InitHttpClient()
	headersSent := make(chan struct{})
	releaseBody := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "11")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		close(headersSent)
		select {
		case <-releaseBody:
			_, _ = io.WriteString(w, "final-usage")
			w.(http.Flusher).Flush()
			return
		case <-r.Context().Done():
			return
		}
	}))
	defer server.Close()

	requestContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{}`)).WithContext(requestContext)
	info := &relaycommon.RelayInfo{
		IsStream:                true,
		RelayMode:               relayconstant.RelayModeResponses,
		FinalRequestRelayFormat: types.RelayFormatClaude,
		ChannelMeta:             &relaycommon.ChannelMeta{ApiType: rootconstant.APITypeAnthropic},
	}

	resp, err := DoApiRequest(&downstreamRequestIDTestAdaptor{url: server.URL}, ctx, info, strings.NewReader(`{}`))
	require.NoError(t, err)
	require.NotNil(t, resp)
	<-headersSent
	cancel()
	close(releaseBody)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "final-usage", string(body), "the upstream body must remain readable for the bounded drain")
	require.NoError(t, resp.Body.Close())
	assert.ErrorIs(t, resp.Request.Context().Err(), context.Canceled, "closing the drain body must release its upstream context")
}

func TestClaudeResponsesRequestStillCancelsBeforeUpstreamHeaders(t *testing.T) {
	service.InitHttpClient()
	requestStarted := make(chan struct{})
	upstreamCanceled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		close(requestStarted)
		<-r.Context().Done()
		close(upstreamCanceled)
	}))
	defer server.Close()
	requestContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{}`)).WithContext(requestContext)
	info := &relaycommon.RelayInfo{
		IsStream:                true,
		RelayMode:               relayconstant.RelayModeResponses,
		FinalRequestRelayFormat: types.RelayFormatClaude,
		ChannelMeta:             &relaycommon.ChannelMeta{ApiType: rootconstant.APITypeAnthropic},
	}
	result := make(chan error, 1)
	go func() {
		_, err := DoApiRequest(&downstreamRequestIDTestAdaptor{url: server.URL}, ctx, info, strings.NewReader(`{}`))
		result <- err
	}()
	select {
	case <-requestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("upstream request did not start")
	}
	cancel()
	select {
	case err := <-result:
		require.Error(t, err)
		assert.ErrorIs(t, requestContext.Err(), context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("upstream request did not stop before response headers")
	}
	select {
	case <-upstreamCanceled:
	case <-time.After(2 * time.Second):
		t.Fatal("client cancellation did not reach the upstream before response headers")
	}
}

func TestClaudeResponsesRequestPreservesErrorAndDeadlineCancellation(t *testing.T) {
	service.InitHttpClient()
	for _, tc := range []struct {
		name     string
		status   int
		deadline bool
	}{
		{name: "HTTP错误不排空", status: http.StatusTooManyRequests},
		{name: "入口截止时间仍终止上游", status: http.StatusOK, deadline: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstreamCanceled := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				w.WriteHeader(tc.status)
				w.(http.Flusher).Flush()
				<-r.Context().Done()
				close(upstreamCanceled)
			}))
			defer func() { server.CloseClientConnections(); server.Close() }()
			requestContext, cancel := context.WithCancel(context.Background())
			if tc.deadline {
				cancel()
				requestContext, cancel = context.WithTimeout(context.Background(), 500*time.Millisecond)
			}
			defer cancel()
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil).WithContext(requestContext)
			info := &relaycommon.RelayInfo{
				IsStream: true, RelayMode: relayconstant.RelayModeResponses,
				FinalRequestRelayFormat: types.RelayFormatClaude,
				ChannelMeta:             &relaycommon.ChannelMeta{ApiType: rootconstant.APITypeAnthropic},
			}
			resp, err := DoApiRequest(&downstreamRequestIDTestAdaptor{url: server.URL}, ctx, info, strings.NewReader(`{}`))
			require.NoError(t, err)
			defer resp.Body.Close()
			if !tc.deadline {
				cancel()
			}
			select {
			case <-upstreamCanceled:
			case <-time.After(2 * time.Second):
				t.Fatal("upstream cancellation was incorrectly detached")
			}
			assert.Error(t, resp.Request.Context().Err())
		})
	}
}
