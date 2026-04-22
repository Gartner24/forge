package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	outputFormat string
	verbose      bool
)

var rootCmd = &cobra.Command{
	Use:           "penforge",
	Short:         "PenForge -- automated security scanning",
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if isJSON() {
			printJSON(map[string]string{"error": err.Error()})
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "", "Output format (json)")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Verbose output")
}

func isJSON() bool {
	return outputFormat == "json"
}

func printJSON(v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

func cmdErr(err error) error {
	return err
}

func printSuccess(msg string) {
	if isJSON() {
		printJSON(map[string]any{"ok": true, "message": msg})
	} else {
		fmt.Println(msg)
	}
}
