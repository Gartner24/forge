# Dev containers

This document describes the dev container model, naming conventions, toolchain images, and security defaults.

## Provisioning unit

Forge provisions one container per `(developer, project)` pair.

Benefits:
- Isolation between developers
- Isolation between projects
- Clear offboarding and cleanup paths

## Naming conventions

- Container: `dev-<project>-<dev>`
  - example: `dev-hemis-santiago`
- Workspace path on host (if bind-mounted):
  - `/opt/data/dev_workspaces/<project>/<dev>/`
- Workspace inside container:
  - `/workspace/<project>`

## Toolchain images

Dev containers are built from a standard base per stack:
- Node stack: node + npm/yarn/pnpm + build essentials + git + sshd
- Python stack: python + pip/venv tools + build essentials + git + sshd
- Mixed stacks can be handled with a custom image or layered images

All toolchain dependencies are inside the container. The container should not rely on host-installed tooling.

## OpenSSH inside container

Each dev container runs OpenSSH server (`sshd`) to support:
- terminal SSH
- VS Code / Cursor Remote-SSH (requires sshd + SFTP subsystem)

## Workspace initialization

On first provision:
- Clone the project repo into `/workspace/<project>` (optional).
- Or provision an empty workspace and require the developer to clone.

Recommendation:
- Bootstrap clone via `devctl` to standardize structure.
- For private repos, use read-only **per-project** deploy keys managed by Forge for **initial clone only**, and let developers push with their own GitHub credentials from inside the container.
See `docs/13-github-deploy-keys.md` for a step-by-step GitHub tutorial.

## Git authentication and Remote-SSH onboarding

Developers authenticate to GitHub from inside the container:
- recommended: SSH keys in the container user’s home (developer-managed)
- optional: GitHub CLI auth (`gh auth login`)

Forge should not require developer GitHub secrets stored on the host.

Deploy keys created by Forge are:
- scoped per **project** and stored under `/opt/data/dev_workspaces/_deploy_keys/<project>/`
- mounted read-only into the dev container for bootstrap cloning
- intended for **read-only access** (do not use them for `git push`)

Developers must still:
- add their own SSH key to their GitHub account
- be granted write access to the repository on GitHub
- configure `user.name` / `user.email` inside the container so commits are attributed correctly.

## Developer SSH keys and gateway keys

Forge tracks developer SSH public keys in two places:

- **Gateway `authorized_keys` (canonical)**: `/opt/infra/forge/gateway/authorized_keys/<dev>.pub`
  - One file per developer id (e.g. `santiago.pub`).
  - Used by the Rust SSH gateway to decide whether to accept `publickey` auth for `<dev>-<project>`.
  - Managed by `devctl add-dev` and `devctl gateway-add-key`; do not edit manually.

- **Container `_keys` (derived)**: `/opt/data/dev_workspaces/_keys/<project>/<dev>/dev`
  - Written automatically by `devctl` when provisioning or updating a dev environment.
  - Mirrors the same key into the dev container’s `authorized_keys` so `sshd` inside the container accepts it.

The gateway store is the **source of truth**; container `_keys` entries are treated as derived and can be safely regenerated from the canonical gateway key.

Remote-SSH onboarding (developer workflow):
1. Admin runs `devctl add-dev` for the developer and project.
2. At the end of the command, `devctl` prints an SSH config snippet like:

   ```sshconfig
   Host <dev>-<project>
     HostName ssh.dev.<dev_base_domain>
     Port 2224
     User <dev>-<project>            # dev-project pair, e.g. santiago-tiap
     IdentityFile ~/.ssh/id_ed25519    # or the path to the developer's SSH key
     StrictHostKeyChecking accept-new
   ```

3. The admin sends this snippet to the developer.
4. On the developer’s machine:
   - paste the snippet into `~/.ssh/config`
   - ensure the `IdentityFile` path points to an SSH key that is added to their GitHub account
5. In Cursor / VS Code:
   - use the Remote-SSH extension to connect to the configured `Host` (e.g. `santiago-tiap`)
   - open `/workspace/<project>` inside the container and work as usual.

## Security defaults

Required:
- run as non-root user (`dev`)
- drop Linux capabilities (`cap_drop: [ALL]`) and add only what is needed
- no privileged containers
- never mount Docker socket
- attach only to `dev-web`
- apply CPU and memory limits

Optional hardening:
- read-only root filesystem
- restrict outbound traffic if needed
- avoid mounting host paths beyond workspace

## Resource limits

Set conservative defaults and allow overrides per project:
- memory limit (example: 1–2GB)
- CPU limit (example: 0.5–2 cores)

## Exposed ports

Dev containers should not publish ports to the host.
Use `expose` only (internal reachability on `dev-web`).

If dev preview HTTP is required:
- define preview ports in `projects.json`
- proxy routes `<dev>-<project>.dev.domain.com` to those ports

