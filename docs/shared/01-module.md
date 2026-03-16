# shared/module — Module Interface

The `module` package defines the contract between Forge Core and every installable module. It is the only place in the codebase where Core knows anything about modules — not their names, not their commands, not their configuration. Just the interface.

## Why This Exists

Forge Core is intentionally dumb about what modules do. It knows how to install them, start them, stop them, and report their status. It does not know that SmeltForge deploys Docker containers or that WatchForge runs goroutine-based health checks. That separation is what makes the suite modular — you can add a new module without touching Core.

The `Module` interface is the handshake that makes this work. Core discovers installed modules at startup, calls `Name()` to identify them, calls `Status()` to know if they're healthy, and calls `Start()` / `Stop()` to control their lifecycle. The module's code does the rest.

## The Interface

```go
// Package module defines the interface every Forge module must implement.
package module

import "context"

// Module is the contract between Forge Core and an installable module.
// Every module binary must expose exactly one implementation of this interface
// via its registration call at startup.
type Module interface {
    // Name returns the module's canonical identifier — lowercase, no spaces.
    // This is the name used in forge install, forge status, and config.toml.
    // Examples: "smeltforge", "watchforge", "hearthforge"
    Name() string

    // Version returns the module's current semantic version string.
    // Example: "0.1.0"
    Version() string

    // Status returns the module's current health state.
    // Called by forge status on every status query.
    Status(ctx context.Context) (Status, error)

    // Start brings the module up. Called by forge install and forge start.
    // Must be idempotent — calling Start on an already-running module must
    // be safe and must not return an error.
    Start(ctx context.Context) error

    // Stop brings the module down gracefully. Called by forge uninstall
    // and forge stop. Must complete all in-progress operations before
    // returning. Receives a context with a deadline — if the deadline
    // passes before shutdown is complete, the module must force-stop and
    // return ctx.Err().
    Stop(ctx context.Context) error

    // Commands returns the cobra subcommands this module contributes to
    // the forge CLI. Core registers them under the module's name automatically.
    // For example, SmeltForge returns its "deploy", "status", "add" commands
    // and Core registers them as "forge smeltforge deploy" etc.
    Commands() []*cobra.Command
}

// Status represents the health state of a module at a point in time.
type Status struct {
    // State is the module's current lifecycle state.
    State State

    // Message is a human-readable description of the current state.
    // Must be non-empty when State is Error or Unknown.
    Message string

    // Since is the time the module entered the current state.
    Since time.Time
}

// State is an enumeration of possible module lifecycle states.
type State string

const (
    StateRunning State = "RUNNING"  // module is up and healthy
    StateStopped State = "STOPPED"  // module is cleanly stopped
    StateError   State = "ERROR"    // module encountered a fatal error
    StateUnknown State = "UNKNOWN"  // module process cannot be reached
)
```

## How Core Discovers Modules

At startup, Core reads `~/.forge/config.toml` to get the list of installed modules and their socket/HTTP addresses. It connects to each module's management endpoint and calls `Status()`. If the module responds, it is listed in `forge status`. If it does not respond, it is marked `UNKNOWN`.

Modules register themselves with Core by calling `module.Register()` at process startup, which announces the module to Core's management endpoint. Core never hardcodes module names — if the registration happens, the module exists; if it doesn't, Core has no knowledge of it.

## How to Implement the Interface

Every module's `main.go` follows the same pattern:

```go
func main() {
    // 1. Create your module implementation
    m := mymodule.New(cfg)

    // 2. Register with Core — this announces the module and its commands
    if err := module.Register(m); err != nil {
        log.Fatalf("failed to register module: %v", err)
    }

    // 3. Block until SIGTERM, then call Stop with a deadline
    module.RunUntilSignal(m)
}
```

`module.RunUntilSignal` handles the signal catching, graceful shutdown deadline, and the `Stop()` call for you. Every module binary uses it so shutdown behaviour is consistent across the suite.

## The --node Flag Convention

When FluxForge is installed, Core automatically adds a `--node <mesh-ip>` flag to every command contributed by every module. This is handled at the Core level, not in the module itself — the module does not need to know about FluxForge to support multi-node targeting. If the `--node` flag is provided, Core routes the command to the appropriate module instance on the target mesh node via the FluxForge network.

## What Does Not Belong Here

The `module` package only defines the interface and the registration/discovery mechanism. It does not contain notification logic (that is `shared/notify`), audit logging (that is `shared/audit`), secret reading (that is `shared/secrets`), or any module-specific behaviour. Keep this package minimal — it is the most critical dependency in the entire codebase and every module imports it.
