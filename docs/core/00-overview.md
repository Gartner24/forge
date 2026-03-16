# Forge Core

Forge core is the CLI binary and built-in secrets store. It is the only component installed by default. Everything else is opt-in.

## Goals

- Single entry point for the entire suite (`forge <module> <command>`)
- Lightweight — one binary, one config file, nothing running
- Encrypted secrets store usable by all modules
- Module lifecycle management (install, uninstall, status, update)
- In-CLI alert banner when active high/critical alerts exist

## What Gets Installed

```
/usr/local/bin/forge         # the CLI binary (~15MB)
~/.forge/config.toml         # global configuration
~/.forge/secrets.age         # encrypted secrets file (created on first secret)
```

Nothing else. No containers, no daemons, no background processes.

## Architecture

```
forge CLI (Go, cobra)
├── cmd/
│   ├── install.go          # forge install <module>
│   ├── uninstall.go        # forge uninstall <module>
│   ├── status.go           # forge status
│   ├── secrets.go          # forge secrets set/get/list/delete
│   └── update.go           # forge update <module>
└── internal/
    ├── installer/           # module install/uninstall logic
    ├── store/               # age-encrypted secrets store
    └── alerts/              # in-CLI alert banner
```

## Module Interface

Every installed module must implement the Module interface defined in `shared/module/`:

```go
type Module interface {
    Name()    string
    Version() string
    Status()  ModuleStatus
    Start()   error
    Stop()    error
}
```

Forge core discovers installed modules by querying this interface — nothing is hardcoded. `forge status` shows whatever is registered.

## Global Config

`~/.forge/config.toml` stores:
- Installed modules and their versions
- Runtime paths for each module
- Global settings (log level, etc.)

Example:
```toml
[core]
log_level = "info"

[modules]
installed = ["smeltforge", "watchforge", "sparkforge"]

[paths]
data = "/opt/data"
infra = "/opt/infra/forge"
```

## Deep Documentation

- [Secrets Store](01-secrets.md) — encryption internals, namespacing, sync
- [Module Installer](02-installer.md) — how modules are installed and managed
- [CLI Reference](03-cli-reference.md) — all forge core commands
- [Config Reference](04-config.md) — ~/.forge/config.toml schema
