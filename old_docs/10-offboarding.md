# Offboarding

This document defines the standard process to revoke developer access and clean up resources.

## Goals

- Remove developer access immediately
- Preserve audit logs
- Optionally preserve workspace data for a retention period

## Steps

1. Disable developer access
   - Remove or disable developer public key in gateway mapping (gateway integration TBD)

2. Partial offboarding (single project)
   - Use: `devctl delete-dev --dev <dev> --project <project> [--purge]`
   - Default: preserves `/opt/data/dev_workspaces/<project>/<dev>/`
   - With `--purge`: deletes the workspace for that `(dev, project)` after tearing down container, vhost, keys, and sshd config.

3. Full offboarding (all projects)
   - Use: `devctl delete-dev --dev <dev> --all-projects [--purge-all]`
   - Default: preserves all `/opt/data/dev_workspaces/<project>/<dev>/` directories for that developer
   - With `--purge-all`: deletes all such workspaces and associated `_keys` / `_sshd` entries for that developer

4. Audit logs
   - Keep audit logs indefinitely (recommended)
   - Do not delete or rewrite historical entries

## Retention policy suggestions

- Keep audit logs indefinitely
- Keep workspace archives 7–30 days unless required longer
- Document policy in `09-operations.md`

