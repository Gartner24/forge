# CLI Reference — SparkForge

## forge sparkforge channel add

```bash
forge sparkforge channel add \
  --type gotify|email|webhook \
  --name <n> \
  --priority-min low|medium|high|critical \
  [--url <url>]               # webhook
  [--smtp-host <host>]        # email
  [--smtp-port <port>]        # email (default: 587)
  [--smtp-user <user>]        # email
  [--smtp-password <pass>]    # email
  [--to <email>]              # email
```

---

## forge sparkforge channel list / show / delete

```bash
forge sparkforge channel list
forge sparkforge channel show <id>
forge sparkforge channel delete <id>
```

---

## forge sparkforge channel enable / disable / test / update

```bash
forge sparkforge channel enable <id>
forge sparkforge channel disable <id>
forge sparkforge channel test <id>
forge sparkforge channel update <id> --priority-min high
```

---

## forge sparkforge send

```bash
forge sparkforge send --title <text> --priority <level>
forge sparkforge send --title <text> --body <text> --priority high
forge sparkforge send --title <text> --priority critical --channel <id>
```

---

## forge sparkforge gotify

```bash
forge sparkforge gotify show          # print Gotify URL and mobile connection info
```

---

## forge sparkforge token

```bash
forge sparkforge token create --name <n>
forge sparkforge token list
forge sparkforge token revoke <token-id>
```

---

## forge sparkforge alerts

```bash
forge sparkforge alerts list
forge sparkforge alerts list --priority high
forge sparkforge alerts acknowledge <id>
forge sparkforge alerts acknowledge --all
```

---

## forge sparkforge logs

```bash
forge sparkforge logs                  # delivery log (tail)
forge sparkforge logs --since 1h
forge sparkforge logs --output json
```
