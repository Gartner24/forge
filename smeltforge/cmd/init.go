package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/gartner24/forge/smeltforge/internal/paths"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise SmeltForge (create directories, start Caddy)",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	dataDir, err := paths.DataDir()
	if err != nil {
		return cmdErr(err)
	}

	for _, sub := range []string{"registry", "workspaces", "caddy/config"} {
		if err := os.MkdirAll(fmt.Sprintf("%s/%s", dataDir, sub), 0755); err != nil {
			return cmdErr(fmt.Errorf("creating %s: %w", sub, err))
		}
	}

	regPath, err := paths.RegistryFile()
	if err != nil {
		return cmdErr(err)
	}
	if _, err := os.Stat(regPath); os.IsNotExist(err) {
		if err := os.WriteFile(regPath, []byte("[]"), 0644); err != nil {
			return cmdErr(fmt.Errorf("creating registry: %w", err))
		}
	}

	d, err := newDocker()
	if err != nil {
		return cmdErr(err)
	}
	defer d.Close()

	ctx := context.Background()

	fmt.Println("Creating Docker network smeltforge-net...")
	if err := d.EnsureNetwork(ctx); err != nil {
		return cmdErr(fmt.Errorf("creating network: %w", err))
	}

	fmt.Println("Pulling and starting Caddy...")
	if _, err := d.StartCaddy(ctx); err != nil {
		return cmdErr(fmt.Errorf("starting caddy: %w", err))
	}

	// Wait for Caddy admin API to be ready.
	caddyClient := newCaddy()
	for i := 0; i < 15; i++ {
		if err := caddyClient.Ping(); err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err := caddyClient.Ping(); err != nil {
		return cmdErr(fmt.Errorf("caddy admin API not reachable: %w", err))
	}

	if err := caddyClient.EnsureServer(); err != nil {
		return cmdErr(fmt.Errorf("configuring caddy: %w", err))
	}

	printSuccess("smeltforge initialised")
	return nil
}
