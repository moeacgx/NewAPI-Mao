package gemini

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestConvertImageRequestNativeImagineModelUsesGenerateContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	info := newGeminiImageRelayInfo("gemini-3-pro-image-preview")
	converted, err := (&Adaptor{}).ConvertImageRequest(c, info, dto.ImageRequest{
		Model:  "gemini-3-pro-image-preview",
		Prompt: "a red apple on a white table",
		N:      lo.ToPtr(uint(1)),
		Size:   "auto",
	})
	require.NoError(t, err)

	geminiRequest, ok := converted.(*dto.GeminiChatRequest)
	require.True(t, ok, "native imagine models must convert to generateContent, got %T", converted)
	require.Len(t, geminiRequest.Contents, 1)
	require.Equal(t, "user", geminiRequest.Contents[0].Role)
	require.Len(t, geminiRequest.Contents[0].Parts, 1)
	require.Equal(t, "a red apple on a white table", geminiRequest.Contents[0].Parts[0].Text)
	require.Equal(t, []string{"TEXT", "IMAGE"}, geminiRequest.GenerationConfig.ResponseModalities)
	require.Empty(t, geminiRequest.GenerationConfig.ImageConfig)
}

func TestConvertImageRequestNativeImagineMapsSizeQualityAndReferenceImage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	info := newGeminiImageRelayInfo("gemini-2.5-flash-image")
	imageJSON, err := json.Marshal("data:image/png;base64,aW1hZ2U=")
	require.NoError(t, err)

	converted, err := (&Adaptor{}).ConvertImageRequest(c, info, dto.ImageRequest{
		Model:   "gemini-2.5-flash-image",
		Prompt:  "make the apple green",
		N:       lo.ToPtr(uint(2)),
		Size:    "1792x1024",
		Quality: "hd",
		Image:   imageJSON,
	})
	require.NoError(t, err)

	encoded, err := common.Marshal(converted)
	require.NoError(t, err)
	require.Equal(t, "user", gjson.GetBytes(encoded, "contents.0.role").String())
	require.Equal(t, "make the apple green", gjson.GetBytes(encoded, "contents.0.parts.0.text").String())
	require.Equal(t, "image/png", gjson.GetBytes(encoded, "contents.0.parts.1.inlineData.mimeType").String())
	require.Equal(t, "aW1hZ2U=", gjson.GetBytes(encoded, "contents.0.parts.1.inlineData.data").String())
	require.False(t, gjson.GetBytes(encoded, "generationConfig.candidateCount").Exists())
	require.Equal(t, []string{"TEXT", "IMAGE"}, stringSlice(gjson.GetBytes(encoded, "generationConfig.responseModalities").Array()))
	require.Equal(t, "16:9", gjson.GetBytes(encoded, "generationConfig.imageConfig.aspectRatio").String())
	require.False(t, gjson.GetBytes(encoded, "generationConfig.imageConfig.imageSize").Exists())
}

func TestConvertImageRequestGemini3QualityMapsImageSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	info := newGeminiImageRelayInfo("gemini-3-pro-image-preview")
	converted, err := (&Adaptor{}).ConvertImageRequest(c, info, dto.ImageRequest{
		Model:   "gemini-3-pro-image-preview",
		Prompt:  "a red apple on a white table",
		Quality: "hd",
		Size:    "1024x1024",
	})
	require.NoError(t, err)

	encoded, err := common.Marshal(converted)
	require.NoError(t, err)
	require.Equal(t, "1:1", gjson.GetBytes(encoded, "generationConfig.imageConfig.aspectRatio").String())
	require.Equal(t, "2K", gjson.GetBytes(encoded, "generationConfig.imageConfig.imageSize").String())
}

func TestConvertImageRequestGeminiFileURIUsesFileData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	info := newGeminiImageRelayInfo("gemini-3-pro-image-preview")
	imageJSON, err := json.Marshal("gs://bucket/apple.png")
	require.NoError(t, err)

	converted, err := (&Adaptor{}).ConvertImageRequest(c, info, dto.ImageRequest{
		Model:  "gemini-3-pro-image-preview",
		Prompt: "edit the apple",
		Image:  imageJSON,
	})
	require.NoError(t, err)

	encoded, err := common.Marshal(converted)
	require.NoError(t, err)
	require.Equal(t, "gs://bucket/apple.png", gjson.GetBytes(encoded, "contents.0.parts.1.fileData.fileUri").String())
	require.Equal(t, "image/png", gjson.GetBytes(encoded, "contents.0.parts.1.fileData.mimeType").String())
	require.False(t, gjson.GetBytes(encoded, "contents.0.parts.1.inlineData").Exists())
}

func TestConvertImageRequestEmptyPrompt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := newGeminiImageRelayInfo("gemini-3-pro-image-preview")

	_, err := (&Adaptor{}).ConvertImageRequest(c, info, dto.ImageRequest{
		Model:  "gemini-3-pro-image-preview",
		Prompt: "   ",
	})
	require.EqualError(t, err, "prompt is required")
}

func TestConvertImageRequestImagenStillUsesPredictInstances(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	info := newGeminiImageRelayInfo("imagen-4.0-generate-001")
	converted, err := (&Adaptor{}).ConvertImageRequest(c, info, dto.ImageRequest{
		Model:  "imagen-4.0-generate-001",
		Prompt: "a red apple on a white table",
		N:      lo.ToPtr(uint(1)),
		Size:   "1024x1024",
	})
	require.NoError(t, err)
	require.IsType(t, dto.GeminiImageRequest{}, converted)
}

func TestConvertImageRequestRejectsNonImagineGeminiChatModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := newGeminiImageRelayInfo("gemini-2.5-pro")

	_, err := (&Adaptor{}).ConvertImageRequest(c, info, dto.ImageRequest{
		Model:  "gemini-2.5-pro",
		Prompt: "a red apple on a white table",
	})
	require.EqualError(t, err, "not supported model for image generation, only imagen models are supported")
}

func TestGetRequestURLNativeImagineUsesGenerateContent(t *testing.T) {
	info := newGeminiImageRelayInfo("gemini-3-pro-image-preview")
	url, err := (&Adaptor{}).GetRequestURL(info)
	require.NoError(t, err)
	require.Contains(t, url, "/models/gemini-3-pro-image-preview:generateContent")
	require.NotContains(t, url, ":predict")
}

func TestDoResponseNativeImagineReturnsOpenAIImageFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	info := newGeminiImageRelayInfo("gemini-3-pro-image-preview")
	info.RelayFormat = types.RelayFormatOpenAIImage

	payload := dto.GeminiChatResponse{
		Candidates: []dto.GeminiChatCandidate{{
			Content: dto.GeminiChatContent{
				Role: "model",
				Parts: []dto.GeminiPart{
					{Text: "here is the apple"},
					{InlineData: &dto.GeminiInlineData{MimeType: "image/png", Data: "aW1hZ2U="}},
				},
			},
		}},
		UsageMetadata: dto.GeminiUsageMetadata{
			PromptTokenCount:     12,
			CandidatesTokenCount: 40,
			TotalTokenCount:      52,
		},
	}
	body, err := common.Marshal(payload)
	require.NoError(t, err)

	usage, newAPIError := (&Adaptor{}).DoResponse(c, &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}, info)
	require.Nil(t, newAPIError)

	var imageResp dto.ImageResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &imageResp))
	require.Len(t, imageResp.Data, 1)
	require.Equal(t, "aW1hZ2U=", imageResp.Data[0].B64Json)
	require.Empty(t, imageResp.Data[0].Url)

	usageValue, ok := usage.(*dto.Usage)
	require.True(t, ok)
	require.Equal(t, 12, usageValue.PromptTokens)
	require.Equal(t, 40, usageValue.CompletionTokens)
	require.Equal(t, 52, usageValue.TotalTokens)
}

func TestDoResponseNativeImagineSupportsGeminiImageCarriers(t *testing.T) {
	tests := []struct {
		name    string
		part    string
		wantB64 string
		wantURL string
	}{
		{name: "inlineData", part: `{"inlineData":{"mimeType":"image/png","data":"aW1hZ2U="}}`, wantB64: "aW1hZ2U="},
		{name: "inline_data", part: `{"inline_data":{"mime_type":"image/png","data":"aW1hZ2U="}}`, wantB64: "aW1hZ2U="},
		{name: "thought inlineData", part: `{"thought":true,"inlineData":{"mimeType":"image/png","data":"aW1hZ2U="}}`, wantB64: "aW1hZ2U="},
		{name: "fileData", part: `{"fileData":{"mimeType":"image/png","fileUri":"https://cdn.example/image.png"}}`, wantURL: "https://cdn.example/image.png"},
		{name: "file_data", part: `{"file_data":{"mime_type":"image/png","file_uri":"https://cdn.example/image.png"}}`, wantURL: "https://cdn.example/image.png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
			info := newGeminiImageRelayInfo("gemini-3-pro-image-preview")
			info.RelayFormat = types.RelayFormatOpenAIImage
			body := `{"candidates":[{"content":{"role":"model","parts":[` + tt.part + `]},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":12,"candidatesTokenCount":40,"totalTokenCount":52}}`

			usage, newAPIError := (&Adaptor{}).DoResponse(c, &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
			}, info)
			require.Nil(t, newAPIError)

			var imageResp dto.ImageResponse
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &imageResp))
			require.Len(t, imageResp.Data, 1)
			require.Equal(t, tt.wantB64, imageResp.Data[0].B64Json)
			require.Equal(t, tt.wantURL, imageResp.Data[0].Url)
			require.Equal(t, 52, usage.(*dto.Usage).TotalTokens)
		})
	}
}

func TestDoResponseNativeImagineRejectsInvalidFileData(t *testing.T) {
	tests := []struct {
		name string
		part string
	}{
		{name: "empty URI", part: `{"fileData":{"mimeType":"image/png","fileUri":""}}`},
		{name: "missing host", part: `{"fileData":{"mimeType":"image/png","fileUri":"https:///image.png"}}`},
		{name: "gs URI", part: `{"fileData":{"mimeType":"image/png","fileUri":"gs://bucket/image.png"}}`},
		{name: "file URI", part: `{"fileData":{"mimeType":"image/png","fileUri":"file:///tmp/image.png"}}`},
		{name: "javascript URI", part: `{"fileData":{"mimeType":"image/png","fileUri":"javascript:alert(1)"}}`},
		{name: "non image MIME", part: `{"fileData":{"mimeType":"text/html","fileUri":"https://cdn.example/image.png"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
			info := newGeminiImageRelayInfo("gemini-3-pro-image-preview")
			info.RelayFormat = types.RelayFormatOpenAIImage
			body := `{"candidates":[{"content":{"role":"model","parts":[` + tt.part + `]},"finishReason":"STOP","index":0}]}`

			_, newAPIError := (&Adaptor{}).DoResponse(c, &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
			}, info)

			require.NotNil(t, newAPIError)
			require.Equal(t, "no images generated", newAPIError.Error())
			require.Empty(t, recorder.Body.Bytes())
		})
	}
}

func TestDoResponseNativeImagineRejectsTextOnlyWithSafeShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	info := newGeminiImageRelayInfo("gemini-3-pro-image-preview")
	info.RelayFormat = types.RelayFormatOpenAIImage
	body := `{"candidates":[{"content":{"role":"model","parts":[{"text":"private refusal text"}]},"finishReason":"SAFETY","index":0}],"promptFeedback":{"blockReason":"PROHIBITED_CONTENT"},"usageMetadata":{"promptTokenCount":12,"candidatesTokenCount":3,"totalTokenCount":15}}`

	usage, newAPIError := (&Adaptor{}).DoResponse(c, &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}, info)

	require.NotNil(t, newAPIError)
	require.Equal(t, "no images generated", newAPIError.Error())
	usageValue, ok := usage.(*dto.Usage)
	require.True(t, ok)
	require.Equal(t, 15, usageValue.TotalTokens)
	require.Equal(t,
		"gemini_native_image_no_media candidates=1 parts=1 text=1 inline_data=0 file_data=0 other=0 finish_reasons=SAFETY block_reason=PROHIBITED_CONTENT",
		common.GetContextKeyString(c, constant.ContextKeyAdminRejectReason),
	)
	require.NotContains(t, common.GetContextKeyString(c, constant.ContextKeyAdminRejectReason), "private refusal text")
}

func TestDoResponseNativeImagineRedactsUnknownDiagnosticEnums(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	c.Set(common.RequestIdKey, "req-shape-test")
	info := newGeminiImageRelayInfo("gemini-3-pro-image-preview")
	info.RelayFormat = types.RelayFormatOpenAIImage
	body := `{"candidates":[{"content":{"role":"model","parts":[{"text":"private refusal text"}]},"finishReason":"SECRETLEAK","index":0}],"promptFeedback":{"blockReason":"ANOTHERSECRET"}}`
	var logBuffer bytes.Buffer
	common.LogWriterMu.Lock()
	originalErrorWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &logBuffer
	common.LogWriterMu.Unlock()
	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultErrorWriter = originalErrorWriter
		common.LogWriterMu.Unlock()
	})

	_, newAPIError := (&Adaptor{}).DoResponse(c, &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}, info)

	require.NotNil(t, newAPIError)
	shape := common.GetContextKeyString(c, constant.ContextKeyAdminRejectReason)
	require.Contains(t, shape, "finish_reasons=OTHER block_reason=OTHER")
	require.NotContains(t, shape, "SECRETLEAK")
	require.NotContains(t, shape, "ANOTHERSECRET")
	require.NotContains(t, shape, "private refusal text")
	require.Contains(t, logBuffer.String(), "req-shape-test")
	require.Contains(t, logBuffer.String(), shape)
	require.NotContains(t, logBuffer.String(), "SECRETLEAK")
	require.NotContains(t, logBuffer.String(), "ANOTHERSECRET")
	require.NotContains(t, logBuffer.String(), "private refusal text")
}

func TestIsGeminiModelSupportImagineIncludesCanvasModels(t *testing.T) {
	for _, model := range []string{
		"gemini-3-pro-image-preview",
		"gemini-3-pro-image",
		"gemini-2.5-flash-image",
		"gemini-3.1-flash-image",
		"gemini-3.1-flash-image-preview",
	} {
		require.True(t, model_setting.IsGeminiModelSupportImagine(model), model)
	}
	require.False(t, model_setting.IsGeminiModelSupportImagine("gemini-2.5-pro"))
}

func newGeminiImageRelayInfo(model string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		OriginModelName: model,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl:    "https://generativelanguage.googleapis.com",
			ApiVersion:        "v1beta",
			UpstreamModelName: model,
		},
	}
}

func stringSlice(results []gjson.Result) []string {
	out := make([]string, 0, len(results))
	for _, result := range results {
		out = append(out, result.String())
	}
	return out
}
