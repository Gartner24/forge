# Architecture

This document describes the canonical VPS filesystem layout, responsibilities of each area, and how traffic and development sessions flow through the system.

## Canonical VPS layout

```text
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
        devctl
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
````

## Separation rationale

* `/opt/infra` contains infrastructure services and admin-only tooling.
* `/opt/apps` contains production application code and production compose stacks.
* `/opt/data` contains runtime state, workspaces, backups, and logs. It is not committed to Git.


Developers should never have host shell access. All developer interaction happens inside dev containers.

## Primary components

### Proxy stack (`/opt/infra/proxy`)

* Public entry point for HTTP/HTTPS (ports 80/443).
* Terminates TLS for production domains and (optionally) developer preview domains.
* Routes requests to upstream containers by Docker DNS name.

### Forge (`/opt/infra/forge`)

* `devctl` provisions dev environments and updates registries.
* `gateway` is a Rust SSH jump host for developer access.
* `registry` defines projects and developer permissions.
* `templates` define how dev containers and proxy vhosts are generated.

### Developer environments

* One container per developer per project:

  * container name: `dev-<project>-<dev>`
  * workspace: `/workspace/<project>`
* Containers attach only to `dev-web`.
* Containers run `sshd` to support VS Code / Cursor Remote-SSH.

## Data flows

### Developer SSH session flow

1. Developer connects to the gateway via SSH (public key auth).
2. Gateway authenticates the key and resolves developer identity.
3. Gateway resolves allowed projects and routes the connection to the appropriate dev container sshd.
4. Developer gets a shell inside the container (terminal) and VS Code can establish Remote-SSH.

### Developer preview HTTP flow (optional, if you expose preview UIs)

1. User requests `https://<dev>-<project>.dev.domain.com`.
2. Global proxy matches `server_name` and routes to the dev container’s preview port(s) on `dev-web`.
3. The dev container serves frontend and backend as configured.

### Production deployment flow

1. Developer pushes code to GitHub.
2. CI builds Docker images and pushes to a registry.
3. VPS pulls images and restarts production services (compose).
4. Global proxy routes production traffic to production containers on `web`.

## Naming conventions

* Developer id: lowercase, alphanumeric plus dashes (e.g. `santiago`, `ana-1`)
* Project id: lowercase, alphanumeric plus dashes (e.g. `hemis`, `tiap`)
* Container: `dev-<project>-<dev>`
* Workspace path: `/opt/data/dev_workspaces/<project>/<dev>/`
* Dev hostname: `<dev>-<project>.dev.domain.com` (Pattern A)

