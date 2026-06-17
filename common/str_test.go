package common

import "testing"

func TestMaskSensitiveInfoInsufficientBalanceErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{
			name: "user quota insufficient",
			in:   "upstream error: 用户额度不足, 剩余额度 0.001, need 0.002",
		},
		{
			name: "pre consume quota failed",
			in:   "预扣费额度失败, 用户剩余额度 0, 预扣费额度 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskSensitiveInfo(tt.in); got != "余额不足" {
				t.Fatalf("MaskSensitiveInfo() = %q, want %q", got, "余额不足")
			}
		})
	}
}
