# Config File

`~/.forge/config.toml` is the global configuration file. Forge Core writes and manages it — modules read from it but never write to it directly.

## Full Reference

```toml
[forge]
domain       = "dev.example.com"     # base domain for the entire suite
data_dir     = "/opt/data"           # root for all runtime data
install_dir  = "/opt/infra/forge"    # root for installed module files
version      = "0.1.0"              # forge core version

[modules.smeltforge]
enabled      = true
api_addr     = "127.0.0.1:7770"
data_dir     = "/opt/data/smeltforge"

[modules.watchforge]
enabled      = true
api_addr     = "127.0.0.1:7771"
data_dir     = "/opt/data/watchforge"
status_page_url = "https://status.dev.example.com"

[modules.sparkforge]
enabled      = true
api_addr     = "127.0.0.1:7778"
gotify_addr  = "127.0.0.1:7779"

[modules.hearthforge]
enabled      = true
api_addr     = "127.0.0.1:7772"
gateway_addr = "0.0.0.0:2224"
data_dir     = "/opt/data/dev_workspaces"
registry_dir = "/opt/infra/forge/registry"

[modules.fluxforge]
enabled         = true
api_addr        = "127.0.0.1:7773"
controller_addr = "127.0.0.1:7777"
mesh_subnet     = "10.forge.0.0/16"

[modules.penforge]
enabled      = true
api_addr     = "127.0.0.1:7774"
data_dir     = "/opt/data/penforge"

[audit.smeltforge]
log_path = "/opt/data/logs/smeltforge/audit.log"

[audit.watchforge]
log_path = "/opt/data/logs/watchforge/audit.log"

[audit.hearthforge]
log_path = "/opt/data/logs/hearthforge/audit.log"

[audit.penforge]
log_path = "/opt/data/logs/penforge/audit.log"
```

## Default Ports

| Module | Purpose | Default port |
|---|---|---|
| SmeltForge | Management API | 7770 |
| WatchForge | Management API | 7771 |
| HearthForge | Management API | 7772 |
| FluxForge | Management API | 7773 |
| PenForge | Management API | 7774 |
| FluxForge | Controller (mesh) | 7777 |
| SparkForge | Notification API | 7778 |
| SparkForge | Gotify | 7779 |
| HearthForge | SSH gateway | 2224 |

## Editing

Use `forge config set` for individual keys. Edit the file directly only for bulk changes. Always run `forge status` after editing to verify all modules are still reachable.
