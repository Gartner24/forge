package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gartner24/forge/watchforge/internal/registry"
	"github.com/gartner24/forge/watchforge/internal/scheduler"
)

// Server serves the WatchForge HTTP API on localhost:7771.
// Endpoints:
//
//	POST /v1/watchforge/pause?monitor=<id>   -- pause a monitor
//	POST /v1/watchforge/resume?monitor=<id>  -- resume a monitor
//	GET  /_watchforge/heartbeat/<id>/<token> -- record heartbeat ping
type Server struct {
	addr      string
	reg       *registry.Registry
	sched     *scheduler.Scheduler
	parentCtx context.Context
	srv       *http.Server
}

func New(addr string, reg *registry.Registry, sched *scheduler.Scheduler, parentCtx context.Context) *Server {
	s := &Server{addr: addr, reg: reg, sched: sched, parentCtx: parentCtx}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/watchforge/pause", s.handlePause)
	mux.HandleFunc("/v1/watchforge/resume", s.handleResume)
	mux.HandleFunc("/_watchforge/heartbeat/", s.handleHeartbeat)
	s.srv = &http.Server{Addr: addr, Handler: mux}
	return s
}

func (s *Server) Start() error {
	go func() {
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			_ = err // daemon logs this via main
		}
	}()
	return nil
}

func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.srv.Shutdown(ctx)
}

func (s *Server) handlePause(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("monitor")
	if id == "" {
		jsonErr(w, "missing monitor query param", http.StatusBadRequest)
		return
	}
	if err := s.reg.UpdateMonitor(id, func(m *registry.Monitor) { m.Paused = true }); err != nil {
		jsonErr(w, err.Error(), http.StatusNotFound)
		return
	}
	s.sched.RefreshMonitor(s.parentCtx, id)
	jsonOK(w, "paused")
}

func (s *Server) handleResume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("monitor")
	if id == "" {
		jsonErr(w, "missing monitor query param", http.StatusBadRequest)
		return
	}
	if err := s.reg.UpdateMonitor(id, func(m *registry.Monitor) { m.Paused = false }); err != nil {
		jsonErr(w, err.Error(), http.StatusNotFound)
		return
	}
	s.sched.RefreshMonitor(s.parentCtx, id)
	jsonOK(w, "resumed")
}

// handleHeartbeat records a ping for /_watchforge/heartbeat/<monitor-id>/<token>
func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/_watchforge/heartbeat/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		jsonErr(w, "invalid heartbeat URL", http.StatusBadRequest)
		return
	}
	monitorID, token := parts[0], parts[1]

	m, err := s.reg.GetMonitor(monitorID)
	if err != nil {
		jsonErr(w, "monitor not found", http.StatusNotFound)
		return
	}
	if m.Type != registry.TypeHeartbeat {
		jsonErr(w, "not a heartbeat monitor", http.StatusBadRequest)
		return
	}
	if m.Token != token {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	now := time.Now().UTC()
	if err := s.reg.UpdateHeartbeatPing(monitorID, now); err != nil {
		jsonErr(w, fmt.Sprintf("recording ping: %v", err), http.StatusInternalServerError)
		return
	}

	jsonOK(w, "ok")
}

func jsonOK(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	b, _ := json.Marshal(map[string]string{"ok": "true", "message": msg})
	_, _ = w.Write(b)
}

func jsonErr(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	b, _ := json.Marshal(map[string]string{"error": msg})
	_, _ = w.Write(b)
}
