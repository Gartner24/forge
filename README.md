# Forge

Forge is a self-hosted development environment platform for a single VPS. It provisions isolated per-developer, per-project Docker workspaces and provides SSH access through a Rust gateway (jump host) so developers can use terminal SSH and VS Code/Cursor Remote-SSH without accessing the host filesystem.

Forge also generates reverse-proxy routing for developer preview domains, manages a project registry, enforces access policies, and records audit logs.

## Repository contents

- `bin/devctl`
  Admin-only CLI to create, list, grant, revoke, disable, and delete developer environments.

- `gateway/`
  Rust SSH bastion/jump host.
  - `keys/`: persistent host keys
  - `authorized_keys/`: developer public keys managed by admin tooling
  - `logs/`: audit logs (append-only)

- `registry/`
  Configuration source of truth.
  - `projects.json`: onboardable projects (repo, stack, ports, defaults)
  - `devs.json`: developers + access mapping (managed by devctl)

- `templates/`
  Generated artifacts come from these templates.
  - `nginx-dev-vhost.conf.tmpl`: dev preview domain routing template
  - `dev-compose.yml.tmpl`: per-dev per-project container template
  - `sshd_config.tmpl`: container OpenSSH configuration for Remote-SSH

- `docs/`
  Architecture and operations documentation.
  - `architecture.md`
  - `operations.md`
  - `troubleshooting.md`

## VPS architecture (canonical)

Forge assumes the VPS follows this structure:

```text
/opt/
  infra/
    proxy/                         # Global reverse proxy (public edge)
      compose.yml
      Dockerfile
      nginx.conf
      conf.d/
        active/                    # live vhosts (not committed)
        examples/                  # templates/examples (committed)
      scripts/
      README.md

    forge/                         # Dev environment platform (admin-only)
      bin/
      gateway/
      registry/
      templates/
      docs/

  apps/                            # Production apps
    hemis/
    tiap/
    ...

  data/                            # Runtime data (not committed)
    dev_workspaces/
    backups/
    logs/
````

## Ownership and permissions

* `/opt/infra/**`: owned by `root:root`, `chmod 750` (admin-only)
* `/opt/apps/**`: owned by `root:root` (or a deploy user), not writable by devs
* `/opt/data/**`: owned by `root:root`, created by admin tooling only
* Developers should never have shell access to the host filesystem.

## Network design

Two external Docker networks:

* `web`: production network
* `dev-web`: development network

The global proxy attaches to both networks:

```text
nginx-proxy
  - attached: web, dev-web
prod containers
  - attached: web
dev containers
  - attached: dev-web
```

This prevents dev containers from seeing production service DNS names by default.

## Domain and TLS design

DNS:

* Production:

  * `hemis.domain.com` → VPS IP
  * `tiap.domain.com` → VPS IP
* Dev:

  * `*.dev.domain.com` → VPS IP (wildcard)

TLS:

* Production: per-domain Let’s Encrypt certificates
* Dev: wildcard certificate for `*.dev.domain.com` using DNS-01 (recommended via Cloudflare API)

## Routing design (Pattern A)

One hostname per developer per project:

* `santiago-hemis.dev.domain.com`
* `ana-tiap.dev.domain.com`

Each vhost routes to exactly one dev container:

* Container: `dev-<project>-<dev>` (example: `dev-hemis-santiago`)
* SSH: container runs `sshd` for VS Code/Cursor Remote-SSH
* Optional HTTP preview:

  * `/` → dev frontend port (e.g., 5173/3000)
  * `/api` → dev backend port (e.g., 5000)

## Dev container model

For each `(dev, project)` pair Forge provisions:

* A dedicated container: `dev-<project>-<dev>`
* A non-root `dev` user
* OpenSSH server configured for Remote-SSH
* A workspace at `/workspace/<project>` (bind mount or named volume)
* Security defaults:

  * non-root
  * no privileged mode
  * no docker socket mounts
  * drop Linux capabilities
  * CPU/memory limits
  * attach only to `dev-web`

Git workflow happens inside the container:

* developer authenticates to GitHub (SSH keys or `gh auth login`)
* clone/pull under `/workspace/<project>`
* push changes normally
* production deployments are handled by CI/CD (not direct `git pull` on the VPS)

## SSH access design (Rust gateway as jump host)

* Developers connect to the gateway only.
* Gateway authenticates by public key and enforces access policies.
* Gateway forwards SSH connections to the correct dev container on `dev-web`.

This supports:

* terminal SSH sessions
* VS Code/Cursor Remote-SSH

Access mapping:

* developer key → developer identity
* developer identity → allowed projects
* allowed project → target container(s)

## Admin CLI (devctl)

Core commands:

* `devctl add-project`
* `devctl add-dev`
* `devctl list-devs`
* `devctl grant <dev> <project>`
* `devctl revoke <dev> <project>`
* `devctl disable-dev <dev>`
* `devctl delete-dev <dev> <project>`

`add-dev` generates:

* dev container + workspace
* proxy vhost for dev domain
* registry updates
* key placement under `authorized_keys/`
* optional repo bootstrap clone

## Production deployment best practice

Production apps should be deployed via CI/CD:

* CI builds Docker images from GitHub
* push to a container registry (e.g., GHCR)
* VPS pulls images and restarts services
* global proxy routes to prod containers on `web`

## Logging and auditing

Forge logs at minimum:

* login attempts (success/fail)
* developer identity
* container target
* source IP
* timestamp

Store locally under:

* `/opt/infra/forge/gateway/logs/audit.log`


