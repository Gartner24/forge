package scheduler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gartner24/forge/shared/audit"
	"github.com/gartner24/forge/shared/notify"
	"github.com/gartner24/forge/watchforge/internal/checker"
	"github.com/gartner24/forge/watchforge/internal/registry"
	"github.com/gartner24/forge/watchforge/internal/statuspage"
)

type monitorState struct {
	consecutiveFails int
	activeIncidentID string
}

type Scheduler struct {
	reg          *registry.Registry
	statusDir    string
	notifyToken  string
	auditLog     *audit.Logger

	mu       sync.Mutex
	states   map[string]*monitorState
	cancels  map[string]context.CancelFunc
	statusMu sync.Mutex
}

func New(reg *registry.Registry, statusDir string, notifyToken string, auditLog *audit.Logger) *Scheduler {
	return &Scheduler{
		reg:         reg,
		statusDir:   statusDir,
		notifyToken: notifyToken,
		auditLog:    auditLog,
		states:      make(map[string]*monitorState),
		cancels:     make(map[string]context.CancelFunc),
	}
}

// Start launches goroutines for all configured monitors and restores incident state.
func (s *Scheduler) Start(ctx context.Context) error {
	monitors, err := s.reg.LoadMonitors()
	if err != nil {
		return fmt.Errorf("loading monitors: %w", err)
	}

	incidents, err := s.reg.LoadIncidents()
	if err != nil {
		return fmt.Errorf("loading incidents: %w", err)
	}

	// Restore open incident IDs so we don't fire duplicate DOWN alerts on restart.
	s.mu.Lock()
	for _, inc := range incidents {
		if inc.ClosedAt == nil {
			if s.states[inc.MonitorID] == nil {
				s.states[inc.MonitorID] = &monitorState{}
			}
			s.states[inc.MonitorID].activeIncidentID = inc.ID
		}
	}
	s.mu.Unlock()

	for _, m := range monitors {
		s.startMonitorGoroutine(ctx, m.ID)
	}
	return nil
}

// AddMonitor starts a goroutine for a newly added monitor.
func (s *Scheduler) AddMonitor(ctx context.Context, id string) {
	s.startMonitorGoroutine(ctx, id)
}

// RemoveMonitor stops the goroutine for a deleted monitor.
func (s *Scheduler) RemoveMonitor(id string) {
	s.mu.Lock()
	if cancel, ok := s.cancels[id]; ok {
		cancel()
		delete(s.cancels, id)
	}
	delete(s.states, id)
	s.mu.Unlock()
}

// RefreshMonitor stops and restarts a monitor's goroutine (after update/pause/resume).
func (s *Scheduler) RefreshMonitor(ctx context.Context, id string) {
	s.mu.Lock()
	if cancel, ok := s.cancels[id]; ok {
		cancel()
		delete(s.cancels, id)
	}
	s.mu.Unlock()
	s.startMonitorGoroutine(ctx, id)
}

// MonitorStatuses returns a map of monitor ID to "healthy"/"down"/"paused" for all monitors.
func (s *Scheduler) MonitorStatuses() map[string]string {
	monitors, _ := s.reg.LoadMonitors()
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]string, len(monitors))
	for _, m := range monitors {
		if m.Paused {
			out[m.ID] = "paused"
			continue
		}
		st := s.states[m.ID]
		if st != nil && st.activeIncidentID != "" {
			out[m.ID] = "down"
		} else {
			out[m.ID] = "healthy"
		}
	}
	return out
}

func (s *Scheduler) startMonitorGoroutine(ctx context.Context, id string) {
	mctx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancels[id] = cancel
	if s.states[id] == nil {
		s.states[id] = &monitorState{}
	}
	s.mu.Unlock()
	go s.runMonitor(mctx, id)
}

func (s *Scheduler) runMonitor(ctx context.Context, id string) {
	for {
		m, err := s.reg.GetMonitor(id)
		if err != nil {
			log.Printf("watchforge: monitor %s gone: %v", id, err)
			return
		}

		if !m.Paused {
			result := s.executeCheck(m)
			s.handleResult(m, result)
		}

		interval := time.Duration(m.IntervalSec) * time.Second
		if interval <= 0 {
			interval = 60 * time.Second
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

func (s *Scheduler) executeCheck(m *registry.Monitor) checker.Result {
	switch m.Type {
	case registry.TypeHTTP:
		timeout := time.Duration(m.TimeoutSec) * time.Second
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		expected := m.ExpectedStatus
		if expected == 0 {
			expected = 200
		}
		return (&checker.HTTPChecker{
			URL:            m.Target,
			ExpectedStatus: expected,
			Contains:       m.Contains,
			Timeout:        timeout,
		}).Check()

	case registry.TypeTCP:
		return (&checker.TCPChecker{
			Address: m.Target,
			Timeout: 10 * time.Second,
		}).Check()

	case registry.TypeDocker:
		return (&checker.DockerChecker{ContainerName: m.Target}).Check()

	case registry.TypeSSL:
		return (&checker.SSLChecker{Host: m.Target}).Check()

	case registry.TypeHeartbeat:
		return (&checker.HeartbeatChecker{
			IntervalSec: m.IntervalSec,
			GraceSec:    m.Grace,
			LastPing:    m.LastPing,
		}).Check()

	default:
		return checker.Result{OK: false, Reason: fmt.Sprintf("unknown type %q", m.Type), CheckedAt: time.Now()}
	}
}

func (s *Scheduler) handleResult(m *registry.Monitor, result checker.Result) {
	threshold := m.Threshold
	if threshold <= 0 {
		threshold = 2
	}

	s.mu.Lock()
	st := s.states[m.ID]
	if st == nil {
		st = &monitorState{}
		s.states[m.ID] = st
	}

	var shouldOpen bool
	var incidentToClose string

	if !result.OK {
		st.consecutiveFails++
		if st.consecutiveFails >= threshold && st.activeIncidentID == "" {
			shouldOpen = true
		}
	} else {
		incidentToClose = st.activeIncidentID
		st.consecutiveFails = 0
	}
	s.mu.Unlock()

	if shouldOpen {
		s.openIncident(m, result)
	} else if incidentToClose != "" {
		s.closeIncident(m, incidentToClose)
	}

	s.regenerateStatusPage()
}

func (s *Scheduler) openIncident(m *registry.Monitor, result checker.Result) {
	id := newID()
	inc := registry.Incident{
		ID:          id,
		MonitorID:   m.ID,
		MonitorName: m.Name,
		OpenedAt:    time.Now().UTC(),
		Reason:      result.Reason,
	}
	if err := s.reg.OpenIncident(inc); err != nil {
		log.Printf("watchforge: opening incident for %s: %v", m.ID, err)
		return
	}

	s.mu.Lock()
	if s.states[m.ID] != nil {
		s.states[m.ID].activeIncidentID = id
	}
	s.mu.Unlock()

	priority := result.Priority
	if priority == "" {
		priority = notify.PriorityHigh
	}

	_ = notify.Send(s.notifyToken, notify.Message{
		Title:    fmt.Sprintf("[DOWN] %s", m.Name),
		Body:     result.Reason,
		Priority: priority,
		Source:   "watchforge",
	})

	if s.auditLog != nil {
		_ = s.auditLog.Write("watchforge", "incident.open", "daemon",
			fmt.Sprintf("monitor=%s incident=%s reason=%s", m.ID, id, result.Reason))
	}
}

func (s *Scheduler) closeIncident(m *registry.Monitor, incidentID string) {
	if err := s.reg.CloseIncident(incidentID, time.Now().UTC()); err != nil {
		log.Printf("watchforge: closing incident %s: %v", incidentID, err)
		return
	}

	s.mu.Lock()
	if s.states[m.ID] != nil {
		s.states[m.ID].activeIncidentID = ""
	}
	s.mu.Unlock()

	_ = notify.Send(s.notifyToken, notify.Message{
		Title:    fmt.Sprintf("[RECOVERED] %s", m.Name),
		Body:     "Monitor has recovered",
		Priority: notify.PriorityMedium,
		Source:   "watchforge",
	})

	if s.auditLog != nil {
		_ = s.auditLog.Write("watchforge", "incident.close", "daemon",
			fmt.Sprintf("monitor=%s incident=%s", m.ID, incidentID))
	}
}

func (s *Scheduler) regenerateStatusPage() {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()

	monitors, err := s.reg.LoadMonitors()
	if err != nil {
		log.Printf("watchforge: loading monitors for status page: %v", err)
		return
	}
	incidents, err := s.reg.LoadIncidents()
	if err != nil {
		log.Printf("watchforge: loading incidents for status page: %v", err)
		return
	}

	statuses := s.MonitorStatuses()

	if err := statuspage.Generate(s.statusDir, monitors, incidents, statuses); err != nil {
		log.Printf("watchforge: generating status page: %v", err)
	}
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
