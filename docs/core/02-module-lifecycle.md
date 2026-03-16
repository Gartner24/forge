# Module Lifecycle

Forge Core manages the install, start, stop, update, and uninstall lifecycle of every module. Modules are discovered dynamically through the `Module` interface — Core never hardcodes module names or capabilities.

## States

```
NOT INSTALLED → INSTALLING → RUNNING
                                ↓        ↑
                              STOPPED ←──┘
                                ↓
                              ERROR
                                ↓
                              UNKNOWN  (process unreachable)
```

`forge status` queries every registered module and displays its current state. If a module process cannot be reached at its configured management address, it is shown as `UNKNOWN`.

## Install

```bash
forge install <module>
```

1. Validates the module name against the known module registry
2. Pulls the module's Docker image(s) if required
3. Writes a `[modules.<name>]` section to `~/.forge/config.toml`
4. Calls the module's `Start()` method
5. Waits up to 30 seconds for the module to report `RUNNING`

Install is idempotent — running it on an already-installed module reports the current state and exits without changes.

If a module requires FluxForge and FluxForge is not installed, Core warns the admin but continues — multi-node features will be unavailable until FluxForge is installed.

## Start and Stop

```bash
forge start <module>
forge stop <module>
```

`Stop` sends SIGTERM to the module process and waits up to 60 seconds for graceful shutdown. If the deadline passes, SIGKILL is sent. All modules must handle SIGTERM by completing in-progress operations before exiting.

## Update

```bash
forge update <module>
forge update --all
```

1. Pulls the latest Docker image for the module
2. Calls `Stop()` with a graceful deadline
3. Starts the new version
4. If the new version fails to reach `RUNNING` within 30 seconds, rolls back to the previous image and restarts

## Uninstall

```bash
forge uninstall <module>
forge uninstall <module> --purge  # also deletes all module data
```

Prompts for confirmation unless `--yes` is provided. Stops the module, removes its `config.toml` section, and removes its Docker images. Without `--purge`, all data in the module's `data_dir` is preserved.

## Module Registration

At startup, each module binary calls `module.Register(m)` which announces the module to Core's management endpoint at the address configured in `config.toml`. Core then begins routing CLI commands to that module and polling it for status.

Core never starts module processes itself — that is the job of the host's process supervisor (systemd or Docker restart policy). Core only communicates with already-running modules.

## --node Flag

When FluxForge is installed, Core automatically adds `--node <mesh-ip>` to every command contributed by every module. The calling module does not need to implement multi-node awareness — Core intercepts the flag and routes the command to the target node's module instance over the mesh network.
