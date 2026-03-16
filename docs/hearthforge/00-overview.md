# HearthForge Overview

HearthForge is a self-hosted remote development environment platform. It provisions isolated per-developer, per-project Docker workspaces and provides SSH access via a Rust gateway (jump host) so developers can use:

- Terminal SSH sessions
- VS Code / Cursor Remote-SSH
- JetBrains Gateway

HearthForge also generates and manages reverse-proxy routing for developer preview domains, maintains a project registry, enforces access policies, and records audit logs.

> HearthForge was the original standalone Forge project. It has been migrated into the Forge suite as a module. The `devctl` CLI is now `forge hearthforge`. A `devctl` alias is created automatically for backwards compatibility.

## Goals

- Isolated developer workspaces (one container per developer per project)
- No host shell access for developers
- Compatibility with VS Code / Cursor / JetBrains Remote-SSH
- Repeatable provisioning and offboarding via a single CLI (`forge hearthforge`)
- Clean separation between:
  - infrastructure (`/opt/infra`)
  - production apps (`/opt/apps`)
  - runtime data (`/opt/data`)
- Strong default security posture (no privileged containers, no Docker socket mounts, non-root)

## Non-Goals

- Kubernetes or multi-node scheduling (use FluxForge + SmeltForge for multi-node)
- Full IAM / SSO integration
- Multi-tenant security guarantees comparable to hardened VMs
- Replacing production CI/CD

## System Summary

- Public edge is handled by a global Nginx proxy stack (`/opt/infra/proxy`) or SmeltForge's Caddy instance if installed.
- Developer environments run on a dedicated Docker network (`dev-web`) separate from production (`web`).
- Developers authenticate with SSH keys to the Rust gateway; the gateway routes them to their dev container's sshd.
- Developer preview hostnames follow Pattern A: one hostname per developer per project.

## Key Design Decision: VS Code Remote-SSH Compatibility

VS Code / Cursor / JetBrains Remote-SSH expect a normal SSH server endpoint with stable filesystem semantics and SFTP support. For that reason, each dev container runs OpenSSH (`sshd`). The gateway acts as a jump host rather than replacing sshd inside the container.

## Where to Start

- Read [Architecture](01-architecture.md) for the full VPS layout and data flows
- Read [CLI Reference](07-devctl.md) for provisioning and admin operations
- Read [SSH Gateway](06-ssh-gateway.md) to understand routing, access policies, and logging
- Read [Threat Model](02-threat-model.md) for security boundaries and non-negotiable rules
