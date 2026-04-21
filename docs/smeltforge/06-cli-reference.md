# CLI Reference — SmeltForge

## forge smeltforge add

```bash
forge smeltforge add --project <id> [flags]

Flags:
  --source git|registry|local
  --repo <url>          Git repo URL (for git source)
  --branch <branch>     Git branch (default: main)
  --image <image>       Docker image (for registry source)
  --path <path>         Local path (for local source)
  --domain <domain>     Public domain for this project
  --port <port>         Container port to proxy to
  --strategy stop-start|blue-green
  --watch               Auto-register WatchForge monitor
```

---

## forge smeltforge deploy

```bash
forge smeltforge deploy --project <id>
forge smeltforge deploy --project <id> --node <mesh-ip>
```

---

## forge smeltforge rollback

```bash
forge smeltforge rollback --project <id>
```

---

## forge smeltforge status

```bash
forge smeltforge status
forge smeltforge status --project <id>
forge smeltforge status --output json
```

---

## forge smeltforge list

```bash
forge smeltforge list
```

---

## forge smeltforge logs

```bash
forge smeltforge logs --project <id>
forge smeltforge logs --project <id> --lines 200
forge smeltforge logs --project <id> --since 1h
```

---

## forge smeltforge env

```bash
forge smeltforge env set <project> <KEY> <value>
forge smeltforge env get <project> <KEY>
forge smeltforge env list <project>
forge smeltforge env unset <project> <KEY>
```

---

## forge smeltforge webhook

```bash
forge smeltforge webhook show <project>       # print current webhook URL
forge smeltforge webhook regenerate <project> # rotate webhook secret
```

---

## forge smeltforge token

```bash
forge smeltforge token create --project <id>
forge smeltforge token list --project <id>
forge smeltforge token revoke <token-id>
```

---

## forge smeltforge polling

```bash
forge smeltforge polling enable --project <id> --interval <seconds>
forge smeltforge polling disable --project <id>
```

---

## forge smeltforge delete

```bash
forge smeltforge delete --project <id>
forge smeltforge delete --project <id> --yes
```

Stops and removes the container and removes the project from the registry. Does not delete env vars from `forge secrets` — run `forge smeltforge env list` first and clean up manually if needed.

---

## forge smeltforge deploy-key

```bash
forge smeltforge deploy-key generate --project <id>   # generates keypair, stores private key in secrets, prints public key
forge smeltforge deploy-key show --project <id>        # print public key again
forge smeltforge deploy-key rotate --project <id>      # generate new keypair
```
