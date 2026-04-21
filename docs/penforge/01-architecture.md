# Architecture

PenForge is the security scanning orchestrator. It runs configurable scan engines as isolated Docker containers against registered targets, aggregates findings, manages their lifecycle, and integrates with SmeltForge for post-deploy scanning.

## Components

### PenForge Daemon (Go)

The core process. Responsibilities:
- Scan target registry management
- Scan orchestration — starts engine containers, collects output, tears them down
- Finding storage and delta detection (new vs known findings)
- Finding lifecycle management (acknowledged, accepted, fixed)
- Scheduled scan execution
- Post-deploy scan hook (called by SmeltForge)
- Scan report generation
- SparkForge integration for critical finding alerts

### Scan Engines (Docker containers)

Each engine is a third-party tool packaged as a Docker image. Engines are started fresh for each scan and torn down immediately after. They never persist between scans.

| Engine | Purpose | Image |
|---|---|---|
| Nuclei | Web vulnerability scanning (OWASP Top 10, CVEs) | `projectdiscovery/nuclei` |
| Nmap | Port and service detection | `instrumentisto/nmap` |
| testssl.sh | SSL/TLS configuration analysis | `drwetter/testssl.sh` |
| dnsx | DNS reconnaissance | `projectdiscovery/dnsx` |
| Trivy | Container image CVE scanning | `aquasec/trivy` |

All engines implement the same `Engine` interface — PenForge never contains engine-specific code outside the engine implementation file.

## Scan Flow

```
forge penforge scan --target <id>
        ↓
Scope validation — target must be in registry/targets.json
        ↓
Pull engine images (if not cached)
        ↓
Start each engine in isolated Docker container
  (no access to dev-web or web networks — scope-restricted only)
        ↓
Each engine runs against declared scope targets
        ↓
Collect findings from all engines
        ↓
Delta detection — compare against previous scan
        ↓
Tear down all engine containers
        ↓
Write findings to finding store
        ↓
Generate scan report
        ↓
SparkForge: alert on new CRITICAL or HIGH findings
        ↓
Write audit log entry
```

## File Layout

```
penforge/
├── registry/
│   └── targets.json        # registered scan targets
└── data/
    ├── findings.json       # finding store (all findings + lifecycle state)
    ├── scans/              # per-scan result files
    │   └── <scan-id>/
    │       ├── nuclei.json
    │       ├── nmap.json
    │       └── report.md
    └── audit.log
```

## Integration Points

- **SmeltForge**: calls PenForge's post-deploy hook after each successful deploy
- **SparkForge**: receives CRITICAL/HIGH alerts for new findings
- **Forge Core**: `--node` flag allows scanning from any mesh node
