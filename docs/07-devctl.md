# devctl (admin CLI)

This document describes the admin CLI responsibilities, commands, expected prompts, and generated artifacts.

## Purpose

`devctl` is an admin-only CLI that provisions and manages developer environments.

It must:
- read project definitions from `registry/projects.json`
- manage developer identities and access in `registry/devs.json`
- generate dev container compose definitions (from templates)
- generate proxy vhost configs for dev domains (from templates)
- trigger safe proxy reloads (after validation)

## Core commands (current implementation)

- `devctl add-project`
- `devctl add-dev`
- `devctl list-devs`
- `devctl gateway-add-key`
- `devctl delete-dev`

## add-project

`devctl add-project` creates or updates a project entry in `registry/projects.json` and can optionally generate a **project-level deploy key**.

Prompts:
- `project id` (slug, e.g. `hemis`; lower-case, alnum + dashes)
- `repo` (HTTPS or SSH URL)
- `default branch` (default `main`)
- `stack` (`node` / `python` / `mixed`)
- Preview settings:
  - enable preview (yes/no)
  - frontend dev port
  - backend dev port (0 to disable)
  - frontend path (default `/`)
  - backend path prefix (default `/api`)
- Resource defaults:
  - cpus (e.g. `1.0`)
  - memory (e.g. `2g`)

Deploy key (optional):
- After saving the project, `devctl` will ask:
  - \"Generate a GitHub deploy key for this project (read-only, for bootstrap clones)? [y/N]\"
- If you choose `y`:
  - A keypair is created under `/opt/data/dev_workspaces/_deploy_keys/<project>/id_ed25519(.pub)`.
  - The public key is printed so you can:
    - add it as a **read-only Deploy key** on GitHub at `Settings → Deploy keys` for the project repo.

## add-dev (provision)

Prompts:
- developer id (e.g. `santiago`)
- developer public key (paste or file)
- select project (1..n)

SSH key handling:

- The Rust SSH gateway’s `authorized_keys` directory is treated as the **canonical** store for developer SSH public keys:
  - If `/opt/infra/forge/gateway/authorized_keys/<dev>.pub` already contains a key, `add-dev` offers to reuse it and copies it into the new container’s `_keys` directory.
  - If no canonical key exists yet, `add-dev` prompts for a public key, writes it to the gateway file, and then populates `_keys` for that `(dev, project)`.
- This means you typically paste a developer’s SSH public key **once**; future `add-dev` runs for additional projects can reuse the same key.

Flags:
- `--recreate`:
  - Recreate the selected `(developer, project)` environment from scratch.
  - Uses the same cleanup logic as `delete-dev` to stop/remove the container, vhost, keys, and sshd config, and **purges the workspace directory** for that `(dev, project)` before re-provisioning.

Generates:
- dev container name: `dev-<project>-<dev>`
- workspace path:
  - host: `/opt/data/dev_workspaces/<project>/<dev>/`
  - container: `/workspace/<project>`
- dev hostname: `<dev>-<project>.dev.domain.com`
- proxy vhost config written to proxy live directory:
  - `/opt/infra/proxy/conf.d/active/dev/<dev>-<project>.conf` (or equivalent)
- registry updates:
  - `devs.json` developer record and access mapping

Repo bootstrap:
- If the selected project has a `repo` configured in `registry/projects.json`:
  - `devctl` starts the container and attempts to clone the repo into `/workspace/<project>` inside the container.
  - The workspace is only wiped when there is **no `.git` directory** present in `/workspace/<project>`.
- Public repos (default):
  - Cloned using the HTTPS URL from `projects.json`.
- Private repos with per-project deploy keys:
  - If a project deploy key exists under `/opt/data/dev_workspaces/_deploy_keys/<project>/id_ed25519(.pub)`:
    - The key directory is mounted read-only into the dev container at `/home/dev/.ssh/forge_deploy`.
    - `devctl` uses `GIT_SSH_COMMAND` with `/home/dev/.ssh/forge_deploy/id_ed25519` and an SSH repo URL (e.g. `git@github.com:owner/repo.git`) for cloning.
- Non-fatal behavior:
  - If cloning fails (auth, network, missing deploy key on Git host, etc.):
    - The environment is still considered provisioned (container, workspace, vhost, `devs.json`).
    - `devctl` prints stdout/stderr from the clone command and a brief \"next steps\" message.
    - Developers or admins can then fix credentials and run `git clone` / `git pull` manually from inside the container.

SSH config snippet:
- After a successful `add-dev` run, `devctl` prints an SSH config snippet that admins can send to the developer.
- The snippet is meant to be pasted into the developer's local `~/.ssh/config` and uses `ProxyJump`:
  - first hop: authenticate to Forge gateway (`ssh.<dev_base_domain>:2224`) as `<dev>-<project>`
  - second hop: SSH from gateway to `dev-<project>-<dev>:22` as `dev`
- Example:

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

- Example for developer `santiago` on project `tiap` with base domain `dev.qyvos.com`:

  ```sshconfig
  Host santiago-tiap-gw
    HostName ssh.dev.qyvos.com
    Port 2224
    User santiago-tiap
    IdentityFile ~/.ssh/id_ed25519
    StrictHostKeyChecking accept-new

  Host santiago-tiap
    HostName dev-tiap-santiago
    Port 22
    User dev
    ProxyJump santiago-tiap-gw
    IdentityFile ~/.ssh/id_ed25519
    StrictHostKeyChecking accept-new
  ```

Outputs to admin:
- DNS checklist (if wildcard not used)
- SSH config snippet for Cursor / VS Code Remote-SSH
- verification commands:
  - container running
  - ssh connection
  - nginx validation/reload status

## list-devs

Lists developers and the projects they are currently associated with, as recorded in `registry/devs.json`:

- Format: `<dev-id>: <project1>, <project2>, ...`

## gateway-add-key

`devctl gateway-add-key` manages the Rust SSH gateway's `authorized_keys` store.

- Command:
  - `devctl gateway-add-key --dev <dev> [--pubkey <path-or-inline>]`
- Behavior:
  - Normalizes `<dev>` via the same slug rules used elsewhere (lower-case, alnum + dashes).
  - Optionally validates that the developer exists in `registry/devs.json` (prints a warning if not found).
  - Reads the developer SSH public key:
    - from `--pubkey` if provided (either a `.pub` file path or an inline `ssh-ed25519 ...` / `ssh-rsa ...` line), or
    - interactively from stdin if `--pubkey` is omitted.
  - Appends the key to `/opt/infra/forge/gateway/authorized_keys/<dev>.pub`, creating the directory/file as needed (idempotent for identical lines).
  - For any projects currently associated with `<dev>` in `registry/devs.json`, also writes the same key into:
    - `/opt/data/dev_workspaces/_keys/<project>/<dev>/dev`
    - so existing dev containers accept the new key without reprovisioning.
- Typical use:
  - Used automatically by `devctl add-dev` to keep the gateway’s canonical key store up to date.
  - Can also be called manually to add extra keys or fix mistakes without reprovisioning containers; existing containers for that developer will be updated.

## delete-dev

`devctl delete-dev` is the single entrypoint for tearing down dev environments.

Flags:
- `--dev <dev>` (required): developer id.
- `--project <project>`: project id (for per-project delete).
- `--all-projects`: delete this developer from **all** projects.
- `--purge`:
  - With `--project`, delete the workspace directory for this `(dev, project)` after teardown.
- `--purge-all`:
  - With `--all-projects`, delete **all** workspace directories for this developer across all projects after teardown.

Per-project delete (soft offboarding):

- Command:
  - `devctl delete-dev --dev <dev> --project <project> [--purge]`
- Behavior:
  - Stops/removes the dev container `dev-<project>-<dev>` (compose down, then `docker rm -f` as fallback).
  - Removes the dev vhost for `<dev>-<project>.dev.<dev_base_domain>` and reloads nginx after validation.
  - Removes keys and `sshd_config` for that `(dev, project)` from:
    - `/opt/data/dev_workspaces/_keys/<project>/<dev>/`
    - `/opt/data/dev_workspaces/_sshd/<project>/<dev>/`
  - Updates `registry/devs.json`:
    - Removes `<project>` from the developer’s `projects` list.
    - If the developer has **no projects left**, sets `status = "disabled"` but keeps the developer record.
  - Workspace:
    - default: preserves `/opt/data/dev_workspaces/<project>/<dev>/`
    - with `--purge`: deletes `/opt/data/dev_workspaces/<project>/<dev>/`

Global delete (hard offboarding):

- Command:
  - `devctl delete-dev --dev <dev> --all-projects [--purge-all]`
- Behavior:
  - Looks up `<dev>` in `registry/devs.json`; if missing, prints a message and exits successfully.
  - For each project in the developer’s `projects` list:
    - Performs the same cleanup as per-project delete.
    - When run in this global mode, the final result is that the developer record is removed entirely from `devs.json`.
  - Workspace:
    - default: preserves all `/opt/data/dev_workspaces/<project>/<dev>/` directories for that developer.
    - with `--purge-all`: deletes all such workspaces and associated `_keys` / `_sshd` entries.

Argument rules:
- Exactly **one** of `--project` or `--all-projects` must be provided.
- `--purge` is only valid with `--project`.
- `--purge-all` is only valid with `--all-projects`.

## Safety requirements

- Validate Nginx config before reload:
  - `nginx -t` in proxy container
- Do not delete workspaces without explicit confirmation flags:
  - `--purge` for per-project deletes
  - `--purge-all` for global deletes
- Keep audit logs immutable and retained beyond workspace deletion.

