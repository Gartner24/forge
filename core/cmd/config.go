package cmd

import (
	"fmt"

	"github.com/BurntSushi/toml"
	sharedconfig "github.com/gartner24/forge/shared/config"
	"github.com/gartner24/forge/core/internal/paths"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Forge configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print the current configuration",
	RunE:  runConfigShow,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value (e.g. forge.domain)",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigGet,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value (e.g. forge.domain dev.example.com)",
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := requireInit()
	if err != nil {
		return cmdErr(err)
	}
	if isJSON() {
		printJSON(cfg)
		return nil
	}
	cfgPath, _ := paths.ConfigFile()
	fmt.Printf("# %s\n", cfgPath)
	if err := toml.NewEncoder(cmd.OutOrStdout()).Encode(cfg); err != nil {
		return cmdErr(err)
	}
	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	cfg, err := requireInit()
	if err != nil {
		return cmdErr(err)
	}
	key := args[0]
	val, ok := getForgeConfigField(cfg, key)
	if !ok {
		return cmdErr(fmt.Errorf("unknown config key %q -- supported keys: forge.domain, forge.data_dir, forge.install_dir, forge.log_level, forge.version", key))
	}
	if isJSON() {
		printJSON(map[string]string{"key": key, "value": val})
	} else {
		fmt.Println(val)
	}
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	cfg, err := requireInit()
	if err != nil {
		return cmdErr(err)
	}
	key, value := args[0], args[1]
	if !setForgeConfigField(cfg, key, value) {
		return cmdErr(fmt.Errorf("unknown or read-only config key %q -- writable keys: forge.domain, forge.data_dir, forge.install_dir, forge.log_level", key))
	}
	cfgPath, err := paths.ConfigFile()
	if err != nil {
		return cmdErr(err)
	}
	if err := writeConfig(cfg, cfgPath); err != nil {
		return cmdErr(fmt.Errorf("saving config: %w", err))
	}
	printSuccess(fmt.Sprintf("%s = %s", key, value))
	return nil
}

func getForgeConfigField(cfg *sharedconfig.Config, key string) (string, bool) {
	switch key {
	case "forge.domain":
		return cfg.Forge.Domain, true
	case "forge.data_dir":
		return cfg.Forge.DataDir, true
	case "forge.install_dir":
		return cfg.Forge.InstallDir, true
	case "forge.log_level":
		return cfg.Forge.LogLevel, true
	case "forge.version":
		return cfg.Forge.Version, true
	}
	return "", false
}

func setForgeConfigField(cfg *sharedconfig.Config, key, value string) bool {
	switch key {
	case "forge.domain":
		cfg.Forge.Domain = value
	case "forge.data_dir":
		cfg.Forge.DataDir = value
	case "forge.install_dir":
		cfg.Forge.InstallDir = value
	case "forge.log_level":
		cfg.Forge.LogLevel = value
	default:
		return false
	}
	return true
}
