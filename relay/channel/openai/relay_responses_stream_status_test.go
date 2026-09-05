package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Responses SSE streams carry no "data: [DONE]" terminator. The handler must
// end the stream itself on the terminal event; otherwise the end reason is
// decided by a race between upstream EOF and the client disconnecting after it
// has already received the final event (codex does exactly that), which
// surfaces as a spurious client_gone / timeout in the usage log.
func TestOaiResponsesStreamHandlerEndsOnTerminalEventWithoutUpstreamEOF(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 2
	t.Cleanup(func() {
		constant.StreamingTimeout = oldTimeout
	})

	cases := []struct {
		name       string
		event      string
		wantReason relaycommon.StreamEndReason
		wantErrors bool
	}{
		{
			name:       "completed",
			event:      `{"type":"response.completed","response":{"status":"completed","output":[],"usage":{"input_tokens":3,"output_tokens":5,"total_tokens":8}}}`,
			wantReason: relaycommon.StreamEndReasonDone,
		},
		{
			name:       "incomplete",
			event:      `{"type":"response.incomplete","response":{"status":"incomplete"}}`,
			wantReason: relaycommon.StreamEndReasonDone,
		},
		{
			name:       "failed",
			event:      `{"type":"response.failed","response":{"status":"failed"}}`,
			wantReason: relaycommon.StreamEndReasonHandlerStop,
			wantErrors: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Pipe writer is intentionally never closed: upstream keeps the
			// connection open after the terminal event.
			pr, pw := io.Pipe()
			go func() {
				_, _ = io.WriteString(pw, "data: "+tc.event+"\n\n")
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
			c.Set(common.RequestIdKey, "responses-stream-status-test")
			info := &relaycommon.RelayInfo{
				OriginModelName: "gpt-5.1",
				IsStream:        true,
				DisablePing:     true,
				ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "gpt-5.1"},
			}
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       pr,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			}

			usage, apiErr := OaiResponsesStreamHandler(c, info, resp)
			require.Nil(t, apiErr)
			require.NotNil(t, info.StreamStatus)
			assert.Equal(t, tc.wantReason, info.StreamStatus.EndReason)
			assert.Equal(t, tc.wantErrors, info.StreamStatus.HasErrors())
			if tc.name == "completed" {
				assert.Equal(t, 3, usage.PromptTokens)
				assert.Equal(t, 5, usage.CompletionTokens)
			}
		})
	}
}
