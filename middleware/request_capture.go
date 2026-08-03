package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

const requestCaptureBodyLimit = 4 * 1024 * 1024

type requestCaptureResponseWriter struct {
	gin.ResponseWriter
	body       bytes.Buffer
	totalBytes int64
	truncated  bool
}

func (w *requestCaptureResponseWriter) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	w.totalBytes += int64(n)
	if remain := requestCaptureBodyLimit - w.body.Len(); remain > 0 {
		captured := n
		if captured > remain {
			captured = remain
		}
		_, _ = w.body.Write(data[:captured])
	}
	if w.totalBytes > int64(requestCaptureBodyLimit) {
		w.truncated = true
	}
	return n, err
}

func (w *requestCaptureResponseWriter) WriteString(data string) (int, error) {
	return w.Write([]byte(data))
}

// captureTargetUserRelay temporarily captures the client-facing relay traffic
// for the user ID being investigated. It runs after token authentication,
// so the user ID cannot be supplied or forged by the client.
func captureTargetUserRelay(c *gin.Context) {
	if !model.RequestCaptureEnabled() || c.GetString(RouteTagKey) != "relay" {
		c.Next()
		return
	}
	userId := c.GetInt("id")
	if userId != 10578 {
		c.Next()
		return
	}
	if strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
		c.Next()
		return
	}

	requestBody, requestBytes, requestTruncated, err := readCapturedRequestBody(c)
	if err != nil {
		common.SysError("failed to capture request body: " + err.Error())
	}

	startedAt := time.Now()
	writer := &requestCaptureResponseWriter{ResponseWriter: c.Writer}
	c.Writer = writer
	c.Next()

	model.EnqueueRequestCapture(model.RequestResponseCapture{
		RequestId:           c.GetString(common.RequestIdKey),
		UserId:              userId,
		TokenId:             c.GetInt("token_id"),
		ChannelId:           common.GetContextKeyInt(c, constant.ContextKeyChannelId),
		CreatedAt:           startedAt,
		DurationMs:          time.Since(startedAt).Milliseconds(),
		Method:              c.Request.Method,
		Path:                c.Request.URL.Path,
		ModelName:           common.GetContextKeyString(c, constant.ContextKeyOriginalModel),
		Status:              writer.Status(),
		RequestContentType:  c.Request.Header.Get("Content-Type"),
		ResponseContentType: writer.Header().Get("Content-Type"),
		RequestBody:         string(requestBody),
		ResponseBody:        writer.body.String(),
		RequestBytes:        requestBytes,
		ResponseBytes:       writer.totalBytes,
		RequestTruncated:    requestTruncated,
		ResponseTruncated:   writer.truncated,
		NodeName:            common.NodeName,
	})
}

func readCapturedRequestBody(c *gin.Context) ([]byte, int64, bool, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, 0, false, err
	}

	requestBytes := storage.Size()
	limit := int64(requestCaptureBodyLimit)
	reader := io.LimitReader(storage, limit)
	body, err := io.ReadAll(reader)
	if _, seekErr := storage.Seek(0, io.SeekStart); err == nil && seekErr != nil {
		err = seekErr
	}
	c.Request.Body = io.NopCloser(storage)
	return body, requestBytes, requestBytes > limit, err
}

var _ http.ResponseWriter = (*requestCaptureResponseWriter)(nil)
