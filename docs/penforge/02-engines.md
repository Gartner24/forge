# Scan Engines

## Engine Interface

Every engine implements:

```go
type Engine interface {
    Name()    string
    Version() string
    Pull(ctx context.Context) error
    Run(ctx context.Context, target ScanTarget) ([]Finding, error)
}
```

No engine-specific code appears outside its implementation file. Adding or replacing an engine requires only creating a new file that implements this interface — no changes to the orchestration layer.

## Nuclei

Runs Nuclei's template library against web targets. Detects OWASP Top 10 vulnerabilities, exposed admin panels, misconfigurations, and known CVEs in web applications.

- Output format: JSON (`-jsonl`)
- Scope: only HTTP/HTTPS endpoints declared in the target scope
- Templates: community templates + any custom templates mounted from `penforge/templates/nuclei/`

## Nmap

Port and service detection against IP addresses and hostnames in the target scope.

- Output format: XML (parsed by PenForge)
- Scope: only IPs and hostnames declared in the target scope
- Default flags: `-sV -sC --script=vuln` (service version + default scripts + vuln scripts)

## testssl.sh

Analyses SSL/TLS configuration for domains in scope. Detects weak cipher suites, deprecated protocol versions (TLS 1.0/1.1, SSLv3), certificate issues, and known TLS vulnerabilities (BEAST, POODLE, Heartbleed, etc.).

- Output format: JSON (`--jsonfile`)
- Scope: only HTTPS domains declared in the target scope

## dnsx

DNS reconnaissance against domains in scope. Enumerates DNS records, detects misconfigured or dangling DNS entries, and identifies subdomain takeover opportunities.

- Output format: JSON (`-json`)
- Scope: only domains declared in the target scope

## Trivy

Container image CVE scanning. Scans the Docker images currently running for registered SmeltForge projects.

- Output format: JSON (`--format json`)
- Scope: images for containers associated with the scan target

## Updating Engine Images

```bash
forge penforge engines update            # pull latest for all engines
forge penforge engines update --engine nuclei
forge penforge engines list              # show current image versions
```

PenForge pins engine images by digest when pulled, ensuring reproducible scans. Run `engines update` deliberately — do not auto-update engine images on every scan.

## Custom Nuclei Templates

Place custom `.yaml` Nuclei templates in `penforge/templates/nuclei/`. They are automatically mounted into the Nuclei container and run alongside the community templates.
