# Dev Containers

This document describes the dev container model, naming conventions, toolchain images, and security defaults.

## Provisioning Unit

HearthForge provisions one container per `(developer, project)` pair.

Benefits:
- Isolation between developers
- Isolation between projects
- Clear offboarding and cleanup paths

## Naming Conventions

| Thing | Pattern | Example |
|---|---|---|
| Container | `dev-<project>-<dev>` | `dev-hemis-santiago` |
| Host workspace | `/opt/data/dev_workspaces/<project>/<dev>/` | |
| Container workspace | `/workspace/<project>` | `/workspace/hemis` |

## Toolchain Images

Dev containers are built from a standard base per stack:

- **Node stack** — node + npm/yarn/pnpm + build essentials + git + sshd + Docker CLI
- **Python stack** — python + pip/venv + build essentials + git + sshd + Docker CLI
- **Mixed stacks** — custom image or layered images

All toolchain dependencies are inside the container. The container does not rely on host-installed tooling.

> The Docker CLI is available inside dev containers, but the Docker daemon is not exposed (no Docker socket mount). Developers can use Docker CLI commands that target a remote daemon if needed.

## OpenSSH Inside Container

Each dev container runs OpenSSH server (`sshd`) to support:
- Terminal SSH
- VS Code / Cursor Remote-SSH (requires sshd + SFTP subsystem)
- JetBrains Gateway (requires sshd + JetBrains backend agent)

## Workspace Initialization

On first provision, `forge hearthforge add-dev` attempts to clone the project repo into `/workspace/<project>`. The workspace is only wiped when there is no `.git` directory present.

**Public repos:** cloned using the HTTPS URL from `projects.json`.

**Private repos with deploy keys:** if a deploy key exists in `forge secrets` for the project, it is mounted read-only into the container at `/home/dev/.ssh/forge_deploy` and used for the bootstrap clone.

If cloning fails for any reason (auth, network, missing deploy key on Git host), the environment is still provisioned. The admin can fix credentials and run `git clone` manually from inside the container.

## Developer SSH Keys and Gateway Keys

Forge tracks developer SSH public keys in two places:

**Gateway `authorized_keys` (canonical):** `/opt/infra/forge/gateway/authorized_keys/<dev>.pub`
- One file per developer id
- Used by the SSH gateway to authenticate developers
- Managed by `forge hearthforge add-dev` and `forge hearthforge gateway-add-key`
- Do not edit manually

**Container `_keys` (derived):** `/opt/data/dev_workspaces/_keys/<project>/<dev>/dev`
- Written automatically by HearthForge when provisioning
- Mirrors the gateway key into the dev container's `authorized_keys`
- Treated as derived — can be safely regenerated from the canonical gateway key

## Remote-SSH Onboarding

1. Admin runs `forge hearthforge add-dev`
2. Admin sends the printed SSH config snippet to the developer
3. Developer pastes snippet into `~/.ssh/config`
4. Developer connects: `ssh <dev>-<project>` or opens in VS Code/Cursor/JetBrains

See [SSH Gateway](06-ssh-gateway.md) for the full golden path.

## Security Defaults

Required:
- Run as non-root user (`dev`)
- Drop Linux capabilities (`cap_drop: [ALL]`)
- No privileged containers
- Never mount Docker socket
- Attach only to `dev-web`
- Apply CPU and memory limits

Optional hardening:
- Read-only root filesystem
- Restrict outbound traffic if needed
- Avoid mounting host paths beyond workspace

## Resource Limits

Set conservative defaults per project in `registry/projects.json`:

```json
"resources": {
  "cpus": "1.0",
  "memory": "2g"
}
```

## Exposed Ports

Dev containers must not publish ports to the host. Use `expose` only (internal reachability on `dev-web`).

For HTTP preview, define preview ports in `projects.json`. The proxy routes `<dev>-<project>.dev.domain.com` to those ports.
