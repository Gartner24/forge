# Forge

Forge is a self-hosted development environment platform for a single VPS. It provisions isolated per-developer, per-project Docker workspaces and provides SSH access via a Rust gateway (jump host) so developers can use terminal SSH and VS Code/Cursor Remote-SSH without accessing the host filesystem.

Forge also generates reverse-proxy routing for developer preview domains, manages a project registry, enforces access policies, and records audit logs.

## Repository contents

- `bin/devctl`
  Admin-only CLI to create, list, grant, revoke, disable, and delete developer environments.

- `gateway/`
  Rust SSH bastion/jump host (host keys, authorized keys, audit logs).

- `registry/`
  Project and developer registries (`projects.json`, `devs.json`).

- `templates/`
  Templates used to generate dev containers, SSH configuration, and proxy vhosts.

- `docs/`
  Architecture and operations documentation.

## Documentation index

Start here:

- `docs/00-overview.md`
- `docs/01-architecture.md`

Core system docs:

- `docs/02-threat-model.md`
- `docs/03-networking-and-routing.md`
- `docs/04-domains-and-tls.md`
- `docs/05-dev-containers.md`
- `docs/06-ssh-gateway.md`
- `docs/07-devctl.md`
- `docs/08-project-registry.md`

Operations:

- `docs/09-operations.md`
- `docs/10-offboarding.md`
- `docs/11-production-deploy.md`
- `docs/12-troubleshooting.md`
