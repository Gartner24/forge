# CLI Reference — WatchForge

## forge watchforge add

```bash
forge watchforge add \
  --type http|tcp|docker|ssl|heartbeat \
  --name <name> \
  --target <url-or-host:port-or-container> \
  --interval <seconds> \
  [--public] \
  [--expected-status <code>] \
  [--contains <string>] \
  [--timeout <seconds>] \
  [--threshold <n>] \
  [--grace <seconds>]   # heartbeat only
```

---

## forge watchforge list

```bash
forge watchforge list
forge watchforge list --output json
```

---

## forge watchforge status

```bash
forge watchforge status                    # all monitors
forge watchforge status --monitor <id>
```

---

## forge watchforge pause / resume

```bash
forge watchforge pause --monitor <id>
forge watchforge resume --monitor <id>
```

---

## forge watchforge delete

```bash
forge watchforge delete --monitor <id>
forge watchforge delete --monitor <id> --yes
```

---

## forge watchforge incidents

```bash
forge watchforge incidents
forge watchforge incidents --monitor <id>
forge watchforge incidents --since 7d
forge watchforge incidents --output json
```

---

## forge watchforge update

```bash
forge watchforge update --monitor <id> --interval 30
forge watchforge update --monitor <id> --public true
forge watchforge update --monitor <id> --threshold 3
```
