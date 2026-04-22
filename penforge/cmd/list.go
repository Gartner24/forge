package cmd

import (
	"fmt"
	"strings"
	"text/tabwriter"
	"os"

	"github.com/gartner24/forge/penforge/internal/paths"
	"github.com/gartner24/forge/shared/registry"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered scan targets",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	targetsFile, err := paths.TargetsFile()
	if err != nil {
		return cmdErr(err)
	}

	targets, err := registry.ReadScanTargets(targetsFile)
	if err != nil {
		return cmdErr(fmt.Errorf("reading targets: %w", err))
	}

	if isJSON() {
		if targets == nil {
			targets = []registry.ScanTarget{}
		}
		printJSON(targets)
		return nil
	}

	if len(targets) == 0 {
		fmt.Println("No targets registered. Run 'penforge add --target <url> --name <n>' to add one.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tURL\tENGINES\tSCHEDULE")
	for _, t := range targets {
		engines := strings.Join(t.Engines, ",")
		if engines == "" {
			engines = "(all)"
		}
		cron := t.Cron
		if cron == "" {
			cron = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", t.ID, t.Name, t.URL, engines, cron)
	}
	w.Flush()
	return nil
}
