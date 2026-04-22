package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

var (
	outputFormat string
	skipConfirm  bool
	errorPrinted bool
)

var rootCmd = &cobra.Command{
	Use:           "sparkforge",
	Short:         "SparkForge -- notification orchestration for Forge",
	Version:       version,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE:          runServe,
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
	rootCmd.PersistentFlags().BoolVar(&skipConfirm, "yes", false, "Skip confirmation prompts")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(channelCmd)
	rootCmd.AddCommand(sendCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(tokenCmd)
	rootCmd.AddCommand(alertsCmd)
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

func newTabWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}
