# Architecture

This document describes the HearthForge VPS filesystem layout, responsibilities of each component, and how traffic and development sessions flow through the system.

## Canonical VPS Layout

```
/opt/
  infra/
    proxy/
      compose.yml
      Dockerfile
      nginx.conf
      conf.d/
        active/              # live vhosts (not committed)
        examples/            # example vhosts (committed)
      scripts/
      README.md

    forge/
      bin/
        forge                # forge CLI binary (replaces devctl)
      gateway/
        Cargo.toml
        src/
        keys/
        authorized_keys/
        logs/
      registry/
        projects.json
        devs.json
      templates/
        nginx-dev-vhost.conf.tmpl
        dev-compose.yml.tmpl
        sshd_config.tmpl
      docs/

  apps/
    hemis/
      frontend/
      backend/
    tiap/
      ...

  data/
    dev_workspaces/
      <project>/
        <dev>/
    backups/
    logs/
      proxy/
      gateway/
```

## Separation Rationale

- `/opt/infra` — infrastructure services and admin-only tooling
- `/opt/apps` — production application code and compose stacks
- `/opt/data` — runtime state, workspaces, backups, and logs (not committed to Git)

Developers never have host shell access. All developer interaction happens inside dev containers.

## Primary Components

### Proxy Stack (`/opt/infra/proxy`)

Public entry point for HTTP/HTTPS (ports 80/443). Terminates TLS for production and developer preview domains. Routes requests to upstream containers by Docker DNS name.

If SmeltForge is installed, HearthForge uses SmeltForge's Caddy instance for dev preview domains instead of managing its own Nginx — one proxy on the server instead of two.

### HearthForge Daemon (Go)

Thin orchestration layer for:
- Workspace provisioning and container lifecycle
- SSH key management
- Deploy key management (via `forge secrets`)
- Vhost config generation

### SSH Gateway (Rust)

Jump host for developer SSH access. Authenticates developers via public key, routes connections to the appropriate dev container's sshd. Never provides a host shell.

### Developer Environments

One container per (developer, project):
- Container name: `dev-<project>-<dev>`
- Workspace: `/workspace/<project>`
- Attaches only to `dev-web` network
- Runs OpenSSH (`sshd`)

## Data Flows

### Developer SSH Session Flow

1. Developer connects to gateway (`ssh.<domain>:2224`) as `<dev>-<project>`
2. Gateway authenticates via public key, resolves developer identity
3. Gateway routes to `dev-<project>-<dev>:22` on `dev-web`
4. Developer gets a shell inside their container

### Developer Preview HTTP Flow

1. User requests `https://<dev>-<project>.dev.domain.com`
2. Proxy matches `server_name`, routes to dev container's preview port on `dev-web`
3. Dev container serves frontend and backend as configured

### Production Deployment Flow

1. Developer pushes code to GitHub
2. CI builds Docker images, pushes to registry
3. VPS pulls images, restarts production services via SmeltForge or compose
4. Proxy routes production traffic to production containers on `web`

## Naming Conventions

| Thing | Convention | Example |
|---|---|---|
| Developer id | lowercase, alphanumeric + dashes | `santiago`, `ana-1` |
| Project id | lowercase, alphanumeric + dashes | `hemis`, `tiap` |
| Container | `dev-<project>-<dev>` | `dev-hemis-santiago` |
| Workspace path | `/opt/data/dev_workspaces/<project>/<dev>/` | |
| Dev hostname | `<dev>-<project>.dev.domain.com` | `santiago-hemis.dev.example.com` |
