package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var penforgeCmd = &cobra.Command{
	Use:                "penforge",
	Short:              "Security scanning (PenForge)",
	Long:               "Delegates to the penforge binary. Install PenForge with: forge install penforge && forge start penforge",
	DisableFlagParsing: true,
	RunE:               runPenforge,
}

func init() {
	rootCmd.AddCommand(penforgeCmd)
}

func runPenforge(cmd *cobra.Command, args []string) error {
	binPath, err := resolvePenforgeBin()
	if err != nil {
		return cmdErr(err)
	}

	c := exec.Command(binPath, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	if err := c.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}

func resolvePenforgeBin() (string, error) {
	cfg, _ := requireInit()

	if cfg != nil {
		binPath := filepath.Join(effectiveInstallDir(cfg), "penforge")
		if _, err := os.Stat(binPath); err == nil {
			return binPath, nil
		}
	}

	// Fall back to PATH.
	if p, err := exec.LookPath("penforge"); err == nil {
		return p, nil
	}

	return "", fmt.Errorf("penforge binary not found -- build it with: cd penforge && go build -o ~/.forge/bin/penforge .")
}
