package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type MonitorType string

const (
	TypeHTTP      MonitorType = "http"
	TypeTCP       MonitorType = "tcp"
	TypeDocker    MonitorType = "docker"
	TypeSSL       MonitorType = "ssl"
	TypeHeartbeat MonitorType = "heartbeat"
)

type Monitor struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Type        MonitorType `json:"type"`
	Target      string      `json:"target"`
	IntervalSec int         `json:"interval"`
	Public      bool        `json:"public"`
	Paused      bool        `json:"paused"`
	Threshold   int         `json:"threshold"`
	// HTTP-specific
	ExpectedStatus int    `json:"expected_status,omitempty"`
	Contains       string `json:"contains,omitempty"`
	TimeoutSec     int    `json:"timeout,omitempty"`
	// Heartbeat-specific
	Grace    int        `json:"grace,omitempty"`
	Token    string     `json:"token,omitempty"`
	LastPing *time.Time `json:"last_ping,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

type Incident struct {
	ID          string     `json:"id"`
	MonitorID   string     `json:"monitor_id"`
	MonitorName string     `json:"monitor_name"`
	OpenedAt    time.Time  `json:"opened_at"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
	Reason      string     `json:"reason"`
}

type Registry struct {
	dir string
	mu  sync.Mutex
}

func New(dir string) (*Registry, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating registry dir: %w", err)
	}
	return &Registry{dir: dir}, nil
}

func (r *Registry) monitorsPath() string { return filepath.Join(r.dir, "monitors.json") }
func (r *Registry) incidentsPath() string { return filepath.Join(r.dir, "incidents.json") }

func (r *Registry) LoadMonitors() ([]Monitor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.loadMonitors()
}

func (r *Registry) loadMonitors() ([]Monitor, error) {
	data, err := os.ReadFile(r.monitorsPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading monitors: %w", err)
	}
	var monitors []Monitor
	if err := json.Unmarshal(data, &monitors); err != nil {
		return nil, fmt.Errorf("parsing monitors: %w", err)
	}
	return monitors, nil
}

func (r *Registry) saveMonitors(monitors []Monitor) error {
	b, err := json.MarshalIndent(monitors, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding monitors: %w", err)
	}
	tmp := r.monitorsPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return fmt.Errorf("writing monitors: %w", err)
	}
	return os.Rename(tmp, r.monitorsPath())
}

func (r *Registry) AddMonitor(m Monitor) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	monitors, err := r.loadMonitors()
	if err != nil {
		return err
	}
	monitors = append(monitors, m)
	return r.saveMonitors(monitors)
}

func (r *Registry) UpdateMonitor(id string, fn func(*Monitor)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	monitors, err := r.loadMonitors()
	if err != nil {
		return err
	}
	for i := range monitors {
		if monitors[i].ID == id {
			fn(&monitors[i])
			return r.saveMonitors(monitors)
		}
	}
	return fmt.Errorf("monitor %q not found", id)
}

func (r *Registry) DeleteMonitor(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	monitors, err := r.loadMonitors()
	if err != nil {
		return err
	}
	out := monitors[:0]
	found := false
	for _, m := range monitors {
		if m.ID == id {
			found = true
			continue
		}
		out = append(out, m)
	}
	if !found {
		return fmt.Errorf("monitor %q not found", id)
	}
	return r.saveMonitors(out)
}

func (r *Registry) GetMonitor(id string) (*Monitor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	monitors, err := r.loadMonitors()
	if err != nil {
		return nil, err
	}
	for i := range monitors {
		if monitors[i].ID == id {
			return &monitors[i], nil
		}
	}
	return nil, fmt.Errorf("monitor %q not found", id)
}

func (r *Registry) LoadIncidents() ([]Incident, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.loadIncidents()
}

func (r *Registry) loadIncidents() ([]Incident, error) {
	data, err := os.ReadFile(r.incidentsPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading incidents: %w", err)
	}
	var incidents []Incident
	if err := json.Unmarshal(data, &incidents); err != nil {
		return nil, fmt.Errorf("parsing incidents: %w", err)
	}
	return incidents, nil
}

func (r *Registry) saveIncidents(incidents []Incident) error {
	b, err := json.MarshalIndent(incidents, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding incidents: %w", err)
	}
	tmp := r.incidentsPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return fmt.Errorf("writing incidents: %w", err)
	}
	return os.Rename(tmp, r.incidentsPath())
}

func (r *Registry) OpenIncident(inc Incident) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	incidents, err := r.loadIncidents()
	if err != nil {
		return err
	}
	incidents = append(incidents, inc)
	return r.saveIncidents(incidents)
}

func (r *Registry) CloseIncident(id string, closedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	incidents, err := r.loadIncidents()
	if err != nil {
		return err
	}
	for i := range incidents {
		if incidents[i].ID == id {
			t := closedAt
			incidents[i].ClosedAt = &t
			return r.saveIncidents(incidents)
		}
	}
	return fmt.Errorf("incident %q not found", id)
}

func (r *Registry) UpdateHeartbeatPing(monitorID string, t time.Time) error {
	return r.UpdateMonitor(monitorID, func(m *Monitor) {
		m.LastPing = &t
	})
}
