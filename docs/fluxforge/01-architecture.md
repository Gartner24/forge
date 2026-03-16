# Architecture

FluxForge creates a private WireGuard mesh network connecting all Forge servers. Every node gets a stable private IP in `10.forge.0.0/16`. Other modules address nodes by mesh IP — no public IP exposure needed for inter-node communication.

## Two Components

### FluxController

Runs on exactly one node (the designated controller). Responsibilities:

- Peer registry — authoritative list of all nodes, their mesh IPs, public keys, and roles
- Join token issuance and validation
- Peer announcement — when a new node joins, the controller pushes the updated peer list to all agents
- NAT relay — when direct peer-to-peer WireGuard fails due to NAT, the controller relays traffic until a direct path is established
- Admin API — handles `forge fluxforge` commands

### FluxAgent

Runs on every node, including the controller node. Responsibilities:

- Registers with the controller at startup
- Receives peer list updates and applies them to the local WireGuard interface
- Sends periodic heartbeats to the controller
- Configures WireGuard routes for all mesh peers

## Resilience

If the controller goes down:
- Existing WireGuard tunnels between agents remain operational — peer-to-peer traffic never routes through the controller
- No new nodes can join until the controller recovers
- Existing nodes cannot be revoked until the controller recovers
- All module operations on existing nodes continue unaffected

Full HA (Raft consensus across 3 controllers) is planned for v2.

## Mesh IP Assignment

IPs are assigned from `10.forge.0.0/16` in sequence:
- First node (controller): `10.forge.1.1`
- Subsequent nodes: `10.forge.2.1`, `10.forge.3.1`, etc.

IPs are stable — a node keeps its IP across restarts. IPs are only reassigned if a node is explicitly revoked and re-joined.

## WireGuard Implementation

FluxForge uses `golang.zx2c4.com/wireguard` (WireGuard-go userspace implementation). If the host kernel has native WireGuard support (kernel 5.6+), FluxForge uses it instead for better performance. The fallback to userspace is automatic.

All traffic between mesh nodes is encrypted with ChaCha20-Poly1305 via WireGuard.

## Port Requirements

| Port | Protocol | Direction | Purpose |
|---|---|---|---|
| 7777 | TCP | Inbound on controller | FluxController API |
| 51820 | UDP | Inbound on all nodes | WireGuard tunnels |

The WireGuard port (51820) must be reachable from all other mesh nodes. The controller API port (7777) must be reachable from any server that will join the mesh.
