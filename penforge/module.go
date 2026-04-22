package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gartner24/forge/penforge/internal/hook"
	"github.com/gartner24/forge/penforge/internal/paths"
	"github.com/gartner24/forge/penforge/internal/scheduler"
	"github.com/gartner24/forge/penforge/internal/store"
	"github.com/gartner24/forge/shared/module"
)

const moduleName = "penforge"
const moduleVersion = "0.1.0"

// PenForgeModule implements shared/module.Module.
type PenForgeModule struct {
	status    module.Status
	hookSrv   *hook.Server
	sched     *scheduler.Scheduler
	cancelFn  context.CancelFunc
}

func NewModule() *PenForgeModule {
	return &PenForgeModule{status: module.StatusStopped}
}

func (m *PenForgeModule) Name() string    { return moduleName }
func (m *PenForgeModule) Version() string { return moduleVersion }
func (m *PenForgeModule) Status() module.Status { return m.status }

func (m *PenForgeModule) Start() error {
	if m.status == module.StatusRunning {
		return fmt.Errorf("penforge is already running")
	}

	if err := paths.EnsureDirs(); err != nil {
		m.status = module.StatusError
		return fmt.Errorf("initialising directories: %w", err)
	}

	targetsFile, err := paths.TargetsFile()
	if err != nil {
		return err
	}
	findingsFile, err := paths.FindingsFile()
	if err != nil {
		return err
	}
	scansDir, err := paths.ScansDir()
	if err != nil {
		return err
	}

	auditFile, err := paths.AuditFile()
	if err != nil {
		return err
	}

	findingStore := store.NewFindingStoreWithAudit(findingsFile, auditFile)
	scanStore := store.NewScanStore(scansDir)

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFn = cancel

	m.hookSrv = hook.New(targetsFile, findingStore, scanStore, "", auditFile)
	m.sched = scheduler.New(targetsFile, findingStore, scanStore, "", auditFile)

	if err := m.sched.Start(ctx); err != nil {
		cancel()
		m.status = module.StatusError
		return fmt.Errorf("starting scheduler: %w", err)
	}

	// Start the HTTP hook server in a goroutine.
	go func() {
		if err := m.hookSrv.ListenAndServe(); err != nil {
			m.status = module.StatusError
		}
	}()

	// Wait a moment for the server to bind.
	time.Sleep(100 * time.Millisecond)
	m.status = module.StatusRunning
	return nil
}

func (m *PenForgeModule) Stop() error {
	if m.status != module.StatusRunning {
		return nil
	}
	if m.cancelFn != nil {
		m.cancelFn()
	}
	if m.sched != nil {
		m.sched.Stop()
	}
	m.status = module.StatusStopped
	return nil
}
