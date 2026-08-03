package model

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/driver/clickhouse"
	"gorm.io/gorm"
)

const requestCaptureTable = "request_response_captures"

var (
	requestCaptureDB    *gorm.DB
	requestCaptureQueue chan RequestResponseCapture
	requestCaptureWG    sync.WaitGroup
)

// RequestResponseCapture is a temporary, targeted record of a relay request and
// the response returned to the client. It is stored in a dedicated ClickHouse
// database so captured payloads never enter the normal application log database.
type RequestResponseCapture struct {
	RequestId           string    `gorm:"column:request_id"`
	UserId              int       `gorm:"column:user_id"`
	TokenId             int       `gorm:"column:token_id"`
	ChannelId           int       `gorm:"column:channel_id"`
	CreatedAt           time.Time `gorm:"column:created_at"`
	DurationMs          int64     `gorm:"column:duration_ms"`
	Method              string    `gorm:"column:method"`
	Path                string    `gorm:"column:path"`
	ModelName           string    `gorm:"column:model_name"`
	Status              int       `gorm:"column:status"`
	RequestContentType  string    `gorm:"column:request_content_type"`
	ResponseContentType string    `gorm:"column:response_content_type"`
	RequestBody         string    `gorm:"column:request_body"`
	ResponseBody        string    `gorm:"column:response_body"`
	RequestBytes        int64     `gorm:"column:request_bytes"`
	ResponseBytes       int64     `gorm:"column:response_bytes"`
	RequestTruncated    bool      `gorm:"column:request_truncated"`
	ResponseTruncated   bool      `gorm:"column:response_truncated"`
	NodeName            string    `gorm:"column:node_name"`
}

func (RequestResponseCapture) TableName() string {
	return requestCaptureTable
}

func InitRequestCaptureDB() error {
	dsn := os.Getenv("REQUEST_CAPTURE_CLICKHOUSE_DSN")
	if dsn == "" {
		return nil
	}
	if !isClickHouseDSN(dsn) {
		return fmt.Errorf("REQUEST_CAPTURE_CLICKHOUSE_DSN must be a ClickHouse connection string")
	}

	db, err := gorm.Open(clickhouse.Open(normalizeClickHouseDSN(dsn)), &gorm.Config{
		PrepareStmt: false,
	})
	if err != nil {
		return fmt.Errorf("open request capture ClickHouse: %w", err)
	}

	createTableSQL := `
CREATE TABLE IF NOT EXISTS request_response_captures (
	request_id String,
	user_id Int32,
	token_id Int32,
	channel_id Int32,
	created_at DateTime64(3),
	duration_ms Int64,
	method LowCardinality(String),
	path String,
	model_name String,
	status UInt16,
	request_content_type String,
	response_content_type String,
	request_body String,
	response_body String,
	request_bytes Int64,
	response_bytes Int64,
	request_truncated UInt8,
	response_truncated UInt8,
	node_name LowCardinality(String)
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (user_id, created_at, request_id)
TTL created_at + INTERVAL 7 DAY DELETE`
	if err := db.Exec(createTableSQL).Error; err != nil {
		return fmt.Errorf("create request capture table: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("configure request capture ClickHouse: %w", err)
	}
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	requestCaptureDB = db
	requestCaptureQueue = make(chan RequestResponseCapture, 32)
	requestCaptureWG.Add(1)
	go runRequestCaptureWriter()
	common.SysLog("targeted request capture enabled for user ID 10578")
	return nil
}

func RequestCaptureEnabled() bool {
	return requestCaptureQueue != nil
}

// EnqueueRequestCapture never blocks a relay request. If ClickHouse cannot keep
// up, the capture is dropped and the normal request continues.
func EnqueueRequestCapture(capture RequestResponseCapture) {
	select {
	case requestCaptureQueue <- capture:
	default:
		common.SysError(fmt.Sprintf("request capture queue full, dropping request_id=%s user_id=%d", capture.RequestId, capture.UserId))
	}
}

func runRequestCaptureWriter() {
	defer requestCaptureWG.Done()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	batch := make([]RequestResponseCapture, 0, 20)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := requestCaptureDB.Create(&batch).Error; err != nil {
			common.SysError(fmt.Sprintf("failed to write %d request captures: %v", len(batch), err))
		}
		batch = batch[:0]
	}

	for {
		select {
		case capture, ok := <-requestCaptureQueue:
			if !ok {
				flush()
				return
			}
			batch = append(batch, capture)
			if len(batch) == cap(batch) {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func closeRequestCaptureDB() error {
	if requestCaptureQueue == nil {
		return nil
	}
	close(requestCaptureQueue)
	requestCaptureWG.Wait()
	requestCaptureQueue = nil

	sqlDB, err := requestCaptureDB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
