package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

var ErrAuditRequestDisabled = errors.New("Request audit is not enabled")

func IsAuditRequestEnabled() bool {
	return AUDIT_DB != nil
}

const (
	AuditBodyTruncatedRequest  = 1
	AuditBodyTruncatedResponse = 2
)

// Business outcome of an audited request. new-api reports most business
// failures (wrong password, invalid params, ...) with HTTP 200 and
// {"success": false}, so the HTTP status code alone cannot distinguish a
// failed login from a successful one.
const (
	AuditResultUnknown = 0
	AuditResultSuccess = 1
	AuditResultFailure = 2
)

const auditRequestLogListSelect = "created_at, request_id, ip, user_agent, method, route, path, query, status_code, result, latency_ms, user_id, username, user_role, body_truncated"

type AuditRequestLog struct {
	CreatedAt     int64  `json:"created_at" gorm:"bigint;index:idx_audit_req_created_at,priority:1;default:0"`
	RequestId     string `json:"request_id" gorm:"type:varchar(64);index;default:''"`
	Ip            string `json:"ip" gorm:"type:varchar(64);index;default:''"`
	UserAgent     string `json:"user_agent" gorm:"type:varchar(512);default:''"`
	Method        string `json:"method" gorm:"type:varchar(16);index;default:''"`
	Route         string `json:"route" gorm:"type:varchar(256);default:''"`
	Path          string `json:"path" gorm:"type:varchar(1024);default:''"`
	Query         string `json:"query" gorm:"type:varchar(2048);default:''"`
	StatusCode    int    `json:"status_code" gorm:"index;default:0"`
	Result        int    `json:"result" gorm:"index;default:0"`
	LatencyMs     int    `json:"latency_ms" gorm:"default:0"`
	UserId        int    `json:"user_id" gorm:"index;default:0"`
	Username      string `json:"username" gorm:"type:varchar(64);index;default:''"`
	UserRole      int    `json:"user_role" gorm:"default:0"`
	RequestBody   string `json:"request_body,omitempty" gorm:"type:text"`
	ResponseBody  string `json:"response_body,omitempty" gorm:"type:text"`
	BodyTruncated int    `json:"body_truncated" gorm:"default:0"`
}

type AuditRequestLogQuery struct {
	StartTimestamp int64
	EndTimestamp   int64
	Ip             string
	Path           string
	Method         string
	StatusCodes    []int
	Result         int
	Username       string
	UserId         *int
	RequestId      string
	StartIdx       int
	Num            int
}

func clickHouseAuditLogOrder(prefix string) string {
	return prefix + "created_at desc, " + prefix + "request_id desc"
}

func applyExplicitAuditTextFilter(tx *gorm.DB, column string, value string) (*gorm.DB, error) {
	if value == "" {
		return tx, nil
	}
	if strings.Contains(value, "%") {
		condition, pattern, err := buildAuditLikeCondition(column, value)
		if err != nil {
			return nil, err
		}
		return tx.Where(condition, pattern), nil
	}
	return tx.Where(column+" = ?", value), nil
}

func buildAuditLikeCondition(column string, value string) (string, string, error) {
	pattern, err := sanitizeClickHouseLikePattern(value)
	if err != nil {
		return "", "", err
	}
	return column + " LIKE ?", pattern, nil
}

func RecordAuditRequestLog(log *AuditRequestLog) {
	if AUDIT_DB == nil || log == nil {
		return
	}
	if log.CreatedAt == 0 {
		log.CreatedAt = common.GetTimestamp()
	}
	if log.RequestId == "" {
		log.RequestId = common.NewRequestId()
	}
	if err := AUDIT_DB.Create(log).Error; err != nil {
		common.SysError("failed to record audit request log: " + err.Error())
	}
}

func GetAuditRequestLogs(query AuditRequestLogQuery) (logs []*AuditRequestLog, total int64, err error) {
	if !IsAuditRequestEnabled() {
		return nil, 0, ErrAuditRequestDisabled
	}

	tx := AUDIT_DB.Model(&AuditRequestLog{})
	if tx, err = applyExplicitAuditTextFilter(tx, "ip", query.Ip); err != nil {
		return nil, 0, err
	}
	if tx, err = applyExplicitAuditTextFilter(tx, "path", query.Path); err != nil {
		return nil, 0, err
	}
	if tx, err = applyExplicitAuditTextFilter(tx, "username", query.Username); err != nil {
		return nil, 0, err
	}
	if query.Method != "" {
		tx = tx.Where("method = ?", query.Method)
	}
	if len(query.StatusCodes) > 0 {
		tx = tx.Where("status_code IN ?", query.StatusCodes)
	}
	if query.Result != AuditResultUnknown {
		tx = tx.Where("result = ?", query.Result)
	}
	if query.UserId != nil {
		tx = tx.Where("user_id = ?", *query.UserId)
	}
	if query.RequestId != "" {
		tx = tx.Where("request_id = ?", query.RequestId)
	}
	if query.StartTimestamp != 0 {
		tx = tx.Where("created_at >= ?", query.StartTimestamp)
	}
	if query.EndTimestamp != 0 {
		tx = tx.Where("created_at <= ?", query.EndTimestamp)
	}

	if err = tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err = tx.Select(auditRequestLogListSelect).Order(clickHouseAuditLogOrder("")).Limit(query.Num).Offset(query.StartIdx).Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

func GetAuditRequestStatusCodes() ([]int, error) {
	if !IsAuditRequestEnabled() {
		return nil, ErrAuditRequestDisabled
	}
	codes := make([]int, 0)
	err := AUDIT_DB.Model(&AuditRequestLog{}).Distinct("status_code").Order("status_code").Pluck("status_code", &codes).Error
	if err != nil {
		return nil, err
	}
	return codes, nil
}

func GetAuditRequestLogDetail(requestId string, createdAt int64) (*AuditRequestLog, error) {
	if !IsAuditRequestEnabled() {
		return nil, ErrAuditRequestDisabled
	}
	if requestId == "" {
		return nil, errors.New("request_id is required")
	}

	tx := AUDIT_DB.Model(&AuditRequestLog{}).Where("request_id = ?", requestId)
	if createdAt != 0 {
		tx = tx.Where("created_at = ?", createdAt)
	}

	var log AuditRequestLog
	if err := tx.Order("created_at desc").Take(&log).Error; err != nil {
		return nil, err
	}
	return &log, nil
}
