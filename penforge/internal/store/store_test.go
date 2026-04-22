package store

import (
	"os"
	"testing"
	"time"

	"github.com/gartner24/forge/penforge/internal/engine"
)

func tmpStore(t *testing.T) *FindingStore {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "findings*.json")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	return NewFindingStore(f.Name())
}

var now = time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC)

func TestUpsertDelta(t *testing.T) {
	s := tmpStore(t)

	findings1 := []engine.Finding{
		{Engine: "nuclei", TargetURL: "https://x.com", Name: "XSS", Host: "x.com", Severity: "high"},
		{Engine: "nmap", TargetURL: "https://x.com", Name: "Open Port 22", Host: "x.com", Severity: "medium"},
	}
	d1, err := s.Upsert("x", "https://x.com", findings1, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(d1.New) != 2 {
		t.Fatalf("scan1 new: want 2, got %d", len(d1.New))
	}
	if len(d1.Recurring) != 0 || len(d1.Resolved) != 0 {
		t.Fatalf("scan1 unexpected delta: recurring=%d resolved=%d", len(d1.Recurring), len(d1.Resolved))
	}

	// XSS disappears, SQLi new, Port 22 recurring
	findings2 := []engine.Finding{
		{Engine: "nmap", TargetURL: "https://x.com", Name: "Open Port 22", Host: "x.com", Severity: "medium"},
		{Engine: "nuclei", TargetURL: "https://x.com", Name: "SQLi", Host: "x.com", Severity: "critical"},
	}
	d2, err := s.Upsert("x", "https://x.com", findings2, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(d2.New) != 1 || d2.New[0].Name != "SQLi" {
		t.Fatalf("scan2 new: want [SQLi], got %v", d2.New)
	}
	if len(d2.Recurring) != 1 {
		t.Fatalf("scan2 recurring: want 1, got %d", len(d2.Recurring))
	}
	if len(d2.Resolved) != 1 || d2.Resolved[0].Name != "XSS" {
		t.Fatalf("scan2 resolved: want [XSS], got %v", d2.Resolved)
	}
}

func TestSeverityIncreaseReopensAcknowledged(t *testing.T) {
	s := tmpStore(t)

	findings1 := []engine.Finding{
		{Engine: "nmap", TargetURL: "https://x.com", Name: "Open Port 22", Host: "x.com", Severity: "medium"},
	}
	d1, err := s.Upsert("x", "https://x.com", findings1, now)
	if err != nil {
		t.Fatal(err)
	}

	portID := d1.New[0].ID
	if err := s.UpdateState(portID, StateAcknowledged, "known", now); err != nil {
		t.Fatal(err)
	}

	// Severity increases from medium to high
	findings2 := []engine.Finding{
		{Engine: "nmap", TargetURL: "https://x.com", Name: "Open Port 22", Host: "x.com", Severity: "high"},
	}
	_, err = s.Upsert("x", "https://x.com", findings2, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	all, _ := s.All()
	for _, f := range all {
		if f.Name == "Open Port 22" {
			if f.State != StateNew {
				t.Errorf("Port 22 state: want new (re-opened), got %s", f.State)
			}
			if f.Severity != "high" {
				t.Errorf("Port 22 severity: want high, got %s", f.Severity)
			}
			if f.PrevSeverity != "medium" {
				t.Errorf("Port 22 prev_severity: want medium, got %s", f.PrevSeverity)
			}
			return
		}
	}
	t.Fatal("Port 22 finding not found after re-scan")
}

func TestUpdateState(t *testing.T) {
	s := tmpStore(t)
	findings := []engine.Finding{
		{Engine: "nuclei", TargetURL: "https://x.com", Name: "XSS", Host: "x.com", Severity: "high"},
	}
	d, _ := s.Upsert("x", "https://x.com", findings, now)
	id := d.New[0].ID

	if err := s.UpdateState(id, StateAccepted, "risk accepted", now); err != nil {
		t.Fatal(err)
	}
	f, err := s.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if f.State != StateAccepted {
		t.Errorf("state: want accepted, got %s", f.State)
	}
	if f.StateReason != "risk accepted" {
		t.Errorf("reason: want 'risk accepted', got %s", f.StateReason)
	}
}

func TestUpdateStateNotFound(t *testing.T) {
	s := tmpStore(t)
	if err := s.UpdateState("nonexistent", StateAccepted, "", now); err == nil {
		t.Fatal("expected error for nonexistent finding")
	}
}
