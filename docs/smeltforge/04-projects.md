# Project Registry

Projects are registered in `registry/projects.json`. Each entry defines the source, routing, strategy, and resource configuration for one deployed application.

## Registering a Project

```bash
forge smeltforge add --project <id>
```

The command prompts for all required values interactively. Flags are also accepted:

```bash
forge smeltforge add \
  --project hemis \
  --source git \
  --repo https://github.com/user/hemis.git \
  --branch main \
  --domain hemis.example.com \
  --port 3000 \
  --strategy blue-green \
  --watch            # auto-register a WatchForge monitor
```

## Project Entry Format

```json
{
  "id": "hemis",
  "source": {
    "type": "git",
    "repo": "https://github.com/user/hemis.git",
    "branch": "main"
  },
  "domain": "hemis.example.com",
  "port": 3000,
  "strategy": "blue-green",
  "health_check": {
    "path": "/health",
    "timeout": 30,
    "interval": 2
  },
  "trigger": {
    "type": "webhook"
  },
  "watch": true
}
```

## Source Types

| Type | Config | Description |
|---|---|---|
| `git` | `repo`, `branch` | Clone/pull from a Git repository |
| `registry` | `image` | Pull a Docker image from a registry |
| `local` | `path` | Use a Compose file or Dockerfile already on the server |

For private Git repos, configure a deploy key:
```bash
forge smeltforge deploy-key generate --project <id>
# prints public key to add on GitHub → Settings → Deploy keys
```

The private key is stored in `forge secrets` under `smeltforge.<id>.deploy_key`.

## Listing and Managing

```bash
forge smeltforge list                      # all projects
forge smeltforge status                    # all projects with current state
forge smeltforge status --project <id>     # single project detail
forge smeltforge delete --project <id>     # remove project and stop container
```
