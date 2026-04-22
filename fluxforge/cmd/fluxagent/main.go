package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gartner24/forge/fluxforge/internal/agent"
	"github.com/gartner24/forge/fluxforge/internal/mesh"
	"github.com/gartner24/forge/fluxforge/internal/store"
	"github.com/gartner24/forge/fluxforge/internal/wg"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var version = "0.1.0"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(version)
		return
	}
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fluxagent: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("finding home dir: %w", err)
	}
	dataDir := filepath.Join(homeDir, ".forge", "fluxforge")

	stateStore, err := store.New[mesh.LocalState](filepath.Join(dataDir, "state.json"))
	if err != nil {
		return err
	}
	state, err := stateStore.Read()
	if err != nil {
		return fmt.Errorf("reading state: %w", err)
	}
	if state == nil {
		return fmt.Errorf("not joined — run: forge fluxforge join")
	}
	if state.IsController {
		return fmt.Errorf("this node is a controller — use fluxcontroller instead")
	}

	privKey, err := wgtypes.ParseKey(state.PrivateKey)
	if err != nil {
		return fmt.Errorf("parsing private key: %w", err)
	}

	wgMgr, err := wg.New(mesh.InterfaceName, state.MeshIP, privKey, mesh.WireGuardPort)
	if err != nil {
		return fmt.Errorf("setting up WireGuard: %w", err)
	}
	defer wgMgr.Close()

	ag := agent.New(*state, wgMgr)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	fmt.Fprintf(os.Stderr, "fluxagent: running, mesh IP %s, controller %s\n",
		state.MeshIP, state.ControllerAddr)
	ag.Run(ctx)
	return nil
}
