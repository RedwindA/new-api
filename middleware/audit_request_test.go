package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeekAndRestoreRequestBodyPreservesBytesOnReadError(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	const payload = `{"username":"alice"}`
	body := io.MultiReader(strings.NewReader(payload), iotest.ErrReader(io.ErrClosedPipe))
	c.Request = httptest.NewRequest(http.MethodPost, "/api/user/login", body)

	captured, truncated := peekAndRestoreRequestBody(c)
	assert.Empty(t, captured)
	assert.False(t, truncated)
	restored, err := io.ReadAll(c.Request.Body)
	require.ErrorIs(t, err, io.ErrClosedPipe)
	assert.Equal(t, payload, string(restored))
}

func TestMatchAuditPathUsesPrefixBoundary(t *testing.T) {
	assert.True(t, matchAuditPath("/api/token"))
	assert.True(t, matchAuditPath("/api/token/12"))
	assert.True(t, matchAuditPath("/api/user/login"))
	assert.True(t, matchAuditPath("/api/user/login/2fa"))
	assert.False(t, matchAuditPath("/api/tokenX"))
	assert.False(t, matchAuditPath("/api/status"))
	assert.False(t, matchAuditPath("/api/user"))
	assert.False(t, matchAuditPath("/v1/chat/completions"))
}

func TestAuditResultOfClassifiesBusinessEnvelope(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   int
	}{
		{"http 200 with success false is a failure", http.StatusOK, `{"success":false,"message":"wrong password"}`, model.AuditResultFailure},
		{"http 200 with success true is a success", http.StatusOK, `{"success":true,"data":{"id":1}}`, model.AuditResultSuccess},
		{"http 4xx is a failure regardless of body", http.StatusTooManyRequests, `{"success":true}`, model.AuditResultFailure},
		{"http 5xx with empty body is a failure", http.StatusInternalServerError, ``, model.AuditResultFailure},
		{"http 200 without success field is unknown", http.StatusOK, `{"data":"x"}`, model.AuditResultUnknown},
		{"http 200 with non-boolean success is unknown", http.StatusOK, `{"success":"yes"}`, model.AuditResultUnknown},
		{"http 200 with non-json body is unknown", http.StatusOK, `<html>`, model.AuditResultUnknown},
		{"http 302 with empty body is unknown", http.StatusFound, ``, model.AuditResultUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, auditResultOf(tc.status, []byte(tc.body)))
		})
	}
}

func TestShouldOmitAuditResponseBodyCoversSecretIssuingRoutes(t *testing.T) {
	assert.True(t, shouldOmitAuditResponseBody(http.MethodGet, "/api/option/", "/api/option/"))
	assert.True(t, shouldOmitAuditResponseBody(http.MethodGet, "/api/channel/:id", "/api/channel/1"))
	assert.True(t, shouldOmitAuditResponseBody(http.MethodGet, "/api/token/:id", "/api/token/9"))
	assert.True(t, shouldOmitAuditResponseBody(http.MethodGet, "/api/redemption/", "/api/redemption/"))
	assert.True(t, shouldOmitAuditResponseBody(http.MethodPost, "/api/user/reset", "/api/user/reset"))
	assert.True(t, shouldOmitAuditResponseBody(http.MethodGet, "/api/user/token", "/api/user/token"))
	assert.True(t, shouldOmitAuditResponseBody(http.MethodPost, "/api/redemption/", "/api/redemption/"))
	assert.True(t, shouldOmitAuditResponseBody(http.MethodPost, "/api/token/batch/keys", "/api/token/batch/keys"))
	assert.True(t, shouldOmitAuditResponseBody(http.MethodPost, "/api/token/:id/key", "/api/token/12/key"))
	assert.True(t, shouldOmitAuditResponseBody(http.MethodPost, "/api/verify", "/api/verify"))
	assert.True(t, shouldOmitAuditResponseBody(http.MethodPost, "/api/user/2fa/setup", "/api/user/2fa/setup"))
	assert.True(t, shouldOmitAuditResponseBody(http.MethodPost, "/api/user/2fa/backup_codes", "/api/user/2fa/backup_codes"))
	assert.True(t, shouldOmitAuditResponseBody(http.MethodPost, "/api/channel/:id/key", "/api/channel/3/key"))
	assert.True(t, shouldOmitAuditResponseBody(http.MethodPost, "/api/oauth/state", "/api/oauth/state"))
	assert.True(t, shouldOmitAuditResponseBody(http.MethodPost, "/api/user/login", "/api/user/login"))
	assert.True(t, shouldOmitAuditResponseBody(http.MethodPost, "/api/user/passkey/verify/finish", "/api/user/passkey/verify/finish"))
	assert.False(t, shouldOmitAuditResponseBody(http.MethodPost, "/api/channel/", "/api/channel/"))
	assert.False(t, shouldOmitAuditResponseBody(http.MethodGet, "/api/user/login/encryption-key", "/api/user/login/encryption-key"))
	assert.False(t, shouldOmitAuditResponseBody(http.MethodGet, "/api/user/2fa/status", "/api/user/2fa/status"))
}

func TestSanitizeAuditBodyRedactsSensitiveKeys(t *testing.T) {
	raw := []byte(`{"username":"alice","password":"plain-secret","nested":{"key":"sk-live","ok":true},"token":"pat-1","proof_token":"proof-1","flow_token":"flow-1"}`)
	got, truncated := sanitizeAuditBody(raw, http.MethodPost, "/api/user/login", "/api/user/login")
	assert.False(t, truncated)
	assert.Contains(t, got, `"username":"alice"`)
	assert.Contains(t, got, `"password":"***"`)
	assert.Contains(t, got, `"key":"***"`)
	assert.Contains(t, got, `"token":"***"`)
	assert.Contains(t, got, `"proof_token":"***"`)
	assert.Contains(t, got, `"flow_token":"***"`)
	assert.NotContains(t, got, "plain-secret")
	assert.NotContains(t, got, "sk-live")
	assert.NotContains(t, got, "pat-1")
	assert.NotContains(t, got, "proof-1")
	assert.NotContains(t, got, "flow-1")
}

func TestSanitizeAuditBodyRedactsOptionValueForSensitiveKey(t *testing.T) {
	raw := []byte(`{"key":"WeChatServerToken","value":"wechat-secret"}`)
	got, truncated := sanitizeAuditBody(raw, http.MethodPut, "/api/option/", "/api/option/")
	assert.False(t, truncated)
	assert.Contains(t, got, `"key":"WeChatServerToken"`)
	assert.Contains(t, got, `"value":"***"`)
	assert.NotContains(t, got, "wechat-secret")

	safe := []byte(`{"key":"QuotaForInviter","value":"100"}`)
	safeGot, _ := sanitizeAuditBody(safe, http.MethodPut, "/api/option/", "/api/option/")
	assert.Contains(t, safeGot, `"value":"100"`)
}

func TestSanitizeAuditBodyRedactsEmbeddedHeaderOverride(t *testing.T) {
	raw := []byte(`{"name":"ch","header_override":"{\"Authorization\":\"Bearer sk-nested\",\"X-Debug\":\"1\"}"}`)
	got, truncated := sanitizeAuditBody(raw, http.MethodPost, "/api/channel/", "/api/channel/")
	assert.False(t, truncated)
	assert.NotContains(t, got, "sk-nested")
	assert.Contains(t, got, "Authorization")
	assert.Contains(t, got, "***")
	assert.Contains(t, got, "X-Debug")

	broken := []byte(`{"header_override":"{not-json"}`)
	brokenGot, _ := sanitizeAuditBody(broken, http.MethodPost, "/api/channel/", "/api/channel/")
	assert.NotContains(t, brokenGot, "{not-json")
	assert.Contains(t, brokenGot, "unparseable nested field omitted")
}

func TestSanitizeAuditBodyOmitsUnparseableRaw(t *testing.T) {
	raw := []byte(`{"password":"still-secret"`)
	got, truncated := sanitizeAuditBody(raw, http.MethodPost, "/api/user/login", "/api/user/login")
	assert.False(t, truncated)
	assert.Equal(t, fmt.Sprintf("<unparseable body omitted, %d bytes>", len(raw)), got)
	assert.NotContains(t, got, "still-secret")
}

func TestSanitizeAuditBodyKeepsEmpty(t *testing.T) {
	got, truncated := sanitizeAuditBody(nil, http.MethodPost, "", "")
	assert.Equal(t, "", got)
	assert.False(t, truncated)
	got, truncated = sanitizeAuditBody([]byte("   "), http.MethodPost, "", "")
	assert.Equal(t, "", got)
	assert.False(t, truncated)
}

func TestSanitizeAuditBodyPreservesLargeInteger(t *testing.T) {
	raw := []byte(`{"quota":9007199254740993,"password":"secret"}`)
	got, truncated := sanitizeAuditBody(raw, http.MethodPost, "/api/user/login", "/api/user/login")
	assert.False(t, truncated)
	assert.Contains(t, got, "9007199254740993")
	assert.NotContains(t, got, "secret")
}

func TestSanitizeAuditBodyRemarshalOverflowUsesPlaceholder(t *testing.T) {
	// encoding/json escapes '<' to \u003c, so a compact body can grow past TEXT.
	payload := `{"note":"` + strings.Repeat("<", 12000) + `"}`
	require.Less(t, len(payload), auditBodyLimit)
	got, truncated := sanitizeAuditBody([]byte(payload), http.MethodPost, "/api/user/login", "/api/user/login")
	require.True(t, truncated)
	assert.Contains(t, got, "unparseable body omitted")
	assert.NotContains(t, got, strings.Repeat("<", 32))
}

func TestSanitizeAuditQueryRedactsSensitiveKeys(t *testing.T) {
	got := sanitizeAuditQuery("username=alice&password=hunter2&code=oauth-code&token=sk-search&proof_token=p1&flow_token=f1")
	assert.Contains(t, got, "username=alice")
	assert.Contains(t, got, "password=%2A%2A%2A")
	assert.Contains(t, got, "code=%2A%2A%2A")
	assert.Contains(t, got, "token=%2A%2A%2A")
	assert.Contains(t, got, "proof_token=%2A%2A%2A")
	assert.Contains(t, got, "flow_token=%2A%2A%2A")
	assert.NotContains(t, got, "hunter2")
	assert.NotContains(t, got, "oauth-code")
	assert.NotContains(t, got, "sk-search")
}

func TestSanitizeAuditPathRedactsFlowToken(t *testing.T) {
	got := sanitizeAuditPath("/api/oauth/telegram/bind/secret-flow-token", "/api/oauth/telegram/bind/:flow_token")
	assert.Equal(t, "/api/oauth/telegram/bind/***", got)
	assert.NotContains(t, got, "secret-flow-token")

	plain := sanitizeAuditPath("/api/token/12", "/api/token/:id")
	assert.Equal(t, "/api/token/12", plain)
}

func TestTryEnqueueAuditLogDropsWhenFullWithoutBlocking(t *testing.T) {
	ch := make(chan *model.AuditRequestLog, 1)
	first := &model.AuditRequestLog{RequestId: "keep"}
	require.True(t, tryEnqueueAuditLog(ch, first))

	dropped := make(chan bool, 1)
	go func() {
		dropped <- tryEnqueueAuditLog(ch, &model.AuditRequestLog{RequestId: "drop-me"})
	}()
	assert.False(t, <-dropped)
	assert.Equal(t, "keep", (<-ch).RequestId)
	select {
	case extra := <-ch:
		t.Fatalf("queue accepted extra entry %q", extra.RequestId)
	default:
	}
}

func TestPeekAndRestoreRequestBodyPreservesHandlerRead(t *testing.T) {
	gin.SetMode(gin.TestMode)
	payload := []byte(`{"username":"alice","password":"peek-secret"}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(payload))
	req.ContentLength = int64(len(payload))
	c.Request = req

	peeked, truncated := peekAndRestoreRequestBody(c)
	require.Equal(t, payload, peeked)
	require.False(t, truncated)
	assert.Equal(t, int64(len(payload)), c.Request.ContentLength)

	restored, err := io.ReadAll(c.Request.Body)
	require.NoError(t, err)
	assert.Equal(t, payload, restored)
	require.NoError(t, c.Request.Body.Close())
}

func TestPeekAndRestoreRequestBodyMarksTruncation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	payload := bytes.Repeat([]byte("a"), auditBodyLimit+8)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(payload))

	peeked, truncated := peekAndRestoreRequestBody(c)
	require.True(t, truncated)
	require.Len(t, peeked, auditBodyLimit)

	restored, err := io.ReadAll(c.Request.Body)
	require.NoError(t, err)
	assert.Equal(t, payload, restored)
}

func TestAuditRequestResponseWriterCapturesPlaintextAndTruncation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	writer := &auditRequestResponseWriter{
		ResponseWriter: c.Writer,
		body:           bytes.NewBuffer(nil),
		maxSize:        auditBodyLimit,
	}
	payload := bytes.Repeat([]byte("x"), auditBodyLimit+16)
	n, err := writer.Write(payload)
	require.NoError(t, err)
	assert.Equal(t, len(payload), n)
	assert.True(t, writer.truncated)
	assert.Equal(t, auditBodyLimit, writer.body.Len())
	assert.Equal(t, bytes.Repeat([]byte("x"), auditBodyLimit), writer.body.Bytes())
}
