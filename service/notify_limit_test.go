package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 限流窗口必须由 NOTIFICATION_LIMIT_DURATION_MINUTE 决定，不能在整点被重置
func TestCheckNotificationLimitWindowSpansClockHours(t *testing.T) {
	originalRedis := common.RedisEnabled
	originalCount := constant.NotifyLimitCount
	originalMinute := constant.NotificationLimitDurationMinute
	t.Cleanup(func() {
		common.RedisEnabled = originalRedis
		constant.NotifyLimitCount = originalCount
		constant.NotificationLimitDurationMinute = originalMinute
	})
	common.RedisEnabled = false
	constant.NotifyLimitCount = 2
	constant.NotificationLimitDurationMinute = 720

	tests := []struct {
		name        string
		windowStart time.Duration
		expected    bool
	}{
		{name: "limit reached two hours ago still blocks", windowStart: 2 * time.Hour, expected: false},
		{name: "limit reached after window expired allows", windowStart: 721 * time.Minute, expected: true},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userId := 900001 + i
			key := fmt.Sprintf("%d:%s", userId, dto.NotifyTypeQuotaExceed)
			notifyLimitStore.Store(key, limitCount{Count: 2, Timestamp: time.Now().Add(-tt.windowStart)})
			t.Cleanup(func() { notifyLimitStore.Delete(key) })

			canSend, err := CheckNotificationLimit(userId, dto.NotifyTypeQuotaExceed)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, canSend)
		})
	}
}
