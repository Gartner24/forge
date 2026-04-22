package engine

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/gartner24/forge/shared/registry"
)

// Finding represents a security finding produced by a scan engine.
type Finding struct {
	Engine      string `json:"engine"`
	TargetURL   string `json:"target_url"`
	Severity    string `json:"severity"` // critical, high, medium, low, info
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CVE         string `json:"cve,omitempty"`
	Remediation string `json:"remediation,omitempty"`
	Host        string `json:"host,omitempty"`
	Port        int    `json:"port,omitempty"`
}

// ID returns a stable, content-addressable identifier for the finding.
// Used for delta detection and deduplication.
func (f Finding) ID() string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s|%s", f.Engine, f.TargetURL, f.Name, f.Host)
	return fmt.Sprintf("f-%x", h.Sum(nil))[:16]
}

// Engine is the interface every scan engine must implement.
// No engine-specific code appears outside the engine's own implementation file.
type Engine interface {
	Name() string
	Version() string
	Pull(ctx context.Context) error
	Run(ctx context.Context, target registry.ScanTarget, networkID string) ([]Finding, error)
}

// All returns the list of all available engines.
func All() []Engine {
	return []Engine{
		NewNuclei(),
		NewNmap(),
		NewTestssl(),
		NewDnsx(),
		NewTrivy(),
	}
}

// ByName returns the engine with the given name, or nil.
func ByName(name string) Engine {
	for _, e := range All() {
		if e.Name() == name {
			return e
		}
	}
	return nil
}

// ForTarget returns the engines to run against the given target.
// If target.Engines is empty, all engines are returned.
func ForTarget(target registry.ScanTarget) []Engine {
	if len(target.Engines) == 0 {
		return All()
	}
	var out []Engine
	for _, name := range target.Engines {
		if e := ByName(name); e != nil {
			out = append(out, e)
		}
	}
	return out
}
