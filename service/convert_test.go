package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestGeminiToOpenAIRequestStopSequences(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		stopSequences []string
		expectedStop  []string
	}{
		{
			name:          "preserve fewer than four stop sequences",
			stopSequences: []string{"stop-a", "stop-b"},
			expectedStop:  []string{"stop-a", "stop-b"},
		},
		{
			name:          "truncate to openai limit",
			stopSequences: []string{"stop-a", "stop-b", "stop-c", "stop-d", "stop-e"},
			expectedStop:  []string{"stop-a", "stop-b", "stop-c", "stop-d"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := &dto.GeminiChatRequest{
				Contents: []dto.GeminiChatContent{
					{
						Role:  "user",
						Parts: []dto.GeminiPart{{Text: "hello"}},
					},
				},
				GenerationConfig: dto.GeminiChatGenerationConfig{
					StopSequences: tc.stopSequences,
				},
			}

			info := &relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{
					UpstreamModelName: "gpt-test",
				},
			}

			openAIReq, err := GeminiToOpenAIRequest(req, info)
			require.NoError(t, err)
			require.Equal(t, tc.expectedStop, openAIReq.Stop)
		})
	}
}
