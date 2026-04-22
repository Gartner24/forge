package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/gartner24/forge/core/internal/paths"
	"github.com/gartner24/forge/shared/config"
	"github.com/spf13/cobra"
)

var version = "0.1.0" // overridden by -ldflags

var (
	outputFormat string
	verbose      bool
	skipConfirm  bool
	noBanner     bool
	nodeTarget   string
	errorPrinted bool
)

var knownModules = map[string]bool{
	"smeltforge":  true,
	"watchforge":  true,
	"sparkforge":  true,
	"fluxforge":   true,
	"hearthforge": true,
	"penforge":    true,
}

var rootCmd = &cobra.Command{
	Use:           "forge",
	Short:         "Forge -- self-hosted infrastructure suite",
	Version:       version,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if !errorPrinted {
			if isJSON() {
				printJSON(map[string]string{"error": err.Error()})
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
		}
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "", "Output format (json)")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Show detailed output")
	rootCmd.PersistentFlags().BoolVar(&skipConfirm, "yes", false, "Skip confirmation prompts")
	rootCmd.PersistentFlags().BoolVar(&noBanner, "no-banner", false, "Suppress the active alert banner")
	rootCmd.PersistentFlags().StringVar(&nodeTarget, "node", "", "Target a specific FluxForge mesh node (requires FluxForge)")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(secretsCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(hearthforgeCmd)
	rootCmd.AddCommand(fluxforgeCmd)
}

func isJSON() bool {
	return outputFormat == "json"
}

func printJSON(v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

func printSuccess(msg string) {
	if isJSON() {
		printJSON(map[string]any{"ok": true, "message": msg})
	} else {
		fmt.Println(msg)
	}
}

// cmdErr prints the error in the appropriate format and returns it so the
// command can return non-nil (causing os.Exit(1) in Execute).
func cmdErr(err error) error {
	errorPrinted = true
	if isJSON() {
		printJSON(map[string]string{"error": err.Error()})
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	return err
}

// mustConfirm prompts the user unless --yes is set. Returns (false, err) when
// in JSON mode without --yes (JSON callers must pass --yes explicitly).
func mustConfirm(prompt string) (bool, error) {
	if skipConfirm {
		return true, nil
	}
	if isJSON() {
		return false, fmt.Errorf("destructive operation requires --yes when using --output json")
	}
	fmt.Printf("%s [y/N] ", prompt)
	var answer string
	fmt.Scanln(&answer)
	return answer == "y" || answer == "Y", nil
}

// requireInit loads config and returns an error if forge has not been initialized.
func requireInit() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	if cfg.Forge.Domain == "" {
		return nil, fmt.Errorf("forge is not initialized -- run: forge init --domain <domain>")
	}
	return cfg, nil
}

func effectiveInstallDir(cfg *config.Config) string {
	if cfg.Forge.InstallDir != "" {
		return cfg.Forge.InstallDir
	}
	dir, _ := paths.BinDir()
	return dir
}

func effectiveDataDir(cfg *config.Config) string {
	if cfg.Forge.DataDir != "" {
		return cfg.Forge.DataDir
	}
	dir, _ := paths.DataDir()
	return dir
}

// newTabWriter returns a tabwriter suitable for CLI table output.
func newTabWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

// formatSince formats a time as a human-readable "time ago" string.
func formatSince(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh%dm", h, m)
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
