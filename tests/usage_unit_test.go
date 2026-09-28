package tests

import (
	"testing"

	"dmama_api/internal/repository"
)

func TestUsername(t *testing.T) {
	if got := repository.Username("10090", "pwa.co.th"); got != "10090@pwa.co.th" {
		t.Fatalf("Username = %q, want %q", got, "10090@pwa.co.th")
	}
}

func TestHumanizeBytes(t *testing.T) {
	cases := []struct {
		bytes     int64
		wantValue float64
		wantUnit  string
	}{
		{0, 0, "KB"},
		{512, 0.5, "KB"},
		{2048, 2, "KB"},
		{1024 * 1024, 1, "MB"},
		{1572864, 1.5, "MB"}, // 1.5 MB
		{1024 * 1024 * 1024, 1, "GB"},
	}
	for _, c := range cases {
		value, unit := repository.HumanizeBytes(c.bytes)
		if unit != c.wantUnit {
			t.Errorf("HumanizeBytes(%d) unit = %q, want %q", c.bytes, unit, c.wantUnit)
		}
		if value != c.wantValue {
			t.Errorf("HumanizeBytes(%d) value = %v, want %v", c.bytes, value, c.wantValue)
		}
	}
}

// Record must never block, even with no worker draining a full buffer.
func TestUsageRecorderRecordNonBlocking(t *testing.T) {
	rec := repository.NewUsageRecorder(nil, 1)
	for i := 0; i < 5; i++ {
		rec.Record(repository.UsageRecord{Method: "GET", Path: "/api/test"})
	}
}
