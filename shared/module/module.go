package module

// Status represents wheter a module is running or not.
type Status string

const (
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
	StatusError Status = "error"
)

// Core interface that each module in forge must implement
// Forge uses this to discover, start, stop, and check modules.
type Module interface {
	// Name return the module's canonical name, e.g. "smeltforge".
	Name() string

	// Version returns the module's current version, e.g. "0.1.0".
	Version() string

	// Status returns wheter the module is running, stopped or errored.
	Status() Status

	// Start starts the module. Returns an error if it fails.
	Start() error

	// Stop stops the module gracefully. Returns an error if it fails.
	Stop() error
}
