package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAbortWithOpenAIMessageMasksModelDetails(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		code       types.ErrorCode
		message    string
	}{
		{
			name:       "no available channel",
			statusCode: http.StatusServiceUnavailable,
			code:       types.ErrorCodeModelNotFound,
			message:    "No available channel for model private-model under group private-group (distributor)",
		},
		{
			name:       "model access denied",
			statusCode: http.StatusForbidden,
			code:       types.ErrorCodeModelAccessDenied,
			message:    "This token has no access to model private-model",
		},
		{
			name:       "retry get channel failed",
			statusCode: http.StatusInternalServerError,
			code:       types.ErrorCodeGetChannelFailed,
			message:    "分组 private-group 下模型 private-model 的可用渠道不存在（retry）",
		},
		{
			name:       "model price not configured",
			statusCode: http.StatusBadRequest,
			code:       types.ErrorCodeModelPriceError,
			message:    "模型 private-model 的价格尚未由管理员配置",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			context.Set(common.RequestIdKey, "request-123")

			abortWithOpenAiMessage(context, tc.statusCode, tc.message, tc.code)

			var response struct {
				Error types.OpenAIError `json:"error"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			assert.Equal(t, tc.statusCode, recorder.Code)
			assert.Equal(t, types.ModelUnavailableMessage+" (request id: request-123)", response.Error.Message)
			assert.Equal(t, string(types.ErrorTypeNewAPIError), response.Error.Type)
			assert.Equal(t, string(tc.code), response.Error.Code)
			assert.NotContains(t, recorder.Body.String(), "private-model")
			assert.NotContains(t, recorder.Body.String(), "private-group")
			assert.True(t, context.IsAborted())
		})
	}
}

func TestAbortWithOpenAIMessageKeepsUnrelatedError(t *testing.T) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	context.Set(common.RequestIdKey, "request-123")

	abortWithOpenAiMessage(context, http.StatusBadRequest, "invalid request", types.ErrorCodeInvalidRequest)

	var response struct {
		Error types.OpenAIError `json:"error"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "invalid request (request id: request-123)", response.Error.Message)
	assert.Equal(t, string(types.ErrorCodeInvalidRequest), response.Error.Code)
}
