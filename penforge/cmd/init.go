package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gartner24/forge/penforge/internal/paths"
	"github.com/gartner24/forge/shared/registry"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise PenForge data directories",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	if err := paths.EnsureDirs(); err != nil {
		return cmdErr(fmt.Errorf("creating directories: %w", err))
	}

	targetsFile, err := paths.TargetsFile()
	if err != nil {
		return cmdErr(err)
	}

	// Create an empty targets.json if it doesn't exist.
	if _, err := os.Stat(targetsFile); os.IsNotExist(err) {
		empty, _ := json.MarshalIndent([]registry.ScanTarget{}, "", "  ")
		if err := os.WriteFile(targetsFile, empty, 0644); err != nil {
			return cmdErr(fmt.Errorf("creating targets.json: %w", err))
		}
	}

	printSuccess("PenForge initialised. Run 'penforge add --target <url> --name <n>' to register a scan target.")
	return nil
}
