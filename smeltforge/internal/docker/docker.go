package docker

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

const NetworkName = "smeltforge-net"
const CaddyContainer = "smeltforge-caddy"
const CaddyImage = "caddy:latest"

type Client struct {
	cli *client.Client
}

func New() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("connecting to Docker: %w", err)
	}
	return &Client{cli: cli}, nil
}

func (c *Client) Close() error {
	return c.cli.Close()
}

// EnsureNetwork creates the smeltforge Docker bridge network if absent.
func (c *Client) EnsureNetwork(ctx context.Context) error {
	networks, err := c.cli.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(filters.Arg("name", NetworkName)),
	})
	if err != nil {
		return fmt.Errorf("listing networks: %w", err)
	}
	for _, n := range networks {
		if n.Name == NetworkName {
			return nil
		}
	}
	_, err = c.cli.NetworkCreate(ctx, NetworkName, network.CreateOptions{Driver: "bridge"})
	return err
}

// PullImage pulls a Docker image, discarding progress output.
func (c *Client) PullImage(ctx context.Context, imageName string) error {
	reader, err := c.cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pulling image %s: %w", imageName, err)
	}
	defer reader.Close()
	_, err = io.Copy(io.Discard, reader)
	return err
}

// BuildImage builds a Docker image from a directory containing a Dockerfile.
func (c *Client) BuildImage(ctx context.Context, contextPath, tag string) error {
	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		errCh <- tarDir(contextPath, pw)
		pw.CloseWithError(<-errCh)
	}()

	resp, err := c.cli.ImageBuild(ctx, pr, dockertypes.ImageBuildOptions{
		Tags:       []string{tag},
		Dockerfile: "Dockerfile",
		Remove:     true,
	})
	if err != nil {
		return fmt.Errorf("building image: %w", err)
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var msg struct {
			Error string `json:"error"`
		}
		if err := decoder.Decode(&msg); err != nil {
			continue
		}
		if msg.Error != "" {
			return fmt.Errorf("build error: %s", msg.Error)
		}
	}
	return nil
}

type ContainerConfig struct {
	Name    string
	Image   string
	Env     []string // ["KEY=VALUE", ...]
	Network string
}

// CreateAndStart creates and starts a container. Returns the container ID.
func (c *Client) CreateAndStart(ctx context.Context, cfg ContainerConfig) (string, error) {
	if cfg.Network == "" {
		cfg.Network = NetworkName
	}

	containerCfg := &container.Config{
		Image: cfg.Image,
		Env:   cfg.Env,
	}
	hostCfg := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
	}
	netCfg := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			cfg.Network: {},
		},
	}

	resp, err := c.cli.ContainerCreate(ctx, containerCfg, hostCfg, netCfg, nil, cfg.Name)
	if err != nil {
		return "", fmt.Errorf("creating container %s: %w", cfg.Name, err)
	}
	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("starting container %s: %w", cfg.Name, err)
	}
	return resp.ID, nil
}

// Stop stops a container gracefully (10 s timeout).
func (c *Client) Stop(ctx context.Context, nameOrID string) error {
	timeout := 10
	err := c.cli.ContainerStop(ctx, nameOrID, container.StopOptions{Timeout: &timeout})
	if err != nil && client.IsErrNotFound(err) {
		return nil
	}
	return err
}

// Remove force-removes a container.
func (c *Client) Remove(ctx context.Context, nameOrID string) error {
	err := c.cli.ContainerRemove(ctx, nameOrID, container.RemoveOptions{Force: true})
	if err != nil && client.IsErrNotFound(err) {
		return nil
	}
	return err
}

// StopAndRemove stops then removes a container.
func (c *Client) StopAndRemove(ctx context.Context, nameOrID string) error {
	if err := c.Stop(ctx, nameOrID); err != nil {
		return err
	}
	return c.Remove(ctx, nameOrID)
}

// ContainerStatus describes a container's current state.
type ContainerStatus struct {
	ID    string
	Name  string
	Image string
	State string
}

// Status returns the container's state, or nil if it doesn't exist.
func (c *Client) Status(ctx context.Context, nameOrID string) (*ContainerStatus, error) {
	info, err := c.cli.ContainerInspect(ctx, nameOrID)
	if err != nil {
		if client.IsErrNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	name := strings.TrimPrefix(info.Name, "/")
	id := info.ID
	if len(id) > 12 {
		id = id[:12]
	}
	return &ContainerStatus{
		ID:    id,
		Name:  name,
		Image: info.Config.Image,
		State: info.State.Status,
	}, nil
}

// Logs returns container log output as a string.
func (c *Client) Logs(ctx context.Context, nameOrID, tail, since string) (string, error) {
	opts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Since:      since,
		Timestamps: true,
	}
	reader, err := c.cli.ContainerLogs(ctx, nameOrID, opts)
	if err != nil {
		if client.IsErrNotFound(err) {
			return "", fmt.Errorf("container %q not found", nameOrID)
		}
		return "", err
	}
	defer reader.Close()

	var out strings.Builder
	if _, err := stdcopy.StdCopy(&out, &out, reader); err != nil {
		return "", err
	}
	return out.String(), nil
}

// HealthCheck polls the container's health endpoint until healthy or timeout.
func (c *Client) HealthCheck(ctx context.Context, containerName, path string, port, timeoutSec, intervalSec int) error {
	if path == "" {
		path = "/health"
	}
	if intervalSec == 0 {
		intervalSec = 2
	}
	if timeoutSec == 0 {
		timeoutSec = 30
	}

	info, err := c.cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return fmt.Errorf("inspecting container: %w", err)
	}

	var containerIP string
	if ep, ok := info.NetworkSettings.Networks[NetworkName]; ok {
		containerIP = ep.IPAddress
	}
	if containerIP == "" {
		return fmt.Errorf("container %s not on network %s", containerName, NetworkName)
	}

	url := fmt.Sprintf("http://%s:%d%s", containerIP, port, path)
	httpClient := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)

	for time.Now().Before(deadline) {
		resp, err := httpClient.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 300 {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(intervalSec) * time.Second):
		}
	}
	return fmt.Errorf("health check timed out after %ds", timeoutSec)
}

// StartCaddy pulls (if needed) and starts the Caddy container.
func (c *Client) StartCaddy(ctx context.Context) (string, error) {
	_ = c.StopAndRemove(ctx, CaddyContainer)

	if err := c.PullImage(ctx, CaddyImage); err != nil {
		return "", fmt.Errorf("pulling caddy: %w", err)
	}

	containerCfg := &container.Config{
		Image: CaddyImage,
		ExposedPorts: nat.PortSet{
			"80/tcp":   {},
			"443/tcp":  {},
			"2019/tcp": {},
		},
	}
	hostCfg := &container.HostConfig{
		PortBindings: nat.PortMap{
			"80/tcp":  []nat.PortBinding{{HostPort: "80"}},
			"443/tcp": []nat.PortBinding{{HostPort: "443"}},
			"2019/tcp": []nat.PortBinding{{
				HostIP:   "127.0.0.1",
				HostPort: "2019",
			}},
		},
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
	}
	netCfg := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			NetworkName: {},
		},
	}

	resp, err := c.cli.ContainerCreate(ctx, containerCfg, hostCfg, netCfg, nil, CaddyContainer)
	if err != nil {
		return "", fmt.Errorf("creating caddy container: %w", err)
	}
	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("starting caddy: %w", err)
	}
	return resp.ID, nil
}

// ContainerName returns the canonical container name for a project.
func ContainerName(projectID string) string {
	return "smeltforge-" + projectID
}

// BlueContainerName returns the blue container name for blue-green deploys.
func BlueContainerName(projectID string) string {
	return "smeltforge-" + projectID + "-blue"
}

// GreenContainerName returns the green container name for blue-green deploys.
func GreenContainerName(projectID string) string {
	return "smeltforge-" + projectID + "-green"
}

// tarDir streams a tar archive of dir to w.
func tarDir(dir string, w io.Writer) error {
	tw := tar.NewWriter(w)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = rel
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
	if err != nil {
		return err
	}
	return tw.Close()
}
