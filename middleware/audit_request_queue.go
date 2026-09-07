package middleware

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	auditPersistQueueSize    = 256
	auditPersistWorkers      = 4
	auditPersistDropLogEvery = 10 * time.Second
)

var (
	auditPersistCh          chan *model.AuditRequestLog
	auditPersistOnce        sync.Once
	auditPersistDropLogUnix atomic.Int64
)

func startAuditPersistWorkers() {
	auditPersistOnce.Do(func() {
		auditPersistCh = make(chan *model.AuditRequestLog, auditPersistQueueSize)
		for i := 0; i < auditPersistWorkers; i++ {
			go auditPersistWorker()
		}
	})
}

func auditPersistWorker() {
	for entry := range auditPersistCh {
		model.RecordAuditRequestLog(entry)
	}
}

func tryEnqueueAuditLog(ch chan *model.AuditRequestLog, entry *model.AuditRequestLog) bool {
	if ch == nil || entry == nil {
		return false
	}
	select {
	case ch <- entry:
		return true
	default:
		return false
	}
}

func enqueueAuditRequestLog(entry *model.AuditRequestLog) bool {
	startAuditPersistWorkers()
	if tryEnqueueAuditLog(auditPersistCh, entry) {
		return true
	}
	logAuditPersistDrop()
	return false
}

func logAuditPersistDrop() {
	now := time.Now().UnixNano()
	last := auditPersistDropLogUnix.Load()
	if last != 0 && now-last < int64(auditPersistDropLogEvery) {
		return
	}
	if !auditPersistDropLogUnix.CompareAndSwap(last, now) {
		return
	}
	common.SysError("audit request persist queue is full; dropping log")
}
