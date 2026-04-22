package controller

import (
	"fmt"
	"sync"
	"time"

	"github.com/gartner24/forge/fluxforge/internal/mesh"
	"github.com/gartner24/forge/fluxforge/internal/store"
)

// Registry manages the in-memory peer registry and flushes it to disk.
// All mutating methods acquire the write lock and bump Version so agents
// know when to re-sync.
type Registry struct {
	mu    sync.RWMutex
	data  mesh.Registry
	store *store.Store[mesh.Registry]
}

func NewRegistry(path string) (*Registry, error) {
	s, err := store.New[mesh.Registry](path)
	if err != nil {
		return nil, err
	}

	r := &Registry{store: s}

	existing, err := s.Read()
	if err != nil {
		return nil, fmt.Errorf("reading registry: %w", err)
	}
	if existing != nil {
		r.data = *existing
	} else {
		r.data = mesh.Registry{NextIPIndex: 2} // 1 reserved for controller
	}

	return r, nil
}

// Version returns the current registry version.
func (r *Registry) Version() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.data.Version
}

// ControllerNodeID returns the ID of the controller node.
func (r *Registry) ControllerNodeID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.data.ControllerNodeID
}

// Nodes returns a snapshot of all nodes (without auth tokens).
func (r *Registry) Nodes() []mesh.NodeInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]mesh.NodeInfo, len(r.data.Nodes))
	for i, n := range r.data.Nodes {
		out[i] = n.Info()
	}
	return out
}

// NodeByToken finds a node by its auth token.
// Returns nil if not found. Refreshes from in-memory state on every call (no disk read).
func (r *Registry) NodeByToken(token string) *mesh.Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for i := range r.data.Nodes {
		if r.data.Nodes[i].AuthToken == token {
			n := r.data.Nodes[i]
			return &n
		}
	}
	return nil
}

// NodeByID returns a copy of the node with the given ID, or nil.
func (r *Registry) NodeByID(id string) *mesh.Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for i := range r.data.Nodes {
		if r.data.Nodes[i].ID == id {
			n := r.data.Nodes[i]
			return &n
		}
	}
	return nil
}

// AddNode adds a node and flushes to disk.
func (r *Registry) AddNode(n mesh.Node) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data.Nodes = append(r.data.Nodes, n)
	r.data.Version++
	return r.flush()
}

// RemoveNode deletes the node with the given ID and flushes to disk.
func (r *Registry) RemoveNode(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	nodes := r.data.Nodes[:0]
	for _, n := range r.data.Nodes {
		if n.ID != id {
			nodes = append(nodes, n)
		}
	}
	if len(nodes) == len(r.data.Nodes) {
		return fmt.Errorf("node %s not found", id)
	}
	r.data.Nodes = nodes
	r.data.Version++
	return r.flush()
}

// SetRole updates the role of the given node and flushes to disk.
func (r *Registry) SetRole(id string, role mesh.Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.data.Nodes {
		if r.data.Nodes[i].ID == id {
			r.data.Nodes[i].Role = role
			r.data.Version++
			return r.flush()
		}
	}
	return fmt.Errorf("node %s not found", id)
}

// SetControllerNodeID updates the designated controller node ID.
func (r *Registry) SetControllerNodeID(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data.ControllerNodeID = id
	r.data.Version++
	return r.flush()
}

// UpdateLastSeen records that a node was seen now.
func (r *Registry) UpdateLastSeen(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.data.Nodes {
		if r.data.Nodes[i].ID == id {
			r.data.Nodes[i].LastSeen = time.Now().UTC()
			// Don't flush for heartbeats — it's too frequent.
			return
		}
	}
}

// NextMeshIP assigns and returns the next available mesh IP, incrementing the counter.
func (r *Registry) NextMeshIP() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	ip := mesh.MeshIPForIndex(r.data.NextIPIndex)
	r.data.NextIPIndex++
	return ip
}

// Tokens returns all pending (unused, non-expired) tokens.
func (r *Registry) Tokens() []mesh.Token {
	r.mu.RLock()
	defer r.mu.RUnlock()
	now := time.Now().UTC()
	out := []mesh.Token{}
	for _, t := range r.data.Tokens {
		if !t.Used && t.ExpiresAt.After(now) {
			out = append(out, t)
		}
	}
	return out
}

// AddToken stores a new join token.
func (r *Registry) AddToken(t mesh.Token) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data.Tokens = append(r.data.Tokens, t)
	return r.flush()
}

// ConsumeToken validates and marks a token as used.
// Returns an error if the token is invalid, expired, or already used.
func (r *Registry) ConsumeToken(value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	for i := range r.data.Tokens {
		t := &r.data.Tokens[i]
		if t.Value != value {
			continue
		}
		if t.Used {
			return fmt.Errorf("token already used")
		}
		if now.After(t.ExpiresAt) {
			return fmt.Errorf("token expired")
		}
		t.Used = true
		return r.flush()
	}
	return fmt.Errorf("invalid token")
}

// RevokeToken marks a token as used (invalidating it before first use).
func (r *Registry) RevokeToken(value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.data.Tokens {
		if r.data.Tokens[i].Value == value && !r.data.Tokens[i].Used {
			r.data.Tokens[i].Used = true
			return r.flush()
		}
	}
	return fmt.Errorf("token not found or already used")
}

func (r *Registry) flush() error {
	cp := r.data
	return r.store.Write(&cp)
}
