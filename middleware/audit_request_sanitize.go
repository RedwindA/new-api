package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const (
	auditNestedFieldOmitted = "<unparseable nested field omitted>"
	auditStoredBodyLimit    = 65535
)

type auditResponseOmitRule struct {
	method string
	route  string
	prefix string
}

// auditOmitResponseRules covers every audited route that issues or displays a
// secret. Matching uses the gin FullPath template when present and falls back
// to the request URL path (including parameterized templates).
var auditOmitResponseRules = []auditResponseOmitRule{
	{method: http.MethodGet, prefix: "/api/option"},
	{method: http.MethodGet, prefix: "/api/channel"},
	{method: http.MethodGet, prefix: "/api/token"},
	{method: http.MethodGet, prefix: "/api/redemption"},
	{method: http.MethodGet, route: "/api/user/token"},
	{method: http.MethodPost, route: "/api/user/reset"},
	{method: http.MethodPost, route: "/api/user/login"},
	{method: http.MethodPost, route: "/api/user/login/2fa"},
	{method: http.MethodPost, route: "/api/user/auth/refresh"},
	{method: http.MethodPost, route: "/api/user/2fa/setup"},
	{method: http.MethodPost, route: "/api/user/2fa/enable"},
	{method: http.MethodPost, route: "/api/user/2fa/disable"},
	{method: http.MethodPost, route: "/api/user/2fa/backup_codes"},
	{method: http.MethodPost, route: "/api/user/passkey/login/begin"},
	{method: http.MethodPost, route: "/api/user/passkey/login/finish"},
	{method: http.MethodPost, route: "/api/user/passkey/register/begin"},
	{method: http.MethodPost, route: "/api/user/passkey/register/finish"},
	{method: http.MethodPost, route: "/api/user/passkey/verify/begin"},
	{method: http.MethodPost, route: "/api/user/passkey/verify/finish"},
	{method: http.MethodDelete, route: "/api/user/passkey"},
	{method: http.MethodPost, route: "/api/oauth/state"},
	{method: http.MethodPost, route: "/api/oauth/telegram/bind/start"},
	{method: http.MethodGet, route: "/api/oauth/wechat"},
	{method: http.MethodGet, route: "/api/oauth/telegram/login"},
	{method: http.MethodGet, route: "/api/oauth/:provider"},
	{method: http.MethodPost, route: "/api/verify"},
	{method: http.MethodPost, route: "/api/redemption/"},
	{method: http.MethodPost, route: "/api/token/:id/key"},
	{method: http.MethodPost, route: "/api/token/batch/keys"},
	{method: http.MethodPost, route: "/api/channel/:id/key"},
}

var auditSensitiveKeys = map[string]struct{}{
	"password":           {},
	"original_password":  {},
	"new_password":       {},
	"old_password":       {},
	"password2":          {},
	"password_encrypted": {},
	"secret":             {},
	"client_secret":      {},
	"key":                {},
	"access_token":       {},
	"refresh_token":      {},
	"session_secret":     {},
	"token":              {},
	"proof_token":        {},
	"flow_token":         {},
	"totp":               {},
	"totp_code":          {},
	"code":               {},
	"backup_code":        {},
	"backup_codes":       {},
	"qr_code_data":       {},
	"private_key":        {},
}

var auditSensitivePathParams = map[string]struct{}{
	"flow_token": {},
}

var auditEmbeddedJSONFields = map[string]struct{}{
	"header_override": {},
	"param_override":  {},
}

func isAuditSensitiveKey(name string) bool {
	_, ok := auditSensitiveKeys[strings.ToLower(name)]
	return ok
}

// isSensitiveOptionKey mirrors controller.GetOptions: suffix Token/Secret/Key
// plus the audit denylist so option names like SMTPPassword are also caught.
func isSensitiveOptionKey(key string) bool {
	if isAuditSensitiveKey(key) {
		return true
	}
	if strings.HasSuffix(key, "Token") ||
		strings.HasSuffix(key, "Secret") ||
		strings.HasSuffix(key, "Key") ||
		strings.HasSuffix(key, "secret") ||
		strings.HasSuffix(key, "api_key") {
		return true
	}
	lower := strings.ToLower(key)
	return strings.HasSuffix(lower, "password")
}

func isSensitiveHeaderName(name string) bool {
	n := strings.ToLower(name)
	if isAuditSensitiveKey(n) {
		return true
	}
	switch n {
	case "authorization", "proxy-authorization", "x-api-key", "api-key":
		return true
	}
	return strings.Contains(n, "api-key") || strings.Contains(n, "apikey")
}

func splitURLPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func matchGinRoute(path, template string) bool {
	if path == template {
		return true
	}
	pathSegs := splitURLPath(path)
	tmplSegs := splitURLPath(template)
	if len(pathSegs) != len(tmplSegs) {
		return false
	}
	for i, tmpl := range tmplSegs {
		if strings.HasPrefix(tmpl, ":") {
			continue
		}
		if pathSegs[i] != tmpl {
			return false
		}
	}
	return true
}

func shouldOmitAuditResponseBody(method string, route string, path string) bool {
	for _, rule := range auditOmitResponseRules {
		if rule.method != "" && !strings.EqualFold(rule.method, method) {
			continue
		}
		if rule.prefix != "" {
			if matchPathPrefix(path, rule.prefix) || matchPathPrefix(route, rule.prefix) {
				return true
			}
			continue
		}
		if rule.route == "" {
			continue
		}
		if route == rule.route || path == rule.route {
			return true
		}
		if matchGinRoute(path, rule.route) || matchGinRoute(route, rule.route) {
			return true
		}
	}
	return false
}

func isOptionUpdateRoute(method string, route string, path string) bool {
	if !strings.EqualFold(method, http.MethodPut) {
		return false
	}
	return matchPathPrefix(path, "/api/option") || matchPathPrefix(route, "/api/option") ||
		path == "/api/option/" || route == "/api/option/"
}

func sanitizeAuditQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return fmt.Sprintf(auditUnparseableBodyFmt, len(rawQuery))
	}
	for key := range values {
		if !isAuditSensitiveKey(key) {
			continue
		}
		for i := range values[key] {
			values[key][i] = "***"
		}
	}
	return truncateAuditString(values.Encode(), auditQueryMaxLen)
}

func sanitizeAuditPath(path string, route string) string {
	if path == "" {
		return ""
	}
	pathSegs := splitURLPath(path)
	if route != "" {
		routeSegs := splitURLPath(route)
		if len(pathSegs) == len(routeSegs) {
			changed := false
			for i, tmpl := range routeSegs {
				name := strings.TrimPrefix(tmpl, ":")
				if !strings.HasPrefix(tmpl, ":") {
					continue
				}
				if _, sensitive := auditSensitivePathParams[name]; !sensitive {
					continue
				}
				pathSegs[i] = "***"
				changed = true
			}
			if changed {
				return truncateAuditString("/"+strings.Join(pathSegs, "/"), auditPathMaxLen)
			}
		}
	}
	if matchPathPrefix(path, "/api/oauth/telegram/bind") && len(pathSegs) >= 5 {
		pathSegs[4] = "***"
		return truncateAuditString("/"+strings.Join(pathSegs, "/"), auditPathMaxLen)
	}
	return truncateAuditString(path, auditPathMaxLen)
}

func redactEmbeddedJSONField(raw string) string {
	var parsed any
	if err := common.UnmarshalUseNumber([]byte(raw), &parsed); err != nil {
		return auditNestedFieldOmitted
	}
	redactAuditValue(parsed, false)
	redactSensitiveHeaders(parsed)
	out, err := common.Marshal(parsed)
	if err != nil {
		return auditNestedFieldOmitted
	}
	return string(out)
}

func redactSensitiveHeaders(v any) {
	node, ok := v.(map[string]any)
	if !ok {
		return
	}
	for key := range node {
		if isSensitiveHeaderName(key) {
			node[key] = "***"
		}
	}
}

func redactAuditValue(v any, optionUpdate bool) {
	switch node := v.(type) {
	case map[string]any:
		if optionUpdate {
			if keyName, ok := node["key"].(string); ok && isSensitiveOptionKey(keyName) {
				if _, exists := node["value"]; exists {
					node["value"] = "***"
				}
			}
		}
		for key, child := range node {
			lower := strings.ToLower(key)
			if optionUpdate && lower == "key" {
				continue
			}
			if isAuditSensitiveKey(key) {
				node[key] = "***"
				continue
			}
			if _, embedded := auditEmbeddedJSONFields[lower]; embedded {
				switch typed := child.(type) {
				case string:
					node[key] = redactEmbeddedJSONField(typed)
				default:
					redactAuditValue(child, false)
					redactSensitiveHeaders(child)
				}
				continue
			}
			redactAuditValue(child, false)
		}
	case []any:
		for _, child := range node {
			redactAuditValue(child, false)
		}
	}
}

func sanitizeAuditBody(raw []byte, method string, route string, path string) (string, bool) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return "", false
	}
	var parsed any
	if err := common.UnmarshalUseNumber(raw, &parsed); err != nil {
		return fmt.Sprintf(auditUnparseableBodyFmt, len(raw)), false
	}
	redactAuditValue(parsed, isOptionUpdateRoute(method, route, path))
	out, err := common.Marshal(parsed)
	if err != nil {
		return fmt.Sprintf(auditUnparseableBodyFmt, len(raw)), false
	}
	if len(out) > auditBodyLimit || len(out) > auditStoredBodyLimit {
		return fmt.Sprintf(auditUnparseableBodyFmt, len(out)), true
	}
	return string(out), false
}
