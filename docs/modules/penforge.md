# PenForge

PenForge is the automated security scanning module. It orchestrates open-source scan engines to detect vulnerabilities in your web applications, networks, SSL configurations, containers, and DNS records.

> **Two hard rules:** (1) PenForge only scans explicitly registered targets — never arbitrary URLs. (2) Every scan, trigger, and finding is logged permanently.

## What It Does

- Web application scanning (OWASP Top 10 via Nuclei)
- Network and port scanning (Nmap)
- SSL/TLS vulnerability checks (testssl.sh)
- DNS reconnaissance (dnsx)
- Container image CVE detection (Trivy)
- Post-deploy security scans (integrated with SmeltForge)
- Finding lifecycle management (new → acknowledged → fixed → verified)
- Structured scan reports

## Installation

```bash
forge install penforge
```

## Quickstart

```bash
# Register a scan target
forge penforge add --target https://myapp.com --name myapp-full

# Run a scan
forge penforge scan --target myapp-full

# View the report
forge penforge report --target myapp-full

# View open findings
forge penforge findings
```

## Scan Engines

All engines run as isolated Docker containers — never directly on the host. Containers are torn down immediately after each scan.

| Engine | Scans for |
|---|---|
| Nuclei | Web app vulnerabilities, OWASP Top 10, CVE templates |
| Nmap | Network/port scanning, service detection |
| testssl.sh | SSL/TLS vulnerabilities and configuration |
| dnsx | DNS reconnaissance, subdomain enumeration |
| Trivy | Container image CVE detection |

## Severity Levels

| Level | Example |
|---|---|
| critical | Remote code execution, auth bypass |
| high | SQL injection, exposed admin panel |
| medium | Outdated TLS, missing security headers |
| low | Information disclosure, weak cipher |
| info | Open ports, tech fingerprinting |

## Scan Triggers

```bash
# Manual
forge penforge scan --target myapp-full

# Single engine only
forge penforge scan --target myapp-full --engine nuclei

# Post-deploy (configure per project in SmeltForge)
# → SmeltForge calls PenForge after a successful deploy
# → If critical findings: SparkForge fires HIGH alert
# → Deploy is NOT rolled back (warn-only)

# Scheduled (configure in target definition)
# → Results compared to previous scan
# → SparkForge only alerts on NEW findings
```

## Finding Lifecycle

```bash
# Acknowledge a finding (working on it)
forge penforge finding acknowledge <id> --reason "investigating"

# Accept risk (known issue, won't fix)
forge penforge finding accept <id> --reason "mitigated by WAF"

# Mark as fixed (triggers re-scan to verify)
forge penforge finding verify <id>
```

Acknowledged and accepted findings do not re-trigger alerts unless severity increases. This prevents alert fatigue from recurring known issues.

## Security Boundaries

- Scan engines run **only** in Docker containers — never directly on the host
- Scan containers have no network access to `dev-web` or internal networks
- Containers are torn down immediately after each scan
- Reports are local only — nothing leaves the server unless explicitly exported
- Scan scope is enforced before every scan — no runtime overrides
- PenForge API requires admin-level Forge token

## CLI Reference

```bash
forge penforge init
forge penforge add --target <url> --name <n>
forge penforge scan --target <id> [--engine <engine>]
forge penforge list
forge penforge report --target <id>
forge penforge report --target <id> --scan <scan-id>
forge penforge findings
forge penforge finding acknowledge <id> --reason <reason>
forge penforge finding accept <id> --reason <reason>
forge penforge finding verify <id>
forge penforge engines
forge penforge engines update
```

## ForgeScanner — Future Custom Engine

PenForge is designed so Nuclei can be replaced by a custom scanner (ForgeScanner) when ready. Any engine just needs to implement the Engine interface:

```go
type Engine interface {
    Name()    string
    Version() string
    Run(target Target) ([]Finding, error)
    Pull() error
}
```

Learning path:
1. Use Nuclei templates, understand what each check does
2. Write custom Nuclei templates for your specific stack
3. Build ForgeScanner in Rust, implement the Engine interface, swap it in

## Deep Documentation

See [`penforge/docs/`](../../penforge/docs/) for:
- Scan target configuration reference
- Engine interface specification
- Finding schema reference
- Report format documentation
- Scope enforcement internals
