package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gartner24/forge/core/internal/paths"
	"github.com/gartner24/forge/core/internal/state"
	"github.com/spf13/cobra"
)

var uninstallPurge bool

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <module>",
	Short: "Uninstall a Forge module",
	Args:  cobra.ExactArgs(1),
	RunE:  runUninstall,
}

func init() {
	uninstallCmd.Flags().BoolVar(&uninstallPurge, "purge", false, "Also delete all module data")
}

func runUninstall(cmd *cobra.Command, args []string) error {
	module := args[0]

	cfg, err := requireInit()
	if err != nil {
		return cmdErr(err)
	}

	mod, ok := cfg.Modules[module]
	if !ok || !mod.Enabled {
		return cmdErr(fmt.Errorf("%s is not installed", module))
	}

	proceed, err := mustConfirm(fmt.Sprintf("Uninstall %s?", module))
	if err != nil {
		return cmdErr(err)
	}
	if !proceed {
		fmt.Println("Aborted.")
		return nil
	}

	// Stop if running.
	s, _ := state.Get(module)
	if s != nil && state.IsAlive(s) {
		if stopErr := stopModule(module, s); stopErr != nil && verbose {
			fmt.Fprintf(os.Stderr, "Warning: stopping %s: %v\n", module, stopErr)
		}
	}
	state.Delete(module)

	// Remove from config.
	mod.Enabled = false
	cfg.Modules[module] = mod

	cfgPath, err := paths.ConfigFile()
	if err != nil {
		return cmdErr(err)
	}
	if err := writeConfig(cfg, cfgPath); err != nil {
		return cmdErr(fmt.Errorf("saving config: %w", err))
	}

	// Purge data directory if requested.
	if uninstallPurge {
		dataDir := mod.DataDir
		if dataDir == "" {
			dataDir = filepath.Join(effectiveDataDir(cfg), module)
		}
		if err := os.RemoveAll(dataDir); err != nil && verbose {
			fmt.Fprintf(os.Stderr, "Warning: removing data dir: %v\n", err)
		}
	}

	msg := fmt.Sprintf("%s uninstalled.", module)
	if !uninstallPurge {
		msg += " Data preserved. Pass --purge to delete it."
	}
	printSuccess(msg)
	return nil
}
