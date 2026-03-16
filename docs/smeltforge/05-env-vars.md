# Environment Variables

SmeltForge injects environment variables into containers at deploy time. Values are stored in `forge secrets` — never in plaintext files on disk.

## Managing Env Vars

```bash
# Set a variable for a project
forge smeltforge env set hemis DB_URL "postgres://user:pass@localhost/hemis"
forge smeltforge env set hemis SECRET_KEY "$(openssl rand -hex 32)"

# List keys (values never shown)
forge smeltforge env list hemis

# Read a specific value
forge smeltforge env get hemis DB_URL

# Remove a variable
forge smeltforge env unset hemis DB_URL
```

## Storage

Variables are stored in `forge secrets` under the namespace `smeltforge.<project-id>.<KEY>`:

```
smeltforge.hemis.DB_URL
smeltforge.hemis.SECRET_KEY
smeltforge.hemis.REDIS_URL
```

## Injection

At deploy time, SmeltForge reads all secrets under `smeltforge.<project-id>.*`, strips the namespace prefix, and passes them to the container as environment variables via Docker's `--env` flag. The container receives plain `DB_URL`, `SECRET_KEY`, etc.

Variables are injected fresh on every deploy — updating a value with `env set` takes effect on the next deploy, no restart needed.

## Multi-Node Sync

On multi-node setups with FluxForge, env vars set on the controller are not automatically synced to other nodes. To deploy the same project on multiple nodes with the same env vars:

```bash
forge smeltforge env set hemis DB_URL "..." 
forge secrets set smeltforge.hemis.DB_URL "..." --sync  # replicate to all nodes
```

Or use `forge secrets set --sync` directly for all env vars on a project.
