package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gartner24/forge/fluxforge/internal/mesh"
	"github.com/gartner24/forge/fluxforge/internal/wg"
)

// Agent registers with the controller and keeps the local WireGuard config
// in sync with the peer registry. Heartbeats run every HeartbeatInterval.
// If the controller goes down, existing WireGuard tunnels stay up.
type Agent struct {
	state          mesh.LocalState
	wgMgr          *wg.Manager
	registryVersion int64
	client         *http.Client
}

func New(state mesh.LocalState, wgMgr *wg.Manager) *Agent {
	return &Agent{
		state: state,
		wgMgr: wgMgr,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Run starts the heartbeat loop. It blocks until ctx is cancelled.
func (a *Agent) Run(ctx context.Context) {
	// Initial peer sync before entering the heartbeat loop.
	a.syncPeers()

	ticker := time.NewTicker(mesh.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.heartbeat()
		}
	}
}

func (a *Agent) heartbeat() {
	body, _ := json.Marshal(map[string]any{
		"node_id":          a.state.NodeID,
		"registry_version": a.registryVersion,
	})

	req, err := http.NewRequest("POST",
		fmt.Sprintf("http://%s/api/heartbeat", a.state.ControllerAddr),
		bytes.NewReader(body),
	)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+a.state.AuthToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		// Controller unreachable — existing tunnels stay up, just keep trying.
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var hbResp struct {
		RegistryVersion int64           `json:"registry_version"`
		Peers           []mesh.NodeInfo `json:"peers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hbResp); err != nil {
		return
	}

	if hbResp.RegistryVersion != a.registryVersion {
		a.registryVersion = hbResp.RegistryVersion
		if len(hbResp.Peers) > 0 {
			a.applyPeers(hbResp.Peers)
		}
	}
}

func (a *Agent) syncPeers() {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("http://%s/api/peers", a.state.ControllerAddr),
		nil,
	)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+a.state.AuthToken)

	resp, err := a.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var pResp struct {
		Version int64           `json:"version"`
		Peers   []mesh.NodeInfo `json:"peers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pResp); err != nil {
		return
	}

	a.registryVersion = pResp.Version
	a.applyPeers(pResp.Peers)
}

func (a *Agent) applyPeers(peers []mesh.NodeInfo) {
	if a.wgMgr == nil {
		return
	}
	configs := make([]wg.PeerConfig, 0, len(peers))
	for _, p := range peers {
		if p.ID == a.state.NodeID {
			continue
		}
		configs = append(configs, wg.PeerConfig{
			PublicKey: p.PublicKey,
			AllowedIP: p.MeshIP + "/32",
			Endpoint:  p.Endpoint,
		})
	}
	a.wgMgr.SetPeers(configs) //nolint:errcheck
}
