# CLI Reference — PenForge

## forge penforge add

```bash
forge penforge add \
  --name <n> \
  --target <url> \
  --scope <domain-or-ip,...> \
  [--engines nuclei,nmap,testssl,dnsx,trivy]
```

---

## forge penforge list / show / delete / update

```bash
forge penforge list
forge penforge show --target <id>
forge penforge delete --target <id>
forge penforge update --target <id> --scope <new-scope>
forge penforge update --target <id> --engines nuclei,testssl
```

---

## forge penforge scan

```bash
forge penforge scan --target <id>
forge penforge scan --target <id> --engine nuclei       # single engine
forge penforge scan --target <id> --node 10.forge.2.1  # from a mesh node
forge penforge scan --target <id> --async
forge penforge scan --target <id> --async --output json
```

| Flag | Description |
|------|-------------|
| `--async` | Return immediately with a `scan_id` instead of blocking until completion. Use `forge penforge scans show <scan_id>` to poll for results. |
| `--output json` | Machine-readable JSON output. Required when using `--async` programmatically. |

**JSON output when `--async` is passed:**

```json
{
  "scan_id": "abc123",
  "target_id": "myapp",
  "started_at": "2026-04-21T10:00:00Z",
  "estimated_seconds": 600,
  "engines": ["nuclei", "nmap", "testssl", "dnsx"]
}
```

> **Why async?** PenForge scans run multiple engines (Nuclei, Nmap, testssl.sh, dnsx)
> and can take 5-15 minutes. Synchronous scans are fine for interactive use. For
> automated callers (CI pipelines, the Forge MCP Server), use `--async` to get a
> `scan_id` immediately and poll with `forge penforge scans show <scan_id> --output json`.

---

## forge penforge report

```bash
forge penforge report --target <id>                    # latest scan report
forge penforge report --target <id> --scan <scan-id>   # specific scan
forge penforge report --target <id> --output json
```

---

## forge penforge findings

```bash
forge penforge findings list
forge penforge findings list --target <id>
forge penforge findings list --severity critical|high|medium|low|info
forge penforge findings list --state new|acknowledged|accepted|fixed|verified
forge penforge findings list --output json
```

---

## forge penforge finding

```bash
forge penforge finding acknowledge <id> --reason <text>
forge penforge finding accept <id> --reason <text>
forge penforge finding verify <id>                     # marks as fixed + triggers re-scan
```

---

## forge penforge schedule

```bash
forge penforge schedule --target <id> --cron "0 2 * * 1"
forge penforge schedule --target <id> --disable
forge penforge schedule list
```

---

## forge penforge engines

```bash
forge penforge engines list                    # show engine versions
forge penforge engines update                  # pull latest for all engines
forge penforge engines update --engine nuclei  # specific engine
```

---

## forge penforge scans

```bash
forge penforge scans list --target <id>        # scan history for a target
forge penforge scans show <scan-id>            # scan summary and finding counts
```
