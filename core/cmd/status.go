package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/gartner24/forge/core/internal/state"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of all installed modules",
	RunE:  runStatus,
}

type moduleStatus struct {
	Module  string     `json:"module"`
	Version string     `json:"version"`
	State   string     `json:"state"`
	Since   *time.Time `json:"since"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := requireInit()
	if err != nil {
		return cmdErr(err)
	}

	var rows []moduleStatus

	// Collect installed modules in deterministic order.
	names := make([]string, 0, len(cfg.Modules))
	for name, mod := range cfg.Modules {
		if mod.Enabled {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	for _, name := range names {
		s, _ := state.Get(name)
		ms := moduleStatus{Module: name}

		if s != nil && state.IsAlive(s) {
			ms.State = "running"
			ms.Version = s.Version
			t := s.StartedAt
			ms.Since = &t
		} else {
			ms.State = "stopped"
			ms.Version = "(none)"
		}
		rows = append(rows, ms)
	}

	if isJSON() {
		if rows == nil {
			rows = []moduleStatus{}
		}
		printJSON(rows)
		return nil
	}

	if len(rows) == 0 {
		fmt.Println("No modules installed. Run 'forge install <module>' to get started.")
		return nil
	}

	w := newTabWriter()
	fmt.Fprintln(w, "MODULE\tVERSION\tSTATE\tSINCE")
	for _, r := range rows {
		since := "-"
		if r.Since != nil {
			since = formatSince(*r.Since)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Module, r.Version, r.State, since)
	}
	w.Flush()
	return nil
}
