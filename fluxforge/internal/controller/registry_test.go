package controller

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gartner24/forge/fluxforge/internal/mesh"
)

func tempRegistry(t *testing.T) *Registry {
	t.Helper()
	dir := t.TempDir()
	reg, err := NewRegistry(filepath.Join(dir, "registry.json"))
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	return reg
}

func addNode(t *testing.T, reg *Registry, id, meshIP string, role mesh.Role, authToken string) {
	t.Helper()
	now := time.Now().UTC()
	err := reg.AddNode(mesh.Node{
		ID: id, MeshIP: meshIP, PublicKey: "key-" + id,
		Role: role, AuthToken: authToken, LastSeen: now, CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("AddNode %s: %v", id, err)
	}
}

// ---- Token tests (Kano Level 1 — security) ----

func TestConsumeToken_Valid(t *testing.T) {
	reg := tempRegistry(t)
	tok, _ := newJoinToken()
	reg.AddToken(tok)

	if err := reg.ConsumeToken(tok.Value); err != nil {
		t.Fatalf("ConsumeToken rejected valid token: %v", err)
	}
}

func TestConsumeToken_SingleUse(t *testing.T) {
	reg := tempRegistry(t)
	tok, _ := newJoinToken()
	reg.AddToken(tok)
	reg.ConsumeToken(tok.Value)

	if err := reg.ConsumeToken(tok.Value); err == nil {
		t.Fatal("expected error on second use of token, got nil")
	}
}

func TestConsumeToken_Expired(t *testing.T) {
	reg := tempRegistry(t)
	now := time.Now().UTC()
	expired := mesh.Token{
		Value:     "expiredtoken",
		CreatedAt: now.Add(-25 * time.Hour),
		ExpiresAt: now.Add(-1 * time.Hour),
	}
	reg.AddToken(expired)

	if err := reg.ConsumeToken(expired.Value); err == nil {
		t.Fatal("expected error on expired token, got nil")
	}
}

func TestConsumeToken_Invalid(t *testing.T) {
	reg := tempRegistry(t)
	if err := reg.ConsumeToken("does-not-exist"); err == nil {
		t.Fatal("expected error on unknown token, got nil")
	}
}

func TestRevokeToken(t *testing.T) {
	reg := tempRegistry(t)
	tok, _ := newJoinToken()
	reg.AddToken(tok)

	if err := reg.RevokeToken(tok.Value); err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}
	if err := reg.ConsumeToken(tok.Value); err == nil {
		t.Fatal("expected error consuming revoked token, got nil")
	}
}

func TestTokenTTL_Is24Hours(t *testing.T) {
	tok, err := newJoinToken()
	if err != nil {
		t.Fatal(err)
	}
	ttl := tok.ExpiresAt.Sub(tok.CreatedAt)
	if ttl < 23*time.Hour+55*time.Minute || ttl > 24*time.Hour+5*time.Minute {
		t.Errorf("token TTL = %v, want ~24h", ttl)
	}
}

func TestTokenSize_AtLeast32Bytes(t *testing.T) {
	tok, _ := newJoinToken()
	// base64url of 32 bytes = 44 chars
	if len(tok.Value) < 43 {
		t.Errorf("token value %q too short (len=%d), expected >=43 for 32 raw bytes", tok.Value, len(tok.Value))
	}
}

func TestAuthToken_AtLeast32Bytes(t *testing.T) {
	tok, err := newAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(tok) < 43 {
		t.Errorf("auth token %q too short", tok)
	}
}

// ---- Registry CRUD tests (Kano Level 2 — core) ----

func TestAddNode_Persists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	reg1, _ := NewRegistry(path)
	addNode(t, reg1, "n1", "10.99.1.1", mesh.RoleOwner, "tok1")

	// Open a new registry from the same file — should see the node.
	reg2, err := NewRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	nodes := reg2.Nodes()
	if len(nodes) != 1 || nodes[0].ID != "n1" {
		t.Fatalf("expected node n1, got %+v", nodes)
	}
}

func TestRemoveNode(t *testing.T) {
	reg := tempRegistry(t)
	addNode(t, reg, "n1", "10.99.1.1", mesh.RoleOwner, "tok1")
	addNode(t, reg, "n2", "10.99.2.1", mesh.RoleNode, "tok2")

	if err := reg.RemoveNode("n2"); err != nil {
		t.Fatal(err)
	}
	nodes := reg.Nodes()
	if len(nodes) != 1 || nodes[0].ID != "n1" {
		t.Fatalf("expected 1 node after removal, got %+v", nodes)
	}
}

func TestRemoveNode_NotFound(t *testing.T) {
	reg := tempRegistry(t)
	if err := reg.RemoveNode("ghost"); err == nil {
		t.Fatal("expected error removing non-existent node")
	}
}

func TestNodeByToken_Found(t *testing.T) {
	reg := tempRegistry(t)
	addNode(t, reg, "n1", "10.99.1.1", mesh.RoleOwner, "secret-tok")

	node := reg.NodeByToken("secret-tok")
	if node == nil || node.ID != "n1" {
		t.Fatalf("NodeByToken: expected n1, got %v", node)
	}
}

func TestNodeByToken_NotFound(t *testing.T) {
	reg := tempRegistry(t)
	if reg.NodeByToken("no-such-token") != nil {
		t.Fatal("expected nil for unknown token")
	}
}

func TestSetRole(t *testing.T) {
	reg := tempRegistry(t)
	addNode(t, reg, "n1", "10.99.1.1", mesh.RoleNode, "tok1")

	if err := reg.SetRole("n1", mesh.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	n := reg.NodeByID("n1")
	if n.Role != mesh.RoleAdmin {
		t.Fatalf("expected RoleAdmin, got %s", n.Role)
	}
}

func TestVersionBumps_OnMutation(t *testing.T) {
	reg := tempRegistry(t)
	v0 := reg.Version()

	addNode(t, reg, "n1", "10.99.1.1", mesh.RoleOwner, "tok1")
	v1 := reg.Version()
	if v1 <= v0 {
		t.Fatalf("version did not increment after AddNode: %d -> %d", v0, v1)
	}

	reg.RemoveNode("n1")
	v2 := reg.Version()
	if v2 <= v1 {
		t.Fatalf("version did not increment after RemoveNode: %d -> %d", v1, v2)
	}
}

func TestNextMeshIP_Sequence(t *testing.T) {
	reg := tempRegistry(t)
	ip1 := reg.NextMeshIP()
	ip2 := reg.NextMeshIP()
	ip3 := reg.NextMeshIP()

	if ip1 == ip2 || ip2 == ip3 {
		t.Fatalf("NextMeshIP returned duplicates: %s %s %s", ip1, ip2, ip3)
	}
	// Should start at index 2 (index 1 is reserved for controller)
	if ip1 != "10.99.2.1" {
		t.Errorf("first agent IP = %s, want 10.99.2.1", ip1)
	}
}

func TestPendingTokens_ExcludesExpiredAndUsed(t *testing.T) {
	reg := tempRegistry(t)
	now := time.Now().UTC()

	good, _ := newJoinToken()
	reg.AddToken(good)

	expired := mesh.Token{Value: "exp", CreatedAt: now.Add(-25 * time.Hour), ExpiresAt: now.Add(-1 * time.Hour)}
	reg.AddToken(expired)

	used := mesh.Token{Value: "used", CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour), Used: true}
	reg.AddToken(used)

	pending := reg.Tokens()
	if len(pending) != 1 || pending[0].Value != good.Value {
		t.Fatalf("Tokens() = %+v, want only the valid token", pending)
	}
}

// ---- RBAC tests (Kano Level 1 — security) ----

func TestRoleAtLeast(t *testing.T) {
	cases := []struct {
		have, need mesh.Role
		want       bool
	}{
		{mesh.RoleOwner, mesh.RoleOwner, true},
		{mesh.RoleOwner, mesh.RoleAdmin, true},
		{mesh.RoleOwner, mesh.RoleNode, true},
		{mesh.RoleAdmin, mesh.RoleOwner, false},
		{mesh.RoleAdmin, mesh.RoleAdmin, true},
		{mesh.RoleAdmin, mesh.RoleNode, true},
		{mesh.RoleNode, mesh.RoleOwner, false},
		{mesh.RoleNode, mesh.RoleAdmin, false},
		{mesh.RoleNode, mesh.RoleNode, true},
	}
	for _, c := range cases {
		got := roleAtLeast(c.have, c.need)
		if got != c.want {
			t.Errorf("roleAtLeast(%s, %s) = %v, want %v", c.have, c.need, got, c.want)
		}
	}
}

// ---- Cleanup verification ----

func TestMain(m *testing.M) {
	// No global state created — t.TempDir() handles per-test cleanup.
	os.Exit(m.Run())
}
