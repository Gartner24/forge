package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/gartner24/forge/watchforge/internal/checker"
	"github.com/gartner24/forge/watchforge/internal/registry"
)

func newTestScheduler(t *testing.T) (*Scheduler, *registry.Registry) {
	t.Helper()
	reg, err := registry.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sched := New(reg, t.TempDir(), "", nil)
	return sched, reg
}

func addMonitor(t *testing.T, reg *registry.Registry, mtype registry.MonitorType) registry.Monitor {
	t.Helper()
	m := registry.Monitor{
		ID:          "test-monitor",
		Name:        "Test",
		Type:        mtype,
		Target:      "example.com",
		IntervalSec: 60,
		Threshold:   2,
		CreatedAt:   time.Now().UTC(),
	}
	if err := reg.AddMonitor(m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestHandleResult_ThresholdNotMet(t *testing.T) {
	sched, reg := newTestScheduler(t)
	m := addMonitor(t, reg, registry.TypeHTTP)

	// 1 fail — below threshold of 2, no incident
	sched.handleResult(&m, checker.Result{OK: false, Reason: "timeout", CheckedAt: time.Now()})
	sched.mu.Lock()
	st := sched.states[m.ID]
	sched.mu.Unlock()
	if st.consecutiveFails != 1 {
		t.Errorf("expected 1 consecutive fail, got %d", st.consecutiveFails)
	}
	if st.activeIncidentID != "" {
		t.Error("no incident should be open below threshold")
	}
	incidents, _ := reg.LoadIncidents()
	if len(incidents) != 0 {
		t.Errorf("expected 0 incidents, got %d", len(incidents))
	}
}

func TestHandleResult_ThresholdMet_OpensIncident(t *testing.T) {
	sched, reg := newTestScheduler(t)
	m := addMonitor(t, reg, registry.TypeHTTP)

	fail := checker.Result{OK: false, Reason: "refused", CheckedAt: time.Now()}
	sched.handleResult(&m, fail)
	sched.handleResult(&m, fail) // 2nd fail = threshold met

	sched.mu.Lock()
	st := sched.states[m.ID]
	sched.mu.Unlock()
	if st.activeIncidentID == "" {
		t.Error("expected incident to be opened after threshold")
	}
	incidents, _ := reg.LoadIncidents()
	if len(incidents) != 1 {
		t.Errorf("expected 1 incident, got %d", len(incidents))
	}
}

func TestHandleResult_Deduplication(t *testing.T) {
	sched, reg := newTestScheduler(t)
	m := addMonitor(t, reg, registry.TypeHTTP)

	fail := checker.Result{OK: false, Reason: "refused", CheckedAt: time.Now()}
	// Open incident
	sched.handleResult(&m, fail)
	sched.handleResult(&m, fail)
	// More failures should not open additional incidents
	sched.handleResult(&m, fail)
	sched.handleResult(&m, fail)

	incidents, _ := reg.LoadIncidents()
	open := 0
	for _, inc := range incidents {
		if inc.ClosedAt == nil {
			open++
		}
	}
	if open != 1 {
		t.Errorf("expected exactly 1 open incident (deduplication), got %d", open)
	}
}

func TestHandleResult_Recovery(t *testing.T) {
	sched, reg := newTestScheduler(t)
	m := addMonitor(t, reg, registry.TypeHTTP)

	fail := checker.Result{OK: false, Reason: "refused", CheckedAt: time.Now()}
	sched.handleResult(&m, fail)
	sched.handleResult(&m, fail) // incident opens

	ok := checker.Result{OK: true, CheckedAt: time.Now()}
	sched.handleResult(&m, ok) // recovery

	sched.mu.Lock()
	st := sched.states[m.ID]
	sched.mu.Unlock()
	if st.activeIncidentID != "" {
		t.Error("incident should be closed after recovery")
	}
	if st.consecutiveFails != 0 {
		t.Errorf("consecutive fails should reset on recovery, got %d", st.consecutiveFails)
	}
	incidents, _ := reg.LoadIncidents()
	if incidents[0].ClosedAt == nil {
		t.Error("incident should be closed in registry")
	}
}

func TestHandleResult_FailResetOnSuccess(t *testing.T) {
	sched, reg := newTestScheduler(t)
	m := addMonitor(t, reg, registry.TypeHTTP)

	fail := checker.Result{OK: false, Reason: "refused", CheckedAt: time.Now()}
	sched.handleResult(&m, fail) // 1 fail

	ok := checker.Result{OK: true, CheckedAt: time.Now()}
	sched.handleResult(&m, ok) // recovery before threshold

	sched.mu.Lock()
	st := sched.states[m.ID]
	sched.mu.Unlock()
	if st.consecutiveFails != 0 {
		t.Errorf("fails should reset to 0 on success, got %d", st.consecutiveFails)
	}
}

func TestStart_RestoresOpenIncidents(t *testing.T) {
	reg, err := registry.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	reg.AddMonitor(registry.Monitor{
		ID: "m1", Name: "Test", Type: registry.TypeHTTP,
		Target: "http://example.com", IntervalSec: 3600,
		Threshold: 2, CreatedAt: time.Now(),
	})
	// Pre-populate an open incident
	reg.OpenIncident(registry.Incident{
		ID: "existing-inc", MonitorID: "m1", MonitorName: "Test",
		OpenedAt: time.Now().UTC(), Reason: "pre-existing",
	})

	sched := New(reg, t.TempDir(), "", nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sched.Start(ctx); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	cancel()

	sched.mu.Lock()
	st := sched.states["m1"]
	sched.mu.Unlock()
	if st == nil || st.activeIncidentID != "existing-inc" {
		t.Errorf("expected restored incident ID 'existing-inc', got %v", st)
	}
}
