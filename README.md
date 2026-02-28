## Target filesystem layout (professional, scalable)

Use this as the canonical structure on the VPS:

```text
/opt/
  infra/
    proxy/                         # Global reverse proxy (public edge)
      compose.yml
      Dockerfile
      nginx.conf

      conf.d/                      # vhosts: prod + dev routing
        hemis.conf
        tiap.conf
        dev/                       # generated dev vhosts
          santiago-hemis.conf
          ana-tiap.conf
      scripts/                     # admin-only helpers
        nginx-validate-reload.sh
      README.md

    dev-platform/                  # Dev environment platform (admin-only)
      bin/
        devctl                      # admin CLI (create/list/revoke/delete dev envs)
      gateway/                      # Rust SSH bastion/jump host
        Cargo.toml
        src/
        keys/                       # host keys (persistent)
        authorized_keys/            # per-dev public keys (admin managed)
        logs/                       # audit logs (append-only)
      registry/
        projects.json               # list of onboardable projects
        devs.json                   # developer identities + access mapping
      templates/
        nginx-dev-vhost.conf.tmpl
        dev-compose.yml.tmpl
        sshd_config.tmpl
      docs/
        architecture.md
        operations.md
        troubleshooting.md

    observability/ (optional later) # prometheus/grafana/loki, etc.

  apps/                            # Production apps (each can be multi-service)
    hemis/
      frontend/
      backend/
      README.md
    tiap/
      ...
    ssh-app/                       # if still used (or move under infra/ if shared)

  data/                            # Data owned by infrastructure (not code)
    dev-workspaces/                # bind mounts OR named-volume backups exports
      hemis/
        santiago/
        ana/
      tiap/
        ana/
    backups/
    logs/
      proxy/
      gateway/
```

### Ownership / permissions

* `/opt/infra/**`: owned by `root:root`, `chmod 750` (admin-only)
* `/opt/apps/**`: owned by `root:root` (or a deploy user), not writable by devs
* `/opt/data/**`: owned by `root:root`, with subfolders created by admin scripts only
* Developers should **never** have shell access to the host filesystem.

---

## Network design

Create and keep two Docker networks:

* `web` = production network (current apps)
* `dev-web` = dev network (all dev containers)

`nginx-proxy` joins **both** so it can route to prod + dev.

```text
nginx-proxy
  - attached: web, dev-web
prod containers
  - attached: web
dev containers
  - attached: dev-web
```

This prevents dev containers from seeing prod service DNS names by default.

---

## Domain + TLS design

### DNS

* Production:

  * `hemis.domain.com` → VPS IP
  * `tiap.domain.com` → VPS IP
* Dev:

  * `*.dev.domain.com` → VPS IP (wildcard)

### TLS

* Production: existing Let’s Encrypt per-domain (your current flow).
* Dev: **wildcard certificate** for `*.dev.domain.com` using DNS-01 (Cloudflare API recommended).

This avoids rate limits and makes dev onboarding fast.

---

## Routing design (Pattern A)

One hostname per developer per project:

* `santiago-hemis.dev.domain.com`
* `ana-tiap.dev.domain.com`

Generated vhost files live here:

* `/opt/infra/proxy/conf.d/dev/<dev>-<project>.conf`

Each vhost routes to exactly one dev container:

* upstream: `dev-<project>-<dev>:22` (SSH) is not via nginx
* upstream: `dev-<project>-<dev>:<appPort>` (HTTP) is via nginx if you also expose preview web UI

Typical dev preview routing (HTTP):

* `/` → dev frontend port (e.g., 5173/3000)
* `/api` → dev backend port (e.g., 5000)

---

## Dev container design (per dev per project)

For each `(dev, project)` pair you provision:

**Container name**

* `dev-<project>-<dev>` (example: `dev-hemis-santiago`)

**What’s inside**

* Toolchain image for the project stack (Node/Python/etc.)
* A non-root `dev` user
* `openssh-server` configured for Remote-SSH
* Workspace at `/workspace/<project>` (bind mount or named volume)

**Security defaults**

* Run as non-root
* No privileged mode
* No docker socket mounts
* Drop caps (`cap_drop: [ALL]` and add only if needed)
* CPU/Mem limits
* Read-only root filesystem if feasible (optional)
* Only attached to `dev-web`

**Git workflow inside container**

* Dev uses SSH keys or `gh auth login`
* Clone/pull occurs inside `/workspace/<project>`
* They push to GitHub as usual
* Production deploy is separate (CI/CD)

---

## SSH access design (Rust gateway as Jump Host)

### Why jump host (Mode 1)

* Works for both:

  * terminal (`ssh`)
  * VS Code/Cursor Remote-SSH
* VS Code expects a normal SSH server endpoint (your dev container’s `sshd`).

### How it works

* Developers SSH to the gateway only.
* Gateway authenticates by public key and enforces access policies.
* Gateway forwards the SSH connection to the correct dev container (internal `dev-web`).

**Result**

* Dev never gets a host shell.
* Dev ends up inside the container’s sshd session.
* VS Code can install its server normally.

### Access policy mapping

* Dev key → dev identity
* Dev identity → allowed projects
* Allowed project → container target(s)

---

## Project registry (multi-project support)

Maintain `/opt/infra/dev-platform/registry/projects.json` with entries like:

* `id`: `hemis`
* `repo`: GitHub repo URL
* `stack`: `node`, `python`, `node+python`, etc.
* `default_branch`: `main`
* `dev_ports`: frontend/back ports
* `startup`: commands (optional)

This allows:

* `devctl add-dev` → select project `1..n` → provision container.

Developers can have:

* access to one project (one container)
* or multiple projects (multiple containers, recommended)

---

## Admin CLI (`devctl`) responsibilities

Admin-only tool that supports:

### Core commands

* `devctl add-project`
  Adds to `projects.json`

* `devctl add-dev`
  Prompts:

  * dev name/id
  * public SSH key (paste or file path)
  * project selection (1..n)
    Outputs:
  * dev domain (hostname)
  * SSH config snippet for VS Code/Cursor
  * status checks to run

* `devctl list-devs`


* `devctl grant <dev> <project>`

* `devctl revoke <dev> <project>`

* `devctl disable-dev <dev>` (key disable)

* `devctl delete-dev <dev> <project>` (remove container + vhost + volumes optionally)

### What `add-dev` generates

* Dev container + volume
* Nginx vhost file in `proxy/conf.d/dev/`
* Gateway registry update (`devs.json`)
* Key storage under gateway `authorized_keys/`
* Optional: bootstrap clone of repo into workspace

### What remains manual

* Only DNS record creation if you don’t use wildcard.
* With wildcard `*.dev.domain.com`, even that becomes unnecessary per dev.

---

## Production deployment best practice

For prod apps in `/opt/apps/<project>`:

* CI builds Docker images from GitHub
* Push to registry (e.g., GHCR)
* VPS pulls images and restarts compose
* `nginx-proxy` routes to prod containers on `web`

No direct `git pull` on production folders.

---

## Operational logging & auditing

Minimum logging:

* gateway login attempts (success/fail)
* dev identity
* container target
* source IP
* timestamp

Store:

* `/opt/infra/dev-platform/gateway/logs/audit.log`
  and optionally ship later.

