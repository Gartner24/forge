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
- Clone the project repo into `/workspace/<project>` (optional)
- Or provision an empty workspace and require the developer to clone

Recommendation:
- Bootstrap clone via `devctl` to standardize structure.

## Git authentication

Developers authenticate to GitHub from inside the container:
- recommended: SSH keys in the container user’s home (developer-managed)
- optional: GitHub CLI auth (`gh auth login`)

Forge should not require developer GitHub secrets stored on the host.

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

