# Forge TODOS

Deferred work identified during planning and review sessions. Ordered by priority.

---

## P1 — Blocks implementation

### [TODO-001] Update HearthForge CLI docs: flag-first design

**What:** Add flag specs to `docs/hearthforge/07-cli-reference.md` for all commands that currently use interactive TTY prompts (`add-dev` and `add-project`). Every required input must be expressible as a flag. `--output json` implies non-interactive mode.

**Why:** The MCP server calls the forge CLI via subprocess. Interactive prompts hang the subprocess. If the Go CLI is implemented before this is fixed, it will need to be refactored.

**Flags to add for `forge hearthforge add-dev`:**
- `--dev <id>` (developer id, e.g. alice)
- `--pubkey <key>` (SSH public key, inline or file path)
- `--project <id>` (project to associate)
- `--ide vscode|jetbrains|both` (already documented)
- `--recreate` (already documented)
- `--node <mesh-ip>` (already documented)

**Flags to add for `forge hearthforge add-project`:**
- `--id <slug>`
- `--repo <url>`
- `--branch <branch>` (default: main)
- `--stack node|python|mixed`
- `--port <port>` (container port to proxy)
- `--domain <domain>`
- `--cpus <n>` (resource defaults)
- `--memory <mb>`

**Design rule (applies to the entire suite):** `--output json` on any command must never prompt. All required inputs must be covered by flags before that flag is accepted.

**Effort:** S (human ~1h / CC ~5min)
**Depends on:** Nothing
**Blocks:** Go implementation of `forge hearthforge add-dev` and `forge hearthforge add-project`, MCP server `hearthforge_add_dev()` and `hearthforge_add_project()` tools.

---

### [TODO-002] Add `--async` flag to PenForge CLI docs

**What:** Add `--async` flag spec to `docs/penforge/05-cli-reference.md` for `forge penforge scan`. When `--async` is passed, the command returns immediately with a `scan_id` and estimated duration rather than blocking until the scan completes.

**Why:** PenForge scans run Nuclei + Nmap + testssl + dnsx + Trivy. A full scan can take 5-15 minutes. The MCP server has a 60-second subprocess timeout. The async flag is the contract the MCP `penforge_scan_start()` tool depends on.

**JSON output expected:**
```json
{
  "scan_id": "abc123",
  "target_id": "myapp",
  "started_at": "2026-04-21T10:00:00Z",
  "estimated_seconds": 600,
  "engines": ["nuclei", "nmap", "testssl"]
}
```

**Status polling:** `forge penforge scans show <scan_id> --output json` already exists in the CLI docs and handles polling. No changes needed there.

**Effort:** S (human ~30min / CC ~5min)
**Depends on:** Nothing
**Blocks:** MCP server `penforge_scan_start()` tool implementation. PenForge Go implementation of the scan runner.

---

### [TODO-003] Verify forge install auto-start behavior

**What:** Confirm whether `forge install <module>` automatically starts the module after
installation. The MCP `forge_install()` tool docstring asserts "The module starts automatically
after installation" but `docs/core/03-cli-reference.md` only says it "registers it with Forge
Core" -- no auto-start mentioned.

**Why:** The MCP tool gives the AI agent false information if auto-start doesn't happen.
The AI will believe a freshly installed module is running, then proceed to make module-specific
calls that fail with "module not running."

**What to do:** When the Go implementation of `forge install` is written, verify the behavior.
If auto-start is true, document it in `docs/core/03-cli-reference.md`. If false, remove the
auto-start claim from the `forge_install()` MCP tool docstring.

**Effort:** XS (human ~10min / CC ~2min)
**Depends on:** Go implementation of `forge install`
**Blocks:** Accurate MCP tool docstring for `forge_install()`

---

### [TODO-004] Audit trail for MCP tool calls

**What:** Log all MCP tool calls that read or modify state. At minimum: tool name, timestamp,
args (with secret values redacted). Write to `~/.forge/audit.log` or an append-only store.

**Why:** Currently `forge_secrets_get` retrieves plaintext secrets with zero audit trail.
An AI agent session that reads all secrets leaves no forensic evidence. The CEO plan says
"all MCP tool calls with side effects should be logged" -- but secret reads are as sensitive
as writes.

**Design option:** Log in `run_forge()` wrapper so every tool call is covered without
per-tool instrumentation. Redact args when the command matches `forge secrets get` or
`forge secrets set`. Alternatively, use FastMCP middleware.

**Acceptance criteria:** Every tool call logs `tool_name` + args (secrets values redacted) +
timestamp to `~/.forge/mcp-audit.log` in JSONL format. Implemented in `run_forge()` wrapper
before first production deployment. `forge secrets get` and `forge secrets set` args must
show the key but never the value.

**Effort:** S (human ~2h / CC ~10min)
**Depends on:** Nothing
**Blocks:** Nothing, but should ship before production use of forge_secrets_get

---

## P3 — Documentation and deployment

### [TODO-005] Document userns-remap Docker incompatibility

**What:** Add a note in `mcp-server/docker-compose.yml` and the MCP server README that
`network_mode: host` is incompatible with Docker user-namespace remapping (`userns-remap`
in `/etc/docker/daemon.json`). Recommend a preflight check.

**Why:** `userns-remap` is a common hardening configuration for Docker hosts. When enabled,
`network_mode: host` silently fails or is refused. A VPS with this configuration will
appear to have the MCP server running (container starts) but tools will fail because forge
can't reach host-level network resources.

**Preflight check to add:**
```bash
docker info | grep -i "userns" && echo "WARNING: userns-remap may break network_mode: host"
```

**Effort:** XS (human ~15min / CC ~3min)
**Depends on:** Nothing
**Blocks:** Nothing
