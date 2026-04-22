package mesh

import "time"

const (
	// MeshSubnet is the WireGuard mesh network CIDR.
	MeshSubnet = "10.99.0.0/16"
	// InterfaceName is the WireGuard interface name on every node.
	InterfaceName = "forge0"
	// WireGuardPort is the default WireGuard UDP listen port.
	WireGuardPort = 51820
	// ControllerPort is the default FluxController HTTP API port.
	ControllerPort = 7777
	// TokenTTL is how long a join token remains valid.
	TokenTTL = 24 * time.Hour
	// HeartbeatInterval is how often agents send heartbeats to the controller.
	HeartbeatInterval = 5 * time.Second
	// ControllerMeshIP is the reserved mesh IP for the controller.
	ControllerMeshIP = "10.99.1.1"
)

type Role string

const (
	RoleOwner Role = "owner"
	RoleAdmin Role = "admin"
	RoleNode  Role = "node"
)

// Node is a mesh member record, stored in the controller registry.
// AuthToken is only sent to the node that owns it; never to other peers.
type Node struct {
	ID        string    `json:"id"`
	MeshIP    string    `json:"mesh_ip"`
	PublicKey string    `json:"public_key"` // WireGuard public key, base64
	Endpoint  string    `json:"endpoint"`   // host:port for WireGuard UDP
	Role      Role      `json:"role"`
	AuthToken string    `json:"auth_token"` // bearer token for this node's API auth
	LastSeen  time.Time `json:"last_seen"`
	CreatedAt time.Time `json:"created_at"`
}

// NodeInfo is the safe-to-share subset of Node (no AuthToken).
type NodeInfo struct {
	ID        string    `json:"id"`
	MeshIP    string    `json:"mesh_ip"`
	PublicKey string    `json:"public_key"`
	Endpoint  string    `json:"endpoint"`
	Role      Role      `json:"role"`
	LastSeen  time.Time `json:"last_seen"`
	CreatedAt time.Time `json:"created_at"`
}

func (n *Node) Info() NodeInfo {
	return NodeInfo{
		ID:        n.ID,
		MeshIP:    n.MeshIP,
		PublicKey: n.PublicKey,
		Endpoint:  n.Endpoint,
		Role:      n.Role,
		LastSeen:  n.LastSeen,
		CreatedAt: n.CreatedAt,
	}
}

// Token is a single-use join credential.
type Token struct {
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
}

// Registry is the authoritative peer registry owned by the controller.
type Registry struct {
	Nodes            []Node  `json:"nodes"`
	Tokens           []Token `json:"tokens"`
	NextIPIndex      int     `json:"next_ip_index"` // third octet of next mesh IP
	ControllerNodeID string  `json:"controller_node_id"`
	Version          int64   `json:"version"` // increments on every mutating operation
}

// LocalState is stored on every node to remember its mesh identity.
type LocalState struct {
	NodeID         string `json:"node_id"`
	MeshIP         string `json:"mesh_ip"`
	PrivateKey     string `json:"private_key"`  // WireGuard private key, base64
	PublicKey      string `json:"public_key"`   // WireGuard public key, base64
	AuthToken      string `json:"auth_token"`   // API bearer token for this node
	ControllerAddr string `json:"controller_addr"` // host:port of the controller API
	IsController   bool   `json:"is_controller"`
}

// MeshIPForIndex returns the mesh IP for the given node index (1-based).
// Index 1 is the controller (10.99.1.1), index 2 is the first agent (10.99.2.1), etc.
func MeshIPForIndex(idx int) string {
	return "10.99." + itoa(idx) + ".1"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [10]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
