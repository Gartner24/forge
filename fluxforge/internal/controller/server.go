package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gartner24/forge/fluxforge/internal/mesh"
	"github.com/gartner24/forge/fluxforge/internal/wg"
	"github.com/gartner24/forge/shared/audit"
)

type ctxKey struct{}

// Server is the FluxController HTTP API server.
type Server struct {
	registry *Registry
	wgMgr    *wg.Manager
	audit    *audit.Logger
	nodeID   string // this controller node's own ID
}

func NewServer(reg *Registry, wgMgr *wg.Manager, auditLog *audit.Logger, nodeID string) *Server {
	return &Server{registry: reg, wgMgr: wgMgr, audit: auditLog, nodeID: nodeID}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/join", s.handleJoin)
	mux.HandleFunc("GET /api/peers", s.requireAuth(s.handlePeers))
	mux.HandleFunc("POST /api/heartbeat", s.requireAuth(s.handleHeartbeat))
	mux.HandleFunc("GET /api/status", s.requireAuth(s.handleStatus))
	mux.HandleFunc("GET /api/nodes", s.requireRole(mesh.RoleAdmin, s.handleNodes))
	mux.HandleFunc("POST /api/tokens", s.requireRole(mesh.RoleAdmin, s.handleTokenCreate))
	mux.HandleFunc("GET /api/tokens", s.requireRole(mesh.RoleAdmin, s.handleTokenList))
	mux.HandleFunc("DELETE /api/tokens/{token}", s.requireRole(mesh.RoleAdmin, s.handleTokenRevoke))
	mux.HandleFunc("DELETE /api/nodes/{id}", s.requireRole(mesh.RoleAdmin, s.handleNodeRevoke))
	mux.HandleFunc("POST /api/nodes/{id}/admin", s.requireRole(mesh.RoleOwner, s.handleAddAdmin))
	mux.HandleFunc("DELETE /api/nodes/{id}/admin", s.requireRole(mesh.RoleOwner, s.handleRemoveAdmin))
	mux.HandleFunc("POST /api/controller/{id}", s.requireRole(mesh.RoleOwner, s.handleSetController))
	return mux
}

// --- join (no auth required) ---

type joinRequest struct {
	NodeID    string `json:"node_id"`
	PublicKey string `json:"public_key"`
	Token     string `json:"token"`
	Endpoint  string `json:"endpoint"` // host:port for WireGuard UDP; auto-detected if empty
}

type joinResponse struct {
	NodeID    string          `json:"node_id"`
	MeshIP    string          `json:"mesh_ip"`
	AuthToken string          `json:"auth_token"`
	Peers     []mesh.NodeInfo `json:"peers"`
}

func (s *Server) handleJoin(w http.ResponseWriter, r *http.Request) {
	var req joinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.NodeID == "" || req.PublicKey == "" || req.Token == "" {
		writeErr(w, http.StatusBadRequest, "node_id, public_key, and token are required")
		return
	}

	endpoint := req.Endpoint
	if endpoint == "" {
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			endpoint = fmt.Sprintf("%s:%d", host, mesh.WireGuardPort)
		}
	}

	if err := s.registry.ConsumeToken(req.Token); err != nil {
		writeErr(w, http.StatusUnauthorized, err.Error())
		return
	}

	authToken, err := newAuthToken()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "generating auth token")
		return
	}

	meshIP := s.registry.NextMeshIP()
	now := time.Now().UTC()
	node := mesh.Node{
		ID:        req.NodeID,
		MeshIP:    meshIP,
		PublicKey: req.PublicKey,
		Endpoint:  endpoint,
		Role:      mesh.RoleNode,
		AuthToken: authToken,
		LastSeen:  now,
		CreatedAt: now,
	}
	if err := s.registry.AddNode(node); err != nil {
		writeErr(w, http.StatusInternalServerError, "registering node")
		return
	}

	s.audit.Write("fluxforge", "node.join", req.NodeID, fmt.Sprintf("mesh_ip=%s endpoint=%s", meshIP, endpoint))
	s.syncWireGuard()

	writeJSON(w, http.StatusOK, joinResponse{
		NodeID:    req.NodeID,
		MeshIP:    meshIP,
		AuthToken: authToken,
		Peers:     s.registry.Nodes(),
	})
}

// --- peers ---

func (s *Server) handlePeers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"version": s.registry.Version(),
		"peers":   s.registry.Nodes(),
	})
}

// --- heartbeat ---

type heartbeatRequest struct {
	NodeID          string `json:"node_id"`
	RegistryVersion int64  `json:"registry_version"`
}

type heartbeatResponse struct {
	RegistryVersion int64           `json:"registry_version"`
	Peers           []mesh.NodeInfo `json:"peers,omitempty"`
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	node := nodeFromCtx(r.Context())
	s.registry.UpdateLastSeen(node.ID)

	var req heartbeatRequest
	json.NewDecoder(r.Body).Decode(&req)

	current := s.registry.Version()
	resp := heartbeatResponse{RegistryVersion: current}
	if req.RegistryVersion != current {
		resp.Peers = s.registry.Nodes()
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- status ---

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	nodes := s.registry.Nodes()
	writeJSON(w, http.StatusOK, map[string]any{
		"controller_node_id": s.registry.ControllerNodeID(),
		"mesh_subnet":        mesh.MeshSubnet,
		"node_count":         len(nodes),
		"registry_version":   s.registry.Version(),
	})
}

// --- nodes ---

func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.registry.Nodes())
}

// --- token create ---

func (s *Server) handleTokenCreate(w http.ResponseWriter, r *http.Request) {
	node := nodeFromCtx(r.Context())
	t, err := newJoinToken()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "generating token")
		return
	}
	if err := s.registry.AddToken(t); err != nil {
		writeErr(w, http.StatusInternalServerError, "storing token")
		return
	}
	s.audit.Write("fluxforge", "token.create", node.ID, "expires="+t.ExpiresAt.Format(time.RFC3339))
	writeJSON(w, http.StatusOK, map[string]string{
		"token":      t.Value,
		"expires_at": t.ExpiresAt.Format(time.RFC3339),
	})
}

// --- token list ---

func (s *Server) handleTokenList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.registry.Tokens())
}

// --- token revoke ---

func (s *Server) handleTokenRevoke(w http.ResponseWriter, r *http.Request) {
	actor := nodeFromCtx(r.Context())
	token := r.PathValue("token")
	if err := s.registry.RevokeToken(token); err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	s.audit.Write("fluxforge", "token.revoke", actor.ID, "token="+safePrefix(token, 8)+"...")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// --- node revoke ---

func (s *Server) handleNodeRevoke(w http.ResponseWriter, r *http.Request) {
	actor := nodeFromCtx(r.Context())
	id := r.PathValue("id")

	target := s.registry.NodeByID(id)
	if target == nil {
		writeErr(w, http.StatusNotFound, "node not found")
		return
	}
	if target.Role == mesh.RoleOwner {
		writeErr(w, http.StatusForbidden, "cannot revoke the owner")
		return
	}
	if id == s.registry.ControllerNodeID() {
		writeErr(w, http.StatusConflict, "cannot revoke the active controller; reassign first with set-controller")
		return
	}

	if err := s.registry.RemoveNode(id); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.audit.Write("fluxforge", "node.revoke", actor.ID, "target="+id)
	s.syncWireGuard()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// --- add admin ---

func (s *Server) handleAddAdmin(w http.ResponseWriter, r *http.Request) {
	actor := nodeFromCtx(r.Context())
	id := r.PathValue("id")
	if s.registry.NodeByID(id) == nil {
		writeErr(w, http.StatusNotFound, "node not found")
		return
	}
	if err := s.registry.SetRole(id, mesh.RoleAdmin); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.audit.Write("fluxforge", "role.add_admin", actor.ID, "target="+id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// --- remove admin ---

func (s *Server) handleRemoveAdmin(w http.ResponseWriter, r *http.Request) {
	actor := nodeFromCtx(r.Context())
	id := r.PathValue("id")

	target := s.registry.NodeByID(id)
	if target == nil {
		writeErr(w, http.StatusNotFound, "node not found")
		return
	}
	if target.Role == mesh.RoleOwner {
		writeErr(w, http.StatusForbidden, "cannot remove owner role")
		return
	}
	if err := s.registry.SetRole(id, mesh.RoleNode); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.audit.Write("fluxforge", "role.remove_admin", actor.ID, "target="+id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// --- set controller ---

func (s *Server) handleSetController(w http.ResponseWriter, r *http.Request) {
	actor := nodeFromCtx(r.Context())
	id := r.PathValue("id")

	if s.registry.NodeByID(id) == nil {
		writeErr(w, http.StatusNotFound, "node not found")
		return
	}
	if err := s.registry.SetControllerNodeID(id); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.audit.Write("fluxforge", "controller.reassign", actor.ID, "new_controller="+id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// --- auth middleware ---

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// NodeByToken reads from in-memory registry — never uses cached auth state.
		node := s.authNode(r)
		if node == nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		ctx := context.WithValue(r.Context(), ctxKey{}, node)
		next(w, r.WithContext(ctx))
	}
}

func (s *Server) requireRole(minRole mesh.Role, next http.HandlerFunc) http.HandlerFunc {
	return s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		node := nodeFromCtx(r.Context())
		if !roleAtLeast(node.Role, minRole) {
			writeErr(w, http.StatusForbidden, "insufficient role")
			return
		}
		next(w, r)
	})
}

func (s *Server) authNode(r *http.Request) *mesh.Node {
	hdr := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(hdr, "Bearer ")
	if !ok || token == "" {
		return nil
	}
	return s.registry.NodeByToken(token)
}

func nodeFromCtx(ctx context.Context) *mesh.Node {
	n, _ := ctx.Value(ctxKey{}).(*mesh.Node)
	return n
}

func roleAtLeast(have, need mesh.Role) bool {
	order := map[mesh.Role]int{mesh.RoleNode: 0, mesh.RoleAdmin: 1, mesh.RoleOwner: 2}
	return order[have] >= order[need]
}

// --- WireGuard sync ---

// SyncWireGuard pushes the current peer list to the local WireGuard interface.
// Called after any peer registry mutation so tunnels stay current within 5 seconds.
// Also exported so the main binary can do an initial sync on startup.
func (s *Server) SyncWireGuard() {
	s.syncWireGuard()
}

func (s *Server) syncWireGuard() {
	if s.wgMgr == nil {
		return
	}
	nodes := s.registry.Nodes()
	peers := make([]wg.PeerConfig, 0, len(nodes))
	for _, n := range nodes {
		if n.ID == s.nodeID {
			continue
		}
		peers = append(peers, wg.PeerConfig{
			PublicKey: n.PublicKey,
			AllowedIP: n.MeshIP + "/32",
			Endpoint:  n.Endpoint,
		})
	}
	s.wgMgr.SetPeers(peers) //nolint:errcheck
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func safePrefix(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[:n]
}
