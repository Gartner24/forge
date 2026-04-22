package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

const dockerSocket = "/var/run/docker.sock"

type DockerChecker struct {
	ContainerName string
}

func (c *DockerChecker) Check() Result {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", dockerSocket)
			},
		},
	}

	url := fmt.Sprintf("http://localhost/containers/%s/json", c.ContainerName)
	resp, err := client.Get(url)
	if err != nil {
		return Result{OK: false, Reason: fmt.Sprintf("docker API: %v", err), CheckedAt: time.Now()}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return Result{OK: false, Reason: "container not found", CheckedAt: time.Now()}
	}
	if resp.StatusCode != http.StatusOK {
		return Result{OK: false, Reason: fmt.Sprintf("docker API status %d", resp.StatusCode), CheckedAt: time.Now()}
	}

	var info struct {
		State struct {
			Running bool   `json:"Running"`
			Health  *struct {
				Status string `json:"Status"`
			} `json:"Health"`
		} `json:"State"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return Result{OK: false, Reason: fmt.Sprintf("parsing container info: %v", err), CheckedAt: time.Now()}
	}

	if !info.State.Running {
		return Result{OK: false, Reason: "container is not running", CheckedAt: time.Now()}
	}
	if info.State.Health != nil && info.State.Health.Status == "unhealthy" {
		return Result{OK: false, Reason: "container healthcheck is unhealthy", CheckedAt: time.Now()}
	}

	return Result{OK: true, CheckedAt: time.Now()}
}
