package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gartner24/forge/fluxforge/internal/mesh"
	"github.com/gartner24/forge/shared/audit"
)

func newTestServer(t *testing.T) (*Server, *Registry) {
	t.Helper()
	dir := t.TempDir()
	reg, err := NewRegistry(filepath.Join(dir, "registry.json"))
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	auditLog, err := audit.New(filepath.Join(dir, "audit.log"))
	if err != nil {
		t.Fatalf("audit.New: %v", err)
	}
	// Seed an owner node.
	now := time.Now().UTC()
	ownerToken := "owner-bearer-token"
	ownerID := "owner-node-id"
	reg.AddNode(mesh.Node{
		ID: ownerID, MeshIP: "10.99.1.1", PublicKey: "owner-pubkey",
		Role: mesh.RoleOwner, AuthToken: ownerToken, LastSeen: now, CreatedAt: now,
	})
	reg.SetControllerNodeID(ownerID)

	srv := NewServer(reg, nil, auditLog, ownerID)
	return srv, reg
}

func do(t *testing.T, handler http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	if body != nil {
		json.NewEncoder(&b).Encode(body)
	}
	req := httptest.NewRequest(method, path, &b)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

// ---- Auth enforcement (Kano Level 1) ----

func TestAuth_RequiredOnProtectedRoutes(t *testing.T) {
	srv, _ := newTestServer(t)
	h := srv.Handler()

	routes := []struct{ method, path string }{
		{"GET", "/api/peers"},
		{"POST", "/api/heartbeat"},
		{"GET", "/api/status"},
	}
	for _, r := range routes {
		w := do(t, h, r.method, r.path, nil, "")
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without auth: got %d, want 401", r.method, r.path, w.Code)
		}
	}
}

func TestAuth_BadTokenRejected(t *testing.T) {
	srv, _ := newTestServer(t)
	w := do(t, srv.Handler(), "GET", "/api/status", nil, "bad-token")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("bad token: got %d, want 401", w.Code)
	}
}

func TestAuth_ValidTokenAccepted(t *testing.T) {
	srv, _ := newTestServer(t)
	w := do(t, srv.Handler(), "GET", "/api/status", nil, "owner-bearer-token")
	if w.Code != http.StatusOK {
		t.Errorf("valid token: got %d, want 200", w.Code)
	}
}

// ---- RBAC enforcement (Kano Level 1) ----

func TestRBAC_NodeCannotCreateToken(t *testing.T) {
	srv, reg := newTestServer(t)
	// Add a plain node.
	now := time.Now().UTC()
	reg.AddNode(mesh.Node{
		ID: "node-2", MeshIP: "10.99.2.1", PublicKey: "pk2",
		Role: mesh.RoleNode, AuthToken: "node-token", LastSeen: now, CreatedAt: now,
	})

	w := do(t, srv.Handler(), "POST", "/api/tokens", nil, "node-token")
	if w.Code != http.StatusForbidden {
		t.Errorf("node creating token: got %d, want 403", w.Code)
	}
}

func TestRBAC_AdminCanCreateToken(t *testing.T) {
	srv, reg := newTestServer(t)
	now := time.Now().UTC()
	reg.AddNode(mesh.Node{
		ID: "admin-node", MeshIP: "10.99.3.1", PublicKey: "pk-admin",
		Role: mesh.RoleAdmin, AuthToken: "admin-token", LastSeen: now, CreatedAt: now,
	})

	w := do(t, srv.Handler(), "POST", "/api/tokens", nil, "admin-token")
	if w.Code != http.StatusOK {
		t.Errorf("admin creating token: got %d, want 200", w.Code)
	}
}

func TestRBAC_AdminCannotAddAdmin(t *testing.T) {
	srv, reg := newTestServer(t)
	now := time.Now().UTC()
	reg.AddNode(mesh.Node{
		ID: "admin-node", MeshIP: "10.99.3.1", PublicKey: "pk-admin",
		Role: mesh.RoleAdmin, AuthToken: "admin-token", LastSeen: now, CreatedAt: now,
	})
	reg.AddNode(mesh.Node{
		ID: "target", MeshIP: "10.99.4.1", PublicKey: "pk-target",
		Role: mesh.RoleNode, AuthToken: "target-tok", LastSeen: now, CreatedAt: now,
	})

	w := do(t, srv.Handler(), "POST", "/api/nodes/target/admin", nil, "admin-token")
	if w.Code != http.StatusForbidden {
		t.Errorf("admin adding admin: got %d, want 403", w.Code)
	}
}

func TestRBAC_OwnerCanAddAdmin(t *testing.T) {
	srv, reg := newTestServer(t)
	now := time.Now().UTC()
	reg.AddNode(mesh.Node{
		ID: "target", MeshIP: "10.99.4.1", PublicKey: "pk-target",
		Role: mesh.RoleNode, AuthToken: "target-tok", LastSeen: now, CreatedAt: now,
	})

	w := do(t, srv.Handler(), "POST", "/api/nodes/target/admin", nil, "owner-bearer-token")
	if w.Code != http.StatusOK {
		t.Errorf("owner adding admin: got %d, want 200 (got body: %s)", w.Code, w.Body.String())
	}
}

// ---- Owner protection (Kano Level 1 — security) ----

func TestCannotRevokeOwner(t *testing.T) {
	srv, _ := newTestServer(t)
	// Owner tries to revoke themselves (or another owner).
	w := do(t, srv.Handler(), "DELETE", "/api/nodes/owner-node-id", nil, "owner-bearer-token")
	if w.Code != http.StatusForbidden {
		t.Errorf("revoking owner: got %d, want 403", w.Code)
	}
}

func TestCannotRemoveOwnerRole(t *testing.T) {
	srv, _ := newTestServer(t)
	w := do(t, srv.Handler(), "DELETE", "/api/nodes/owner-node-id/admin", nil, "owner-bearer-token")
	if w.Code != http.StatusForbidden {
		t.Errorf("removing owner role: got %d, want 403 (body: %s)", w.Code, w.Body.String())
	}
}

// ---- Join flow (Kano Level 2 — core) ----

func TestJoin_ValidToken(t *testing.T) {
	srv, reg := newTestServer(t)
	tok, _ := newJoinToken()
	reg.AddToken(tok)

	body := map[string]string{
		"node_id":    "new-node",
		"public_key": "newpubkey",
		"token":      tok.Value,
	}
	w := do(t, srv.Handler(), "POST", "/api/join", body, "")
	if w.Code != http.StatusOK {
		t.Fatalf("join with valid token: got %d body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["mesh_ip"] == "" || resp["auth_token"] == "" {
		t.Errorf("join response missing fields: %+v", resp)
	}
}

func TestJoin_InvalidToken(t *testing.T) {
	srv, _ := newTestServer(t)
	body := map[string]string{
		"node_id": "n", "public_key": "pk", "token": "bad",
	}
	w := do(t, srv.Handler(), "POST", "/api/join", body, "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("join with bad token: got %d, want 401", w.Code)
	}
}

func TestJoin_TokenConsumedAfterUse(t *testing.T) {
	srv, reg := newTestServer(t)
	tok, _ := newJoinToken()
	reg.AddToken(tok)

	body := map[string]string{
		"node_id": "n1", "public_key": "pk1", "token": tok.Value,
	}
	do(t, srv.Handler(), "POST", "/api/join", body, "") // first join consumes token

	body["node_id"] = "n2"
	w := do(t, srv.Handler(), "POST", "/api/join", body, "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("second join with same token: got %d, want 401", w.Code)
	}
}

// ---- Heartbeat version diffing (Kano Level 2) ----

func TestHeartbeat_ReturnsPeersOnVersionMismatch(t *testing.T) {
	srv, _ := newTestServer(t)
	body := map[string]any{"node_id": "owner-node-id", "registry_version": int64(0)}

	w := do(t, srv.Handler(), "POST", "/api/heartbeat", body, "owner-bearer-token")
	if w.Code != http.StatusOK {
		t.Fatalf("heartbeat: %d", w.Code)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	// Version 0 != current version (1+), so peers should be returned.
	if resp["peers"] == nil {
		t.Error("heartbeat with stale version should return peers")
	}
}

func TestHeartbeat_NoPeersWhenVersionCurrent(t *testing.T) {
	srv, reg := newTestServer(t)
	currentVersion := reg.Version()

	body := map[string]any{"node_id": "owner-node-id", "registry_version": currentVersion}
	w := do(t, srv.Handler(), "POST", "/api/heartbeat", body, "owner-bearer-token")
	if w.Code != http.StatusOK {
		t.Fatalf("heartbeat: %d", w.Code)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["peers"] != nil {
		t.Error("heartbeat with current version should not return peers")
	}
}
