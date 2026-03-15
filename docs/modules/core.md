# Forge Core

Forge core is the CLI binary and built-in secrets store. It is the only thing installed by default. No background daemons, no containers, no extra processes.

## What It Does

- Installs, uninstalls, and manages the lifecycle of all other modules
- Provides `forge secrets` — an encrypted key-value store used by all modules
- Shows the in-CLI alert banner when active high/critical alerts exist
- Reads and writes the global Forge config at `~/.forge/config.toml`

## Installation

```bash
just core/install
# → builds the forge binary
# → installs to /usr/local/bin/forge
# → creates ~/.forge/config.toml
```

## forge secrets

The secrets store is a core built-in — not a separate module. It uses age encryption and stores everything in a single encrypted file.

```bash
forge secrets set DATABASE_URL "postgres://..."
forge secrets get DATABASE_URL
forge secrets list
forge secrets delete DATABASE_URL
```

Secrets are namespaced per module:
```
smeltforge.myapp.DATABASE_URL
hearthforge.deploykeys.myproject
fluxforge.controller.token
```

When FluxForge is installed, add `--sync` to replicate a secret across all mesh nodes:
```bash
forge secrets set DATABASE_URL "postgres://..." --sync
```

## Module Management

```bash
forge install <module>       # install a module
forge uninstall <module>     # uninstall a module
forge status                 # show all installed modules and state
forge update <module>        # update a module
```

## In-CLI Alert Banner

When SparkForge is installed and active alerts exist, every `forge` command is prepended with:

```
⚠  2 active alerts:
   [HIGH]     myapp-web is DOWN since 10:04 (14m ago)
   [CRITICAL] SSL cert for myapp.com expires in 2 days

Run 'forge sparkforge alerts' to view details.
```

This clears automatically when incidents resolve.

## Storage Footprint

- `forge` binary: ~15MB
- Secrets file: negligible (one encrypted file)
- Config file: negligible

## Source

`core/` — see [Project Structure](../02-project-structure.md) for layout details.
