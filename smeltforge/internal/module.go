package internal

import (
	"context"
	"fmt"
	"net/http"

	sharedmodule "github.com/gartner24/forge/shared/module"
	sharedsecrets "github.com/gartner24/forge/shared/secrets"
	"github.com/gartner24/forge/smeltforge/internal/caddy"
	"github.com/gartner24/forge/smeltforge/internal/deploy"
	"github.com/gartner24/forge/smeltforge/internal/docker"
	"github.com/gartner24/forge/smeltforge/internal/paths"
	"github.com/gartner24/forge/smeltforge/internal/registry"
	"github.com/gartner24/forge/smeltforge/internal/server"
)

// SmeltForge implements the shared Module interface.
type SmeltForge struct {
	cancel  context.CancelFunc
	srv     *http.Server
	status  sharedmodule.Status
}

var _ sharedmodule.Module = (*SmeltForge)(nil)

func (m *SmeltForge) Name() string    { return "smeltforge" }
func (m *SmeltForge) Version() string { return "0.1.0" }

func (m *SmeltForge) Status() sharedmodule.Status {
	if m.cancel == nil {
		return sharedmodule.StatusStopped
	}
	return m.status
}

func (m *SmeltForge) Start() error {
	regPath, err := paths.RegistryFile()
	if err != nil {
		return err
	}
	reg, err := registry.Load(regPath)
	if err != nil {
		return fmt.Errorf("loading registry: %w", err)
	}

	d, err := docker.New()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}

	caddyClient := caddy.New("")
	engine, err := deploy.NewEngine(reg, d, caddyClient)
	if err != nil {
		return fmt.Errorf("deploy engine: %w", err)
	}

	secretsPath, err := paths.SecretsFile()
	if err != nil {
		return err
	}
	store, err := sharedsecrets.New(secretsPath)
	if err != nil {
		return fmt.Errorf("secrets store: %w", err)
	}

	deployFn := func(ctx context.Context, projectID, trigger string) (*deploy.Result, error) {
		return engine.Deploy(ctx, projectID, trigger)
	}
	secretLookup := func(key string) (string, error) {
		return store.Get(key)
	}

	srv := server.New(reg, deployFn, secretLookup)

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.status = sharedmodule.StatusRunning

	httpSrv := &http.Server{Addr: ":9320", Handler: srv}
	m.srv = httpSrv

	go srv.RunWorker(ctx)
	go srv.RunPoller(ctx)
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			m.status = sharedmodule.StatusError
		}
	}()

	return nil
}

func (m *SmeltForge) Stop() error {
	if m.cancel != nil {
		m.cancel()
	}
	if m.srv != nil {
		return m.srv.Shutdown(context.Background())
	}
	m.status = sharedmodule.StatusStopped
	return nil
}
