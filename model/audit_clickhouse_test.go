package model

import (
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func closeDedicatedAuditDB() {
	if AUDIT_DB == nil || AUDIT_DB == LOG_DB || AUDIT_DB == DB {
		return
	}
	sqlDB, err := AUDIT_DB.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}

func requireAuditClickHouse(t *testing.T) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("AUDIT_TEST_DSN"))
	if dsn == "" {
		t.Skip("set AUDIT_TEST_DSN to a dedicated ClickHouse test database")
	}
	if !isClickHouseDSN(dsn) {
		t.Skip("ClickHouse audit tests require a ClickHouse DSN")
	}

	previousAudit := AUDIT_DB
	previousMaster := common.IsMasterNode
	previousEnv, hadEnv := os.LookupEnv("AUDIT_SQL_DSN")
	t.Cleanup(func() {
		if AUDIT_DB != previousAudit {
			closeDedicatedAuditDB()
		}
		AUDIT_DB = previousAudit
		common.IsMasterNode = previousMaster
		if hadEnv {
			require.NoError(t, os.Setenv("AUDIT_SQL_DSN", previousEnv))
		} else {
			require.NoError(t, os.Unsetenv("AUDIT_SQL_DSN"))
		}
	})

	common.IsMasterNode = true
	require.NoError(t, os.Setenv("AUDIT_SQL_DSN", dsn))
	if err := InitAuditDB(); err != nil {
		require.NoError(t, err, "initialize ClickHouse audit test database")
	}
	require.NotNil(t, AUDIT_DB)
	require.True(t, IsAuditRequestEnabled())
}

func truncateAuditRequestLogs(t *testing.T) {
	t.Helper()
	require.NotNil(t, AUDIT_DB)
	require.NoError(t, AUDIT_DB.Exec("TRUNCATE TABLE IF EXISTS audit_request_logs").Error)
}

func TestInitAuditDBMigratesIdempotentlyOnClickHouse(t *testing.T) {
	requireAuditClickHouse(t)

	require.NoError(t, migrateAUDITDB(), "second migrate must be idempotent")
	var createSQL string
	require.NoError(t, AUDIT_DB.Raw("SHOW CREATE TABLE audit_request_logs").Scan(&createSQL).Error)
	assert.Contains(t, createSQL, "audit_request_logs")
	assert.Contains(t, strings.ToUpper(createSQL), "MERGETREE")
}

func TestMigrateAuditDBAddsResultColumnToLegacyTable(t *testing.T) {
	requireAuditClickHouse(t)
	require.NoError(t, AUDIT_DB.Exec("DROP TABLE IF EXISTS audit_request_logs").Error)
	legacySQL := strings.Replace(clickHouseAuditCreateTableSQL(0), "\tresult Int32 DEFAULT 0,\n", "", 1)
	require.NotContains(t, legacySQL, "result Int32")
	require.NoError(t, AUDIT_DB.Exec(legacySQL).Error)
	require.NoError(t, AUDIT_DB.Exec("INSERT INTO audit_request_logs (created_at, request_id, status_code) VALUES (1, 'legacy-row', 200)").Error)

	require.NoError(t, migrateAUDITDB())
	require.NoError(t, migrateAUDITDB(), "second migrate must be idempotent")

	var createSQL string
	require.NoError(t, AUDIT_DB.Raw("SHOW CREATE TABLE audit_request_logs").Scan(&createSQL).Error)
	assert.Contains(t, createSQL, "`result` Int32")

	legacy, err := GetAuditRequestLogDetail("legacy-row", 1)
	require.NoError(t, err)
	assert.Equal(t, AuditResultUnknown, legacy.Result)
	assert.Equal(t, 200, legacy.StatusCode)

	logs, total, err := GetAuditRequestLogs(AuditRequestLogQuery{Result: AuditResultFailure, Num: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, logs)
}

func TestClickHouseAuditTTLSyncIsIdempotent(t *testing.T) {
	requireAuditClickHouse(t)

	previousTTL, hadTTL := os.LookupEnv("AUDIT_SQL_CLICKHOUSE_TTL_DAYS")
	t.Cleanup(func() {
		if hadTTL {
			require.NoError(t, os.Setenv("AUDIT_SQL_CLICKHOUSE_TTL_DAYS", previousTTL))
		} else {
			require.NoError(t, os.Unsetenv("AUDIT_SQL_CLICKHOUSE_TTL_DAYS"))
		}
	})

	require.NoError(t, os.Setenv("AUDIT_SQL_CLICKHOUSE_TTL_DAYS", "14"))
	require.NoError(t, migrateAUDITDB())
	require.NoError(t, migrateAUDITDB())
	hasTTL, err := clickHouseAuditTableHasTTL()
	require.NoError(t, err)
	assert.True(t, hasTTL, "TTL should be present after setting 14 days")

	require.NoError(t, os.Setenv("AUDIT_SQL_CLICKHOUSE_TTL_DAYS", "0"))
	require.NoError(t, migrateAUDITDB())
	require.NoError(t, migrateAUDITDB())
	hasTTL, err = clickHouseAuditTableHasTTL()
	require.NoError(t, err)
	assert.False(t, hasTTL, "TTL should be removed when days is 0")

	require.NoError(t, os.Setenv("AUDIT_SQL_CLICKHOUSE_TTL_DAYS", "7"))
	require.NoError(t, migrateAUDITDB())
	hasTTL, err = clickHouseAuditTableHasTTL()
	require.NoError(t, err)
	assert.True(t, hasTTL, "TTL should return after setting 7 days")
}

func TestClickHouseAuditWriteFilterAndPagination(t *testing.T) {
	requireAuditClickHouse(t)
	truncateAuditRequestLogs(t)

	createdAt := common.GetTimestamp()
	largeJSON := `{"username":"alice","password":"***","pad":"` + strings.Repeat("x", 60000) + `"}`
	require.NoError(t, AUDIT_DB.Create(&AuditRequestLog{
		CreatedAt:     createdAt,
		RequestId:     "ch-login-fail",
		Ip:            "1.2.3.4",
		Method:        "POST",
		Path:          "/api/user/login",
		StatusCode:    401,
		Result:        AuditResultFailure,
		UserId:        0,
		RequestBody:   largeJSON,
		ResponseBody:  `{"success":false}`,
		BodyTruncated: 0,
	}).Error)
	require.NoError(t, AUDIT_DB.Create(&AuditRequestLog{
		CreatedAt:    createdAt + 1,
		RequestId:    "ch-channel-get",
		Ip:           "8.8.8.8",
		Method:       "GET",
		Path:         "/api/channel/",
		StatusCode:   401,
		Result:       AuditResultFailure,
		RequestBody:  "",
		ResponseBody: "<response body omitted by policy>",
	}).Error)
	require.NoError(t, AUDIT_DB.Create(&AuditRequestLog{
		CreatedAt:    createdAt + 2,
		RequestId:    "ch-admin-ok",
		Ip:           "1.2.3.4",
		Method:       "GET",
		Path:         "/api/option/",
		StatusCode:   200,
		Result:       AuditResultSuccess,
		UserId:       1,
		Username:     "root",
		RequestBody:  "",
		ResponseBody: "<response body omitted by policy>",
	}).Error)

	logs, total, err := GetAuditRequestLogs(AuditRequestLogQuery{
		Ip:          "1.2.3.4",
		StatusCodes: []int{401},
		StartIdx:    0,
		Num:         20,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	assert.Equal(t, "ch-login-fail", logs[0].RequestId)
	assert.Empty(t, logs[0].RequestBody)
	assert.Empty(t, logs[0].ResponseBody)

	pathLogs, pathTotal, err := GetAuditRequestLogs(AuditRequestLogQuery{
		Path:     "/api/channel/%",
		StartIdx: 0,
		Num:      20,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), pathTotal)
	require.Len(t, pathLogs, 1)
	assert.Equal(t, "ch-channel-get", pathLogs[0].RequestId)

	// ClickHouse LIKE treats `_` as a wildcard unless escaped; sanitizer must keep it literal.
	underscoreLogs, underscoreTotal, err := GetAuditRequestLogs(AuditRequestLogQuery{
		Path:     "/api/user_login%",
		StartIdx: 0,
		Num:      20,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), underscoreTotal)
	assert.Empty(t, underscoreLogs)

	multiStatusLogs, multiStatusTotal, err := GetAuditRequestLogs(AuditRequestLogQuery{
		StatusCodes: []int{200, 401},
		StartIdx:    0,
		Num:         20,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), multiStatusTotal)
	assert.Len(t, multiStatusLogs, 3)

	statusCodes, err := GetAuditRequestStatusCodes()
	require.NoError(t, err)
	assert.Equal(t, []int{200, 401}, statusCodes)

	failedLogs, failedTotal, err := GetAuditRequestLogs(AuditRequestLogQuery{
		Result:   AuditResultFailure,
		StartIdx: 0,
		Num:      20,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), failedTotal)
	require.Len(t, failedLogs, 2)
	assert.Equal(t, AuditResultFailure, failedLogs[0].Result)
	assert.Equal(t, AuditResultFailure, failedLogs[1].Result)

	successLogs, successTotal, err := GetAuditRequestLogs(AuditRequestLogQuery{
		Result:   AuditResultSuccess,
		StartIdx: 0,
		Num:      20,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), successTotal)
	require.Len(t, successLogs, 1)
	assert.Equal(t, "ch-admin-ok", successLogs[0].RequestId)

	page, pageTotal, err := GetAuditRequestLogs(AuditRequestLogQuery{StartIdx: 1, Num: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(3), pageTotal)
	require.Len(t, page, 1)
	assert.Equal(t, "ch-channel-get", page[0].RequestId)

	detail, err := GetAuditRequestLogDetail("ch-login-fail", createdAt)
	require.NoError(t, err)
	assert.Equal(t, largeJSON, detail.RequestBody)
	assert.GreaterOrEqual(t, len(detail.RequestBody), 60000)
	assert.NotContains(t, detail.RequestBody, "plain-login-secret")
	assert.Contains(t, detail.RequestBody, `"password":"***"`)
}

func TestClickHouseAuditRecordAndListRoundTrip(t *testing.T) {
	requireAuditClickHouse(t)
	truncateAuditRequestLogs(t)

	RecordAuditRequestLog(&AuditRequestLog{
		RequestId:  "ch-record-1",
		Path:       "/api/user/login",
		CreatedAt:  11,
		Method:     "POST",
		StatusCode: 401,
	})

	got, err := GetAuditRequestLogDetail("ch-record-1", 11)
	require.NoError(t, err)
	assert.Equal(t, "/api/user/login", got.Path)

	_, err = GetAuditRequestLogDetail("", 0)
	require.Error(t, err)
}
