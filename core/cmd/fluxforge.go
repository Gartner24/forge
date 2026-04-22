package cmd

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gartner24/forge/core/internal/paths"
	"github.com/spf13/cobra"
)

// fluxforgeState mirrors fluxforge/internal/mesh.LocalState.
// Defined here to avoid a cross-module dependency.
type fluxforgeState struct {
	NodeID         string `json:"node_id"`
	MeshIP         string `json:"mesh_ip"`
	PrivateKey     string `json:"private_key"`
	PublicKey      string `json:"public_key"`
	AuthToken      string `json:"auth_token"`
	ControllerAddr string `json:"controller_addr"`
	IsController   bool   `json:"is_controller"`
}

const fluxforgeControllerPort = 7777
const fluxforgeWireGuardPort = 51820

var fluxforgeCmd = &cobra.Command{
	Use:   "fluxforge",
	Short: "Manage the WireGuard mesh network",
}

func init() {
	// init
	ffInitCmd.Flags().IntVar(&ffInitPort, "port", fluxforgeControllerPort, "Controller API port")
	fluxforgeCmd.AddCommand(ffInitCmd)

	// join
	ffJoinCmd.Flags().StringVar(&ffJoinController, "controller", "", "Controller address (host:port)")
	ffJoinCmd.Flags().StringVar(&ffJoinToken, "token", "", "Join token")
	ffJoinCmd.Flags().StringVar(&ffJoinEndpoint, "endpoint", "", "This node's public WireGuard endpoint (host:port); auto-detected if empty")
	ffJoinCmd.MarkFlagRequired("controller")
	ffJoinCmd.MarkFlagRequired("token")
	fluxforgeCmd.AddCommand(ffJoinCmd)

	// status / nodes
	fluxforgeCmd.AddCommand(ffStatusCmd)
	fluxforgeCmd.AddCommand(ffNodesCmd)

	// token subcommand
	ffTokenCmd.AddCommand(ffTokenCreateCmd)
	ffTokenCmd.AddCommand(ffTokenListCmd)
	ffTokenRevokeCmd.Args = cobra.ExactArgs(1)
	ffTokenCmd.AddCommand(ffTokenRevokeCmd)
	fluxforgeCmd.AddCommand(ffTokenCmd)

	// revoke
	ffRevokeCmd.Args = cobra.ExactArgs(1)
	fluxforgeCmd.AddCommand(ffRevokeCmd)

	// add-admin / remove-admin
	ffAddAdminCmd.Args = cobra.ExactArgs(1)
	ffRemoveAdminCmd.Args = cobra.ExactArgs(1)
	fluxforgeCmd.AddCommand(ffAddAdminCmd)
	fluxforgeCmd.AddCommand(ffRemoveAdminCmd)

	// set-controller
	ffSetControllerCmd.Args = cobra.ExactArgs(1)
	fluxforgeCmd.AddCommand(ffSetControllerCmd)

	// ping
	ffPingCmd.Args = cobra.ExactArgs(1)
	fluxforgeCmd.AddCommand(ffPingCmd)
}

// ---- init ----

var ffInitPort int

var ffInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise the mesh controller on this node",
	RunE:  runFluxforgeInit,
}

func runFluxforgeInit(cmd *cobra.Command, args []string) error {
	dataDir, err := fluxforgeDataDir()
	if err != nil {
		return cmdErr(err)
	}

	statePath := filepath.Join(dataDir, "state.json")
	if _, err := os.Stat(statePath); err == nil {
		printSuccess("fluxforge already initialised on this node")
		return nil
	}

	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return cmdErr(fmt.Errorf("creating data dir: %w", err))
	}

	privKey, pubKey, err := generateWireGuardKeys()
	if err != nil {
		return cmdErr(fmt.Errorf("generating WireGuard keys: %w", err))
	}

	nodeID, err := generateID()
	if err != nil {
		return cmdErr(fmt.Errorf("generating node ID: %w", err))
	}

	ownerToken, err := generateToken()
	if err != nil {
		return cmdErr(fmt.Errorf("generating owner token: %w", err))
	}

	controllerAddr := fmt.Sprintf("localhost:%d", ffInitPort)
	meshIP := "10.99.1.1"

	state := fluxforgeState{
		NodeID:         nodeID,
		MeshIP:         meshIP,
		PrivateKey:     privKey,
		PublicKey:      pubKey,
		AuthToken:      ownerToken,
		ControllerAddr: controllerAddr,
		IsController:   true,
	}
	if err := writeJSON(statePath, state); err != nil {
		return cmdErr(fmt.Errorf("writing state: %w", err))
	}

	// Write the initial registry with the owner node and a first join token.
	joinToken, err := generateSecureToken()
	if err != nil {
		return cmdErr(fmt.Errorf("generating join token: %w", err))
	}
	now := time.Now().UTC()
	registry := map[string]any{
		"nodes": []map[string]any{{
			"id":         nodeID,
			"mesh_ip":    meshIP,
			"public_key": pubKey,
			"endpoint":   "",
			"role":       "owner",
			"auth_token": ownerToken,
			"last_seen":  now,
			"created_at": now,
		}},
		"tokens": []map[string]any{{
			"value":      joinToken,
			"created_at": now,
			"expires_at": now.Add(24 * time.Hour),
			"used":       false,
		}},
		"next_ip_index":      2,
		"controller_node_id": nodeID,
		"version":            1,
	}
	if err := writeJSON(filepath.Join(dataDir, "registry.json"), registry); err != nil {
		return cmdErr(fmt.Errorf("writing registry: %w", err))
	}

	// Start the fluxcontroller daemon.
	if err := startDaemon("fluxcontroller"); err != nil {
		// Non-fatal: state is written, user can start manually.
		fmt.Fprintf(os.Stderr, "Warning: could not start fluxcontroller: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run manually: ~/.forge/bin/fluxcontroller\n")
	}

	if isJSON() {
		printJSON(map[string]any{
			"ok":              true,
			"node_id":         nodeID,
			"mesh_ip":         meshIP,
			"controller_addr": controllerAddr,
			"join_token":      joinToken,
		})
	} else {
		fmt.Printf("FluxForge controller initialised\n")
		fmt.Printf("  Node ID:    %s\n", nodeID)
		fmt.Printf("  Mesh IP:    %s\n", meshIP)
		fmt.Printf("  API:        http://%s\n", controllerAddr)
		fmt.Printf("\nJoin token (valid 24h):\n  %s\n", joinToken)
		fmt.Printf("\nTo add a node:\n  forge fluxforge join --controller <this-ip>:%d --token <token>\n", ffInitPort)
	}
	return nil
}

// ---- join ----

var (
	ffJoinController string
	ffJoinToken      string
	ffJoinEndpoint   string
)

var ffJoinCmd = &cobra.Command{
	Use:   "join",
	Short: "Join this server to an existing mesh",
	RunE:  runFluxforgeJoin,
}

func runFluxforgeJoin(cmd *cobra.Command, args []string) error {
	dataDir, err := fluxforgeDataDir()
	if err != nil {
		return cmdErr(err)
	}

	statePath := filepath.Join(dataDir, "state.json")
	if _, err := os.Stat(statePath); err == nil {
		return cmdErr(fmt.Errorf("already initialised — check state at %s", statePath))
	}

	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return cmdErr(fmt.Errorf("creating data dir: %w", err))
	}

	privKey, pubKey, err := generateWireGuardKeys()
	if err != nil {
		return cmdErr(fmt.Errorf("generating WireGuard keys: %w", err))
	}

	nodeID, err := generateID()
	if err != nil {
		return cmdErr(fmt.Errorf("generating node ID: %w", err))
	}

	endpoint := ffJoinEndpoint
	if endpoint == "" {
		if ip, err := detectOutboundIP(); err == nil {
			endpoint = fmt.Sprintf("%s:%d", ip, fluxforgeWireGuardPort)
		}
	}

	// Register with the controller.
	joinReq := map[string]string{
		"node_id":    nodeID,
		"public_key": pubKey,
		"token":      ffJoinToken,
		"endpoint":   endpoint,
	}
	body, _ := json.Marshal(joinReq)

	resp, err := http.Post(
		fmt.Sprintf("http://%s/api/join", ffJoinController),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return cmdErr(fmt.Errorf("contacting controller: %w", err))
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var errResp struct{ Error string `json:"error"` }
		json.Unmarshal(raw, &errResp)
		return cmdErr(fmt.Errorf("join failed: %s", errResp.Error))
	}

	var joinResp struct {
		NodeID    string `json:"node_id"`
		MeshIP    string `json:"mesh_ip"`
		AuthToken string `json:"auth_token"`
	}
	if err := json.Unmarshal(raw, &joinResp); err != nil {
		return cmdErr(fmt.Errorf("parsing join response: %w", err))
	}

	state := fluxforgeState{
		NodeID:         nodeID,
		MeshIP:         joinResp.MeshIP,
		PrivateKey:     privKey,
		PublicKey:      pubKey,
		AuthToken:      joinResp.AuthToken,
		ControllerAddr: ffJoinController,
		IsController:   false,
	}
	if err := writeJSON(statePath, state); err != nil {
		return cmdErr(fmt.Errorf("writing state: %w", err))
	}

	if err := startDaemon("fluxagent"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not start fluxagent: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run manually: ~/.forge/bin/fluxagent\n")
	}

	if isJSON() {
		printJSON(map[string]any{
			"ok":      true,
			"node_id": nodeID,
			"mesh_ip": joinResp.MeshIP,
		})
	} else {
		fmt.Printf("Joined mesh\n")
		fmt.Printf("  Node ID:  %s\n", nodeID)
		fmt.Printf("  Mesh IP:  %s\n", joinResp.MeshIP)
	}
	return nil
}

// ---- status ----

var ffStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show controller status and mesh summary",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := ffAPIGet("/api/status")
		if err != nil {
			return cmdErr(err)
		}
		if isJSON() {
			fmt.Println(string(data))
			return nil
		}
		var s struct {
			ControllerNodeID string `json:"controller_node_id"`
			MeshSubnet       string `json:"mesh_subnet"`
			NodeCount        int    `json:"node_count"`
			RegistryVersion  int64  `json:"registry_version"`
		}
		json.Unmarshal(data, &s)
		w := newTabWriter()
		fmt.Fprintln(w, "CONTROLLER\tSUBNET\tNODES\tVERSION")
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\n",
			s.ControllerNodeID, s.MeshSubnet, s.NodeCount, s.RegistryVersion)
		w.Flush()
		return nil
	},
}

// ---- nodes ----

var ffNodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "List all mesh nodes",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := ffAPIGet("/api/nodes")
		if err != nil {
			return cmdErr(err)
		}
		if isJSON() {
			fmt.Println(string(data))
			return nil
		}
		var nodes []struct {
			ID        string    `json:"id"`
			MeshIP    string    `json:"mesh_ip"`
			Role      string    `json:"role"`
			Endpoint  string    `json:"endpoint"`
			LastSeen  time.Time `json:"last_seen"`
		}
		json.Unmarshal(data, &nodes)
		w := newTabWriter()
		fmt.Fprintln(w, "NODE ID\tMESH IP\tROLE\tENDPOINT\tLAST SEEN")
		for _, n := range nodes {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				n.ID, n.MeshIP, n.Role, n.Endpoint, formatSince(n.LastSeen))
		}
		w.Flush()
		return nil
	},
}

// ---- token ----

var ffTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage join tokens",
}

var ffTokenCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Generate a new 24h single-use join token",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := ffAPIPost("/api/tokens", nil)
		if err != nil {
			return cmdErr(err)
		}
		if isJSON() {
			fmt.Println(string(data))
			return nil
		}
		var t struct {
			Token     string `json:"token"`
			ExpiresAt string `json:"expires_at"`
		}
		json.Unmarshal(data, &t)
		fmt.Printf("Token:      %s\nExpires at: %s\n", t.Token, t.ExpiresAt)
		return nil
	},
}

var ffTokenListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pending join tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := ffAPIGet("/api/tokens")
		if err != nil {
			return cmdErr(err)
		}
		if isJSON() {
			fmt.Println(string(data))
			return nil
		}
		var tokens []struct {
			Value     string    `json:"value"`
			ExpiresAt time.Time `json:"expires_at"`
		}
		json.Unmarshal(data, &tokens)
		if len(tokens) == 0 {
			fmt.Println("No pending tokens.")
			return nil
		}
		w := newTabWriter()
		fmt.Fprintln(w, "TOKEN\tEXPIRES")
		for _, t := range tokens {
			fmt.Fprintf(w, "%s\t%s\n", t.Value, formatSince(t.ExpiresAt))
		}
		w.Flush()
		return nil
	},
}

var ffTokenRevokeCmd = &cobra.Command{
	Use:   "revoke <token>",
	Short: "Revoke a join token before it is used",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := ffAPIDelete("/api/tokens/" + args[0])
		if err != nil {
			return cmdErr(err)
		}
		if isJSON() {
			fmt.Println(string(data))
			return nil
		}
		fmt.Println("Token revoked.")
		return nil
	},
}

// ---- revoke node ----

var ffRevokeCmd = &cobra.Command{
	Use:   "revoke <node-id>",
	Short: "Remove a node from the mesh",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := ffAPIDelete("/api/nodes/" + args[0])
		if err != nil {
			return cmdErr(err)
		}
		if isJSON() {
			fmt.Println(string(data))
			return nil
		}
		fmt.Printf("Node %s revoked.\n", args[0])
		return nil
	},
}

// ---- add-admin / remove-admin ----

var ffAddAdminCmd = &cobra.Command{
	Use:   "add-admin <node-id>",
	Short: "Promote a node to Admin role (Owner only)",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := ffAPIPost("/api/nodes/"+args[0]+"/admin", nil)
		if err != nil {
			return cmdErr(err)
		}
		if isJSON() {
			fmt.Println(string(data))
			return nil
		}
		fmt.Printf("Node %s promoted to admin.\n", args[0])
		return nil
	},
}

var ffRemoveAdminCmd = &cobra.Command{
	Use:   "remove-admin <node-id>",
	Short: "Remove Admin role from a node (Owner only)",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := ffAPIDelete("/api/nodes/" + args[0] + "/admin")
		if err != nil {
			return cmdErr(err)
		}
		if isJSON() {
			fmt.Println(string(data))
			return nil
		}
		fmt.Printf("Admin role removed from node %s.\n", args[0])
		return nil
	},
}

// ---- set-controller ----

var ffSetControllerCmd = &cobra.Command{
	Use:   "set-controller <node-id>",
	Short: "Reassign the FluxController to a different node (Owner only)",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := ffAPIPost("/api/controller/"+args[0], nil)
		if err != nil {
			return cmdErr(err)
		}
		if isJSON() {
			fmt.Println(string(data))
			return nil
		}
		fmt.Printf("Controller reassigned to node %s.\n", args[0])
		return nil
	},
}

// ---- ping ----

var ffPingCmd = &cobra.Command{
	Use:   "ping <node-id|mesh-ip>",
	Short: "Test reachability to a mesh node over WireGuard",
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		// If target looks like a node ID (not an IP), resolve it via the API.
		if net.ParseIP(target) == nil {
			meshIP, err := ffResolveMeshIP(target)
			if err != nil {
				return cmdErr(err)
			}
			target = meshIP
		}

		pingCmd := exec.Command("ping", "-c", "1", "-W", "5", target)
		pingCmd.Stdout = os.Stdout
		pingCmd.Stderr = os.Stderr
		if err := pingCmd.Run(); err != nil {
			return cmdErr(fmt.Errorf("ping %s: unreachable", target))
		}
		return nil
	},
}

// ---- API helpers ----

func ffAPIGet(path string) ([]byte, error) {
	state, err := loadFluxforgeState()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", "http://"+state.ControllerAddr+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+state.AuthToken)
	return doRequest(req)
}

func ffAPIPost(path string, body any) ([]byte, error) {
	state, err := loadFluxforgeState()
	if err != nil {
		return nil, err
	}
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	} else {
		r = bytes.NewReader([]byte("{}"))
	}
	req, err := http.NewRequest("POST", "http://"+state.ControllerAddr+path, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+state.AuthToken)
	req.Header.Set("Content-Type", "application/json")
	return doRequest(req)
}

func ffAPIDelete(path string) ([]byte, error) {
	state, err := loadFluxforgeState()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("DELETE", "http://"+state.ControllerAddr+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+state.AuthToken)
	return doRequest(req)
}

func doRequest(req *http.Request) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("contacting controller: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		var e struct{ Error string `json:"error"` }
		json.Unmarshal(data, &e)
		if e.Error != "" {
			return nil, fmt.Errorf("%s", e.Error)
		}
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return data, nil
}

func ffResolveMeshIP(nodeID string) (string, error) {
	data, err := ffAPIGet("/api/nodes")
	if err != nil {
		return "", err
	}
	var nodes []struct {
		ID     string `json:"id"`
		MeshIP string `json:"mesh_ip"`
	}
	if err := json.Unmarshal(data, &nodes); err != nil {
		return "", err
	}
	for _, n := range nodes {
		if n.ID == nodeID {
			return n.MeshIP, nil
		}
	}
	return "", fmt.Errorf("node %s not found", nodeID)
}

// ---- state / key helpers ----

func loadFluxforgeState() (*fluxforgeState, error) {
	dataDir, err := fluxforgeDataDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dataDir, "state.json"))
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("fluxforge not initialised — run: forge fluxforge init")
	}
	if err != nil {
		return nil, fmt.Errorf("reading fluxforge state: %w", err)
	}
	var s fluxforgeState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing fluxforge state: %w", err)
	}
	return &s, nil
}

func fluxforgeDataDir() (string, error) {
	forgeDir, err := paths.ForgeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(forgeDir, "fluxforge"), nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// generateWireGuardKeys returns (privKeyBase64, pubKeyBase64, error).
// Uses crypto/ecdh (stdlib since Go 1.20) for correct Curve25519 key generation.
func generateWireGuardKeys() (priv, pub string, err error) {
	curve := ecdh.X25519()
	privKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return
	}
	priv = base64.StdEncoding.EncodeToString(privKey.Bytes())
	pub = base64.StdEncoding.EncodeToString(privKey.PublicKey().Bytes())
	return
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

func generateToken() (string, error) {
	return generateSecureToken()
}

func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func startDaemon(name string) error {
	binPath, err := paths.ModuleBin(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		return fmt.Errorf("binary not found at %s — run: forge install fluxforge", binPath)
	}

	forgeDir, _ := paths.ForgeDir()
	logPath := filepath.Join(forgeDir, "data", name, "forge.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return err
	}
	lf, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	c := exec.Command(binPath)
	c.Stdout = lf
	c.Stderr = lf
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := c.Start(); err != nil {
		lf.Close()
		return err
	}
	lf.Close()
	return nil
}

func detectOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String(), nil
}
