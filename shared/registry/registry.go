package registry

import (
	"encoding/json"
	"os"
)

// ScanTarget represents a registered scan target in PenForge's targets.json.
type ScanTarget struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	URL     string   `json:"url"`
	Scope   []string `json:"scope"`
	Engines []string `json:"engines"` // empty means all
	Cron    string   `json:"cron,omitempty"`
}

// ReadScanTargets reads targets.json from path.
// Returns an empty slice (not an error) if the file does not exist.
func ReadScanTargets(path string) ([]ScanTarget, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []ScanTarget{}, nil
	}
	if err != nil {
		return nil, err
	}
	var targets []ScanTarget
	if err := json.Unmarshal(data, &targets); err != nil {
		return nil, err
	}
	return targets, nil
}

// WriteScanTargets atomically writes targets to path.
func WriteScanTargets(path string, targets []ScanTarget) error {
	data, err := json.MarshalIndent(targets, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
