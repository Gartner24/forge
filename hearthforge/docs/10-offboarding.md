# Offboarding

This document defines the standard process to revoke developer access and clean up resources.

## Goals

- Remove developer access immediately
- Preserve audit logs
- Optionally preserve workspace data for a retention period

## Steps

### 1. Disable Developer Access

Remove or disable the developer's public key in the gateway mapping:

```bash
# The safest approach is full offboarding (step 3)
# which handles key removal automatically.
# For immediate emergency access revocation, remove the key file:
sudo rm /opt/infra/forge/hearthforge/gateway/authorized_keys/<dev>.pub
sudo docker compose restart gateway
```

### 2. Partial Offboarding (single project)

```bash
forge hearthforge delete-dev --dev <dev> --project <project> [--purge]
```

- Default: preserves `/opt/data/dev_workspaces/<project>/<dev>/`
- With `--purge`: deletes the workspace for that `(dev, project)` after tearing down container, vhost, keys, and sshd config

### 3. Full Offboarding (all projects)

```bash
forge hearthforge delete-dev --dev <dev> --all-projects [--purge-all]
```

- Default: preserves all `/opt/data/dev_workspaces/<project>/<dev>/` directories for that developer
- With `--purge-all`: deletes all workspaces and associated `_keys` / `_sshd` entries

### 4. Audit Logs

Keep audit logs indefinitely. Do not delete or rewrite historical entries. Logs are append-only by design.

## Retention Policy Suggestions

- Keep audit logs indefinitely
- Keep workspace archives 7–30 days unless required longer
- Document your retention policy in `09-operations.md`

## What Gets Cleaned Up

`forge hearthforge delete-dev` handles:

| Item | Cleaned up |
|---|---|
| Docker container | ✅ Stopped and removed |
| Proxy vhost config | ✅ Removed, proxy reloaded |
| Container SSH keys (`_keys/`) | ✅ Removed |
| Container sshd config (`_sshd/`) | ✅ Removed |
| `devs.json` entry | ✅ Project removed (or record disabled if no projects remain) |
| Workspace directory | Only with `--purge` / `--purge-all` |
| Audit logs | ❌ Never deleted |
| Gateway `authorized_keys` | ❌ Not removed by default — remove manually if needed |
