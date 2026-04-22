package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gartner24/forge/penforge/internal/engine"
	"github.com/gartner24/forge/shared/audit"
)

const (
	StateNew          = "new"
	StateAcknowledged = "acknowledged"
	StateAccepted     = "accepted"
	StateFixed        = "fixed"
	StateVerified     = "verified"
)

// StoredFinding is a finding persisted in findings.json.
type StoredFinding struct {
	ID             string     `json:"id"`
	Engine         string     `json:"engine"`
	TargetID       string     `json:"target_id"`
	TargetURL      string     `json:"target_url"`
	Severity       string     `json:"severity"`
	Name           string     `json:"name"`
	Description    string     `json:"description,omitempty"`
	CVE            string     `json:"cve,omitempty"`
	Remediation    string     `json:"remediation,omitempty"`
	Host           string     `json:"host,omitempty"`
	Port           int        `json:"port,omitempty"`
	FirstSeen      time.Time  `json:"first_seen"`
	LastSeen       time.Time  `json:"last_seen"`
	State          string     `json:"state"`
	StateReason    string     `json:"state_reason,omitempty"`
	StateChangedAt *time.Time `json:"state_changed_at,omitempty"`
	PrevSeverity   string     `json:"prev_severity,omitempty"`
}

// ScanRecord is persisted per scan under scans/<scan_id>/status.json.
type ScanRecord struct {
	ID                string     `json:"id"`
	TargetID          string     `json:"target_id"`
	TargetURL         string     `json:"target_url,omitempty"`
	Engines           []string   `json:"engines"`
	Status            string     `json:"status"` // running, completed, failed
	StartedAt         time.Time  `json:"started_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	EstimatedSeconds  int        `json:"estimated_seconds,omitempty"`
	Error             string     `json:"error,omitempty"`
	NewFindings       int        `json:"new_findings,omitempty"`
	RecurringFindings int        `json:"recurring_findings,omitempty"`
	ResolvedFindings  int        `json:"resolved_findings,omitempty"`
	TotalFindings     int        `json:"total_findings,omitempty"`
}

// Delta holds the results of comparing two scans.
type Delta struct {
	New       []StoredFinding
	Resolved  []StoredFinding
	Recurring []StoredFinding
}

// FindingStore manages the findings.json file.
type FindingStore struct {
	mu        sync.Mutex
	path      string
	auditPath string
}

func NewFindingStore(path string) *FindingStore {
	return &FindingStore{path: path}
}

// NewFindingStoreWithAudit creates a FindingStore that writes audit entries on state changes.
func NewFindingStoreWithAudit(path, auditPath string) *FindingStore {
	return &FindingStore{path: path, auditPath: auditPath}
}

func (s *FindingStore) All() ([]StoredFinding, error) {
	return readFindings(s.path)
}

func (s *FindingStore) ByTarget(targetID string) ([]StoredFinding, error) {
	all, err := s.All()
	if err != nil {
		return nil, err
	}
	var out []StoredFinding
	for _, f := range all {
		if f.TargetID == targetID {
			out = append(out, f)
		}
	}
	return out, nil
}

func (s *FindingStore) Get(id string) (*StoredFinding, error) {
	all, err := s.All()
	if err != nil {
		return nil, err
	}
	for _, f := range all {
		if f.ID == id {
			cp := f
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("finding %q not found", id)
}

// Upsert merges engine findings into the store, applying delta detection.
// Returns the delta (new, resolved, recurring).
func (s *FindingStore) Upsert(targetID string, targetURL string, incoming []engine.Finding, now time.Time) (Delta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := readFindings(s.path)
	if err != nil {
		return Delta{}, err
	}

	// Build an index of existing findings by ID for this target.
	existing := map[string]int{} // id -> index in all
	for i, f := range all {
		if f.TargetID == targetID {
			existing[f.ID] = i
		}
	}

	// Compute the IDs seen in the incoming scan.
	incomingIDs := map[string]bool{}
	for _, f := range incoming {
		ef := engine.Finding{
			Engine:    f.Engine,
			TargetURL: f.TargetURL,
			Name:      f.Name,
			Host:      f.Host,
		}
		id := ef.ID()
		incomingIDs[id] = true
	}

	var delta Delta

	// Update or create findings from the incoming scan.
	for _, f := range incoming {
		ef := engine.Finding{
			Engine:    f.Engine,
			TargetURL: f.TargetURL,
			Name:      f.Name,
			Host:      f.Host,
		}
		id := ef.ID()
		if idx, exists := existing[id]; exists {
			prev := all[idx]
			all[idx].LastSeen = now

			// Detect severity increase for acknowledged/accepted findings.
			if (prev.State == StateAcknowledged || prev.State == StateAccepted) &&
				severityRank(f.Severity) > severityRank(prev.Severity) {
				all[idx].PrevSeverity = prev.Severity
				all[idx].Severity = f.Severity
				all[idx].State = StateNew // re-open with new severity
				t := now
				all[idx].StateChangedAt = &t
			}

			delta.Recurring = append(delta.Recurring, all[idx])
		} else {
			sf := StoredFinding{
				ID:          id,
				Engine:      f.Engine,
				TargetID:    targetID,
				TargetURL:   f.TargetURL,
				Severity:    f.Severity,
				Name:        f.Name,
				Description: f.Description,
				CVE:         f.CVE,
				Remediation: f.Remediation,
				Host:        f.Host,
				Port:        f.Port,
				FirstSeen:   now,
				LastSeen:    now,
				State:       StateNew,
			}
			all = append(all, sf)
			delta.New = append(delta.New, sf)
		}
	}

	// Mark findings not seen in this scan as resolved (if previously new/acknowledged).
	for id, idx := range existing {
		if !incomingIDs[id] {
			f := all[idx]
			if f.State == StateNew || f.State == StateAcknowledged {
				delta.Resolved = append(delta.Resolved, f)
			}
		}
	}

	return delta, writeFindings(s.path, all)
}

func (s *FindingStore) UpdateState(id, state, reason string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := readFindings(s.path)
	if err != nil {
		return err
	}
	for i, f := range all {
		if f.ID == id {
			all[i].State = state
			all[i].StateReason = reason
			t := now
			all[i].StateChangedAt = &t
			if err := writeFindings(s.path, all); err != nil {
				return err
			}
			if s.auditPath != "" {
				if al, aerr := audit.New(s.auditPath); aerr == nil {
					al.Write("penforge", "finding_state_changed", "", fmt.Sprintf("finding=%s state=%s reason=%s", id, state, reason))
				}
			}
			return nil
		}
	}
	return fmt.Errorf("finding %q not found", id)
}

// ScanStore manages scan records under scans/<scan_id>/status.json.
type ScanStore struct {
	scansDir string
}

func NewScanStore(scansDir string) *ScanStore {
	return &ScanStore{scansDir: scansDir}
}

func (s *ScanStore) Save(rec ScanRecord) error {
	dir := filepath.Join(s.scansDir, rec.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return writeJSON(filepath.Join(dir, "status.json"), rec)
}

func (s *ScanStore) Get(id string) (*ScanRecord, error) {
	var rec ScanRecord
	path := filepath.Join(s.scansDir, id, "status.json")
	if err := readJSON(path, &rec); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("scan %q not found", id)
		}
		return nil, err
	}
	return &rec, nil
}

// LatestForTarget returns the most recent completed scan for a target.
func (s *ScanStore) LatestForTarget(targetID string) (*ScanRecord, error) {
	scans, err := s.AllForTarget(targetID)
	if err != nil {
		return nil, err
	}
	if len(scans) == 0 {
		return nil, nil
	}
	// Sort by StartedAt descending.
	sort.Slice(scans, func(i, j int) bool {
		return scans[i].StartedAt.After(scans[j].StartedAt)
	})
	return &scans[0], nil
}

// AllForTarget returns all scan records for a target.
func (s *ScanStore) AllForTarget(targetID string) ([]ScanRecord, error) {
	entries, err := os.ReadDir(s.scansDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []ScanRecord
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(s.scansDir, e.Name(), "status.json")
		var rec ScanRecord
		if err := readJSON(path, &rec); err != nil {
			continue
		}
		if targetID == "" || rec.TargetID == targetID {
			out = append(out, rec)
		}
	}
	return out, nil
}

func severityRank(s string) int {
	switch strings.ToLower(s) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func readFindings(path string) ([]StoredFinding, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []StoredFinding{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return []StoredFinding{}, nil
	}
	var findings []StoredFinding
	if err := json.Unmarshal(data, &findings); err != nil {
		return nil, err
	}
	return findings, nil
}

func writeFindings(path string, findings []StoredFinding) error {
	return writeJSON(path, findings)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
