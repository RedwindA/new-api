package types

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewErrorMasksUnavailableModelDetails(t *testing.T) {
	testCases := []struct {
		name       string
		message    string
		errorCode  ErrorCode
		statusCode int
	}{
		{
			name:       "retry get channel failed",
			message:    "分组 default 下模型 gpt-test 的可用渠道不存在（retry）",
			errorCode:  ErrorCodeGetChannelFailed,
			statusCode: http.StatusInternalServerError,
		},
		{
			name:       "model price not configured",
			message:    "模型 gpt-test 的价格尚未由管理员配置，暂时无法使用，请联系站点管理员开启该模型",
			errorCode:  ErrorCodeModelPriceError,
			statusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			newAPIError := NewError(errors.New(tc.message), tc.errorCode, ErrOptionWithStatusCode(tc.statusCode))

			require.NotNil(t, newAPIError)
			assert.Equal(t, tc.message, newAPIError.Error())
			assert.Equal(t, ModelUnavailableMessage, newAPIError.ToOpenAIError().Message)
			assert.Equal(t, ModelUnavailableMessage, newAPIError.ToClaudeError().Message)
			assert.Equal(t, tc.errorCode, newAPIError.GetErrorCode())
			assert.Equal(t, tc.statusCode, newAPIError.StatusCode)
			assert.NotContains(t, newAPIError.ToOpenAIError().Message, "gpt-test")
			assert.NotContains(t, newAPIError.ToClaudeError().Message, "default")
		})
	}
}

func TestNewErrorKeepsUnrelatedMessage(t *testing.T) {
	message := "invalid request"
	newAPIError := NewError(errors.New(message), ErrorCodeInvalidRequest)

	require.NotNil(t, newAPIError)
	assert.Equal(t, message, newAPIError.Error())
	assert.Equal(t, message, newAPIError.ToOpenAIError().Message)
	assert.Equal(t, message, newAPIError.ToClaudeError().Message)
}
