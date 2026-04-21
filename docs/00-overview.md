# Overview

Forge is a self-hosted infrastructure suite designed for solo developers and small teams. It provisions isolated dev environments, deploys applications, monitors uptime, sends alerts, and scans for security vulnerabilities — all from a single CLI on your own servers.

## What Forge Replaces

| Forge Module | Replaces |
|---|---|
| FluxForge | Tailscale / ZeroTier |
| SmeltForge | Dokploy / Coolify / Render |
| WatchForge | Uptime Kuma / BetterUptime |
| SparkForge | Gotify / basic PagerDuty |
| HearthForge | GitHub Codespaces / Coder |
| PenForge | Manual Nuclei runs / paid scanners |

## What Forge Is Not

- A Kubernetes platform or multi-node scheduler
- A replacement for production CI/CD pipelines
- A full IAM or SSO solution
- A multi-tenant platform with VM-level security guarantees

## Core Design Principles

**Modular by default.**
Install only what you need. A developer who only wants remote dev workspaces installs HearthForge. Someone who only wants deployments installs SmeltForge. No module is ever a hard dependency of another.

**FluxForge is always optional.**
Every module works fully on a single VPS with zero mesh networking. FluxForge only adds the ability to connect multiple servers together. If you have one server, you never need FluxForge.

**Lightweight core.**
`forge` installs as a single Go binary with a small encrypted secrets file. No background daemons, no Docker containers, nothing running until you install a module.

**CLI-first.**
The web dashboard is a future feature. Everything in Forge is designed to be fully operable from the terminal. This makes Forge scriptable, SSH-friendly, and CI-compatible from day one.

**Audit everything.**
Every module writes to an append-only audit log. Logs are never truncated or deleted by Forge. This is a hard rule, not a configuration option.

## Single VPS vs Multi-VPS

Forge works on both. The experience is the same — the only difference is whether FluxForge is installed.

**Single VPS (no FluxForge needed):**
```bash
forge install smeltforge
forge smeltforge add --project myapp
forge smeltforge deploy --project myapp
```

**Multi-VPS (with FluxForge):**
```bash
# Primary VPS — initialize the mesh
forge install fluxforge
forge fluxforge init

# Each additional VPS — join the mesh
forge fluxforge join --controller <ip>:7777 --token <token>

# Deploy to any node
forge smeltforge deploy --project myapp --node vps2
```

The module commands are identical. FluxForge just makes `--node` work.

## Module Install/Uninstall

```bash
forge install <module>       # install and start a module
forge uninstall <module>     # stop and remove a module
forge status                 # show all installed modules and their state
forge update <module>        # update a module to latest version
```

Each module self-registers with Forge core on install. `forge status` discovers what is installed by querying the module registry — nothing is hardcoded.

## Where to Go Next

- [Architecture](01-architecture.md) — full system design, components, and data flows
- [Project Structure](02-project-structure.md) — monorepo layout and conventions
- [Contributing](03-contributing.md) — how to set up a dev environment and contribute
- [Module docs](modules/) — deep documentation for each module
