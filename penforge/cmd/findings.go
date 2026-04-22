package cmd

import (
	"fmt"
	"text/tabwriter"
	"os"

	"github.com/gartner24/forge/penforge/internal/paths"
	"github.com/gartner24/forge/penforge/internal/store"
	"github.com/spf13/cobra"
)

var (
	findingsTargetID  string
	findingsSeverity  string
	findingsState     string
)

var findingsCmd = &cobra.Command{
	Use:   "findings",
	Short: "List security findings",
	RunE:  runFindings,
}

func init() {
	findingsCmd.Flags().StringVar(&findingsTargetID, "target", "", "Filter by target ID")
	findingsCmd.Flags().StringVar(&findingsSeverity, "severity", "", "Filter by severity (critical|high|medium|low|info)")
	findingsCmd.Flags().StringVar(&findingsState, "state", "", "Filter by state (new|acknowledged|accepted|fixed|verified)")
	rootCmd.AddCommand(findingsCmd)
}

func runFindings(cmd *cobra.Command, args []string) error {
	findingsFile, err := paths.FindingsFile()
	if err != nil {
		return cmdErr(err)
	}

	findingStore := store.NewFindingStore(findingsFile)

	var findings []store.StoredFinding
	if findingsTargetID != "" {
		findings, err = findingStore.ByTarget(findingsTargetID)
	} else {
		findings, err = findingStore.All()
	}
	if err != nil {
		return cmdErr(fmt.Errorf("reading findings: %w", err))
	}

	// Apply filters.
	findings = filterFindings(findings, findingsSeverity, findingsState)

	if isJSON() {
		if findings == nil {
			findings = []store.StoredFinding{}
		}
		printJSON(findings)
		return nil
	}

	if len(findings) == 0 {
		fmt.Println("No findings.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSEVERITY\tENGINE\tTARGET\tNAME\tSTATE")
	for _, f := range findings {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			f.ID, f.Severity, f.Engine, f.TargetID, truncate(f.Name, 45), f.State)
	}
	w.Flush()
	return nil
}

func filterFindings(findings []store.StoredFinding, severity, state string) []store.StoredFinding {
	if severity == "" && state == "" {
		return findings
	}
	var out []store.StoredFinding
	for _, f := range findings {
		if severity != "" && f.Severity != severity {
			continue
		}
		if state != "" && f.State != state {
			continue
		}
		out = append(out, f)
	}
	return out
}
