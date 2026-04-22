package registry

import (
	"testing"
	"time"
)

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	reg, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestRegistry_AddAndLoad(t *testing.T) {
	reg := newTestRegistry(t)
	m := Monitor{
		ID: "m1", Name: "Test", Type: TypeHTTP,
		Target: "https://example.com", IntervalSec: 60,
		CreatedAt: time.Now().UTC(),
	}
	if err := reg.AddMonitor(m); err != nil {
		t.Fatal(err)
	}
	monitors, err := reg.LoadMonitors()
	if err != nil {
		t.Fatal(err)
	}
	if len(monitors) != 1 || monitors[0].ID != "m1" {
		t.Fatalf("expected 1 monitor, got %d", len(monitors))
	}
}

func TestRegistry_UpdateMonitor(t *testing.T) {
	reg := newTestRegistry(t)
	reg.AddMonitor(Monitor{ID: "m1", Name: "Old", Type: TypeHTTP, CreatedAt: time.Now()})

	if err := reg.UpdateMonitor("m1", func(m *Monitor) { m.Name = "New" }); err != nil {
		t.Fatal(err)
	}

	monitors, _ := reg.LoadMonitors()
	if monitors[0].Name != "New" {
		t.Errorf("expected name 'New', got %q", monitors[0].Name)
	}
}

func TestRegistry_UpdateMonitor_NotFound(t *testing.T) {
	reg := newTestRegistry(t)
	err := reg.UpdateMonitor("nonexistent", func(m *Monitor) {})
	if err == nil {
		t.Error("expected error for missing monitor")
	}
}

func TestRegistry_DeleteMonitor(t *testing.T) {
	reg := newTestRegistry(t)
	reg.AddMonitor(Monitor{ID: "m1", Name: "A", Type: TypeHTTP, CreatedAt: time.Now()})
	reg.AddMonitor(Monitor{ID: "m2", Name: "B", Type: TypeTCP, CreatedAt: time.Now()})

	if err := reg.DeleteMonitor("m1"); err != nil {
		t.Fatal(err)
	}
	monitors, _ := reg.LoadMonitors()
	if len(monitors) != 1 || monitors[0].ID != "m2" {
		t.Fatalf("expected 1 monitor after delete, got %d", len(monitors))
	}
}

func TestRegistry_DeleteMonitor_NotFound(t *testing.T) {
	reg := newTestRegistry(t)
	if err := reg.DeleteMonitor("x"); err == nil {
		t.Error("expected error when deleting missing monitor")
	}
}

func TestRegistry_GetMonitor(t *testing.T) {
	reg := newTestRegistry(t)
	reg.AddMonitor(Monitor{ID: "m1", Name: "A", Type: TypeHTTP, CreatedAt: time.Now()})

	m, err := reg.GetMonitor("m1")
	if err != nil || m.ID != "m1" {
		t.Fatalf("GetMonitor failed: %v", err)
	}

	_, err = reg.GetMonitor("missing")
	if err == nil {
		t.Error("expected error for missing monitor")
	}
}

func TestRegistry_Incidents(t *testing.T) {
	reg := newTestRegistry(t)
	inc := Incident{
		ID: "i1", MonitorID: "m1", MonitorName: "Test",
		OpenedAt: time.Now().UTC(), Reason: "timeout",
	}
	if err := reg.OpenIncident(inc); err != nil {
		t.Fatal(err)
	}

	incidents, err := reg.LoadIncidents()
	if err != nil || len(incidents) != 1 {
		t.Fatalf("expected 1 incident, got %d: %v", len(incidents), err)
	}
	if incidents[0].ClosedAt != nil {
		t.Error("expected open incident (nil ClosedAt)")
	}

	closedAt := time.Now().UTC()
	if err := reg.CloseIncident("i1", closedAt); err != nil {
		t.Fatal(err)
	}

	incidents, _ = reg.LoadIncidents()
	if incidents[0].ClosedAt == nil {
		t.Error("expected ClosedAt to be set after closing")
	}
}

func TestRegistry_CloseIncident_NotFound(t *testing.T) {
	reg := newTestRegistry(t)
	if err := reg.CloseIncident("nope", time.Now()); err == nil {
		t.Error("expected error closing missing incident")
	}
}

func TestRegistry_UpdateHeartbeatPing(t *testing.T) {
	reg := newTestRegistry(t)
	reg.AddMonitor(Monitor{ID: "m1", Name: "HB", Type: TypeHeartbeat, CreatedAt: time.Now()})

	now := time.Now().UTC()
	if err := reg.UpdateHeartbeatPing("m1", now); err != nil {
		t.Fatal(err)
	}

	m, _ := reg.GetMonitor("m1")
	if m.LastPing == nil || m.LastPing.IsZero() {
		t.Error("expected LastPing to be set")
	}
}

func TestRegistry_EmptyLoad(t *testing.T) {
	reg := newTestRegistry(t)
	monitors, err := reg.LoadMonitors()
	if err != nil || monitors != nil {
		t.Errorf("expected nil monitors on empty registry, got %v, err %v", monitors, err)
	}
	incidents, err := reg.LoadIncidents()
	if err != nil || incidents != nil {
		t.Errorf("expected nil incidents on empty registry, got %v, err %v", incidents, err)
	}
}
