# Deploy Triggers

SmeltForge supports four ways to trigger a deployment.

## Manual (CLI)

```bash
forge smeltforge deploy --project <id>
forge smeltforge deploy --project <id> --node 10.forge.2.1  # specific mesh node
```

## Webhook

SmeltForge exposes an HTTP endpoint per project. The CI system sends a POST with the HMAC-signed secret in the URL path.

```
POST https://<your-domain>/_smeltforge/webhook/<project-id>/<webhook-secret>
```

HMAC validation: the secret in the URL is validated against the stored webhook secret. Invalid secrets return HTTP 403 and are logged.

Regenerate a webhook secret:
```bash
forge smeltforge webhook regenerate <project-id>
```

Configure in GitHub Actions:
```yaml
- name: Deploy
  run: |
    curl -X POST https://example.com/_smeltforge/webhook/myapp/${{ secrets.FORGE_WEBHOOK_SECRET }}
```

## CI Token

For CI systems that prefer a standard Authorization header:

```
POST https://<your-domain>/_smeltforge/deploy
Authorization: Bearer <ci-token>
Content-Type: application/json

{"project": "<project-id>"}
```

Manage CI tokens:
```bash
forge smeltforge token create --project <id>
forge smeltforge token list --project <id>
forge smeltforge token revoke <token-id>
```

## Git Polling

SmeltForge polls the configured Git repository on a configurable interval and deploys when new commits appear on the tracked branch.

```json
"trigger": {
  "type": "polling",
  "interval": 60,
  "branch": "main"
}
```

```bash
forge smeltforge polling enable --project <id> --interval 60
forge smeltforge polling disable --project <id>
```

Polling is the simplest integration — no webhook setup required, but has up to `interval` seconds of latency before a new commit is deployed.
