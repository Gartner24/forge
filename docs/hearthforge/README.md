# HearthForge

HearthForge provisions isolated per-developer, per-project Docker workspaces and provides SSH access via a Rust gateway. Developers connect using VS Code, Cursor, or JetBrains Gateway with Remote-SSH — no host shell access is ever granted.

> HearthForge is the original Forge project, migrated into the suite. The existing `devctl` CLI has been replaced by `forge hearthforge` subcommands. A `devctl` shell alias is created automatically for backwards compatibility.

## What It Does

- One isolated Docker container per (developer, project)
- SSH access via a Rust jump-host gateway
- VS Code / Cursor / JetBrains Gateway Remote-SSH support
- Automatic workspace bootstrap from Git repo
- Deploy key management for private repos
- Per-developer public key management
- **HTTP preview routing** — `<dev>-<project>.dev.<forge-domain>.com` for stable shareable preview URLs
- **SSH port forwarding** — `localhost:<port>` access to any dev server (npm run dev, APIs, etc.) with no proxy and no public exposure
- Full offboarding (soft and hard) with optional workspace purge

## Installation

```bash
forge install hearthforge
```

## Quickstart

```bash
# Register a project
forge hearthforge add-project

# Provision a developer environment
forge hearthforge add-dev

# List all developers and their projects
forge hearthforge list-devs
```

## Security Model

- Developers **never** receive host shell access
- All developer work happens inside containers
- Containers run as non-root user (`dev`)
- `cap_drop: [ALL]` — all Linux capabilities dropped
- No `--privileged` containers
- Docker socket is never mounted
- Containers attach only to `dev-web` network
- CPU and memory limits enforced per container

## SSH Connection Flow

1. Developer connects to gateway (`ssh.<domain>:2224`) as `<dev>-<project>`
2. Gateway authenticates via public key
3. Gateway routes connection to `dev-<project>-<dev>:22` on `dev-web`
4. Developer gets a shell inside their container

The gateway is transport-only. It never provides a shell on the host.

## Developer SSH Config

After `forge hearthforge add-dev`, an SSH config snippet is printed for the developer. It includes `LocalForward` directives for every port defined in the project's `dev_ports` configuration:

```sshconfig
Host <dev>-<project>-gw
  HostName ssh.<dev_base_domain>
  Port 2224
  User <dev>-<project>
  IdentityFile ~/.ssh/id_ed25519
  StrictHostKeyChecking accept-new

Host <dev>-<project>
  HostName dev-<project>-<dev>
  Port 22
  User dev
  ProxyJump <dev>-<project>-gw
  IdentityFile ~/.ssh/id_ed25519
  StrictHostKeyChecking accept-new
  # Dev server port forwarding — auto-generated from registry/projects.json
  LocalForward 3000 localhost:3000
  LocalForward 5000 localhost:5000
```

Paste into `~/.ssh/config`, then `ssh <dev>-<project>` gives a shell inside the container and automatically makes all configured dev ports available at `localhost:<port>` on the developer's local machine.

## Two Ways to Access a Running App

HearthForge provides two distinct access patterns — they serve different purposes:

| Feature | URL | Who can access | Use case |
|---|---|---|---|
| **Preview domain** | `https://<dev>-<project>.dev.<forge-domain>.com` | Anyone with the URL | Sharing builds, demo links, stable staging |
| **SSH port forwarding** | `localhost:<port>` on developer's machine | Only the authenticated developer | Active development, hot reload, WebSockets |

The preview domain requires the app to be running and the proxy to route to it. SSH port forwarding works with any dev server the moment it starts — no proxy configuration needed.

## IDE Support

```bash
forge hearthforge add-dev --ide vscode      # default — VS Code / Cursor
forge hearthforge add-dev --ide jetbrains   # JetBrains Gateway
forge hearthforge add-dev --ide both        # both
```

## CLI Reference

```bash
forge hearthforge add-project
forge hearthforge add-dev [--ide vscode|jetbrains|both] [--node <n>]
forge hearthforge list-devs
forge hearthforge gateway-add-key --dev <dev>
forge hearthforge delete-dev --dev <dev> --project <project> [--purge]
forge hearthforge delete-dev --dev <dev> --all-projects [--purge-all]
forge hearthforge migrate-secrets          # migrate _deploy_keys/ to forge secrets
```

## Offboarding

```bash
# Remove from one project (preserve workspace by default)
forge hearthforge delete-dev --dev alice --project myapp

# Remove from one project and delete workspace
forge hearthforge delete-dev --dev alice --project myapp --purge

# Remove from all projects
forge hearthforge delete-dev --dev alice --all-projects

# Remove from all projects and delete all workspaces
forge hearthforge delete-dev --dev alice --all-projects --purge-all
```

## Migrating from devctl

A `devctl` shell alias is created automatically on `forge install hearthforge`. Every `devctl` command maps directly:

| Old command | New command |
|---|---|
| `devctl add-project` | `forge hearthforge add-project` |
| `devctl add-dev` | `forge hearthforge add-dev` |
| `devctl list-devs` | `forge hearthforge list-devs` |
| `devctl gateway-add-key` | `forge hearthforge gateway-add-key` |
| `devctl delete-dev` | `forge hearthforge delete-dev` |

## Deep Documentation

See [`docs/hearthforge/`](./) for the full original documentation:
- [Overview](00-overview.md)
- [Architecture](01-architecture.md)
- [Threat Model](02-threat-model.md)
- [Networking and Routing](03-networking-and-routing.md)
- [Domains and TLS](04-domains-and-tls.md)
- [Dev Containers](05-dev-containers.md)
- [SSH Gateway](06-ssh-gateway.md)
- [CLI Reference](07-cli-reference.md)
- [Project Registry](08-project-registry.md)
- [Operations](09-operations.md)
- [Offboarding](10-offboarding.md)
- [Production Deploy](11-production-deploy.md)
- [Troubleshooting](12-troubleshooting.md)
- [GitHub Deploy Keys](13-github-deploy-keys.md)

## Installation

```bash
forge install hearthforge
```

## Quickstart

```bash
# Register a project
forge hearthforge add-project

# Provision a developer environment
forge hearthforge add-dev

# List all developers and their projects
forge hearthforge list-devs
```

## Security Model

- Developers **never** receive host shell access
- All developer work happens inside containers
- Containers run as non-root user (`dev`)
- `cap_drop: [ALL]` — all Linux capabilities dropped
- No `--privileged` containers
- Docker socket is never mounted
- Containers attach only to `dev-web` network
- CPU and memory limits enforced per container

## SSH Connection Flow

1. Developer connects to gateway (`ssh.<domain>:2224`) as `<dev>-<project>`
2. Gateway authenticates via public key
3. Gateway routes connection to `dev-<project>-<dev>:22` on `dev-web`
4. Developer gets a shell inside their container

The gateway is transport-only. It never provides a shell on the host.

## Developer SSH Config

After `forge hearthforge add-dev`, an SSH config snippet is printed for the developer:

```sshconfig
Host <dev>-<project>-gw
  HostName ssh.<dev_base_domain>
  Port 2224
  User <dev>-<project>
  IdentityFile ~/.ssh/id_ed25519
  StrictHostKeyChecking accept-new

Host <dev>-<project>
  HostName dev-<project>-<dev>
  Port 22
  User dev
  ProxyJump <dev>-<project>-gw
  IdentityFile ~/.ssh/id_ed25519
  StrictHostKeyChecking accept-new
```

Paste into `~/.ssh/config`, then connect with `ssh <dev>-<project>` or open in VS Code/Cursor via Remote-SSH.

## IDE Support

```bash
forge hearthforge add-dev --ide vscode      # default — VS Code / Cursor
forge hearthforge add-dev --ide jetbrains   # JetBrains Gateway
forge hearthforge add-dev --ide both        # both
```

## CLI Reference

```bash
forge hearthforge add-project
forge hearthforge add-dev [--ide vscode|jetbrains|both] [--node <n>]
forge hearthforge list-devs
forge hearthforge gateway-add-key --dev <dev>
forge hearthforge delete-dev --dev <dev> --project <project> [--purge]
forge hearthforge delete-dev --dev <dev> --all-projects [--purge-all]
forge hearthforge migrate-secrets          # migrate _deploy_keys/ to forge secrets
```

## Offboarding

```bash
# Remove from one project (preserve workspace by default)
forge hearthforge delete-dev --dev alice --project myapp

# Remove from one project and delete workspace
forge hearthforge delete-dev --dev alice --project myapp --purge

# Remove from all projects
forge hearthforge delete-dev --dev alice --all-projects

# Remove from all projects and delete all workspaces
forge hearthforge delete-dev --dev alice --all-projects --purge-all
```

## Migrating from devctl

A `devctl` shell alias is created automatically on `forge install hearthforge`. Every `devctl` command maps directly:

| Old command | New command |
|---|---|
| `devctl add-project` | `forge hearthforge add-project` |
| `devctl add-dev` | `forge hearthforge add-dev` |
| `devctl list-devs` | `forge hearthforge list-devs` |
| `devctl gateway-add-key` | `forge hearthforge gateway-add-key` |
| `devctl delete-dev` | `forge hearthforge delete-dev` |

## Deep Documentation

See [`docs/hearthforge/`](./) for the full original documentation:
- [Overview](00-overview.md)
- [Architecture](01-architecture.md)
- [Threat Model](02-threat-model.md)
- [Networking and Routing](03-networking-and-routing.md)
- [Domains and TLS](04-domains-and-tls.md)
- [Dev Containers](05-dev-containers.md)
- [SSH Gateway](06-ssh-gateway.md)
- [CLI Reference](07-cli-reference.md)
- [Project Registry](08-project-registry.md)
- [Operations](09-operations.md)
- [Offboarding](10-offboarding.md)
- [Production Deploy](11-production-deploy.md)
- [Troubleshooting](12-troubleshooting.md)
- [GitHub Deploy Keys](13-github-deploy-keys.md)
