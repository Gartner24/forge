package deliverylog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gartner24/forge/sparkforge/internal/model"
	"github.com/gartner24/forge/sparkforge/internal/paths"
)

type Log struct {
	path string
}

func New() (*Log, error) {
	p, err := paths.DeliveryLogFile()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return nil, fmt.Errorf("creating delivery log dir: %w", err)
	}
	return &Log{path: p}, nil
}

func (l *Log) Append(r model.DeliveryRecord) error {
	line, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("encoding delivery record: %w", err)
	}
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening delivery log: %w", err)
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s\n", line)
	return err
}

func (l *Log) Read(since time.Time) ([]model.DeliveryRecord, error) {
	if _, err := os.Stat(l.path); os.IsNotExist(err) {
		return []model.DeliveryRecord{}, nil
	}
	f, err := os.Open(l.path)
	if err != nil {
		return nil, fmt.Errorf("opening delivery log: %w", err)
	}
	defer f.Close()

	var records []model.DeliveryRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var r model.DeliveryRecord
		if err := json.Unmarshal(line, &r); err != nil {
			continue
		}
		if !since.IsZero() && r.Timestamp.Before(since) {
			continue
		}
		records = append(records, r)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading delivery log: %w", err)
	}
	return records, nil
}
