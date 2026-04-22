package fluxforge

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/gartner24/forge/fluxforge/internal/mesh"
	"github.com/gartner24/forge/fluxforge/internal/store"
	"github.com/gartner24/forge/shared/module"
)

const moduleName = "fluxforge"
const moduleVersion = "0.1.0"

// Module implements the shared module.Module interface for FluxForge.
// It reports status by checking whether the appropriate daemon binary is running.
type Module struct {
	dataDir string
}

func New(dataDir string) *Module {
	return &Module{dataDir: dataDir}
}

func (m *Module) Name() string    { return moduleName }
func (m *Module) Version() string { return moduleVersion }

func (m *Module) Status() module.Status {
	pid, err := m.runningPID()
	if err != nil || pid == 0 {
		return module.StatusStopped
	}
	if isAlive(pid) {
		return module.StatusRunning
	}
	return module.StatusStopped
}

func (m *Module) Start() error {
	return nil // lifecycle managed by forge fluxforge init/join
}

func (m *Module) Stop() error {
	pid, err := m.runningPID()
	if err != nil || pid == 0 {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGTERM)
}

func (m *Module) runningPID() (int, error) {
	s, err := store.New[mesh.LocalState](filepath.Join(m.dataDir, "state.json"))
	if err != nil {
		return 0, err
	}
	state, err := s.Read()
	if err != nil || state == nil {
		return 0, err
	}
	// The PID file would be at ~/.forge/run/fluxcontroller.json or fluxagent.json.
	// For now, we return 0 indicating the caller should check the run dir.
	return 0, nil
}

func isAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
