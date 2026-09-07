package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	ginGzip "github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	auditCHOnce    sync.Once
	auditCHSkipErr error
)

func requireAuditClickHouse(t *testing.T) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("AUDIT_TEST_DSN"))
	if dsn == "" {
		t.Skip("set AUDIT_TEST_DSN to a dedicated ClickHouse test database")
	}
	if !strings.HasPrefix(dsn, "clickhouse://") &&
		!strings.HasPrefix(dsn, "tcp://") &&
		!strings.HasPrefix(dsn, "http://") &&
		!strings.HasPrefix(dsn, "https://") {
		t.Skip("ClickHouse audit tests require a ClickHouse DSN")
	}

	auditCHOnce.Do(func() {
		common.IsMasterNode = true
		if err := os.Setenv("AUDIT_SQL_DSN", dsn); err != nil {
			auditCHSkipErr = err
			return
		}
		if err := model.InitAuditDB(); err != nil {
			auditCHSkipErr = err
		}
	})
	if auditCHSkipErr != nil {
		require.NoError(t, auditCHSkipErr, "initialize ClickHouse audit test database")
	}
	require.True(t, model.IsAuditRequestEnabled())
	require.NoError(t, model.AUDIT_DB.Exec("TRUNCATE TABLE IF EXISTS audit_request_logs").Error)
}

func setupAuditIntegration(t *testing.T) *gin.Engine {
	t.Helper()
	requireAuditClickHouse(t)
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(RequestId())
	api := engine.Group("/api")
	api.Use(ginGzip.Gzip(ginGzip.DefaultCompression))
	api.Use(AuditRequest())
	// Mirrors production: common.ApiErrorI18n answers business failures with HTTP 200.
	api.POST("/user/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid password"})
	})
	api.POST("/user/login/2fa", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"access_token": "sk-login-access-token"}})
	})
	api.POST("/user/register", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid password"})
	})
	api.POST("/user/reset", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": "reset-plain-password"})
	})
	api.GET("/user/token", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": "sk-user-access-token"})
	})
	api.POST("/token/:id/key", func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "no permission", "key": "sk-should-be-redacted"})
	})
	api.POST("/redemption/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": []string{"redeem-code-aaa", "redeem-code-bbb"}})
	})
	api.POST("/token/batch/keys", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"keys": map[string]string{"1": "sk-batch-one"}}})
	})
	api.POST("/verify", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"proof_token": "proof-secret", "expires_at": 1}})
	})
	api.POST("/user/2fa/setup", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
			"secret":       "JBSWY3DPEHPK3PXP",
			"qr_code_data": "otpauth://totp/new-api?secret=JBSWY3DPEHPK3PXP",
			"backup_codes": []string{"backup-aa"},
		}})
	})
	api.GET("/channel/", func(c *gin.Context) {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized"})
	})
	api.GET("/option/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"ChannelSecret": "plaintext-option-secret"}})
	})
	api.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
	return engine
}

func TestAuditRequestRecordsRateLimitedRequest(t *testing.T) {
	requireAuditClickHouse(t)
	previousRedis := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() { common.RedisEnabled = previousRedis })
	engine := gin.New()
	engine.Use(RequestId(), AuditRequest(), rateLimitFactory(1, 60, t.Name()))
	handled := 0
	engine.POST("/api/user/login", func(c *gin.Context) {
		handled++
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/user/login", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	waitForAuditLog(t, rec.Header().Get(common.RequestIdKey))
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/user/login", nil))
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, 1, handled, "rate-limited request must not reach the handler")
	entry := waitForAuditLog(t, rec.Header().Get(common.RequestIdKey))
	assert.Equal(t, http.StatusTooManyRequests, entry.StatusCode)
	assert.Equal(t, model.AuditResultFailure, entry.Result)
}

func waitForAuditLog(t *testing.T, requestID string) *model.AuditRequestLog {
	t.Helper()
	var log *model.AuditRequestLog
	var lastErr error
	require.Eventually(t, func() bool {
		got, err := model.GetAuditRequestLogDetail(requestID, 0)
		if err != nil {
			lastErr = err
			return false
		}
		log = got
		return true
	}, 15*time.Second, 50*time.Millisecond, "request_id=%s last_err=%v", requestID, lastErr)
	require.NotNil(t, log)
	return log
}

func TestAuditRequestLoginRedactsPasswordAndStoresJSON(t *testing.T) {
	engine := setupAuditIntegration(t)
	body := `{"username":"alice","password":"plain-login-secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	requestID := rec.Header().Get(common.RequestIdKey)
	require.NotEmpty(t, requestID)

	log := waitForAuditLog(t, requestID)
	assert.Equal(t, http.StatusOK, log.StatusCode)
	assert.Equal(t, model.AuditResultFailure, log.Result)
	assert.Equal(t, 0, log.UserId)
	assert.Contains(t, log.RequestBody, `"password":"***"`)
	assert.NotContains(t, log.RequestBody, "plain-login-secret")
	assert.Contains(t, log.ResponseBody, `"message":"invalid password"`)
	assert.NotEqual(t, "<response body omitted by policy>", log.ResponseBody)

	var stored []model.AuditRequestLog
	require.NoError(t, model.AUDIT_DB.Find(&stored).Error)
	raw, err := common.Marshal(stored)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "plain-login-secret")
}

func TestAuditRequestSuccessfulLoginOmitsBodyAndMarksSuccess(t *testing.T) {
	engine := setupAuditIntegration(t)
	req := httptest.NewRequest(http.MethodPost, "/api/user/login/2fa", strings.NewReader(`{"code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "sk-login-access-token")

	log := waitForAuditLog(t, rec.Header().Get(common.RequestIdKey))
	assert.Equal(t, http.StatusOK, log.StatusCode)
	assert.Equal(t, model.AuditResultSuccess, log.Result)
	assert.Equal(t, "<response body omitted by policy>", log.ResponseBody)
	assert.NotContains(t, log.ResponseBody, "sk-login-access-token")
}

func TestAuditRequestRecordsUnauthorizedChannelGet(t *testing.T) {
	engine := setupAuditIntegration(t)
	req := httptest.NewRequest(http.MethodGet, "/api/channel/", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	log := waitForAuditLog(t, rec.Header().Get(common.RequestIdKey))
	assert.Equal(t, 0, log.UserId)
	assert.Equal(t, http.StatusUnauthorized, log.StatusCode)
	assert.Equal(t, model.AuditResultFailure, log.Result)
	assert.Equal(t, "/api/channel/", log.Path)
}

func TestAuditRequestCapturesGzipPlaintextResponse(t *testing.T) {
	engine := setupAuditIntegration(t)
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(`{"username":"bob"}`))
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Contains(t, rec.Header().Get("Content-Encoding"), "gzip")
	gr, err := gzip.NewReader(rec.Body)
	require.NoError(t, err)
	plain, err := io.ReadAll(gr)
	require.NoError(t, err)
	assert.Contains(t, string(plain), `"success":false`)

	log := waitForAuditLog(t, rec.Header().Get(common.RequestIdKey))
	assert.True(t, strings.HasPrefix(strings.TrimSpace(log.ResponseBody), "{"))
	assert.Contains(t, log.ResponseBody, `"success":false`)
	assert.NotContains(t, log.ResponseBody, "\x1f\x8b")
}

func TestAuditRequestTruncatedBodyUsesPlaceholder(t *testing.T) {
	engine := setupAuditIntegration(t)
	huge := `{"username":"alice","password":"` + strings.Repeat("s", auditBodyLimit) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(huge))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	log := waitForAuditLog(t, rec.Header().Get(common.RequestIdKey))
	assert.Equal(t, model.AuditBodyTruncatedRequest, log.BodyTruncated&model.AuditBodyTruncatedRequest)
	assert.Contains(t, log.RequestBody, "unparseable body omitted")
	assert.NotContains(t, log.RequestBody, strings.Repeat("s", 32))
}

func TestAuditRequestOmitsSecretIssuingResponseBodies(t *testing.T) {
	engine := setupAuditIntegration(t)
	cases := []struct {
		method string
		path   string
		secret string
	}{
		{http.MethodPost, "/api/user/reset", "reset-plain-password"},
		{http.MethodGet, "/api/user/token", "sk-user-access-token"},
		{http.MethodPost, "/api/redemption/", "redeem-code-aaa"},
		{http.MethodPost, "/api/token/batch/keys", "sk-batch-one"},
		{http.MethodPost, "/api/verify", "proof-secret"},
		{http.MethodPost, "/api/user/2fa/setup", "JBSWY3DPEHPK3PXP"},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)
			assert.Contains(t, rec.Body.String(), tc.secret)

			log := waitForAuditLog(t, rec.Header().Get(common.RequestIdKey))
			assert.Equal(t, "<response body omitted by policy>", log.ResponseBody)
			assert.NotContains(t, log.ResponseBody, tc.secret)
		})
	}
}

func TestAuditRequestCapturesNon200ResponseOnOmitRoute(t *testing.T) {
	engine := setupAuditIntegration(t)
	req := httptest.NewRequest(http.MethodPost, "/api/token/12/key", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	log := waitForAuditLog(t, rec.Header().Get(common.RequestIdKey))
	assert.Equal(t, http.StatusForbidden, log.StatusCode)
	assert.Contains(t, log.ResponseBody, `"message":"no permission"`)
	assert.Contains(t, log.ResponseBody, `"key":"***"`)
	assert.NotContains(t, log.ResponseBody, "sk-should-be-redacted")
	assert.Equal(t, 0, log.BodyTruncated&model.AuditBodyTruncatedResponse)
}

func TestAuditRequestOmitsOptionResponseBody(t *testing.T) {
	engine := setupAuditIntegration(t)
	req := httptest.NewRequest(http.MethodGet, "/api/option/", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	assert.Contains(t, rec.Body.String(), "plaintext-option-secret")

	log := waitForAuditLog(t, rec.Header().Get(common.RequestIdKey))
	assert.Equal(t, "<response body omitted by policy>", log.ResponseBody)
	assert.NotContains(t, log.ResponseBody, "plaintext-option-secret")
}

func TestAuditRequestSkipsUnmatchedStatusRoute(t *testing.T) {
	engine := setupAuditIntegration(t)
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var count int64
	require.NoError(t, model.AUDIT_DB.Model(&model.AuditRequestLog{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestAuditRequestNoOpWhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previous := model.AUDIT_DB
	model.AUDIT_DB = nil
	t.Cleanup(func() { model.AUDIT_DB = previous })

	var seen []byte
	engine := gin.New()
	engine.Use(AuditRequest())
	engine.POST("/api/user/login", func(c *gin.Context) {
		seen, _ = io.ReadAll(c.Request.Body)
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
	payload := []byte(`{"username":"alice","password":"plain-login-secret"}`)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(payload)))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, payload, seen)
	assert.False(t, model.IsAuditRequestEnabled())
}

func TestAuditRequestRestoresBodyForHandler(t *testing.T) {
	requireAuditClickHouse(t)
	gin.SetMode(gin.TestMode)

	var seen []byte
	engine := gin.New()
	engine.Use(AuditRequest())
	engine.POST("/api/user/login", func(c *gin.Context) {
		seen, _ = io.ReadAll(c.Request.Body)
		c.Status(http.StatusOK)
	})
	payload := []byte(`{"username":"alice","password":"handler-secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(payload))
	engine.ServeHTTP(httptest.NewRecorder(), req)
	assert.Equal(t, payload, seen)
}
