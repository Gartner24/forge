# Scope Enforcement

PenForge will not scan anything that has not been explicitly registered as a scan target. This is a hard constraint enforced before any engine starts.

## Registering a Target

```bash
forge penforge add \
  --name "production-api" \
  --target https://api.example.com \
  --scope "api.example.com,203.0.113.10" \
  --engines nuclei,testssl,nmap
```

The `--scope` flag declares the exact domains and IP addresses the engines are permitted to reach. Engines are started with Docker network configuration that prevents access to any address outside this scope.

Target entry format:
```json
{
  "id": "production-api",
  "name": "production-api",
  "url": "https://api.example.com",
  "scope": [
    "api.example.com",
    "203.0.113.10"
  ],
  "engines": ["nuclei", "testssl", "nmap"]
}
```

Omit `--engines` to run all available engines against the target.

## What Scope Enforcement Does

1. Before any scan starts, PenForge validates that the target exists in `registry/targets.json`. If it does not, the scan is refused immediately.

2. Each engine container is started with explicit Docker `--network` configuration:
   - No access to `dev-web` (HearthForge developer network)
   - No access to `web` (SmeltForge production network)
   - No access to `bridge` (default Docker network)
   - Only outbound access to the IP addresses resolved from the declared scope

3. Engine output is validated after the scan — any finding referencing a host outside the declared scope is discarded.

## Why This Matters

Without scope enforcement, a misconfigured scan could reach internal infrastructure — developer containers, databases, or other servers on the mesh network. PenForge is designed to scan external-facing surfaces only. The scope declaration is the admin's explicit statement of what is in bounds.

## Managing Targets

```bash
forge penforge list                        # all registered targets
forge penforge show --target <id>
forge penforge delete --target <id>
forge penforge update --target <id> --scope "api.example.com,new-ip"
```

## Scheduled Scans

```bash
# Schedule a weekly scan every Monday at 02:00
forge penforge schedule --target <id> --cron "0 2 * * 1"

# Disable scheduled scanning for a target
forge penforge schedule --target <id> --disable

# View schedules
forge penforge schedule list
```

Scheduled scans run with the same scope enforcement and engine isolation as manual scans.
