package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/gartner24/forge/core/internal/paths"
	"github.com/spf13/cobra"
)

var updateAll bool

var updateCmd = &cobra.Command{
	Use:   "update [<module>]",
	Short: "Update a module to the latest version",
	RunE:  runUpdate,
}

func init() {
	updateCmd.Flags().BoolVar(&updateAll, "all", false, "Update every installed module")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	if !updateAll && len(args) == 0 {
		return cmdErr(fmt.Errorf("specify a module name or pass --all"))
	}

	cfg, err := requireInit()
	if err != nil {
		return cmdErr(err)
	}

	var targets []string
	if updateAll {
		for name, mod := range cfg.Modules {
			if mod.Enabled {
				targets = append(targets, name)
			}
		}
		if len(targets) == 0 {
			printSuccess("No modules installed.")
			return nil
		}
	} else {
		targets = []string{args[0]}
	}

	type result struct {
		Module  string `json:"module"`
		Error   string `json:"error"`
	}
	var results []result
	for _, module := range targets {
		if err := updateModule(module, cfg.Forge.InstallDir); err != nil {
			if isJSON() {
				results = append(results, result{Module: module, Error: err.Error()})
			} else {
				fmt.Printf("  %s: %v\n", module, err)
			}
		}
	}
	if isJSON() && len(results) > 0 {
		printJSON(results)
	}
	return nil
}

func updateModule(module, installDir string) error {
	if installDir == "" {
		dir, _ := paths.BinDir()
		installDir = dir
	}
	binPath := filepath.Join(installDir, module)
	// Automated download is not yet implemented. Guide the user to place the
	// updated binary manually.
	return fmt.Errorf(
		"automated update not available -- download the latest %s binary from "+
			"the forge releases and place it at %s, then run 'forge stop %s && forge start %s'",
		module, binPath, module, module,
	)
}
