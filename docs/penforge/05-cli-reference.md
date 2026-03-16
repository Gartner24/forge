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
```

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
