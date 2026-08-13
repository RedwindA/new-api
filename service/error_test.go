package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResetStatusCode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		statusCode       int
		statusCodeConfig string
		expectedCode     int
	}{
		{
			name:             "map string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"503"}`,
			expectedCode:     503,
		},
		{
			name:             "map int value",
			statusCode:       429,
			statusCodeConfig: `{"429":503}`,
			expectedCode:     503,
		},
		{
			name:             "skip invalid string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"bad-code"}`,
			expectedCode:     429,
		},
		{
			name:             "skip status code 200",
			statusCode:       200,
			statusCodeConfig: `{"200":503}`,
			expectedCode:     200,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			newAPIError := &types.NewAPIError{
				StatusCode: tc.statusCode,
			}
			ResetStatusCode(newAPIError, tc.statusCodeConfig)
			require.Equal(t, tc.expectedCode, newAPIError.StatusCode)
		})
	}
}

func TestRelayErrorHandlerTruncatesInvalidJSONBodyInLog(t *testing.T) {
	withDebugEnabled(t, false)

	body := strings.Repeat("b", common.LocalLogContentLimit+256)
	var logBuffer bytes.Buffer

	common.LogWriterMu.Lock()
	oldWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &logBuffer
	common.LogWriterMu.Unlock()
	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultErrorWriter = oldWriter
		common.LogWriterMu.Unlock()
	})

	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.Equal(t, "bad response status code 500", newAPIError.Error())
	require.Contains(t, logBuffer.String(), "[truncated")
	require.Contains(t, logBuffer.String(), fmt.Sprintf("original_length=%d", len(body)))
	require.NotContains(t, logBuffer.String(), strings.Repeat("b", common.LocalLogContentLimit+1))
}

func TestRelayErrorHandlerKeepsStructuredErrorMessage(t *testing.T) {
	message := strings.Repeat("c", common.LocalLogContentLimit+256)
	body := `{"message":"` + message + `"}`
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.Equal(t, message, newAPIError.Error())
}

func TestRelayErrorHandlerKeepsOpenAIErrorMessage(t *testing.T) {
	message := strings.Repeat("d", common.LocalLogContentLimit+256)
	body := `{"error":{"message":"` + message + `","type":"server_error","code":"server_error"}}`
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.Equal(t, message, newAPIError.Error())
}

func TestRelayErrorHandlerMasksNewAPIModelDetails(t *testing.T) {
	testCases := []struct {
		name            string
		statusCode      int
		errorCode       types.ErrorCode
		originalMessage string
	}{
		{
			name:            "no available channel",
			statusCode:      http.StatusServiceUnavailable,
			errorCode:       types.ErrorCodeModelNotFound,
			originalMessage: "分组 default 下模型 gpt-test 无可用渠道（distributor）",
		},
		{
			name:            "get channel failed",
			statusCode:      http.StatusServiceUnavailable,
			errorCode:       types.ErrorCodeModelNotFound,
			originalMessage: "獲取分組 default 下模型 gpt-test 的可用管道失敗（distributor）：快取錯誤",
		},
		{
			name:            "model access denied",
			statusCode:      http.StatusForbidden,
			errorCode:       types.ErrorCodeModelAccessDenied,
			originalMessage: "This token has no access to model gpt-test",
		},
		{
			name:            "legacy model access denied in English",
			statusCode:      http.StatusForbidden,
			originalMessage: "This token has no access to model gpt-test",
		},
		{
			name:            "legacy model access denied in Simplified Chinese",
			statusCode:      http.StatusForbidden,
			originalMessage: "该令牌无权访问模型 gpt-test",
		},
		{
			name:            "legacy model access denied in Traditional Chinese",
			statusCode:      http.StatusForbidden,
			originalMessage: "該令牌無權存取模型 gpt-test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body := fmt.Sprintf(
				`{"error":{"message":%q,"type":"new_api_error","code":%q}}`,
				tc.originalMessage,
				tc.errorCode,
			)
			resp := &http.Response{
				StatusCode: tc.statusCode,
				Body:       io.NopCloser(strings.NewReader(body)),
			}

			newAPIError := RelayErrorHandler(context.Background(), resp, false)

			require.NotNil(t, newAPIError)
			assert.Equal(t, tc.originalMessage, newAPIError.Error())
			assert.Equal(t, types.ModelUnavailableMessage, newAPIError.ToOpenAIError().Message)
			assert.Equal(t, types.ModelUnavailableMessage, newAPIError.ToClaudeError().Message)
			assert.Equal(t, tc.errorCode, newAPIError.GetErrorCode())
			assert.Equal(t, tc.statusCode, newAPIError.StatusCode)
		})
	}
}

func TestRelayErrorHandlerKeepsOtherUpstreamModelNotFoundMessage(t *testing.T) {
	message := "The model 'gpt-test' does not exist"
	body := `{"error":{"message":"` + message + `","type":"invalid_request_error","code":"model_not_found"}}`
	resp := &http.Response{
		StatusCode: http.StatusServiceUnavailable,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	assert.Equal(t, message, newAPIError.Error())
	assert.Equal(t, message, newAPIError.ToOpenAIError().Message)
	assert.Equal(t, message, newAPIError.ToClaudeError().Message)
}

func TestRelayErrorHandlerKeepsOtherNewAPIForbiddenMessage(t *testing.T) {
	message := "No permission to access this group"
	body := `{"error":{"message":"` + message + `","type":"new_api_error","code":""}}`
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	assert.Equal(t, message, newAPIError.Error())
	assert.Equal(t, message, newAPIError.ToOpenAIError().Message)
	assert.Equal(t, message, newAPIError.ToClaudeError().Message)
}

func TestRelayErrorHandlerKeepsInvalidJSONBodyInDebugLog(t *testing.T) {
	withDebugEnabled(t, true)

	body := strings.Repeat("e", common.LocalLogContentLimit+256)
	var logBuffer bytes.Buffer

	common.LogWriterMu.Lock()
	oldWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &logBuffer
	common.LogWriterMu.Unlock()
	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultErrorWriter = oldWriter
		common.LogWriterMu.Unlock()
	})

	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.NotContains(t, logBuffer.String(), "[truncated")
	require.Contains(t, logBuffer.String(), body)
}

func withDebugEnabled(t *testing.T, enabled bool) {
	t.Helper()

	oldDebug := common.DebugEnabled
	common.DebugEnabled = enabled
	t.Cleanup(func() {
		common.DebugEnabled = oldDebug
	})
}
