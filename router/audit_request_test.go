package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditRequestRoutesRequireAdministrator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousAuditDB := model.AUDIT_DB
	previousRateLimit := common.GlobalApiRateLimitEnable
	model.AUDIT_DB = nil
	common.GlobalApiRateLimitEnable = false
	t.Cleanup(func() {
		model.AUDIT_DB = previousAuditDB
		common.GlobalApiRateLimitEnable = previousRateLimit
	})

	for _, role := range []int{0, common.RoleCommonUser, common.RoleAdminUser} {
		engine := gin.New()
		engine.Use(sessions.Sessions("audit-test", cookie.NewStore([]byte("audit-route-test-secret"))))
		engine.Use(func(c *gin.Context) {
			if role != 0 {
				session := sessions.Default(c)
				session.Set("id", 1)
				session.Set("username", "audit-test")
				session.Set("role", role)
				session.Set("status", common.UserStatusEnabled)
				session.Set("group", "default")
			}
			c.Next()
		})
		SetApiRouter(engine)

		for _, path := range []string{"/api/audit_request/", "/api/audit_request/detail?request_id=test", "/api/audit_request/status_codes"} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("New-Api-User", "1")
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)

			var response struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
			}
			require.NoError(t, common.Unmarshal(rec.Body.Bytes(), &response))
			assert.False(t, response.Success)
			if role == 0 {
				assert.Equal(t, http.StatusUnauthorized, rec.Code)
			} else {
				assert.Equal(t, http.StatusOK, rec.Code)
			}
			if role == common.RoleAdminUser {
				assert.Equal(t, model.ErrAuditRequestDisabled.Error(), response.Message)
			} else {
				assert.NotEqual(t, model.ErrAuditRequestDisabled.Error(), response.Message)
				assert.Empty(t, rec.Header().Get("Auth-Version"))
			}
		}
	}
}
