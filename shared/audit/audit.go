package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Entry is a single audit log record.
type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	Module    string    `json:"module"`
	Action    string    `json:"action"`
	Actor     string    `json:"actor,omitempty"`
	Detail    string    `json:"detail,omitempty"`
}

// Logger writes audit entries to a file.
// It only ever appends — never truncates or overwrites.
type Logger struct {
	path string
}

// New creates a new Logger that writes to the given file path.
// It creates the file and any parent directories if they don't exist.
func New(path string) (*Logger, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating audit log directory: %w", err)
	}

	// os.O_APPEND means every write goes to the end of the file.
	// os.O_CREATE means create the file if it doesn't exist.
	// We open and close just to verify we can write, then reopen per entry.
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening audit log: %w", err)
	}
	_ = f.Close()

	return &Logger{path: path}, nil
}

// Write appends a single entry to the audit log as a JSON line.
func (l *Logger) Write(module, action, actor, detail string) error {
	entry := Entry{
		Timestamp: time.Now().UTC(),
		Module:    module,
		Action:    action,
		Actor:     actor,
		Detail:    detail,
	}

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encoding audit entry: %w", err)
	}

	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening audit log for write: %w", err)
	}
	//nolint:errcheck
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s\n", line)
	return err
}
