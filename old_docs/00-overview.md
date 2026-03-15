# Forge overview

Forge is a self-hosted development environment platform designed for a single VPS. It provisions isolated per-developer, per-project Docker workspaces and provides SSH access via a Rust gateway (jump host) so developers can use:


- Terminal SSH sessions
- VS Code / Cursor Remote-SSH

Forge also generates and manages reverse-proxy routing for developer preview domains, maintains a project registry, enforces access policies, and records audit logs.

## Goals

- Isolated developer workspaces (one container per developer per project)
- No host shell access for developers
- Compatibility with VS Code / Cursor Remote-SSH
- Repeatable provisioning and offboarding via a single admin CLI (`devctl`)
- Clean separation between:
  - infrastructure (`/opt/infra`)
  - production apps (`/opt/apps`)
  - runtime data (`/opt/data`)
- Strong default security posture (no privileged containers, no Docker socket mounts, non-root)

## Non-goals

- Kubernetes or multi-node scheduling
- Full IAM / SSO integration
- Multi-tenant security guarantees comparable to hardened VMs
- Replacing production CI/CD; Forge is for developer environments

## System summary

- Public edge is handled by a global Nginx proxy stack (`/opt/infra/proxy`).
- Developer environments run on a dedicated Docker network (`dev-web`) separate from production (`web`).
- Developers authenticate with SSH keys to the Rust gateway; the gateway routes them to their dev container’s sshd.
- Developer preview hostnames follow Pattern A: one hostname per developer per project (recommended).

## Key design decision: VS Code Remote-SSH compatibility

VS Code / Cursor Remote-SSH expects a normal SSH server endpoint with stable filesystem semantics and SFTP support. For that reason, each dev container runs OpenSSH (`sshd`). The gateway acts as a jump host rather than replacing sshd inside the container.

## Where to start

- Read `01-architecture.md` for the full VPS layout and data flows.
- Read `07-devctl.md` for provisioning and admin operations.
- Read `06-ssh-gateway.md` to understand routing, access policies, and logging.
