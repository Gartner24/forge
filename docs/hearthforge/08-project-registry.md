# Project Registry

The project registry defines which projects can be provisioned into developer environments and how to configure them.

**Location:** `hearthforge/registry/projects.json`

## Required Fields

```json
{
  "id": "hemis",
  "repo": "https://github.com/owner/hemis",
  "default_branch": "main",
  "stack": "mixed",
  "dev_ports": {
    "frontend": 5173,
    "backend": 5000
  },
  "bootstrap": {
    "clone": true,
    "path": "/workspace/hemis"
  },
  "resources": {
    "cpus": "1.0",
    "memory": "2g"
  },
  "preview": true
}
```

## Field Reference

| Field | Type | Description |
|---|---|---|
| `id` | string | Unique project id. Lowercase, dashes allowed. |
| `repo` | string | Git repository URL (HTTPS or SSH) |
| `default_branch` | string | Branch to clone on bootstrap (default: `main`) |
| `stack` | string | `node`, `python`, or `mixed` |
| `dev_ports.frontend` | number | Frontend dev server port |
| `dev_ports.backend` | number | Backend dev server port (optional) |
| `bootstrap.clone` | bool | Whether to clone repo on provision |
| `bootstrap.path` | string | Workspace path inside container |
| `resources.cpus` | string | CPU limit (e.g. `"1.0"`) |
| `resources.memory` | string | Memory limit (e.g. `"2g"`) |
| `preview` | bool | Whether to expose HTTP preview via proxy |

## Optional Fields

```json
{
  "startup": [
    "npm install",
    "npm run dev"
  ]
}
```

## Validation Rules

- `id` must match naming convention (lowercase, alphanumeric + dashes)
- `repo` must be a valid URL or SSH remote
- Ports must be numeric and not conflict with system ports
- If `preview` is enabled, proxy template must support routing

## devs.json Relationship

`registry/devs.json` references project `id` values to grant access. No access should be granted to unknown project ids.

## Change Management

- Changes to `projects.json` are admin-only
- Do not commit real data — commit only example entries or empty arrays
- Runtime data lives at the path configured in `~/.forge/config.toml`
