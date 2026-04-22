package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gartner24/forge/hearthforge/daemon"
)

var version = "0.1.0" // overridden by -ldflags

func main() {
	showVersion := flag.Bool("version", false, "Print version and exit")
	devNetwork := flag.String("network", "dev-web", "Docker network for dev containers")
	gatewayImage := flag.String("gateway-image", "forge-gateway:latest", "Docker image for the SSH gateway")
	composeFile := flag.String("compose", defaultComposeFile(), "Path to gateway docker-compose.yml")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	d := daemon.New(*devNetwork, *gatewayImage, *composeFile)

	if err := d.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "hearthforge: start failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "hearthforge %s started (gateway: %s)\n", version, d.Name())

	// Wait for SIGTERM or SIGINT.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	fmt.Fprintln(os.Stderr, "hearthforge: shutting down...")

	// Give Stop up to 10 seconds.
	done := make(chan error, 1)
	go func() { done <- d.Stop() }()

	select {
	case err := <-done:
		if err != nil {
			fmt.Fprintf(os.Stderr, "hearthforge: stop error: %v\n", err)
			os.Exit(1)
		}
	case <-time.After(10 * time.Second):
		fmt.Fprintln(os.Stderr, "hearthforge: stop timed out, forcing exit")
		os.Exit(1)
	}
}

func defaultComposeFile() string {
	// Derive from the binary location: <binary-dir>/../compose.yml
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), "..", "compose.yml")
}
