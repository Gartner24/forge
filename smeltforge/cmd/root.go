package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gartner24/forge/shared/secrets"
	"github.com/gartner24/forge/smeltforge/internal/caddy"
	"github.com/gartner24/forge/smeltforge/internal/docker"
	"github.com/gartner24/forge/smeltforge/internal/paths"
	"github.com/gartner24/forge/smeltforge/internal/registry"
)

var version = "0.1.0"

var (
	outputFormat string
	projectID    string
	yesFlag      bool
	errorPrinted bool
)

var rootCmd = &cobra.Command{
	Use:           "smeltforge",
	Short:         "SmeltForge -- Docker deployment platform",
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
	rootCmd.PersistentFlags().BoolVar(&yesFlag, "yes", false, "Skip confirmation prompts")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(envCmd)
	rootCmd.AddCommand(webhookCmd)
	rootCmd.AddCommand(tokenCmd)
	rootCmd.AddCommand(pollingCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(deployKeyCmd)
}

func isJSON() bool { return outputFormat == "json" }

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

func cmdErr(err error) error {
	errorPrinted = true
	if isJSON() {
		printJSON(map[string]string{"error": err.Error()})
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	return err
}

func mustConfirm(prompt string) (bool, error) {
	if yesFlag {
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

func loadRegistry() (*registry.Registry, error) {
	regPath, err := paths.RegistryFile()
	if err != nil {
		return nil, err
	}
	return registry.Load(regPath)
}

func loadSecrets() (*secrets.Store, error) {
	p, err := paths.SecretsFile()
	if err != nil {
		return nil, err
	}
	return secrets.New(p)
}

func newDocker() (*docker.Client, error) {
	return docker.New()
}

func newCaddy() *caddy.Client {
	return caddy.New("")
}
