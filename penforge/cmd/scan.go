package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/gartner24/forge/penforge/internal/engine"
	"github.com/gartner24/forge/penforge/internal/paths"
	"github.com/gartner24/forge/penforge/internal/scanner"
	"github.com/gartner24/forge/penforge/internal/store"
	"github.com/gartner24/forge/shared/registry"
	"github.com/spf13/cobra"
)

var (
	scanTargetID string
	scanEngine   string
	scanAsync    bool
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Run a security scan against a registered target",
	RunE:  runScan,
}

// runScanBgCmd is the hidden subcommand used for async background scans.
var runScanBgCmd = &cobra.Command{
	Use:    "_run-scan",
	Hidden: true,
	RunE:   runScanBackground,
}

var bgScanID string

func init() {
	scanCmd.Flags().StringVar(&scanTargetID, "target", "", "Target ID (required)")
	scanCmd.Flags().StringVar(&scanEngine, "engine", "", "Run a single engine only")
	scanCmd.Flags().BoolVar(&scanAsync, "async", false, "Return immediately with scan_id, don't wait for completion")
	scanCmd.MarkFlagRequired("target")

	runScanBgCmd.Flags().StringVar(&bgScanID, "scan-id", "", "Scan ID to run")
	runScanBgCmd.MarkFlagRequired("scan-id")

	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(runScanBgCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	target, engines, err := resolveScanParams(scanTargetID, scanEngine)
	if err != nil {
		return cmdErr(err)
	}

	scanID := newScanID()

	if scanAsync {
		return runScanAsync(scanID, target, engines)
	}

	return runScanSync(cmd.Context(), scanID, target, engines)
}

func runScanAsync(scanID string, target registry.ScanTarget, engines []engine.Engine) error {
	// Save an initial record so the scan is visible immediately.
	scansDir, err := paths.ScansDir()
	if err != nil {
		return cmdErr(err)
	}
	findingsFile, err := paths.FindingsFile()
	if err != nil {
		return cmdErr(err)
	}

	findingStore := store.NewFindingStore(findingsFile)
	scanStore := store.NewScanStore(scansDir)

	rec := store.ScanRecord{
		ID:               scanID,
		TargetID:         target.ID,
		TargetURL:        target.URL,
		Engines:          engineNames(engines),
		Status:           "running",
		StartedAt:        time.Now().UTC(),
		EstimatedSeconds: len(engines) * 120,
	}
	_ = findingStore
	if err := scanStore.Save(rec); err != nil {
		return cmdErr(fmt.Errorf("saving scan record: %w", err))
	}

	// Fork a detached subprocess to run the actual scan.
	self := os.Args[0]
	bgCmd := exec.Command(self, "_run-scan", "--scan-id", scanID)
	bgCmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := bgCmd.Start(); err != nil {
		// Fall back to synchronous if we can't fork.
		return runScanSync(context.Background(), scanID, target, engines)
	}
	// Detach — do not wait.

	resp := map[string]any{
		"scan_id":           scanID,
		"target_id":         target.ID,
		"started_at":        rec.StartedAt.Format(time.RFC3339),
		"estimated_seconds": rec.EstimatedSeconds,
		"engines":           rec.Engines,
	}
	printJSON(resp)
	return nil
}

func runScanSync(ctx context.Context, scanID string, target registry.ScanTarget, engines []engine.Engine) error {
	scansDir, err := paths.ScansDir()
	if err != nil {
		return cmdErr(err)
	}
	findingsFile, err := paths.FindingsFile()
	if err != nil {
		return cmdErr(err)
	}
	auditFile, err := paths.AuditFile()
	if err != nil {
		return cmdErr(err)
	}
	findingStore := store.NewFindingStore(findingsFile)
	scanStore := store.NewScanStore(scansDir)

	if !isJSON() {
		fmt.Printf("Scanning target %q with engines: %s\n", target.ID, strings.Join(engineNames(engines), ", "))
	}

	result, err := scanner.RunScan(ctx, scanID, target, engines, findingStore, scanStore, "", auditFile)
	if err != nil {
		scanner.MarkFailed(scanStore, scanID, err)
		return cmdErr(fmt.Errorf("scan failed: %w", err))
	}

	if isJSON() {
		printJSON(result)
	} else {
		fmt.Printf("Scan %s completed in %s\n", result.ScanID, result.Duration.Round(time.Second))
		fmt.Printf("  New:       %d\n", result.NewFindings)
		fmt.Printf("  Recurring: %d\n", result.RecurringFindings)
		fmt.Printf("  Resolved:  %d\n", result.ResolvedFindings)
		fmt.Printf("  Total:     %d\n", result.TotalFindings)
	}
	return nil
}

// runScanBackground is called by the hidden _run-scan command to execute an async scan.
func runScanBackground(cmd *cobra.Command, args []string) error {
	scansDir, err := paths.ScansDir()
	if err != nil {
		return err
	}
	findingsFile, err := paths.FindingsFile()
	if err != nil {
		return err
	}

	scanStore := store.NewScanStore(scansDir)
	findingStore := store.NewFindingStore(findingsFile)

	rec, err := scanStore.Get(bgScanID)
	if err != nil {
		return fmt.Errorf("loading scan %s: %w", bgScanID, err)
	}

	targetsFile, err := paths.TargetsFile()
	if err != nil {
		return err
	}
	targets, err := registry.ReadScanTargets(targetsFile)
	if err != nil {
		return err
	}

	var target *registry.ScanTarget
	for _, t := range targets {
		if t.ID == rec.TargetID {
			cp := t
			target = &cp
			break
		}
	}
	if target == nil {
		return fmt.Errorf("target %q not found for scan %s", rec.TargetID, bgScanID)
	}

	var engines []engine.Engine
	if len(rec.Engines) > 0 {
		for _, name := range rec.Engines {
			if e := engine.ByName(name); e != nil {
				engines = append(engines, e)
			}
		}
	} else {
		engines = engine.ForTarget(*target)
	}

	auditFile, _ := paths.AuditFile()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if _, err := scanner.RunScan(ctx, bgScanID, *target, engines, findingStore, scanStore, "", auditFile); err != nil {
		scanner.MarkFailed(scanStore, bgScanID, err)
	}
	return nil
}

func resolveScanParams(targetID, engineName string) (registry.ScanTarget, []engine.Engine, error) {
	targetsFile, err := paths.TargetsFile()
	if err != nil {
		return registry.ScanTarget{}, nil, err
	}
	targets, err := registry.ReadScanTargets(targetsFile)
	if err != nil {
		return registry.ScanTarget{}, nil, fmt.Errorf("reading targets: %w", err)
	}

	var target *registry.ScanTarget
	for _, t := range targets {
		if t.ID == targetID {
			cp := t
			target = &cp
			break
		}
	}
	if target == nil {
		return registry.ScanTarget{}, nil, fmt.Errorf("target %q not registered -- run 'penforge add' to register it", targetID)
	}

	var engines []engine.Engine
	if engineName != "" {
		e := engine.ByName(engineName)
		if e == nil {
			return registry.ScanTarget{}, nil, fmt.Errorf("unknown engine %q", engineName)
		}
		engines = []engine.Engine{e}
	} else {
		engines = engine.ForTarget(*target)
	}

	return *target, engines, nil
}

func engineNames(engines []engine.Engine) []string {
	names := make([]string, len(engines))
	for i, e := range engines {
		names[i] = e.Name()
	}
	return names
}

func newScanID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return fmt.Sprintf("scan-%s-%s", time.Now().UTC().Format("20060102"), hex.EncodeToString(b))
}
