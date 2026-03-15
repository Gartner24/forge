# PenForge

PenForge is the automated security scanning module for the Forge suite. It orchestrates open-source scan engines to detect vulnerabilities in web applications, networks, SSL configurations, containers, and DNS records.

## Goals

- Automated security scanning on every deploy (opt-in)
- Scheduled recurring scans
- Finding lifecycle management — never lose track of a finding
- Scope enforcement — never scan targets you did not register
- Full audit trail of every scan and finding
- Pluggable engine interface for future custom scanner (ForgeScanner)

## Non-Goals

- Manual penetration testing (PenForge is automated)
- Replacing a human security review
- Scanning targets outside your registered scope

## Security Philosophy

PenForge has two hard rules that cannot be configured away:

1. **Scope enforcement** — PenForge only scans targets explicitly registered by an admin. It validates scope before every scan and refuses to proceed if the target is not registered. There are no runtime overrides.

2. **Audit everything** — every scan trigger, engine run, finding, and finding state change is logged permanently. Logs are append-only.

## Architecture

```
PenForge Daemon
├── scan target registry (registry/targets.json)
├── scope enforcer
├── scan scheduler
│   ├── manual trigger
│   ├── SmeltForge post-deploy hook
│   └── cron schedule
├── engine runner
│   ├── Nuclei (Docker container)
│   ├── Nmap (Docker container)
│   ├── testssl.sh (Docker container)
│   ├── dnsx (Docker container)
│   └── Trivy (Docker container)
├── finding store
├── report generator
└── SparkForge alert client
```

All scan engines run as isolated Docker containers. They are pulled on demand and torn down immediately after each scan. Engines have no access to `dev-web` or other internal networks — only to declared scope targets.

## Engine Interface

Every engine implements a simple Go interface. This is the integration point for ForgeScanner when it is ready:

```go
type Engine interface {
    Name()    string
    Version() string
    Run(target Target) ([]Finding, error)
    Pull() error
}
```

## ForgeScanner Roadmap

PenForge is designed so Nuclei can be replaced by a custom scanner:

1. Use Nuclei templates — understand what each check does
2. Write custom Nuclei templates for your specific stack
3. Build ForgeScanner in Rust, implement the Engine interface
4. Register ForgeScanner alongside other engines

Everything else (scheduling, finding management, reports, alerts) stays unchanged.

## Deep Documentation

- [Architecture](01-architecture.md) — component details
- [Scan Target Configuration](02-targets.md) — targets.json schema reference
- [Engine Reference](03-engines.md) — all engines, what they scan, how to update them
- [Finding Lifecycle](04-findings.md) — states, transitions, alert deduplication
- [Report Format](05-reports.md) — report schema and output formats
- [Security Boundaries](06-security.md) — isolation model and hard rules
- [Operations](07-operations.md) — day-2 management
