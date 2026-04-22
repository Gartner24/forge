package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gartner24/forge/core/internal/paths"
	"github.com/gartner24/forge/core/internal/state"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start <module>",
	Short: "Start a Forge module",
	Args:  cobra.ExactArgs(1),
	RunE:  runStart,
}

func runStart(cmd *cobra.Command, args []string) error {
	module := args[0]

	cfg, err := requireInit()
	if err != nil {
		return cmdErr(err)
	}

	mod, ok := cfg.Modules[module]
	if !ok || !mod.Enabled {
		return cmdErr(fmt.Errorf("%s is not installed -- run: forge install %s", module, module))
	}

	// Check if already running.
	s, _ := state.Get(module)
	if s != nil && state.IsAlive(s) {
		return cmdErr(fmt.Errorf("%s is already running (pid %d)", module, s.PID))
	}

	binPath, err := paths.ModuleBin(module)
	if err != nil {
		return cmdErr(err)
	}
	if cfg.Forge.InstallDir != "" {
		binPath = filepath.Join(cfg.Forge.InstallDir, module)
	}

	logPath, err := paths.ModuleLogFile(module)
	if err != nil {
		return cmdErr(err)
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return cmdErr(fmt.Errorf("creating log dir: %w", err))
	}

	lf, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return cmdErr(fmt.Errorf("opening log file: %w", err))
	}

	c := exec.Command(binPath)
	c.Stdout = lf
	c.Stderr = lf
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := c.Start(); err != nil {
		lf.Close()
		if os.IsNotExist(err) {
			return cmdErr(fmt.Errorf("module binary not found at %s -- binary must be placed there before starting", binPath))
		}
		return cmdErr(fmt.Errorf("starting %s: %w", module, err))
	}
	lf.Close()

	ver := moduleVersion(binPath)
	if err := state.Set(module, state.ModuleState{
		PID:       c.Process.Pid,
		Version:   ver,
		StartedAt: time.Now().UTC(),
	}); err != nil {
		return cmdErr(fmt.Errorf("writing state: %w", err))
	}

	printSuccess(fmt.Sprintf("%s started (pid %d)", module, c.Process.Pid))
	return nil
}

// moduleVersion runs <binary> --version with a short timeout and returns the trimmed output.
func moduleVersion(binPath string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, binPath, "--version").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// stopModule sends SIGTERM to the process in s and removes the state file.
// Shared by uninstall and stop commands.
func stopModule(module string, s *state.ModuleState) error {
	proc, err := os.FindProcess(s.PID)
	if err != nil {
		return err
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("sending SIGTERM: %w", err)
	}
	// Give the process up to 5 seconds to exit cleanly.
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		if proc.Signal(syscall.Signal(0)) != nil {
			break // process exited
		}
	}
	return nil
}
