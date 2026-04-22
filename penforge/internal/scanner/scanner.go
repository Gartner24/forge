package scanner

import (
	"context"
	"fmt"
	"time"

	dockertypes "github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
	"github.com/gartner24/forge/penforge/internal/engine"
	"github.com/gartner24/forge/penforge/internal/store"
	"github.com/gartner24/forge/shared/audit"
	"github.com/gartner24/forge/shared/notify"
	"github.com/gartner24/forge/shared/registry"
)

// Result is returned after a synchronous scan completes.
type Result struct {
	ScanID            string
	TargetID          string
	NewFindings       int
	RecurringFindings int
	ResolvedFindings  int
	TotalFindings     int
	Duration          time.Duration
}

// RunScan executes a full scan against the given target using the specified engines.
// It enforces network isolation, runs each engine, performs delta detection,
// saves results, and sends SparkForge alerts for new critical/high findings.
func RunScan(
	ctx context.Context,
	scanID string,
	target registry.ScanTarget,
	engines []engine.Engine,
	findingStore *store.FindingStore,
	scanStore *store.ScanStore,
	sparkForgeAddr string,
	auditPath string,
) (*Result, error) {
	start := time.Now()

	var auditLog *audit.Logger
	if auditPath != "" {
		if al, err := audit.New(auditPath); err == nil {
			auditLog = al
		}
	}
	if auditLog != nil {
		auditLog.Write("penforge", "scan_started", "", fmt.Sprintf("scan=%s target=%s engines=%v", scanID, target.ID, engineNames(engines)))
	}

	rec := store.ScanRecord{
		ID:               scanID,
		TargetID:         target.ID,
		TargetURL:        target.URL,
		Engines:          engineNames(engines),
		Status:           "running",
		StartedAt:        start,
		EstimatedSeconds: estimatedSeconds(len(engines)),
	}
	if err := scanStore.Save(rec); err != nil {
		return nil, fmt.Errorf("saving scan record: %w", err)
	}

	// Create an isolated Docker network for this scan.
	cli, err := dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	defer cli.Close()

	networkID, err := createNetwork(ctx, cli, scanID)
	if err != nil {
		return nil, fmt.Errorf("scan network: %w", err)
	}
	defer func() {
		rmCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		cli.NetworkRemove(rmCtx, networkID)
	}()

	// Run each engine, collecting all findings.
	var allFindings []engine.Finding
	for _, eng := range engines {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		findings, err := eng.Run(ctx, target, networkID)
		if err != nil {
			// Log engine failure but continue with other engines.
			fmt.Printf("engine %s failed: %v\n", eng.Name(), err)
			continue
		}
		allFindings = append(allFindings, findings...)
	}

	// Filter out findings for hosts outside the declared scope.
	allFindings = filterByScope(allFindings, target)

	now := time.Now()
	delta, err := findingStore.Upsert(target.ID, target.URL, allFindings, now)
	if err != nil {
		return nil, fmt.Errorf("persisting findings: %w", err)
	}

	// Alert on new critical/high findings.
	if sparkForgeAddr != "" {
		for _, f := range delta.New {
			if f.Severity == "critical" || f.Severity == "high" {
				prio := notify.PriorityHigh
				if f.Severity == "critical" {
					prio = notify.PriorityCritical
				}
				notify.Send(sparkForgeAddr, notify.Message{
					Title:    fmt.Sprintf("[PenForge] New %s finding: %s", f.Severity, f.Name),
					Body:     fmt.Sprintf("Target: %s\nEngine: %s\nHost: %s", target.ID, f.Engine, f.Host),
					Priority: prio,
					Source:   "penforge",
				})
			}
		}
		// Alert if severity increased for acknowledged/accepted findings.
		for _, f := range delta.Recurring {
			if f.PrevSeverity != "" && severityRank(f.Severity) > severityRank(f.PrevSeverity) {
				notify.Send(sparkForgeAddr, notify.Message{
					Title:    fmt.Sprintf("[PenForge] Severity increased: %s (%s -> %s)", f.Name, f.PrevSeverity, f.Severity),
					Body:     fmt.Sprintf("Target: %s\nFinding: %s", target.ID, f.ID),
					Priority: notify.PriorityHigh,
					Source:   "penforge",
				})
			}
		}
	}

	completedAt := time.Now()
	total, _ := findingStore.ByTarget(target.ID)
	rec.Status = "completed"
	rec.CompletedAt = &completedAt
	rec.NewFindings = len(delta.New)
	rec.RecurringFindings = len(delta.Recurring)
	rec.ResolvedFindings = len(delta.Resolved)
	rec.TotalFindings = len(total)
	if err := scanStore.Save(rec); err != nil {
		return nil, fmt.Errorf("updating scan record: %w", err)
	}
	if auditLog != nil {
		auditLog.Write("penforge", "scan_completed", "", fmt.Sprintf("scan=%s target=%s new=%d recurring=%d resolved=%d", scanID, target.ID, len(delta.New), len(delta.Recurring), len(delta.Resolved)))
	}

	return &Result{
		ScanID:            scanID,
		TargetID:          target.ID,
		NewFindings:       len(delta.New),
		RecurringFindings: len(delta.Recurring),
		ResolvedFindings:  len(delta.Resolved),
		TotalFindings:     len(total),
		Duration:          time.Since(start),
	}, nil
}

// MarkFailed marks a scan as failed in the scan store.
func MarkFailed(scanStore *store.ScanStore, scanID string, scanErr error) {
	rec, err := scanStore.Get(scanID)
	if err != nil {
		return
	}
	t := time.Now()
	rec.Status = "failed"
	rec.CompletedAt = &t
	rec.Error = scanErr.Error()
	scanStore.Save(*rec)
}

func createNetwork(ctx context.Context, cli *dockerclient.Client, scanID string) (string, error) {
	name := "penforge-scan-" + scanID
	resp, err := cli.NetworkCreate(ctx, name, dockertypes.CreateOptions{
		Driver:     "bridge",
		Attachable: false,
	})
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func engineNames(engines []engine.Engine) []string {
	names := make([]string, len(engines))
	for i, e := range engines {
		names[i] = e.Name()
	}
	return names
}

func estimatedSeconds(numEngines int) int {
	return numEngines * 120
}

func filterByScope(findings []engine.Finding, target registry.ScanTarget) []engine.Finding {
	if len(target.Scope) == 0 {
		return findings
	}
	scopeSet := map[string]bool{}
	for _, s := range target.Scope {
		scopeSet[s] = true
	}
	var out []engine.Finding
	for _, f := range findings {
		if f.Host == "" || scopeSet[f.Host] {
			out = append(out, f)
		}
	}
	return out
}

func severityRank(s string) int {
	switch s {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}
