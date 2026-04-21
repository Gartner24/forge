# CLI Reference — Forge Core

## Global Flags

| Flag | Description |
|---|---|
| `--verbose` | Show detailed output |
| `--output json` | Machine-readable JSON output |
| `--yes` | Skip confirmation prompts |
| `--no-banner` | Suppress the active alert banner |
| `--node <mesh-ip>` | Target a specific FluxForge mesh node (requires FluxForge) |

---

## forge init

Initialises Forge on a new server. Creates `~/.forge/`, generates the age identity key, and writes `config.toml` with the provided domain.

```bash
forge init --domain dev.example.com
```

Must be run once before installing any modules. Safe to run again — reports current state if already initialised.

---

## forge install

```bash
forge install <module>
forge install hearthforge
forge install hearthforge --node 10.forge.2.1  # install on a mesh node
```

> **Auto-start behaviour:** `forge install` registers and configures the module but
> does not start it. Run `forge start <module>` after installation to bring the module
> online. `forge status` will show the module as `stopped` until started.

```bash
# Install then start
forge install smeltforge
forge start smeltforge

# Verify
forge status
```

---

## forge uninstall

```bash
forge uninstall <module>
forge uninstall <module> --purge   # also delete all module data
forge uninstall <module> --yes     # skip confirmation
```

---

## forge update

```bash
forge update <module>
forge update --all                 # update every installed module
```

---

## forge start / stop

```bash
forge start <module>
forge stop <module>
```

---

## forge status

```bash
forge status                       # all modules
forge status --output json
```

Output columns: `MODULE`, `VERSION`, `STATE`, `SINCE`.

---

## forge secrets

```bash
forge secrets set <key> <value>
forge secrets set <key> <value> --sync     # replicate to mesh nodes

forge secrets get <key>

forge secrets list
forge secrets list --prefix smeltforge.hemis

forge secrets delete <key>
```

Key format: `<module>.<scope>.<name>` — e.g. `smeltforge.hemis.DB_URL`.

---

## forge logs

```bash
forge logs <module>                # tail live logs
forge logs <module> --lines 200   # last N lines
forge logs <module> --since 1h    # last hour
```

---

## forge config

```bash
forge config show                  # print current config.toml
forge config get forge.domain
forge config set forge.domain dev.newdomain.com
```
