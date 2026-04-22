package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/gartner24/forge/penforge/internal/paths"
	"github.com/gartner24/forge/penforge/internal/store"
	"github.com/spf13/cobra"
)

var (
	scansListTargetID string
)

var scansCmd = &cobra.Command{
	Use:   "scans",
	Short: "Manage scan history",
}

var scansListCmd = &cobra.Command{
	Use:   "list",
	Short: "List scan history for a target",
	RunE:  runScansList,
}

var scansShowCmd = &cobra.Command{
	Use:   "show <scan-id>",
	Short: "Show scan details and finding counts",
	Args:  cobra.ExactArgs(1),
	RunE:  runScansShow,
}

func init() {
	scansListCmd.Flags().StringVar(&scansListTargetID, "target", "", "Target ID")
	scansCmd.AddCommand(scansListCmd)
	scansCmd.AddCommand(scansShowCmd)
	rootCmd.AddCommand(scansCmd)
}

func runScansList(cmd *cobra.Command, args []string) error {
	scansDir, err := paths.ScansDir()
	if err != nil {
		return cmdErr(err)
	}

	scanStore := store.NewScanStore(scansDir)
	scans, err := scanStore.AllForTarget(scansListTargetID)
	if err != nil {
		return cmdErr(err)
	}

	sort.Slice(scans, func(i, j int) bool {
		return scans[i].StartedAt.After(scans[j].StartedAt)
	})

	if isJSON() {
		if scans == nil {
			scans = []store.ScanRecord{}
		}
		printJSON(scans)
		return nil
	}

	if len(scans) == 0 {
		fmt.Println("No scans found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SCAN ID\tTARGET\tSTATUS\tSTARTED\tNEW\tTOTAL")
	for _, s := range scans {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%d\n",
			s.ID, s.TargetID, s.Status,
			s.StartedAt.Format(time.RFC3339),
			s.NewFindings, s.TotalFindings)
	}
	w.Flush()
	return nil
}

func runScansShow(cmd *cobra.Command, args []string) error {
	scanID := args[0]

	scansDir, err := paths.ScansDir()
	if err != nil {
		return cmdErr(err)
	}

	scanStore := store.NewScanStore(scansDir)
	rec, err := scanStore.Get(scanID)
	if err != nil {
		return cmdErr(err)
	}

	if isJSON() {
		printJSON(rec)
		return nil
	}

	fmt.Printf("Scan ID:    %s\n", rec.ID)
	fmt.Printf("Target:     %s\n", rec.TargetID)
	fmt.Printf("Status:     %s\n", rec.Status)
	fmt.Printf("Started:    %s\n", rec.StartedAt.Format(time.RFC3339))
	if rec.CompletedAt != nil {
		dur := rec.CompletedAt.Sub(rec.StartedAt).Round(time.Second)
		fmt.Printf("Completed:  %s (%s)\n", rec.CompletedAt.Format(time.RFC3339), dur)
	}
	fmt.Printf("Engines:    %v\n", rec.Engines)
	if rec.Error != "" {
		fmt.Printf("Error:      %s\n", rec.Error)
	}
	fmt.Printf("\nNew:        %d\n", rec.NewFindings)
	fmt.Printf("Recurring:  %d\n", rec.RecurringFindings)
	fmt.Printf("Resolved:   %d\n", rec.ResolvedFindings)
	fmt.Printf("Total:      %d\n", rec.TotalFindings)
	return nil
}
