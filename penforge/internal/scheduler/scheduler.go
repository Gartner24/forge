package scheduler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/gartner24/forge/penforge/internal/engine"
	"github.com/gartner24/forge/penforge/internal/scanner"
	"github.com/gartner24/forge/penforge/internal/store"
	"github.com/gartner24/forge/shared/registry"
)

// Scheduler runs cron-scheduled scans for registered targets.
type Scheduler struct {
	targetsPath string
	findings    *store.FindingStore
	scans       *store.ScanStore
	sparkAddr   string
	auditPath   string
	cron        *cron.Cron
}

func New(targetsPath string, findings *store.FindingStore, scans *store.ScanStore, sparkAddr, auditPath string) *Scheduler {
	return &Scheduler{
		targetsPath: targetsPath,
		findings:    findings,
		scans:       scans,
		sparkAddr:   sparkAddr,
		auditPath:   auditPath,
		cron:        cron.New(),
	}
}

// Start loads targets and registers cron jobs for those with a cron expression.
func (s *Scheduler) Start(ctx context.Context) error {
	targets, err := registry.ReadScanTargets(s.targetsPath)
	if err != nil {
		return fmt.Errorf("reading targets: %w", err)
	}

	for _, t := range targets {
		if t.Cron == "" {
			continue
		}
		target := t // capture for closure
		_, err := s.cron.AddFunc(target.Cron, func() {
			s.runTarget(ctx, target)
		})
		if err != nil {
			log.Printf("penforge: invalid cron expression for target %s (%q): %v", target.ID, target.Cron, err)
		}
	}

	s.cron.Start()
	log.Printf("penforge: scheduler started with %d scheduled targets", len(targets))
	return nil
}

// Stop halts the scheduler.
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

func (s *Scheduler) runTarget(ctx context.Context, target registry.ScanTarget) {
	engines := engine.ForTarget(target)
	scanID := newScanID()
	log.Printf("penforge: scheduled scan %s started for target %s", scanID, target.ID)

	scanCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	if _, err := scanner.RunScan(scanCtx, scanID, target, engines, s.findings, s.scans, s.sparkAddr, s.auditPath); err != nil {
		log.Printf("penforge: scheduled scan failed for %s: %v", target.ID, err)
		scanner.MarkFailed(s.scans, scanID, err)
	}
}

func newScanID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return fmt.Sprintf("scan-%s-%s", time.Now().UTC().Format("20060102"), hex.EncodeToString(b))
}
