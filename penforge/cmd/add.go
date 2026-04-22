package cmd

import (
	"fmt"
	"strings"

	"github.com/gartner24/forge/penforge/internal/paths"
	"github.com/gartner24/forge/shared/registry"
	"github.com/spf13/cobra"
)

var (
	addTarget  string
	addName    string
	addScope   string
	addEngines string
	addCron    string
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Register a scan target",
	RunE:  runAdd,
}

func init() {
	addCmd.Flags().StringVar(&addTarget, "target", "", "Target URL (required)")
	addCmd.Flags().StringVar(&addName, "name", "", "Target name/ID (required)")
	addCmd.Flags().StringVar(&addScope, "scope", "", "Comma-separated domains and IPs in scope")
	addCmd.Flags().StringVar(&addEngines, "engines", "", "Comma-separated engines to run (default: all)")
	addCmd.Flags().StringVar(&addCron, "cron", "", "Cron expression for scheduled scans (e.g. '0 2 * * *')")
	addCmd.MarkFlagRequired("target")
	addCmd.MarkFlagRequired("name")
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	targetsFile, err := paths.TargetsFile()
	if err != nil {
		return cmdErr(err)
	}

	targets, err := registry.ReadScanTargets(targetsFile)
	if err != nil {
		return cmdErr(fmt.Errorf("reading targets: %w", err))
	}

	id := slugify(addName)
	for _, t := range targets {
		if t.ID == id {
			return cmdErr(fmt.Errorf("target %q already exists -- use 'penforge update' to modify it", id))
		}
	}

	var scope []string
	if addScope != "" {
		for _, s := range strings.Split(addScope, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				scope = append(scope, s)
			}
		}
	}

	var engines []string
	if addEngines != "" {
		for _, e := range strings.Split(addEngines, ",") {
			e = strings.TrimSpace(e)
			if e != "" {
				engines = append(engines, e)
			}
		}
	}

	t := registry.ScanTarget{
		ID:      id,
		Name:    addName,
		URL:     addTarget,
		Scope:   scope,
		Engines: engines,
		Cron:    addCron,
	}
	targets = append(targets, t)

	if err := registry.WriteScanTargets(targetsFile, targets); err != nil {
		return cmdErr(fmt.Errorf("saving target: %w", err))
	}

	if isJSON() {
		printJSON(t)
	} else {
		fmt.Printf("Target %q registered. Run 'penforge scan --target %s' to scan it.\n", t.Name, t.ID)
	}
	return nil
}

func slugify(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), " ", "-"))
}
