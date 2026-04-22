package cmd

import (
	"fmt"
	"text/tabwriter"
	"os"
	"time"

	"github.com/gartner24/forge/penforge/internal/paths"
	"github.com/gartner24/forge/penforge/internal/store"
	"github.com/spf13/cobra"
)

var (
	reportTargetID string
	reportScanID   string
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Show scan report for a target",
	RunE:  runReport,
}

func init() {
	reportCmd.Flags().StringVar(&reportTargetID, "target", "", "Target ID (required)")
	reportCmd.Flags().StringVar(&reportScanID, "scan", "", "Specific scan ID (default: latest)")
	reportCmd.MarkFlagRequired("target")
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) error {
	scansDir, err := paths.ScansDir()
	if err != nil {
		return cmdErr(err)
	}
	findingsFile, err := paths.FindingsFile()
	if err != nil {
		return cmdErr(err)
	}

	scanStore := store.NewScanStore(scansDir)
	findingStore := store.NewFindingStore(findingsFile)

	var rec *store.ScanRecord
	if reportScanID != "" {
		rec, err = scanStore.Get(reportScanID)
		if err != nil {
			return cmdErr(err)
		}
	} else {
		rec, err = scanStore.LatestForTarget(reportTargetID)
		if err != nil {
			return cmdErr(err)
		}
		if rec == nil {
			if isJSON() {
				printJSON(map[string]any{"target_id": reportTargetID, "scans": 0})
			} else {
				fmt.Printf("No scans found for target %q.\n", reportTargetID)
			}
			return nil
		}
	}

	findings, err := findingStore.ByTarget(reportTargetID)
	if err != nil {
		return cmdErr(err)
	}

	// Count by severity.
	counts := map[string]int{"critical": 0, "high": 0, "medium": 0, "low": 0, "info": 0}
	newCounts := map[string]int{}
	for _, f := range findings {
		counts[f.Severity]++
		if f.State == store.StateNew {
			newCounts[f.Severity]++
		}
	}

	type reportOutput struct {
		ScanID     string         `json:"scan_id"`
		TargetID   string         `json:"target_id"`
		Status     string         `json:"status"`
		StartedAt  time.Time      `json:"started_at"`
		Duration   string         `json:"duration,omitempty"`
		Engines    []string       `json:"engines"`
		Summary    map[string]int `json:"summary"`
		NewSummary map[string]int `json:"new_findings"`
		Findings   []store.StoredFinding `json:"findings,omitempty"`
	}

	out := reportOutput{
		ScanID:     rec.ID,
		TargetID:   rec.TargetID,
		Status:     rec.Status,
		StartedAt:  rec.StartedAt,
		Engines:    rec.Engines,
		Summary:    counts,
		NewSummary: newCounts,
	}
	if rec.CompletedAt != nil {
		out.Duration = rec.CompletedAt.Sub(rec.StartedAt).Round(time.Second).String()
	}
	if isJSON() {
		out.Findings = findings
		printJSON(out)
		return nil
	}

	fmt.Printf("Scan Report: %s\n", rec.ID)
	fmt.Printf("Target:      %s\n", rec.TargetID)
	fmt.Printf("Status:      %s\n", rec.Status)
	fmt.Printf("Started:     %s\n", rec.StartedAt.Format(time.RFC3339))
	if out.Duration != "" {
		fmt.Printf("Duration:    %s\n", out.Duration)
	}
	fmt.Printf("Engines:     %v\n\n", rec.Engines)

	fmt.Println("Finding Summary:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SEVERITY\tTOTAL\tNEW")
	for _, sev := range []string{"critical", "high", "medium", "low", "info"} {
		fmt.Fprintf(w, "%s\t%d\t%d\n", sev, counts[sev], newCounts[sev])
	}
	w.Flush()

	if len(findings) > 0 {
		fmt.Println("\nFindings:")
		w2 := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w2, "ID\tSEVERITY\tENGINE\tNAME\tSTATE")
		for _, f := range findings {
			fmt.Fprintf(w2, "%s\t%s\t%s\t%s\t%s\n", f.ID, f.Severity, f.Engine, truncate(f.Name, 50), f.State)
		}
		w2.Flush()
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
