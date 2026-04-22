package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gartner24/forge/shared/audit"
	"github.com/gartner24/forge/shared/secrets"
	"github.com/gartner24/forge/smeltforge/internal/caddy"
	"github.com/gartner24/forge/smeltforge/internal/docker"
	"github.com/gartner24/forge/smeltforge/internal/paths"
	"github.com/gartner24/forge/smeltforge/internal/registry"
)

// Result holds the outcome of a deploy operation.
type Result struct {
	ProjectID string
	Image     string
	Strategy  string
	Duration  time.Duration
	Trigger   string
	Success   bool
	Error     error
}

// Engine orchestrates deploys for a single project.
type Engine struct {
	reg    *registry.Registry
	docker *docker.Client
	caddy  *caddy.Client
	audit  *audit.Logger
	store  *secrets.Store
}

func NewEngine(reg *registry.Registry, d *docker.Client, caddyClient *caddy.Client) (*Engine, error) {
	auditPath, err := paths.AuditLog()
	if err != nil {
		return nil, err
	}
	al, err := audit.New(auditPath)
	if err != nil {
		return nil, fmt.Errorf("opening audit log: %w", err)
	}

	secretsPath, err := paths.SecretsFile()
	if err != nil {
		return nil, err
	}
	store, err := secrets.New(secretsPath)
	if err != nil {
		return nil, fmt.Errorf("opening secrets: %w", err)
	}

	return &Engine{
		reg:    reg,
		docker: d,
		caddy:  caddyClient,
		audit:  al,
		store:  store,
	}, nil
}

// Deploy runs a full deploy for the given project ID.
func (e *Engine) Deploy(ctx context.Context, projectID, trigger string) (*Result, error) {
	start := time.Now()

	p, err := e.reg.Get(projectID)
	if err != nil {
		return nil, err
	}

	imageName, err := e.resolveImage(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("resolving image: %w", err)
	}

	envVars, err := e.loadEnvVars(projectID)
	if err != nil {
		return nil, fmt.Errorf("loading env vars: %w", err)
	}

	strategy := p.Strategy
	if strategy == "" {
		strategy = "stop-start"
	}

	var deployErr error
	switch strategy {
	case "blue-green":
		deployErr = e.deployBlueGreen(ctx, p, imageName, envVars)
	default:
		deployErr = e.deployStopStart(ctx, p, imageName, envVars)
	}

	dur := time.Since(start)
	result := &Result{
		ProjectID: projectID,
		Image:     imageName,
		Strategy:  strategy,
		Duration:  dur,
		Trigger:   trigger,
		Success:   deployErr == nil,
		Error:     deployErr,
	}

	detail := fmt.Sprintf("image=%s strategy=%s trigger=%s duration=%s", imageName, strategy, trigger, dur.Round(time.Millisecond))
	auditAction := "deploy.success"
	if deployErr != nil {
		auditAction = "deploy.failure"
		detail += " error=" + deployErr.Error()
	}
	_ = e.audit.Write("smeltforge", auditAction, trigger, detail)

	return result, deployErr
}

// Rollback performs a stop-start of the previous image.
func (e *Engine) Rollback(ctx context.Context, projectID string) error {
	p, err := e.reg.Get(projectID)
	if err != nil {
		return err
	}

	if p.State.PreviousImage == "" {
		return fmt.Errorf("no previous image available for rollback")
	}

	envVars, err := e.loadEnvVars(projectID)
	if err != nil {
		return fmt.Errorf("loading env vars: %w", err)
	}

	prevImage := p.State.PreviousImage

	if err := e.deployStopStart(ctx, p, prevImage, envVars); err != nil {
		return fmt.Errorf("rollback deploy failed: %w", err)
	}

	_ = e.audit.Write("smeltforge", "rollback", "cli", fmt.Sprintf("project=%s image=%s", projectID, prevImage))
	return nil
}

// deployStopStart performs a stop-start deploy strategy.
func (e *Engine) deployStopStart(ctx context.Context, p *registry.Project, imageName string, envVars []string) error {
	containerName := docker.ContainerName(p.ID)

	// Save rollback reference before stopping.
	prevImage := p.State.Image
	prevID := p.State.ID

	// Always attempt removal: idempotent (no-op if absent), but prevents
	// "container name already in use" on first deploy after a registry wipe.
	_ = e.docker.StopAndRemove(ctx, containerName)

	// Start new container.
	newID, err := e.docker.CreateAndStart(ctx, docker.ContainerConfig{
		Name:  containerName,
		Image: imageName,
		Env:   envVars,
	})
	if err != nil {
		return err
	}

	// Update Caddy routing.
	upstreamDial := fmt.Sprintf("%s:%d", containerName, p.Port)
	if p.State.ID == "" {
		if err := e.caddy.AddRoute(p.ID, p.Domain, upstreamDial); err != nil {
			return fmt.Errorf("adding caddy route: %w", err)
		}
	} else {
		if err := e.caddy.UpdateUpstream(p.ID, upstreamDial); err != nil {
			return fmt.Errorf("updating caddy route: %w", err)
		}
	}

	// Persist updated state with rollback reference.
	p.State = registry.ContainerState{
		ID:            newID,
		Image:         imageName,
		PreviousID:    prevID,
		PreviousImage: prevImage,
	}
	return e.reg.Update(p)
}

// deployBlueGreen performs a zero-downtime blue-green deploy.
func (e *Engine) deployBlueGreen(ctx context.Context, p *registry.Project, imageName string, envVars []string) error {
	currentColor := p.State.Color
	if currentColor == "" {
		currentColor = "blue"
	}
	newColor := "green"
	if currentColor == "green" {
		newColor = "blue"
	}

	newName := blueGreenName(p.ID, newColor)
	oldName := blueGreenName(p.ID, currentColor)

	// Remove stale new-color container if leftover from failed deploy.
	_ = e.docker.StopAndRemove(ctx, newName)

	// Start new container.
	newID, err := e.docker.CreateAndStart(ctx, docker.ContainerConfig{
		Name:  newName,
		Image: imageName,
		Env:   envVars,
	})
	if err != nil {
		return fmt.Errorf("starting %s container: %w", newColor, err)
	}

	// Health check new container.
	hc := p.HealthCheck
	healthErr := e.docker.HealthCheck(ctx, newName, hc.Path, p.Port, hc.Timeout, hc.Interval)
	if healthErr != nil {
		// Health check failed: stop new container, keep old one.
		_ = e.docker.StopAndRemove(ctx, newName)
		notifySparkForge("smeltforge", fmt.Sprintf("blue-green health check failed for %s: %v -- rolled back", p.ID, healthErr), "HIGH")
		return fmt.Errorf("health check failed: %w", healthErr)
	}

	// Atomic Caddy route switch.
	upstreamDial := fmt.Sprintf("%s:%d", newName, p.Port)
	if p.State.ID == "" {
		if err := e.caddy.AddRoute(p.ID, p.Domain, upstreamDial); err != nil {
			_ = e.docker.StopAndRemove(ctx, newName)
			return fmt.Errorf("adding caddy route: %w", err)
		}
	} else {
		if err := e.caddy.UpdateUpstream(p.ID, upstreamDial); err != nil {
			_ = e.docker.StopAndRemove(ctx, newName)
			return fmt.Errorf("switching caddy route: %w", err)
		}
	}

	// Stop old container.
	prevImage := p.State.Image
	prevID := p.State.ID
	_ = e.docker.StopAndRemove(ctx, oldName)

	p.State = registry.ContainerState{
		ID:            newID,
		Image:         imageName,
		Color:         newColor,
		PreviousID:    prevID,
		PreviousImage: prevImage,
	}
	return e.reg.Update(p)
}

// resolveImage returns the image name to use, pulling or building as needed.
func (e *Engine) resolveImage(ctx context.Context, p *registry.Project) (string, error) {
	switch p.Source.Type {
	case "registry":
		if err := e.docker.PullImage(ctx, p.Source.Image); err != nil {
			return "", err
		}
		return p.Source.Image, nil

	case "git":
		wsPath, err := paths.Workspace(p.ID)
		if err != nil {
			return "", err
		}
		if err := syncGit(p.Source.Repo, p.Source.Branch, wsPath); err != nil {
			return "", fmt.Errorf("git sync: %w", err)
		}
		tag := "smeltforge-" + p.ID + ":latest"
		if err := e.docker.BuildImage(ctx, wsPath, tag); err != nil {
			return "", fmt.Errorf("docker build: %w", err)
		}
		return tag, nil

	case "local":
		tag := "smeltforge-" + p.ID + ":latest"
		if err := e.docker.BuildImage(ctx, p.Source.Path, tag); err != nil {
			return "", fmt.Errorf("docker build: %w", err)
		}
		return tag, nil

	default:
		return "", fmt.Errorf("unknown source type %q", p.Source.Type)
	}
}

// loadEnvVars reads env vars from secrets for a project.
// Keys stored as smeltforge.<project>.<KEY>; internal keys are prefixed with _.
func (e *Engine) loadEnvVars(projectID string) ([]string, error) {
	prefix := "smeltforge." + projectID + "."
	allKeys, err := e.store.List()
	if err != nil {
		return nil, err
	}

	var envVars []string
	for _, k := range allKeys {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		key := strings.TrimPrefix(k, prefix)
		if strings.HasPrefix(key, "_") {
			continue // internal smeltforge secret
		}
		val, err := e.store.Get(k)
		if err != nil {
			continue
		}
		envVars = append(envVars, key+"="+val)
	}
	return envVars, nil
}

func blueGreenName(projectID, color string) string {
	return "smeltforge-" + projectID + "-" + color
}

// syncGit clones or pulls the repo at the given branch into wsPath.
func syncGit(repo, branch, wsPath string) error {
	if branch == "" {
		branch = "main"
	}
	if _, err := os.Stat(filepath.Join(wsPath, ".git")); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(wsPath), 0755); err != nil {
			return err
		}
		return run("git", "clone", "--branch", branch, "--depth", "1", repo, wsPath)
	}
	if err := run("git", "-C", wsPath, "fetch", "origin", branch); err != nil {
		return err
	}
	return run("git", "-C", wsPath, "reset", "--hard", "origin/"+branch)
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// notifySparkForge sends a notification to SparkForge if available.
// This is a best-effort no-op if SparkForge is not installed.
func notifySparkForge(module, message, priority string) {
	// SparkForge integration: call its API if installed.
	// No-op if not configured.
}
