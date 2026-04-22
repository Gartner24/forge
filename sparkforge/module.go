package main

import (
	"github.com/gartner24/forge/shared/module"
)

// Compile-time check that sparkForge implements module.Module.
var _ module.Module = (*sparkForge)(nil)

type sparkForge struct {
	running bool
	stop    chan struct{}
}

func (s *sparkForge) Name() string { return "sparkforge" }

func (s *sparkForge) Version() string { return version }

func (s *sparkForge) Status() module.Status {
	if s.running {
		return module.StatusRunning
	}
	return module.StatusStopped
}

func (s *sparkForge) Start() error {
	s.running = true
	return nil
}

func (s *sparkForge) Stop() error {
	s.running = false
	if s.stop != nil {
		close(s.stop)
	}
	return nil
}
