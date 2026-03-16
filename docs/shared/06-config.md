# shared/config — Global Config Reader

The `config` package gives every Forge module read access to the global Forge configuration file (`~/.forge/config.toml`). This file is the single source of truth for installation-wide settings: which modules are installed, where their log files live, what the forge domain is, and how inter-module communication is addressed.

## What the Config File Contains

`~/.forge/config.toml` is created and managed by Forge Core. Modules never write to it directly — they read from it. When a module is installed with `forge install`, Core writes a section for that module into the config file. When it is uninstalled, Core removes that section. A module should treat the config file as read-only.

A typical config file looks like this:

```toml
[forge]
domain       = "dev.qyvos.com"           # base domain for the entire suite
data_dir     = "/opt/data"               # root for all runtime data
install_dir  = "/opt/infra/forge"        # root for all installed module files
version      = "0.1.0"

[modules.smeltforge]
enabled      = true
api_addr     = "127.0.0.1:7770"          # SmeltForge management API address
data_dir     = "/opt/data/smeltforge"

[modules.watchforge]
enabled      = true
api_addr     = "127.0.0.1:7771"
data_dir     = "/opt/data/watchforge"
status_page_url = "https://status.dev.qyvos.com"

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
enabled          = true
controller_addr  = "127.0.0.1:7777"
mesh_subnet      = "10.forge.0.0/16"

[modules.penforge]
enabled      = true
api_addr     = "127.0.0.1:7773"
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

## The Config Type

```go
package config

// Config is the parsed representation of ~/.forge/config.toml.
// Obtain it via Load() — do not construct it directly.
type Config struct {
    Forge   ForgeConfig              `toml:"forge"`
    Modules map[string]ModuleConfig  `toml:"modules"`
    Audit   map[string]AuditConfig   `toml:"audit"`
}

// ForgeConfig holds installation-wide settings.
type ForgeConfig struct {
    Domain     string `toml:"domain"`      // base domain, e.g. "dev.qyvos.com"
    DataDir    string `toml:"data_dir"`    // e.g. "/opt/data"
    InstallDir string `toml:"install_dir"` // e.g. "/opt/infra/forge"
    Version    string `toml:"version"`
}

// ModuleConfig holds per-module settings written by Core on install.
type ModuleConfig struct {
    Enabled    bool   `toml:"enabled"`
    APIAddr    string `toml:"api_addr"`    // host:port of the module's management API
    DataDir    string `toml:"data_dir"`    // module-specific data directory
    // Additional module-specific fields are parsed into the raw map
    // and accessed via Extra()
    extra      map[string]interface{}
}

// AuditConfig holds the log file path for a module's audit log.
type AuditConfig struct {
    LogPath string `toml:"log_path"`
}
```

## Loading the Config

```go
// Load reads and parses ~/.forge/config.toml.
// Returns an error if the file does not exist (Forge is not installed)
// or if the TOML is malformed.
// Load is safe to call concurrently — it reads the file on each call
// and does not cache. For performance-sensitive code that calls Load
// repeatedly, use LoadCached which caches with a configurable TTL.
func Load() (*Config, error)

// LoadCached returns a cached config, refreshing it if the cache is
// older than the given TTL. A TTL of zero disables caching.
// Suitable for modules that read config frequently, like on every
// incoming API request.
func LoadCached(ttl time.Duration) (*Config, error)

// MustLoad is like Load but panics if the config cannot be read.
// Use only in module startup code where a missing config is genuinely
// unrecoverable.
func MustLoad() *Config
```

## How Modules Use It

The most common use of `config` in a module is startup — reading the relevant section to know where data files live, what address to bind to, and how to reach other modules.

```go
// In SmeltForge's initialization:
cfg := config.MustLoad()

// Read SmeltForge's own config section
smeltCfg := cfg.Modules["smeltforge"]
listenAddr := smeltCfg.APIAddr // "127.0.0.1:7770"

// Read the forge-wide domain for generating project URLs
domain := cfg.Forge.Domain // "dev.qyvos.com"

// Check if SparkForge is installed before trying to send notifications
if cfg.IsModuleEnabled("sparkforge") {
    // Safe to use shared/notify — it will find SparkForge's API addr
}

// Read the audit log path for this module
auditPath := cfg.Audit["smeltforge"].LogPath
```

The `IsModuleEnabled` helper is a convenience method on `Config` that checks both the `enabled` flag and whether the module section exists:

```go
// IsModuleEnabled returns true if the named module is installed and enabled.
// This is the canonical check for opt-in integration — before calling
// another module's API, always check IsModuleEnabled first.
func (c *Config) IsModuleEnabled(name string) bool
```

## Why Modules Should Not Cache Config Aggressively

The config file can change while a module is running — most commonly when another module is installed or uninstalled. If SmeltForge caches the config at startup and holds it forever, it will miss the fact that PenForge was installed after SmeltForge started, and the post-deploy scan integration will never activate.

The recommended pattern is to use `LoadCached` with a short TTL (5–30 seconds) for settings that are read frequently, and `Load` for settings that are read infrequently (like at the start of a deploy operation). Never use `MustLoad` at startup and then hold the result for the module's entire lifetime unless the settings genuinely cannot change without a module restart.

## The Domain Helper

A common need across modules is constructing URLs from the forge domain. The `config` package provides a helper for this so modules produce consistent URLs:

```go
// DevPreviewURL returns the preview URL for a developer's project.
// Example: DevPreviewURL("alice", "hemis") → "https://alice-hemis.dev.qyvos.com"
func (c *Config) DevPreviewURL(dev, project string) string

// StatusPageURL returns the WatchForge status page URL.
// Example: StatusPageURL() → "https://status.dev.qyvos.com"
func (c *Config) StatusPageURL() string
```

These helpers ensure that every module that constructs these URLs uses the same subdomain format, avoiding subtle inconsistencies if the format ever changes — only the helper needs to be updated, not every module that builds a URL.
