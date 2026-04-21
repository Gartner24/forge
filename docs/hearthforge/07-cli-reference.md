# CLI Reference

This document describes all `forge hearthforge` subcommands, prompts, flags, and generated artifacts.

> The original `devctl` CLI has been replaced by `forge hearthforge`. A `devctl` shell alias is created automatically on install for backwards compatibility. See the [migration table](#migration-from-devctl) at the bottom of this page.

> **Machine-readable mode:** Passing `--output json` on any command disables all
> interactive prompts. Every required input must be supplied as a flag. This is the
> mode used by the Forge MCP Server and any other automated callers.

## add-project

Creates or updates a project entry in `registry/projects.json`. Optionally generates a project-level deploy key.

```bash
forge hearthforge add-project \
  --id <slug> \
  --repo <url> \
  [--branch <branch>] \
  [--stack node|python|mixed] \
  [--port <port>] \
  [--domain <domain>] \
  [--cpus <n>] \
  [--memory <mb>] \
  [--output json]
```

| Flag | Required | Description |
|------|----------|-------------|
| `--id <slug>` | Yes | Project slug. Lowercase, alphanumeric, dashes. e.g. `myapp`. |
| `--repo <url>` | Yes | Git repository URL. HTTPS or SSH format. |
| `--branch <branch>` | No | Default branch for clones and deploys. Default: `main`. |
| `--stack node\|python\|mixed` | No | Toolchain stack. Default: `node`. |
| `--port <port>` | No | Container port the dev server runs on, e.g. `3000`. |
| `--domain <domain>` | No | Preview domain for the project. |
| `--cpus <n>` | No | CPU limit for the dev container. Default: `1`. |
| `--memory <mb>` | No | Memory limit in MB. Default: `512`. |
| `--output json` | No | Return machine-readable JSON. Disables all prompts. |

**Interactive mode (human-friendly alternative):**

```bash
forge hearthforge add-project
```

**Prompts:**
- Project id (slug, e.g. `hemis` — lowercase, alphanumeric + dashes)
- Repo (HTTPS or SSH URL)
- Default branch (default: `main`)
- Stack (`node` / `python` / `mixed`)
- Preview settings: enable preview, frontend dev port, backend dev port, frontend path, backend path prefix
- Resource defaults: cpus, memory

**Deploy key (optional):**

After saving the project, you are asked: `Generate a GitHub deploy key for this project? [y/N]`

If `y`:
- A keypair is stored in `forge secrets` under `hearthforge.deploykeys.<project>`
- The public key is printed for you to add as a read-only Deploy key on GitHub at `Settings → Deploy keys`

See [GitHub Deploy Keys](13-github-deploy-keys.md) for the full tutorial.

---

## add-dev

Provisions a new developer environment for a (developer, project) pair.

```bash
forge hearthforge add-dev \
  --dev <id> \
  --pubkey <key-or-path> \
  --project <project-id> \
  [--ide vscode|jetbrains|both] \
  [--recreate] \
  [--node <mesh-ip>] \
  [--output json]
```

| Flag | Required | Description |
|------|----------|-------------|
| `--dev <id>` | Yes | Developer identifier, e.g. `alice`. Lowercase, alphanumeric, dashes. |
| `--pubkey <key-or-path>` | Yes | SSH public key string or path to a `.pub` file. |
| `--project <id>` | Yes | Project slug to associate the dev with, e.g. `myapp`. |
| `--ide vscode\|jetbrains\|both` | No | IDE to optimise the workspace for. Default: `vscode`. |
| `--recreate` | No | Tear down and recreate the workspace if it already exists. |
| `--node <mesh-ip>` | No | FluxForge mesh IP of the node to provision on. Default: local node. |
| `--output json` | No | Return machine-readable JSON. Disables all prompts. |

**Interactive mode (human-friendly alternative):**

```bash
forge hearthforge add-dev
```

**Prompts:**
- Developer id (e.g. `santiago`)
- Developer public key (paste or file path — reused if already in gateway store)
- Select project (1..n)

**SSH key handling:**
- If `gateway/authorized_keys/<dev>.pub` already exists, `add-dev` offers to reuse it
- If not, prompts for a public key, writes it to the gateway store, then populates `_keys` for this `(dev, project)`
- Malformed or placeholder keys are rejected

**Flags:**
- `--ide vscode` — default. Prints VS Code / Cursor SSH config snippet
- `--ide jetbrains` — installs JetBrains backend agent in container. Prints JetBrains connection details
- `--ide both` — installs both
- `--recreate` — tears down the existing `(dev, project)` environment and re-provisions from scratch
- `--node <node>` — (requires FluxForge) provision the container on a specific mesh node

**Generates:**
- Container: `dev-<project>-<dev>`
- Workspace: `/opt/data/dev_workspaces/<project>/<dev>/` (host), `/workspace/<project>` (container)
- Dev hostname: `<dev>-<project>.dev.domain.com`
- Proxy vhost config for the preview domain
- `devs.json` developer record and access mapping

**Output to admin:**
- DNS checklist (if wildcard not used)
- SSH config snippet for developer — includes `LocalForward` lines for every port in the project's `dev_ports` configuration, enabling automatic SSH port forwarding to `localhost:<port>` on the developer's machine
- Verification commands

---

## list-devs

Lists all developers and the projects they are associated with.

```bash
forge hearthforge list-devs
```

Format: `<dev-id>: <project1>, <project2>, ...`

---

## gateway-add-key

Adds or updates an SSH public key for an existing developer in the gateway's canonical store.

```bash
forge hearthforge gateway-add-key --dev <dev> [--pubkey <path-or-inline>]
```

**Behavior:**
- Reads the key from `--pubkey` (file path or inline `ssh-ed25519 ...` string) or from stdin if omitted
- Appends the key to `gateway/authorized_keys/<dev>.pub` (idempotent for identical lines)
- For all projects currently associated with `<dev>`, also updates `_keys/<project>/<dev>/dev`

**Typical uses:**
- Run automatically by `add-dev` to keep the gateway store up to date
- Run manually to add a second key (e.g. new laptop) without reprovisioning

---

## delete-dev

Tears down developer environments. Single entry point for all offboarding.

```bash
# Remove from one project
forge hearthforge delete-dev --dev <dev> --project <project> [--purge]

# Remove from all projects
forge hearthforge delete-dev --dev <dev> --all-projects [--purge-all]
```

**Per-project delete:**
- Stops/removes container `dev-<project>-<dev>`
- Removes dev vhost and reloads proxy
- Removes `_keys/<project>/<dev>/` and `_sshd/<project>/<dev>/`
- Removes project from developer's `projects` list in `devs.json`
- If no projects remain, sets `status = "disabled"` (keeps the developer record)
- Default: preserves workspace at `/opt/data/dev_workspaces/<project>/<dev>/`
- With `--purge`: deletes the workspace

**Global delete:**
- Performs per-project delete for every project in the developer's list
- Removes the developer record entirely from `devs.json`
- Default: preserves all workspaces
- With `--purge-all`: deletes all workspaces and associated `_keys` / `_sshd` entries

**Argument rules:**
- Exactly one of `--project` or `--all-projects` must be provided
- `--purge` is only valid with `--project`
- `--purge-all` is only valid with `--all-projects`

---

## migrate-secrets

Migrates existing `_deploy_keys/` directories into `forge secrets`. Run once when upgrading from standalone Forge to the Forge suite.

```bash
forge hearthforge migrate-secrets
```

- Reads all keys from `/opt/data/dev_workspaces/_deploy_keys/<project>/id_ed25519`
- Imports each into `forge secrets` under `hearthforge.deploykeys.<project>`
- Removes the plaintext key files
- Prints a migration report

---

## Safety Requirements

- Validate Nginx/Caddy config before reload
- Do not delete workspaces without explicit `--purge` or `--purge-all` flags
- Keep audit logs immutable and retained beyond workspace deletion

---

## Migration from devctl

A `devctl` shell alias is created automatically on `forge install hearthforge`. All flags and prompts are identical.

| Old command | New command |
|---|---|
| `devctl add-project` | `forge hearthforge add-project` |
| `devctl add-dev` | `forge hearthforge add-dev` |
| `devctl list-devs` | `forge hearthforge list-devs` |
| `devctl gateway-add-key` | `forge hearthforge gateway-add-key` |
| `devctl delete-dev` | `forge hearthforge delete-dev` |
