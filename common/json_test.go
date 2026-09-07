package common

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalUseNumberPreservesLargeInteger(t *testing.T) {
	const raw = `{"id":9007199254740993,"ok":true}`
	var parsed any
	require.NoError(t, UnmarshalUseNumber([]byte(raw), &parsed))
	out, err := Marshal(parsed)
	require.NoError(t, err)
	require.Contains(t, string(out), "9007199254740993")
}

func TestJsonRawMessageToString(t *testing.T) {
	tests := []struct {
		name string
		data json.RawMessage
		want string
	}{
		{
			name: "object",
			data: json.RawMessage(`{"city":"Paris","days":0,"strict":false}`),
			want: `{"city":"Paris","days":0,"strict":false}`,
		},
		{
			name: "string",
			data: json.RawMessage(`"{\"city\":\"Paris\",\"days\":0,\"strict\":false}"`),
			want: `{"city":"Paris","days":0,"strict":false}`,
		},
		{
			name: "null",
			data: json.RawMessage(`null`),
			want: "",
		},
		{
			name: "empty",
			data: nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, JsonRawMessageToString(tt.data))
		})
	}
}
