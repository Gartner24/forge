package dedup

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gartner24/forge/sparkforge/internal/model"
	"github.com/gartner24/forge/sparkforge/internal/paths"
)

type Store struct {
	path string
}

func New() (*Store, error) {
	p, err := paths.AlertsFile()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return nil, fmt.Errorf("creating alerts dir: %w", err)
	}
	return &Store{path: p}, nil
}

func (s *Store) IsDuplicate(source, eventType string) bool {
	if eventType == "" {
		return false
	}
	alerts, err := s.load()
	if err != nil {
		return false
	}
	for _, a := range alerts {
		if a.Source == source && a.EventType == eventType {
			return true
		}
	}
	return false
}

func (s *Store) Add(msg model.Message) (model.Alert, error) {
	alerts, err := s.load()
	if err != nil {
		return model.Alert{}, err
	}
	alert := model.Alert{
		ID:        newID(),
		Source:    msg.Source,
		EventType: msg.EventType,
		Title:     msg.Title,
		Priority:  msg.Priority,
		FiredAt:   time.Now().UTC(),
	}
	alerts = append(alerts, alert)
	return alert, s.save(alerts)
}

func (s *Store) Remove(id string) error {
	alerts, err := s.load()
	if err != nil {
		return err
	}
	next := alerts[:0]
	found := false
	for _, a := range alerts {
		if a.ID == id {
			found = true
			continue
		}
		next = append(next, a)
	}
	if !found {
		return fmt.Errorf("alert %q not found", id)
	}
	return s.save(next)
}

func (s *Store) List() ([]model.Alert, error) {
	return s.load()
}

func (s *Store) load() ([]model.Alert, error) {
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		return []model.Alert{}, nil
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("reading alerts: %w", err)
	}
	var alerts []model.Alert
	if err := json.Unmarshal(data, &alerts); err != nil {
		return nil, fmt.Errorf("parsing alerts: %w", err)
	}
	return alerts, nil
}

func (s *Store) save(alerts []model.Alert) error {
	if alerts == nil {
		alerts = []model.Alert{}
	}
	data, err := json.MarshalIndent(alerts, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding alerts: %w", err)
	}
	return os.WriteFile(s.path, data, 0644)
}

func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return fmt.Sprintf("%x", b)
}
