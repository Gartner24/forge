package deliverylog

import (
	"testing"
	"time"

	"github.com/gartner24/forge/sparkforge/internal/model"
)

// Kano 3 (Auxiliary): delivery log is the audit trail.

func newTestLog(t *testing.T) *Log {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	l, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return l
}

func TestLog_AppendAndRead(t *testing.T) {
	l := newTestLog(t)

	r := model.DeliveryRecord{
		Timestamp:   time.Now().UTC(),
		ChannelID:   "ch-001",
		ChannelName: "slack-alerts",
		Title:       "API is down",
		Priority:    model.PriorityHigh,
		Source:      "watchforge",
		Status:      "ok",
	}

	if err := l.Append(r); err != nil {
		t.Fatalf("Append() error: %v", err)
	}

	records, err := l.Read(time.Time{})
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("Read() returned %d records, want 1", len(records))
	}
	got := records[0]
	if got.ChannelID != "ch-001" {
		t.Errorf("ChannelID = %q, want %q", got.ChannelID, "ch-001")
	}
	if got.Status != "ok" {
		t.Errorf("Status = %q, want %q", got.Status, "ok")
	}
}

func TestLog_EmptyReadReturnsSlice(t *testing.T) {
	l := newTestLog(t)
	records, err := l.Read(time.Time{})
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if records == nil {
		t.Error("Read() on empty log returned nil, want empty slice")
	}
}

func TestLog_AppendIsAppendOnly(t *testing.T) {
	l := newTestLog(t)

	for i := 0; i < 5; i++ {
		_ = l.Append(model.DeliveryRecord{
			Timestamp:   time.Now().UTC(),
			ChannelID:   "ch-1",
			ChannelName: "test",
			Title:       "msg",
			Priority:    model.PriorityLow,
			Source:      "test",
			Status:      "ok",
		})
	}

	records, _ := l.Read(time.Time{})
	if len(records) != 5 {
		t.Errorf("Read() returned %d records after 5 appends, want 5", len(records))
	}
}

func TestLog_FilterBySince(t *testing.T) {
	l := newTestLog(t)

	old := model.DeliveryRecord{
		Timestamp:   time.Now().Add(-2 * time.Hour).UTC(),
		ChannelID:   "ch-1",
		ChannelName: "test",
		Title:       "old msg",
		Priority:    model.PriorityLow,
		Source:      "test",
		Status:      "ok",
	}
	recent := model.DeliveryRecord{
		Timestamp:   time.Now().UTC(),
		ChannelID:   "ch-1",
		ChannelName: "test",
		Title:       "recent msg",
		Priority:    model.PriorityLow,
		Source:      "test",
		Status:      "ok",
	}
	_ = l.Append(old)
	_ = l.Append(recent)

	since := time.Now().Add(-30 * time.Minute)
	records, err := l.Read(since)
	if err != nil {
		t.Fatalf("Read(since) error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("Read(since) returned %d records, want 1", len(records))
	}
	if records[0].Title != "recent msg" {
		t.Errorf("Read(since) returned wrong record: %q", records[0].Title)
	}
}

func TestLog_ErrorRecordPreservesError(t *testing.T) {
	l := newTestLog(t)

	_ = l.Append(model.DeliveryRecord{
		Timestamp:   time.Now().UTC(),
		ChannelID:   "ch-1",
		ChannelName: "gotify",
		Title:       "test",
		Priority:    model.PriorityHigh,
		Source:      "watchforge",
		Status:      "error",
		Error:       "connection refused",
	})

	records, _ := l.Read(time.Time{})
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Status != "error" {
		t.Errorf("Status = %q, want %q", records[0].Status, "error")
	}
	if records[0].Error != "connection refused" {
		t.Errorf("Error = %q, want %q", records[0].Error, "connection refused")
	}
}
