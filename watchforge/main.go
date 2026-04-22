package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gartner24/forge/shared/audit"
	"github.com/gartner24/forge/shared/config"
	"github.com/gartner24/forge/watchforge/internal/api"
	"github.com/gartner24/forge/watchforge/internal/registry"
	"github.com/gartner24/forge/watchforge/internal/scheduler"
)

const version = "0.1.0"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Println(version)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("watchforge: loading config: %v", err)
	}

	dataDir := moduleDataDir(cfg)

	registryDir := filepath.Join(dataDir, "registry")
	statusDir := filepath.Join(dataDir, "data", "status")
	auditPath := filepath.Join(dataDir, "data", "audit.log")

	reg, err := registry.New(registryDir)
	if err != nil {
		log.Fatalf("watchforge: initialising registry: %v", err)
	}

	auditLog, err := audit.New(auditPath)
	if err != nil {
		log.Fatalf("watchforge: initialising audit log: %v", err)
	}

	apiAddr := "127.0.0.1:7771"
	if m, ok := cfg.Modules["watchforge"]; ok && m.APIAddr != "" {
		apiAddr = m.APIAddr
	}

	// Notify token for SparkForge. Empty string = no auth (SparkForge not installed).
	notifyToken := os.Getenv("WATCHFORGE_NOTIFY_TOKEN")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	sched := scheduler.New(reg, statusDir, notifyToken, auditLog)

	apiSrv := api.New(apiAddr, reg, sched, ctx)
	if err := apiSrv.Start(); err != nil {
		log.Fatalf("watchforge: starting API server: %v", err)
	}

	if err := sched.Start(ctx); err != nil {
		log.Fatalf("watchforge: starting scheduler: %v", err)
	}

	log.Printf("watchforge %s started (api=%s)", version, apiAddr)
	_ = auditLog.Write("watchforge", "daemon.start", "system", fmt.Sprintf("version=%s api=%s", version, apiAddr))

	<-ctx.Done()

	apiSrv.Stop()
	_ = auditLog.Write("watchforge", "daemon.stop", "system", "")
	log.Println("watchforge: stopped")
}

func moduleDataDir(cfg *config.Config) string {
	if m, ok := cfg.Modules["watchforge"]; ok && m.DataDir != "" {
		return m.DataDir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".forge", "data", "watchforge")
}
