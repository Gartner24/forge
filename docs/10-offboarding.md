# Offboarding

This document defines the standard process to revoke developer access and clean up resources.

## Goals

- Remove developer access immediately
- Preserve audit logs
- Optionally preserve workspace data for a retention period

## Steps

1. Disable developer access
   - Remove or disable developer public key in gateway mapping
   - Use: `devctl disable-dev <dev>`

2. Revoke project access (if partial offboarding)
   - Use: `devctl revoke <dev> <project>`

3. Stop/remove containers (optional, depending on policy)
   - Use: `devctl delete-dev <dev> <project>`

4. Remove dev vhost routing
   - Remove the proxy vhost file generated for:
     - `<dev>-<project>.dev.domain.com`
   - Validate and reload proxy

5. Workspace handling
   - Option A: archive workspace to backups
   - Option B: delete workspace immediately (requires explicit confirmation)

6. Audit logs
   - Keep audit logs indefinitely (recommended)
   - Do not delete or rewrite historical entries

## Retention policy suggestions

- Keep audit logs indefinitely
- Keep workspace archives 7–30 days unless required longer
- Document policy in `09-operations.md`

