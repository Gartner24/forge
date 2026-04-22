package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gartner24/forge/watchforge/internal/registry"
	"github.com/gartner24/forge/watchforge/internal/scheduler"
)

func newTestServer(t *testing.T) (*Server, *registry.Registry) {
	t.Helper()
	reg, err := registry.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sched := scheduler.New(reg, t.TempDir(), "", nil)
	ctx := context.Background()
	srv := New("127.0.0.1:0", reg, sched, ctx)
	return srv, reg
}

func addTestMonitor(t *testing.T, reg *registry.Registry, mtype registry.MonitorType, token string) registry.Monitor {
	t.Helper()
	m := registry.Monitor{
		ID: "test-m", Name: "Test", Type: mtype,
		Target: "example.com", Token: token,
		IntervalSec: 60, Threshold: 2,
		CreatedAt: time.Now().UTC(),
	}
	if err := reg.AddMonitor(m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestHandlePause_OK(t *testing.T) {
	srv, reg := newTestServer(t)
	addTestMonitor(t, reg, registry.TypeHTTP, "")

	req := httptest.NewRequest(http.MethodPost, "/v1/watchforge/pause?monitor=test-m", nil)
	w := httptest.NewRecorder()
	srv.handlePause(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body)
	}
	m, _ := reg.GetMonitor("test-m")
	if !m.Paused {
		t.Error("expected monitor to be paused")
	}
}

func TestHandlePause_MissingParam(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/watchforge/pause", nil)
	w := httptest.NewRecorder()
	srv.handlePause(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlePause_WrongMethod(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/watchforge/pause?monitor=x", nil)
	w := httptest.NewRecorder()
	srv.handlePause(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleResume_OK(t *testing.T) {
	srv, reg := newTestServer(t)
	m := addTestMonitor(t, reg, registry.TypeHTTP, "")
	// Pause first
	reg.UpdateMonitor(m.ID, func(mon *registry.Monitor) { mon.Paused = true })

	req := httptest.NewRequest(http.MethodPost, "/v1/watchforge/resume?monitor=test-m", nil)
	w := httptest.NewRecorder()
	srv.handleResume(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body)
	}
	updated, _ := reg.GetMonitor("test-m")
	if updated.Paused {
		t.Error("expected monitor to be unpaused")
	}
}

func TestHandleHeartbeat_OK(t *testing.T) {
	srv, reg := newTestServer(t)
	addTestMonitor(t, reg, registry.TypeHeartbeat, "secret-token")

	req := httptest.NewRequest(http.MethodGet, "/_watchforge/heartbeat/test-m/secret-token", nil)
	req.SetPathValue("id", "test-m")
	w := httptest.NewRecorder()
	// Manually set path for the handler
	req = req.Clone(req.Context())
	srv.handleHeartbeat(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body)
	}
	m, _ := reg.GetMonitor("test-m")
	if m.LastPing == nil {
		t.Error("expected LastPing to be set after heartbeat")
	}
}

func TestHandleHeartbeat_WrongToken(t *testing.T) {
	srv, reg := newTestServer(t)
	addTestMonitor(t, reg, registry.TypeHeartbeat, "correct-token")

	req := httptest.NewRequest(http.MethodGet, "/_watchforge/heartbeat/test-m/wrong-token", nil)
	w := httptest.NewRecorder()
	srv.handleHeartbeat(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestHandleHeartbeat_NotHeartbeatType(t *testing.T) {
	srv, reg := newTestServer(t)
	addTestMonitor(t, reg, registry.TypeHTTP, "tok")

	req := httptest.NewRequest(http.MethodGet, "/_watchforge/heartbeat/test-m/tok", nil)
	w := httptest.NewRecorder()
	srv.handleHeartbeat(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleHeartbeat_InvalidPath(t *testing.T) {
	srv, _ := newTestServer(t)
	for _, path := range []string{
		"/_watchforge/heartbeat/",
		"/_watchforge/heartbeat/onlyone",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		srv.handleHeartbeat(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("path %q: expected 400, got %d", path, w.Code)
		}
	}
}

func TestHandlePause_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/watchforge/pause?monitor=missing", nil)
	w := httptest.NewRecorder()
	srv.handlePause(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	var body map[string]string
	json.NewDecoder(strings.NewReader(w.Body.String())).Decode(&body)
	if body["error"] == "" {
		t.Error("expected JSON error body")
	}
}
