package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/gartner24/forge/penforge/internal/engine"
	"github.com/spf13/cobra"
)

var (
	enginesUpdateEngine string
)

var enginesCmd = &cobra.Command{
	Use:   "engines",
	Short: "Manage scan engines",
	RunE:  runEnginesList,
}

var enginesUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Pull latest engine images",
	RunE:  runEnginesUpdate,
}

func init() {
	enginesUpdateCmd.Flags().StringVar(&enginesUpdateEngine, "engine", "", "Update a specific engine only")
	enginesCmd.AddCommand(enginesUpdateCmd)
	rootCmd.AddCommand(enginesCmd)
}

func runEnginesList(cmd *cobra.Command, args []string) error {
	engines := engine.All()

	type engineInfo struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	var infos []engineInfo
	for _, e := range engines {
		infos = append(infos, engineInfo{Name: e.Name(), Version: e.Version()})
	}

	if isJSON() {
		printJSON(infos)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ENGINE\tIMAGE")
	for _, info := range infos {
		fmt.Fprintf(w, "%s\t%s\n", info.Name, info.Version)
	}
	w.Flush()
	return nil
}

func runEnginesUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	engines := engine.All()

	if enginesUpdateEngine != "" {
		e := engine.ByName(enginesUpdateEngine)
		if e == nil {
			return cmdErr(fmt.Errorf("unknown engine %q", enginesUpdateEngine))
		}
		engines = []engine.Engine{e}
	}

	var results []map[string]any
	for _, e := range engines {
		if !isJSON() {
			fmt.Printf("Pulling %s...\n", e.Name())
		}
		err := e.Pull(ctx)
		result := map[string]any{"engine": e.Name(), "ok": err == nil}
		if err != nil {
			result["error"] = err.Error()
			if !isJSON() {
				fmt.Fprintf(os.Stderr, "  Failed: %v\n", err)
			}
		} else if !isJSON() {
			fmt.Printf("  OK\n")
		}
		results = append(results, result)
	}

	if isJSON() {
		printJSON(results)
	}
	return nil
}
