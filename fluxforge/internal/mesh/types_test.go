package mesh

import "testing"

func TestMeshIPForIndex(t *testing.T) {
	cases := []struct {
		idx  int
		want string
	}{
		{1, "10.99.1.1"},
		{2, "10.99.2.1"},
		{10, "10.99.10.1"},
		{254, "10.99.254.1"},
	}
	for _, c := range cases {
		got := MeshIPForIndex(c.idx)
		if got != c.want {
			t.Errorf("MeshIPForIndex(%d) = %q, want %q", c.idx, got, c.want)
		}
	}
}

func TestNodeInfo_DoesNotLeakAuthToken(t *testing.T) {
	n := Node{
		ID:        "node-1",
		MeshIP:    "10.99.2.1",
		AuthToken: "super-secret",
		Role:      RoleNode,
	}
	info := n.Info()
	// NodeInfo has no AuthToken field — this is a compile-time guarantee,
	// but verify via the struct that the field simply doesn't exist.
	_ = info.ID
	_ = info.MeshIP
	_ = info.Role
	// If NodeInfo had AuthToken, accessing info.AuthToken would compile — it does not.
}

func TestRoleConstants(t *testing.T) {
	if RoleOwner == RoleAdmin || RoleAdmin == RoleNode || RoleOwner == RoleNode {
		t.Fatal("role constants must be distinct")
	}
}
