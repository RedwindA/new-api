package middleware

import (
	"bytes"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestCaptureResponseLimitDoesNotTruncateClientResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	writer := &requestCaptureResponseWriter{ResponseWriter: context.Writer}
	payload := bytes.Repeat([]byte("a"), requestCaptureBodyLimit+1024)

	written, err := writer.Write(payload)

	require.NoError(t, err)
	assert.Equal(t, len(payload), written)
	assert.Equal(t, payload, recorder.Body.Bytes())
	assert.Len(t, writer.body.Bytes(), requestCaptureBodyLimit)
	assert.Equal(t, int64(len(payload)), writer.totalBytes)
	assert.True(t, writer.truncated)
}

func TestReadCapturedRequestBodyKeepsBodyReadable(t *testing.T) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	payload := []byte(`{"model":"gpt-5","input":"hello"}`)
	context.Request = httptest.NewRequest("POST", "/v1/responses", bytes.NewReader(payload))
	t.Cleanup(func() { common.CleanupBodyStorage(context) })

	captured, totalBytes, truncated, err := readCapturedRequestBody(context)
	require.NoError(t, err)
	forwarded, err := io.ReadAll(context.Request.Body)
	require.NoError(t, err)

	assert.Equal(t, payload, captured)
	assert.Equal(t, payload, forwarded)
	assert.Equal(t, int64(len(payload)), totalBytes)
	assert.False(t, truncated)
}
