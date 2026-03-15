# FluxForge

FluxForge is a WireGuard-based private mesh network. It connects multiple servers into a single private network so Forge modules on different nodes can communicate securely without public exposure.

> **FluxForge is always optional.** Every other module works on a single VPS with no mesh networking. Install FluxForge only when you need to connect multiple servers.

## What It Does

- Creates an encrypted WireGuard mesh between all your servers
- Assigns every node a stable private IP in the `10.forge.x.x` range
- Handles NAT traversal for home servers behind firewalls
- Provides join token-based node enrollment
- Enables all other modules to target specific nodes with `--node <name>`

## Installation

```bash
forge install fluxforge
```

## Setup

```bash
# On your primary VPS — initialize the mesh (run once)
forge fluxforge init
# → starts FluxController
# → prints a join token

# On each additional server — join the mesh
forge fluxforge join --controller <ip>:7777 --token <token>
```

Join tokens are 24-hour, single-use.

## CLI Reference

```bash
forge fluxforge init                          # initialize mesh on this node (first run)
forge fluxforge join --controller <ip> --token <token>  # join an existing mesh
forge fluxforge status                        # show mesh status and all nodes
forge fluxforge nodes                         # list all nodes and their mesh IPs
forge fluxforge token create                  # generate a new join token
forge fluxforge token revoke <token>          # revoke a join token
forge fluxforge add-admin <node>              # promote a node to admin role
forge fluxforge revoke <node>                 # remove a node from the mesh
forge fluxforge set-controller <node>         # move the controller to a different node
```

## Access Control

| Role | Permissions |
|---|---|
| Owner | Full access. Created on `forge fluxforge init`. Cannot be removed. |
| Admin | Add/remove nodes, generate join tokens, view mesh status. |
| Node | Join mesh only. No management access. |

## Architecture

FluxForge has two binaries:

- **FluxController** — coordination server. One per mesh. Manages peer registry, token issuance, admin API. Acts as a DERP-style relay of last resort for NAT traversal. Does NOT route production traffic.
- **FluxAgent** — runs on every node. Registers with controller, syncs peer list, configures WireGuard locally.

The controller can live on any node. Default is the first node where `forge fluxforge init` was run. Reassign with `forge fluxforge set-controller <node>`.

## Resilience

Currently: basic. If the controller goes down, existing tunnels stay up but new nodes cannot join until it recovers.

Planned: full HA via Raft consensus across 3 controller nodes.

## Deep Documentation

See [`fluxforge/docs/`](../../fluxforge/docs/) for:
- Detailed architecture
- NAT traversal internals
- WireGuard configuration reference
- Controller HA design (future)
