package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store is a thread-safe JSON file store for a single value.
type Store[T any] struct {
	mu   sync.RWMutex
	path string
}

// New creates or opens a Store at the given path.
// Parent directories are created if they do not exist.
func New[T any](path string) (*Store[T], error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating store dir: %w", err)
	}
	return &Store[T]{path: path}, nil
}

// Read returns the stored value, or nil if the file does not exist yet.
func (s *Store[T]) Read() (*T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", s.path, err)
	}

	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", s.path, err)
	}

	return &v, nil
}

// Write persists v atomically (write to temp file, rename).
func (s *Store[T]) Write(v *T) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("writing %s: %w", tmp, err)
	}

	if err := os.Rename(tmp, s.path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("committing %s: %w", s.path, err)
	}

	return nil
}

// Path returns the underlying file path.
func (s *Store[T]) Path() string {
	return s.path
}
