# Forge Docs Update — MCP Server Prerequisites

This file instructs Claude Code to make four documentation changes required before
the Forge MCP Server can be built. Read this file fully before making any changes.
Then read every file referenced. Then make the changes. Then open a PR.

---

## Background

A Forge MCP Server is being added to the suite. It is a Python/FastMCP sidecar service
that wraps the Go `forge` CLI binary via subprocess. AI agents (Claude, etc.) connect to
it and issue infrastructure commands like "deploy my app" or "scan this target".

Because the MCP server calls the CLI via subprocess, two hard constraints apply:

1. Every CLI command the MCP server calls must be non-interactive — no TTY prompts.
   `--output json` on any command must never prompt. All required inputs must be flags.
2. Long-running commands (like PenForge scans) must support `--async` so the subprocess
   returns immediately with a scan_id instead of blocking for 5-15 minutes.

These constraints are documented in `jjstack/TODOS.md`. Read that file now.
Also read `jjstack/ceo-plans/2026-04-21-forge-mcp-server.md`.
Also read `jjstack/eng-plans/2026-04-21-forge-mcp-server.md`.

---

## Change 1 — docs/hearthforge/07-cli-reference.md

**File to edit:** `docs/hearthforge/07-cli-reference.md`
**Read first:** `jjstack/TODOS.md` (TODO-001 section)

### What to change

The `forge hearthforge add-dev` and `forge hearthforge add-project` commands currently
use interactive TTY prompts. This blocks the MCP server. Add full flag specifications
for both commands so every input can be passed non-interactively.

### Design rule to document at the top of the CLI reference

Add this note near the top of the file, after the intro paragraph:

> **Machine-readable mode:** Passing `--output json` on any command disables all
> interactive prompts. Every required input must be supplied as a flag. This is the
> mode used by the Forge MCP Server and any other automated callers.

### Flags to add for `forge hearthforge add-dev`

Current docs show prompts. Replace with this flag table and example:

```
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

### Flags to add for `forge hearthforge add-project`

```
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

### Keep all existing content

Do not remove or replace any existing content in the file. Add the new flag tables
and the machine-readable note. The interactive prompt flow can remain documented
as the "human-friendly" alternative, but the flags must be the primary documented
interface.

---

## Change 2 — docs/core/03-cli-reference.md

**File to edit:** `docs/core/03-cli-reference.md`
**Read first:** `jjstack/TODOS.md` (TODO-003 section)

### What to change

The `forge install` section currently says it "registers the module with Forge Core"
but does not say whether it auto-starts the module. The MCP server needs a definitive
answer because it will tell the AI agent what to do after installation.

**Decision: `forge install` does NOT auto-start the module.**

This is consistent with the lightweight core design (no background daemons, nothing
running until explicitly started). The user must run `forge start <module>` after
installing.

### Edit the `forge install` section

Find the `forge install` section and add this paragraph after the existing description:

> **Auto-start behaviour:** `forge install` registers and configures the module but
> does not start it. Run `forge start <module>` after installation to bring the module
> online. `forge status` will show the module as `stopped` until started.

Also add a usage example showing the two-step flow:

```bash
# Install then start
forge install smeltforge
forge start smeltforge

# Verify
forge status
```

### Keep all existing content

Do not remove any existing content. Only add the auto-start clarification and example.

---

## Change 3 — docs/penforge/ CLI reference

**File to find:** Look for the PenForge CLI reference. It is likely at
`docs/penforge/05-cli-reference.md` or similar. Check what files exist under
`docs/penforge/` and find the one that documents `forge penforge scan`.
**Read first:** `jjstack/TODOS.md` (TODO-002 section)

### What to change

Add `--async` flag documentation to the `forge penforge scan` command.

### Add this to the `forge penforge scan` section

```bash
forge penforge scan <target-id>
forge penforge scan <target-id> --async
forge penforge scan <target-id> --async --output json
```

| Flag | Description |
|------|-------------|
| `--async` | Return immediately with a `scan_id` instead of blocking until completion. Use `forge penforge scans show <scan_id>` to poll for results. |
| `--output json` | Machine-readable JSON output. Required when using `--async` programmatically. |

### JSON output when `--async` is passed

When `--async --output json` is used, the command returns immediately with:

```json
{
  "scan_id": "abc123",
  "target_id": "myapp",
  "started_at": "2026-04-21T10:00:00Z",
  "estimated_seconds": 600,
  "engines": ["nuclei", "nmap", "testssl", "dnsx"]
}
```

### Add a note explaining why this exists

> **Why async?** PenForge scans run multiple engines (Nuclei, Nmap, testssl.sh, dnsx)
> and can take 5–15 minutes. Synchronous scans are fine for interactive use. For
> automated callers (CI pipelines, the Forge MCP Server), use `--async` to get a
> `scan_id` immediately and poll with `forge penforge scans show <scan_id> --output json`.

### Keep all existing content

Do not remove any existing content. Add the `--async` flag to the existing scan
command documentation.

---

## Change 4 — docs/02-project-structure.md

**File to edit:** `docs/02-project-structure.md`

### What to change

The `mcp-server/` directory is not listed in the monorepo layout. Add it.

### In the Root Layout section

Find the directory tree under "Root Layout" and add `mcp-server/` as a top-level
entry, after `penforge/` and before the closing of the tree:

```
├── mcp-server/             # Python/FastMCP MCP server (AI agent control plane)
```

### Add a new section after the existing module descriptions

Add this section after the HearthForge description and before any closing content:

---

### mcp-server/

The Forge MCP Server is a Python sidecar service that wraps the `forge` CLI binary
via subprocess and exposes all Forge functionality as MCP tools. AI agents (Claude,
etc.) connect via Streamable HTTP and issue natural-language infrastructure commands.

It is not a Go module and is not part of `go.work`. It has its own `pyproject.toml`
and `Dockerfile`.

```
mcp-server/
├── pyproject.toml
├── Dockerfile
├── docker-compose.yml
├── .mcp.json
├── VERSION
└── src/forge_mcp/
    ├── __init__.py
    ├── server.py              # FastMCP entry point, lifespan, /health
    └── tools/
        ├── __init__.py
        ├── utils.py           # run_forge() subprocess wrapper
        ├── core.py            # forge_* tools (status, install, secrets, config)
        ├── smeltforge.py      # smeltforge_* tools
        ├── watchforge.py      # watchforge_* tools
        ├── sparkforge.py      # sparkforge_* tools
        ├── fluxforge.py       # fluxforge_* tools
        ├── hearthforge.py     # hearthforge_* tools
        └── penforge.py        # penforge_* tools
```

**Platform constraint:** `network_mode: host` is Linux-only. Docker Desktop (Mac/Windows)
is not supported. This service is VPS-only by design.

**Security:** The MCP port (default 8008) must be firewalled to localhost only.
`forge_secrets_get` returns plaintext values — restrict port access accordingly.

---

## PR Instructions

After making all four changes:

1. Stage all changed files
2. Create branch: `docs/mcp-server-prerequisites`
3. Commit message:
   ```
   docs: add MCP server prerequisites and flag-first CLI specs

   - hearthforge: add flag-first design for add-dev and add-project (TODO-001)
   - core: clarify forge install does not auto-start modules (TODO-003)
   - penforge: add --async flag to forge penforge scan (TODO-002)
   - project-structure: add mcp-server/ to monorepo layout
   ```
4. Push branch and open PR to `main`
5. PR title: `docs: add MCP server prerequisites and flag-first CLI specs`
6. PR description:
   ```
   Resolves TODO-001, TODO-002, and TODO-003 from jjstack/TODOS.md.

   These changes are required before the Forge MCP Server can be built.
   The MCP server calls the forge CLI via subprocess and cannot handle
   interactive prompts or blocking long-running commands.

   Changes:
   - docs/hearthforge/07-cli-reference.md: flag-first design for add-dev and add-project
   - docs/core/03-cli-reference.md: forge install does not auto-start, must run forge start separately
   - docs/penforge/[cli-reference]: --async flag for forge penforge scan with JSON output shape
   - docs/02-project-structure.md: mcp-server/ added to monorepo layout
   ```
