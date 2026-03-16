# shared/

The `shared/` directory is a Go module (`github.com/<user>/forge/shared`) imported by every Forge module. It contains no business logic — only the reusable infrastructure code that modules need to be consistent with each other: how they identify themselves to Core, how they send notifications, how they write audit logs, how they read secrets, how they parse registry files, and how they read global config.

Understanding `shared/` is the most important prerequisite for building a new module. Every module depends on it. Every integration between modules flows through it.

## Why a Shared Library Exists

Each Forge module is an independent Go binary with its own `go.mod`. They are intentionally decoupled at the code level — SmeltForge never imports HearthForge's packages and vice versa. But they still need to behave consistently: audit logs must have the same shape, secrets must be read the same way, notifications must arrive at SparkForge through the same interface.

The shared library solves this without coupling. Every module imports `shared/`, not each other. `shared/` is the only place where cross-cutting behaviour is defined.

**The rule for what belongs here:** if more than one module needs the same code, it goes in `shared/`. If only one module needs it, it stays in that module's `internal/`.

## Packages

```
shared/
├── go.mod
├── module/     # Module interface — how Core discovers and manages modules
├── notify/     # SparkForge client — how modules send notifications
├── audit/      # Audit log writer — how modules write structured audit events
├── secrets/    # Secrets client — how modules read encrypted secrets
├── registry/   # Registry parsers — how modules read/write JSON registry files
└── config/     # Config reader — how modules read global Forge config
```

Each package is documented in detail in its own file. Start with `module/` if you are building a new module, or `notify/` if you are adding alerting to an existing one.

## Deep Documentation

- [module/ — Module Interface](01-module.md)
- [notify/ — SparkForge Notification Client](02-notify.md)
- [audit/ — Audit Log Writer](03-audit.md)
- [secrets/ — Secrets Client](04-secrets.md)
- [registry/ — Registry File Parsers](05-registry.md)
- [config/ — Global Config Reader](06-config.md)
