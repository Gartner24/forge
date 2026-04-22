package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gartner24/forge/fluxforge/internal/agent"
	"github.com/gartner24/forge/fluxforge/internal/controller"
	"github.com/gartner24/forge/fluxforge/internal/mesh"
	"github.com/gartner24/forge/fluxforge/internal/store"
	"github.com/gartner24/forge/fluxforge/internal/wg"
	"github.com/gartner24/forge/shared/audit"
	sharedconfig "github.com/gartner24/forge/shared/config"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func fluxDataDir() string {
	cfg, err := sharedconfig.Load()
	if err == nil && cfg != nil {
		if m, ok := cfg.Modules["fluxforge"]; ok && m.DataDir != "" {
			return m.DataDir
		}
		if cfg.Forge.DataDir != "" {
			return filepath.Join(cfg.Forge.DataDir, "fluxforge")
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".forge", "data", "fluxforge")
	}
	return "/opt/data/fluxforge"
}

var version = "0.1.0"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(version)
		return
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fluxcontroller: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	dataDir := fluxDataDir()

	stateStore, err := store.New[mesh.LocalState](filepath.Join(dataDir, "state.json"))
	if err != nil {
		return err
	}
	state, err := stateStore.Read()
	if err != nil {
		return fmt.Errorf("reading state: %w", err)
	}
	if state == nil {
		return fmt.Errorf("not initialized — run: forge fluxforge init")
	}
	if !state.IsController {
		return fmt.Errorf("this node is not a controller — run: forge fluxforge join")
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

	reg, err := controller.NewRegistry(filepath.Join(dataDir, "registry.json"))
	if err != nil {
		return fmt.Errorf("loading registry: %w", err)
	}

	auditLog, err := audit.New(filepath.Join(dataDir, "logs", "fluxforge", "audit.log"))
	if err != nil {
		return fmt.Errorf("opening audit log: %w", err)
	}

	srv := controller.NewServer(reg, wgMgr, auditLog, state.NodeID)

	// Do an initial WireGuard sync so the controller's own interface has all peers.
	srv.SyncWireGuard()

	// Run the agent loop alongside the controller so this node participates in the mesh.
	agentState := mesh.LocalState{
		NodeID:         state.NodeID,
		MeshIP:         state.MeshIP,
		AuthToken:      state.AuthToken,
		ControllerAddr: state.ControllerAddr,
	}
	ag := agent.New(agentState, nil) // wgMgr already managed by server's syncWireGuard

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	go ag.Run(ctx)

	addr := state.ControllerAddr
	if addr == "" {
		addr = fmt.Sprintf(":%d", mesh.ControllerPort)
	} else {
		// Bind on all interfaces, not just the mesh IP.
		_, port, err := splitHostPort(addr)
		if err != nil {
			port = fmt.Sprintf("%d", mesh.ControllerPort)
		}
		addr = ":" + port
	}

	httpSrv := &http.Server{Addr: addr, Handler: srv.Handler()}

	go func() {
		<-ctx.Done()
		httpSrv.Shutdown(context.Background()) //nolint:errcheck
	}()

	fmt.Fprintf(os.Stderr, "fluxcontroller: listening on %s\n", addr)
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func splitHostPort(addr string) (host, port string, err error) {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i], addr[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("no port in %q", addr)
}
