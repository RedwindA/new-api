package model

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitAuditDBDisabledWhenUnset(t *testing.T) {
	previousAudit := AUDIT_DB
	previousEnv, hadEnv := os.LookupEnv("AUDIT_SQL_DSN")
	t.Cleanup(func() {
		AUDIT_DB = previousAudit
		if hadEnv {
			require.NoError(t, os.Setenv("AUDIT_SQL_DSN", previousEnv))
		} else {
			require.NoError(t, os.Unsetenv("AUDIT_SQL_DSN"))
		}
	})

	require.NoError(t, os.Unsetenv("AUDIT_SQL_DSN"))
	require.NoError(t, InitAuditDB())
	assert.Nil(t, AUDIT_DB)
	assert.False(t, IsAuditRequestEnabled())

	_, _, err := GetAuditRequestLogs(AuditRequestLogQuery{StartIdx: 0, Num: 10})
	require.ErrorIs(t, err, ErrAuditRequestDisabled)
	_, err = GetAuditRequestLogDetail("any", 0)
	require.ErrorIs(t, err, ErrAuditRequestDisabled)
}

func TestInitAuditDBRejectsNonClickHouseDSN(t *testing.T) {
	previousAudit := AUDIT_DB
	previousEnv, hadEnv := os.LookupEnv("AUDIT_SQL_DSN")
	t.Cleanup(func() {
		AUDIT_DB = previousAudit
		if hadEnv {
			require.NoError(t, os.Setenv("AUDIT_SQL_DSN", previousEnv))
		} else {
			require.NoError(t, os.Unsetenv("AUDIT_SQL_DSN"))
		}
	})

	cases := []string{
		"local",
		"user:pass@tcp(127.0.0.1:3306)/audit?parseTime=true",
		"postgres://user:pass@127.0.0.1:5432/audit",
		"postgresql://user:pass@127.0.0.1:5432/audit",
	}
	for _, dsn := range cases {
		t.Run(dsn, func(t *testing.T) {
			require.NoError(t, os.Setenv("AUDIT_SQL_DSN", dsn))
			err := InitAuditDB()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "only supports ClickHouse")
			assert.Nil(t, AUDIT_DB)
			assert.False(t, IsAuditRequestEnabled())
		})
	}
}
