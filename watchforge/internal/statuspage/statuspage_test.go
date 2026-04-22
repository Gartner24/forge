package statuspage

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gartner24/forge/watchforge/internal/registry"
)

func TestGenerate_BasicPage(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()
	closed := now.Add(-1 * time.Hour)

	monitors := []registry.Monitor{
		{ID: "m1", Name: "API", Type: "http", Public: true, CreatedAt: now},
		{ID: "m2", Name: "DB", Type: "tcp", Public: true, CreatedAt: now},
		{ID: "m3", Name: "Private", Type: "ssl", Public: false, CreatedAt: now},
	}
	incidents := []registry.Incident{
		{ID: "i1", MonitorID: "m1", MonitorName: "API", OpenedAt: now.Add(-2 * time.Hour), ClosedAt: &closed, Reason: "timeout"},
		{ID: "i2", MonitorID: "m2", MonitorName: "DB", OpenedAt: now.Add(-30 * time.Minute), Reason: "connection refused"},
	}
	statuses := map[string]string{"m1": "healthy", "m2": "down", "m3": "healthy"}

	if err := Generate(dir, monitors, incidents, statuses); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	b, err := os.ReadFile(dir + "/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(b)

	if !strings.Contains(content, "Major Outage") {
		t.Error("expected Major Outage banner when a monitor is down")
	}
	if !strings.Contains(content, "API") {
		t.Error("expected public monitor 'API' to be shown")
	}
	if !strings.Contains(content, "DB") {
		t.Error("expected public monitor 'DB' to be shown")
	}
	if strings.Contains(content, "Private") {
		t.Error("private monitor should not appear on status page")
	}
	if !strings.Contains(content, "badge down") {
		t.Error("expected down badge for DB monitor")
	}
	if !strings.Contains(content, "Recent Incidents") {
		t.Error("expected incident history section")
	}
	if !strings.Contains(content, "timeout") {
		t.Error("expected closed incident reason in history")
	}
	// Open incident (connection refused) should not be in history (not closed)
	if strings.Contains(content, "connection refused") && strings.Contains(content, "Recent Incidents") {
		// Check it's not in the history table (open incidents excluded from history)
		histIdx := strings.Index(content, "Recent Incidents")
		afterHist := content[histIdx:]
		if strings.Contains(afterHist, "connection refused") {
			t.Error("open incident should not appear in history")
		}
	}
	t.Logf("HTML length: %d bytes", len(b))
}

func TestGenerate_AllHealthy(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()
	monitors := []registry.Monitor{
		{ID: "m1", Name: "API", Type: "http", Public: true, CreatedAt: now},
	}
	statuses := map[string]string{"m1": "healthy"}
	if err := Generate(dir, monitors, nil, statuses); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	b, _ := os.ReadFile(dir + "/index.html")
	if !strings.Contains(string(b), "All Systems Operational") {
		t.Error("expected operational banner when all healthy")
	}
}

func TestGenerate_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	monitors := []registry.Monitor{}
	if err := Generate(dir, monitors, nil, nil); err != nil {
		t.Fatal(err)
	}
	// No .tmp file should remain
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("tmp file left behind: %s", e.Name())
		}
	}
}
