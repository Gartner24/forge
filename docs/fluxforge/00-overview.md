# FluxForge

FluxForge is a WireGuard-based private mesh network for the Forge suite. It connects multiple servers into a single encrypted private network so that Forge modules on different nodes can communicate without public exposure.

## Goals

- Connect any number of servers into a private mesh network
- Assign every node a stable private IP (`10.forge.x.x`)
- Handle NAT traversal for home servers behind firewalls
- Enable all other Forge modules to target specific nodes with `--node`
- Simple join flow: one command per new node

## Non-Goals

- Replacing a production firewall or security perimeter
- Routing all internet traffic through the mesh (not a VPN exit node)
- High availability on day one (planned for future release)

## Architecture

```
FluxController (one per mesh)
├── peer registry
├── join token issuance
├── admin HTTP API
└── NAT relay of last resort

FluxAgent (one per node)
├── WireGuard config manager
├── controller heartbeat
└── peer list sync
```

The controller is the coordination brain. It does **not** route production traffic — it only manages peer discovery and authentication. All actual traffic goes peer-to-peer over WireGuard tunnels.

## Mesh IP Addressing

Every node gets a stable private IP in the `10.forge.x.x` range assigned by the controller on join. Other modules reference nodes by this IP. The IP is stable — it does not change when the node restarts.

## NAT Traversal

WireGuard's UDP hole punching handles most NAT scenarios automatically. For cases where direct peer-to-peer fails (strict firewalls, double NAT), the FluxController acts as a relay of last resort (DERP-style). Once a direct path is established, traffic bypasses the controller.

## Security Model

- All traffic between nodes is encrypted by WireGuard (ChaCha20-Poly1305)
- Join tokens are 24-hour, single-use
- Three access roles: Owner, Admin, Node
- The controller node is the most sensitive node in the mesh — treat it accordingly

## Deep Documentation

- [Architecture](01-architecture.md) — component details and data flows
- [Join Flow](02-join-flow.md) — how nodes join and leave the mesh
- [Access Control](03-access-control.md) — roles and token management
- [NAT Traversal](04-nat-traversal.md) — how peer-to-peer connections are established
- [Operations](05-operations.md) — day-2 management
