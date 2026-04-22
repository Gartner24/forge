package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/gartner24/forge/core/internal/paths"
	sharedconfig "github.com/gartner24/forge/shared/config"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install <module>",
	Short: "Install a Forge module",
	Long:  "Registers the module with Forge Core. Does not start it -- run forge start <module> after installation.",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstall,
}

func runInstall(cmd *cobra.Command, args []string) error {
	module := args[0]

	if !knownModules[module] {
		return cmdErr(fmt.Errorf("unknown module %q -- known modules: smeltforge, watchforge, sparkforge, fluxforge, hearthforge, penforge", module))
	}

	cfg, err := requireInit()
	if err != nil {
		return cmdErr(err)
	}

	if nodeTarget != "" && verbose {
		fmt.Printf("Note: --node targeting requires FluxForge. Running locally.\n")
	}

	if cfg.Modules == nil {
		cfg.Modules = make(map[string]sharedconfig.ModuleConfig)
	}

	if m, ok := cfg.Modules[module]; ok && m.Enabled {
		printSuccess(fmt.Sprintf("%s is already installed", module))
		return nil
	}

	dataDir := filepath.Join(effectiveDataDir(cfg), module)
	cfg.Modules[module] = sharedconfig.ModuleConfig{
		Enabled: true,
		DataDir: dataDir,
	}

	cfgPath, err := paths.ConfigFile()
	if err != nil {
		return cmdErr(err)
	}
	if err := writeConfig(cfg, cfgPath); err != nil {
		return cmdErr(fmt.Errorf("saving config: %w", err))
	}

	msg := fmt.Sprintf("%s installed. Run 'forge start %s' to bring it online.", module, module)
	printSuccess(msg)
	return nil
}
