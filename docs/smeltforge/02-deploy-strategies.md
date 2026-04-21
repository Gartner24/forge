# Deploy Strategies

SmeltForge supports two deployment strategies per project, configured in `registry/projects.json`.

## Stop-Start (default)

```
current container → STOP → pull new image → START new container → update Caddy routing
```

- Simplest strategy — always correct, no extra resource usage
- Has a brief downtime window (seconds) during the stop→start transition
- Suitable for non-critical projects or those with low traffic
- Default when no strategy is specified

## Blue-Green

```
current (blue) serving traffic
        ↓
pull new image → START new container (green)
        ↓
health check green (configurable timeout, default 30s)
        ↓
[pass] Caddy switches routing blue → green (zero-downtime)
       STOP blue container
        ↓
[fail] STOP green container — blue continues serving
       SparkForge alert fired
```

- Zero in-flight request drops — Caddy's config reload is atomic
- Requires enough memory for two containers simultaneously during the switch
- Health check: HTTP GET to the container's health endpoint, or TCP connect check
- Configure per-project:

```json
"strategy": "blue-green",
"health_check": {
  "path": "/health",
  "timeout": 30,
  "interval": 2
}
```

## Rollback

Both strategies support rollback to the previous image:

```bash
forge smeltforge rollback --project <id>
```

SmeltForge keeps a reference to the previous container image. Rollback performs a stop-start of the previous image regardless of the originally configured strategy.

The rollback reference is cleared after a second successful deploy. Only one rollback generation is retained.
