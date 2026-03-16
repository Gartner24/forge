# Finding Lifecycle

## Finding States

```
NEW → ACKNOWLEDGED → FIXED → VERIFIED
          ↓
       ACCEPTED (risk accepted — permanent suppression)
```

| State | Description |
|---|---|
| `new` | Finding appeared in the latest scan and has not been reviewed |
| `acknowledged` | Admin has seen it and is working on remediation. Alerts suppressed until severity increases. |
| `accepted` | Risk permanently accepted. Alerts suppressed unless severity increases. |
| `fixed` | Admin believes it is remediated. Awaiting verification scan. |
| `verified` | Confirmed fixed by a re-scan. |

## Finding Record Format

```json
{
  "id": "f-abc123",
  "engine": "nuclei",
  "target": "https://api.example.com",
  "severity": "high",
  "name": "SQL Injection in /api/users",
  "description": "...",
  "cve": "CVE-2024-1234",
  "remediation": "Sanitise user input in the query builder",
  "first_seen": "2026-03-14T10:23:01Z",
  "last_seen": "2026-03-14T10:23:01Z",
  "state": "new",
  "state_reason": "",
  "state_changed_at": null
}
```

## Managing Findings

```bash
# Acknowledge — you've seen it, working on it
forge penforge finding acknowledge <id> --reason "Ticket #1234 raised"

# Accept risk — won't fix, suppress permanently
forge penforge finding accept <id> --reason "Internal tool, not exposed externally"

# Mark as fixed — triggers optional re-scan
forge penforge finding verify <id>

# View all findings
forge penforge findings list
forge penforge findings list --severity critical
forge penforge findings list --state new
forge penforge findings list --target <id>
forge penforge findings list --output json
```

## Delta Detection

After every scan, PenForge compares results against the previous scan:

- **New** — appeared in latest scan, not in previous scan
- **Resolved** — was in previous scan, not in latest (may be genuinely fixed or intermittent)
- **Recurring** — appeared in both scans

Only **new** findings trigger SparkForge alerts. Recurring findings that have already been acknowledged or accepted do not re-alert unless their severity increases.

## SparkForge Alerts

| Condition | Priority |
|---|---|
| New CRITICAL finding | `critical` |
| New HIGH finding | `high` |
| New MEDIUM finding | `medium` |
| Severity of acknowledged/accepted finding increased | `high` |

New LOW and INFO findings do not trigger alerts — they appear only in the scan report.
