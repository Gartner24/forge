package engine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// dockerClient returns a Docker client configured from the environment.
func dockerClient() (*dockerclient.Client, error) {
	return dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
}

// pullImage pulls a Docker image, discarding progress output.
func pullImage(ctx context.Context, cli *dockerclient.Client, ref string) error {
	rc, err := cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pulling %s: %w", ref, err)
	}
	defer rc.Close()
	io.Copy(io.Discard, rc)
	return nil
}

// containerOpts holds options for running a scan container.
type containerOpts struct {
	image     string
	cmd       []string
	env       []string
	mounts    []mount.Mount
	networkID string
	name      string
}

// runContainer creates, starts, waits for, and removes a container.
// Returns stdout output. The container is always removed within 60 s of completion.
func runContainer(ctx context.Context, cli *dockerclient.Client, opts containerOpts) ([]byte, error) {
	cfg := &container.Config{
		Image: opts.image,
		Cmd:   opts.cmd,
		Env:   opts.env,
	}

	hostCfg := &container.HostConfig{
		Mounts:     opts.mounts,
		AutoRemove: false,
		// No privileged access, no capabilities beyond defaults.
	}

	netCfg := &network.NetworkingConfig{}
	if opts.networkID != "" {
		netCfg.EndpointsConfig = map[string]*network.EndpointSettings{
			opts.networkID: {},
		}
	}

	resp, err := cli.ContainerCreate(ctx, cfg, hostCfg, netCfg, nil, opts.name)
	if err != nil {
		return nil, fmt.Errorf("creating container %s: %w", opts.name, err)
	}

	containerID := resp.ID

	// Ensure the container is removed within 60 s regardless of outcome.
	defer func() {
		rmCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		cli.ContainerStop(rmCtx, containerID, container.StopOptions{})
		cli.ContainerRemove(rmCtx, containerID, container.RemoveOptions{Force: true})
	}()

	if err := cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("starting container %s: %w", opts.name, err)
	}

	waitCh, errCh := cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return nil, fmt.Errorf("waiting for container %s: %w", opts.name, err)
		}
	case result := <-waitCh:
		if result.Error != nil {
			return nil, fmt.Errorf("container %s exited with error: %s", opts.name, result.Error.Message)
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	logOpts := container.LogsOptions{ShowStdout: true, ShowStderr: false}
	logsRC, err := cli.ContainerLogs(ctx, containerID, logOpts)
	if err != nil {
		return nil, fmt.Errorf("reading logs for %s: %w", opts.name, err)
	}
	defer logsRC.Close()

	var stdout bytes.Buffer
	stdcopy.StdCopy(&stdout, io.Discard, logsRC)
	return stdout.Bytes(), nil
}

// createScanNetwork creates an isolated Docker bridge network for a scan.
// Returns the network ID.
func createScanNetwork(ctx context.Context, cli *dockerclient.Client, scanID string) (string, error) {
	name := "penforge-scan-" + scanID
	resp, err := cli.NetworkCreate(ctx, name, network.CreateOptions{
		Driver:     "bridge",
		Attachable: false,
	})
	if err != nil {
		return "", fmt.Errorf("creating scan network: %w", err)
	}
	return resp.ID, nil
}

// removeScanNetwork removes the scan network.
func removeScanNetwork(cli *dockerclient.Client, networkID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli.NetworkRemove(ctx, networkID)
}

// tempDir creates a temporary directory for engine output, returns the path and a cleanup func.
func tempDir(prefix string) (string, func(), error) {
	dir, err := os.MkdirTemp("", "penforge-"+prefix+"-")
	if err != nil {
		return "", nil, err
	}
	return dir, func() { os.RemoveAll(dir) }, nil
}
