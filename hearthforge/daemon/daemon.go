// Package daemon implements the HearthForge module interface.
// HearthForge manages developer SSH gateway and dev container lifecycle.
package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gartner24/forge/shared/module"
)

const moduleName = "hearthforge"
const moduleVersion = "0.1.0"

// HearthForge implements module.Module. It manages the SSH gateway container
// and validates that the dev-web Docker network exists.
type HearthForge struct {
	devNetwork    string
	gatewayImage  string
	composeFile   string
}

// New creates a HearthForge daemon instance.
func New(devNetwork, gatewayImage, composeFile string) *HearthForge {
	if devNetwork == "" {
		devNetwork = "dev-web"
	}
	if gatewayImage == "" {
		gatewayImage = "forge-gateway:latest"
	}
	return &HearthForge{
		devNetwork:   devNetwork,
		gatewayImage: gatewayImage,
		composeFile:  composeFile,
	}
}

func (h *HearthForge) Name() string    { return moduleName }
func (h *HearthForge) Version() string { return moduleVersion }

func (h *HearthForge) Status() module.Status {
	if !h.isGatewayRunning() {
		return module.StatusStopped
	}
	return module.StatusRunning
}

func (h *HearthForge) Start() error {
	if err := h.ensureNetwork(); err != nil {
		return fmt.Errorf("ensuring dev-web network: %w", err)
	}

	if h.isGatewayRunning() {
		return nil // already up, idempotent
	}

	if h.composeFile != "" {
		out, err := exec.Command("docker", "compose", "-f", h.composeFile, "up", "-d", "gateway").CombinedOutput()
		if err != nil {
			return fmt.Errorf("starting gateway: %w\n%s", err, out)
		}
		return nil
	}

	// No compose file — start the gateway container directly.
	out, err := exec.Command("docker", "start", "forge-gateway").CombinedOutput()
	if err != nil {
		return fmt.Errorf("starting gateway container: %w\n%s", err, out)
	}
	return nil
}

func (h *HearthForge) Stop() error {
	if h.composeFile != "" {
		out, err := exec.Command("docker", "compose", "-f", h.composeFile, "stop", "gateway").CombinedOutput()
		if err != nil {
			return fmt.Errorf("stopping gateway: %w\n%s", err, out)
		}
		return nil
	}

	out, err := exec.Command("docker", "stop", "forge-gateway").CombinedOutput()
	if err != nil {
		return fmt.Errorf("stopping gateway container: %w\n%s", err, out)
	}
	return nil
}

func (h *HearthForge) isGatewayRunning() bool {
	out, err := exec.Command("docker", "ps",
		"--filter", "name=forge-gateway",
		"--filter", "status=running",
		"--format", "{{.Names}}",
	).Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "forge-gateway")
}

func (h *HearthForge) ensureNetwork() error {
	out, err := exec.Command("docker", "network", "ls",
		"--filter", "name="+h.devNetwork,
		"--format", "{{.Name}}",
	).Output()
	if err != nil {
		return err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.TrimSpace(line) == h.devNetwork {
			return nil // already exists
		}
	}

	// Create the network.
	createOut, createErr := exec.Command("docker", "network", "create", h.devNetwork).CombinedOutput()
	if createErr != nil {
		return fmt.Errorf("creating network %s: %w\n%s", h.devNetwork, createErr, createOut)
	}
	fmt.Fprintf(os.Stderr, "Created Docker network: %s\n", h.devNetwork)
	return nil
}
