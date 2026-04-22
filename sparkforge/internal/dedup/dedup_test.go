package dedup

import (
	"testing"

	"github.com/gartner24/forge/sparkforge/internal/model"
)

// Kano 2 (Core): deduplication prevents alert storms. Must be correct.

func newTestStore(t *testing.T) *Store {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	s, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return s
}

func TestDedup_NewAlert(t *testing.T) {
	s := newTestStore(t)
	msg := model.Message{
		Title:     "api is down",
		Priority:  model.PriorityHigh,
		Source:    "watchforge",
		EventType: "monitor:api-health",
	}

	alert, err := s.Add(msg)
	if err != nil {
		t.Fatalf("Add() error: %v", err)
	}
	if alert.ID == "" {
		t.Error("Add() returned empty ID")
	}
	if alert.FiredAt.IsZero() {
		t.Error("Add() returned zero FiredAt")
	}
}

func TestDedup_IsDuplicate(t *testing.T) {
	s := newTestStore(t)
	msg := model.Message{
		Title:     "api is down",
		Priority:  model.PriorityHigh,
		Source:    "watchforge",
		EventType: "monitor:api-health",
	}

	// Before adding: not a duplicate.
	if s.IsDuplicate("watchforge", "monitor:api-health") {
		t.Error("IsDuplicate() = true before any alert, want false")
	}

	if _, err := s.Add(msg); err != nil {
		t.Fatalf("Add() error: %v", err)
	}

	// After adding: is a duplicate.
	if !s.IsDuplicate("watchforge", "monitor:api-health") {
		t.Error("IsDuplicate() = false after Add(), want true")
	}

	// Different source: not a duplicate.
	if s.IsDuplicate("other", "monitor:api-health") {
		t.Error("IsDuplicate() = true for different source, want false")
	}

	// Different event_type: not a duplicate.
	if s.IsDuplicate("watchforge", "monitor:db") {
		t.Error("IsDuplicate() = true for different event_type, want false")
	}
}

func TestDedup_EmptyEventTypeNeverDuplicate(t *testing.T) {
	s := newTestStore(t)
	// A message with no event_type should never be considered a duplicate.
	if s.IsDuplicate("watchforge", "") {
		t.Error("IsDuplicate() = true with empty event_type, want false")
	}
	// Even after adding a message with empty event_type.
	_, _ = s.Add(model.Message{
		Title: "deploy started", Priority: model.PriorityLow, Source: "smeltforge",
	})
	if s.IsDuplicate("smeltforge", "") {
		t.Error("IsDuplicate() = true with empty event_type after Add(), want false")
	}
}

func TestDedup_Remove(t *testing.T) {
	s := newTestStore(t)
	msg := model.Message{
		Title:     "ssl expiring",
		Priority:  model.PriorityCritical,
		Source:    "watchforge",
		EventType: "monitor:ssl-hemis",
	}
	alert, _ := s.Add(msg)

	if err := s.Remove(alert.ID); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}

	// After removal: no longer a duplicate.
	if s.IsDuplicate("watchforge", "monitor:ssl-hemis") {
		t.Error("IsDuplicate() = true after Remove(), want false")
	}

	// Remove nonexistent.
	if err := s.Remove("nonexistent"); err == nil {
		t.Error("Remove() nonexistent ID should return error")
	}
}

func TestDedup_List(t *testing.T) {
	s := newTestStore(t)

	alerts, err := s.List()
	if err != nil {
		t.Fatalf("List() error on empty store: %v", err)
	}
	if alerts == nil {
		t.Error("List() on empty store returned nil, want empty slice")
	}
	if len(alerts) != 0 {
		t.Errorf("List() on empty store returned %d entries, want 0", len(alerts))
	}

	s.Add(model.Message{Title: "a", Priority: model.PriorityHigh, Source: "watchforge", EventType: "monitor:a"})
	s.Add(model.Message{Title: "b", Priority: model.PriorityMedium, Source: "watchforge", EventType: "monitor:b"})

	alerts, _ = s.List()
	if len(alerts) != 2 {
		t.Errorf("List() returned %d alerts, want 2", len(alerts))
	}
}

func TestDedup_MultipleAlerts_IndependentKeys(t *testing.T) {
	s := newTestStore(t)

	s.Add(model.Message{Title: "api down", Priority: model.PriorityHigh, Source: "watchforge", EventType: "monitor:api"})
	s.Add(model.Message{Title: "db down", Priority: model.PriorityHigh, Source: "watchforge", EventType: "monitor:db"})

	if !s.IsDuplicate("watchforge", "monitor:api") {
		t.Error("monitor:api should be active")
	}
	if !s.IsDuplicate("watchforge", "monitor:db") {
		t.Error("monitor:db should be active")
	}
	if s.IsDuplicate("watchforge", "monitor:cache") {
		t.Error("monitor:cache should not be active")
	}
}
