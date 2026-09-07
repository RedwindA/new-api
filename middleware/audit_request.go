package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

const (
	auditBodyLimit   = 61440
	auditUAMaxLen    = 512
	auditPathMaxLen  = 1024
	auditQueryMaxLen = 2048
)

const (
	auditUnparseableBodyFmt      = "<unparseable body omitted, %d bytes>"
	auditResponseOmittedByPolicy = "<response body omitted by policy>"
)

var auditRequestPathPrefixes = []string{
	"/api/user/login", "/api/user/register", "/api/user/reset",
	"/api/user/2fa", "/api/user/passkey", "/api/user/auth", "/api/user/token",
	"/api/user/topup", "/api/user/epay",
	"/api/oauth", "/api/verification", "/api/reset_password", "/api/verify",
	"/api/token", "/api/redemption", "/api/channel", "/api/option",
}

type restoreReadCloser struct {
	io.Reader
	closer io.Closer
}

func (r *restoreReadCloser) Close() error {
	if r.closer == nil {
		return nil
	}
	return r.closer.Close()
}

type auditRequestResponseWriter struct {
	gin.ResponseWriter
	body      *bytes.Buffer
	maxSize   int
	truncated bool
}

func (w *auditRequestResponseWriter) Write(b []byte) (int, error) {
	if w.body.Len() < w.maxSize {
		remain := w.maxSize - w.body.Len()
		if remain >= len(b) {
			w.body.Write(b)
		} else {
			w.body.Write(b[:remain])
			w.truncated = true
		}
	} else if len(b) > 0 {
		w.truncated = true
	}
	return w.ResponseWriter.Write(b)
}

func (w *auditRequestResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func matchPathPrefix(path string, prefix string) bool {
	if path == prefix {
		return true
	}
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	return path[len(prefix)] == '/'
}

func matchAuditPath(path string) bool {
	for _, prefix := range auditRequestPathPrefixes {
		if matchPathPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// auditResultOf classifies the business outcome of an audited response.
// new-api answers most business failures with HTTP 200 and
// {"success": false}, so the JSON envelope is consulted in addition to the
// status code. Responses that are not a JSON object with a boolean "success"
// and have a non-error status are left as unknown.
func auditResultOf(statusCode int, responseBody []byte) int {
	if statusCode >= http.StatusBadRequest {
		return model.AuditResultFailure
	}
	var envelope struct {
		Success *bool `json:"success"`
	}
	if err := common.Unmarshal(responseBody, &envelope); err != nil || envelope.Success == nil {
		return model.AuditResultUnknown
	}
	if *envelope.Success {
		return model.AuditResultSuccess
	}
	return model.AuditResultFailure
}

func truncateAuditString(value string, maxLen int) string {
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	runes := []rune(value)
	if len(runes) <= maxLen {
		return value
	}
	return string(runes[:maxLen])
}

func peekAndRestoreRequestBody(c *gin.Context) ([]byte, bool) {
	if c.Request == nil || c.Request.Body == nil {
		return nil, false
	}
	origBody := c.Request.Body
	peek := make([]byte, auditBodyLimit+1)
	n, err := io.ReadFull(origBody, peek)
	c.Request.Body = &restoreReadCloser{
		Reader: io.MultiReader(bytes.NewReader(peek[:n]), origBody),
		closer: origBody,
	}
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, false
	}
	if n > auditBodyLimit {
		return peek[:auditBodyLimit], true
	}
	return peek[:n], false
}

// AuditRequest records full request/response metadata for critical management
// endpoints. Mount after gzip (so this wrapper sees plaintext) and before the
// global rate limiter (so 429s are still recorded). When AUDIT_SQL_DSN is unset
// the handler is a no-op so capture has no request-path cost.
func AuditRequest() gin.HandlerFunc {
	if !model.IsAuditRequestEnabled() {
		return func(c *gin.Context) {
			c.Next()
		}
	}
	startAuditPersistWorkers()
	return auditRequest
}

func auditRequest(c *gin.Context) {
	path := ""
	if c.Request != nil && c.Request.URL != nil {
		path = c.Request.URL.Path
	}
	if !matchAuditPath(path) {
		c.Next()
		return
	}

	start := time.Now()
	reqBody, reqTruncated := peekAndRestoreRequestBody(c)
	writer := &auditRequestResponseWriter{
		ResponseWriter: c.Writer,
		body:           bytes.NewBuffer(nil),
		maxSize:        auditBodyLimit,
	}
	c.Writer = writer

	c.Next()

	method := c.Request.Method
	route := c.FullPath()
	query := ""
	if c.Request.URL != nil {
		query = c.Request.URL.RawQuery
	}
	ua := ""
	if c.Request != nil {
		ua = c.Request.UserAgent()
	}

	requestBody, reqOverflow := sanitizeAuditBody(reqBody, method, route, path)
	responseBody := auditResponseOmittedByPolicy
	respTruncated := false
	// Omit-by-policy routes only hide successful (secret-issuing) responses;
	// error responses are always captured (sanitized and size-capped) for
	// diagnosis. A business failure reported as HTTP 200 counts as an error.
	statusCode := writer.Status()
	result := auditResultOf(statusCode, writer.body.Bytes())
	if statusCode != http.StatusOK || result == model.AuditResultFailure || !shouldOmitAuditResponseBody(method, route, path) {
		var respOverflow bool
		responseBody, respOverflow = sanitizeAuditBody(writer.body.Bytes(), method, route, path)
		respTruncated = writer.truncated || respOverflow
	}

	bodyTruncated := 0
	if reqTruncated || reqOverflow {
		bodyTruncated |= model.AuditBodyTruncatedRequest
	}
	if respTruncated {
		bodyTruncated |= model.AuditBodyTruncatedResponse
	}

	entry := &model.AuditRequestLog{
		CreatedAt:     common.GetTimestamp(),
		RequestId:     c.GetString(common.RequestIdKey),
		Ip:            c.ClientIP(),
		UserAgent:     truncateAuditString(ua, auditUAMaxLen),
		Method:        method,
		Route:         route,
		Path:          sanitizeAuditPath(path, route),
		Query:         sanitizeAuditQuery(query),
		StatusCode:    statusCode,
		Result:        result,
		LatencyMs:     int(time.Since(start).Milliseconds()),
		UserId:        c.GetInt("id"),
		Username:      c.GetString("username"),
		UserRole:      c.GetInt("role"),
		RequestBody:   requestBody,
		ResponseBody:  responseBody,
		BodyTruncated: bodyTruncated,
	}

	enqueueAuditRequestLog(entry)
}
