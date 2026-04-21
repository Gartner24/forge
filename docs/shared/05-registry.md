# shared/registry — Registry File Parsers

The `registry` package provides typed Go structs and read/write functions for every JSON registry file that Forge modules share. Rather than having each module define its own struct for `projects.json` or its own file I/O code, the shared registry package ensures that all modules read and write the same data shapes with the same validation rules and the same atomic write behaviour.

## What the Registry Files Are

Registry files are the persistent configuration layer of Forge. They are plain JSON files stored under each module's `registry/` directory on the server. They are not a database — they are human-readable, git-committable (in their example form), and directly editable by an admin in an emergency. They are the source of truth for everything that persists between module restarts.

The key registry files are:

```
hearthforge/registry/projects.json   # projects available for dev environments
hearthforge/registry/devs.json       # developers and their project access
watchforge/registry/monitors.json    # configured monitors
penforge/registry/targets.json       # registered scan targets
sparkforge/registry/channels.json    # notification channels
```

Each file is owned by one module and only that module writes to it. Other modules that need to read it import the appropriate type from `shared/registry` and call the read function — they never write to another module's registry file.

## The Data Types

```go
package registry

// Project represents a project registered in HearthForge's projects.json.
// This same type is also referenced by SmeltForge for deployment projects,
// with different fields populated depending on which module owns the entry.
type Project struct {
    // ID is the project's canonical slug. Lowercase, alphanumeric, dashes.
    // Example: "hemis", "tiap", "my-api"
    ID string `json:"id"`

    // Repo is the Git repository URL. Can be HTTPS or SSH format.
    Repo string `json:"repo,omitempty"`

    // DefaultBranch is the branch used for bootstrap clones and deploys.
    DefaultBranch string `json:"default_branch,omitempty"`

    // Stack describes the toolchain. One of: "node", "python", "mixed".
    Stack string `json:"stack,omitempty"`

    // DevPorts defines the ports the project exposes for development access.
    DevPorts DevPorts `json:"dev_ports,omitempty"`

    // Preview controls whether the project gets a public preview domain.
    Preview bool `json:"preview,omitempty"`

    // Resources sets the container resource limits for this project.
    Resources Resources `json:"resources,omitempty"`
}

// DevPorts holds the port configuration for a project's dev servers.
type DevPorts struct {
    // Frontend is the port the frontend dev server runs on (e.g. 3000).
    Frontend int `json:"frontend,omitempty"`

    // Backend is the port the backend API runs on (e.g. 5000).
    // Zero means no backend port is configured.
    Backend int `json:"backend,omitempty"`
}

// Resources defines CPU and memory limits for a dev container.
type Resources struct {
    CPUs   string `json:"cpus,omitempty"`   // e.g. "1.0"
    Memory string `json:"memory,omitempty"` // e.g. "2g"
}

// Developer represents a registered developer in HearthForge's devs.json.
type Developer struct {
    // ID is the developer's canonical slug. Lowercase, alphanumeric, dashes.
    ID string `json:"id"`

    // Projects is the list of project IDs this developer has access to.
    Projects []string `json:"projects"`

    // Status is "active" for a developer with at least one project, or
    // "disabled" for a developer whose last project was removed but whose
    // record was not purged.
    Status string `json:"status"`
}

// Monitor represents a configured monitor in WatchForge's monitors.json.
type Monitor struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Type     string `json:"type"`     // "http", "tcp", "docker", "ssl", "heartbeat"
    Target   string `json:"target"`
    Interval int    `json:"interval"` // check interval in seconds
    Public   bool   `json:"public"`   // whether to show on public status page
    Paused   bool   `json:"paused"`
}

// ScanTarget represents a registered scan target in PenForge's targets.json.
type ScanTarget struct {
    ID      string   `json:"id"`
    Name    string   `json:"name"`
    URL     string   `json:"url"`
    Scope   []string `json:"scope"`   // domains and IPs in scope
    Engines []string `json:"engines"` // engines to run; empty means all
}

// Channel represents a notification channel in SparkForge's channels.json.
type Channel struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Type        string `json:"type"`         // "gotify", "email", "webhook"
    Enabled     bool   `json:"enabled"`
    PriorityMin string `json:"priority_min"` // minimum priority to deliver
}
```

## Read and Write Functions

Every registry type has a corresponding pair of functions:

```go
// ReadProjects reads and validates projects.json for the given registry path.
// Returns an empty slice (not an error) if the file does not exist.
func ReadProjects(path string) ([]Project, error)

// WriteProjects writes a project list to the given path atomically.
// The write is atomic — it writes to a temp file first and renames into
// place, so a partial write can never corrupt the live file.
func WriteProjects(path string, projects []Project) error

// ReadDevelopers / WriteDevelopers — same pattern for devs.json
// ReadMonitors   / WriteMonitors   — same pattern for monitors.json
// ReadScanTargets / WriteScanTargets — same pattern for targets.json
// ReadChannels   / WriteChannels   — same pattern for channels.json
```

## Atomic Writes

Every write function uses the write-to-temp-then-rename pattern. This is critical because registry files are read by multiple goroutines concurrently, and a partial write (e.g. from a crash mid-write) must never leave the file in a corrupted state.

```go
// Inside WriteProjects — the atomic write pattern used by all write functions:
func WriteProjects(path string, projects []Project) error {
    data, err := json.MarshalIndent(projects, "", "  ")
    if err != nil {
        return err
    }

    // Write to a temp file in the same directory (same filesystem = atomic rename)
    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, data, 0644); err != nil {
        return err
    }

    // Rename is atomic on POSIX systems — readers either see the old file
    // or the new file, never a partial state
    return os.Rename(tmp, path)
}
```

## Validation

Read functions validate the data they parse. A project with a missing `id`, a monitor with an unrecognised `type`, or a developer with an empty `projects` field are all rejected at read time with descriptive errors. This prevents silent corruption from spreading — if a registry file contains an invalid entry, the module that reads it will surface the error immediately rather than silently ignoring it or operating on bad data.

Validation is intentionally strict about required fields and permissive about optional ones. A future version of a registry file may contain fields this version does not know about — those fields are ignored rather than causing a parse error.

## Locking

Registry files are not protected by a distributed lock — they are single-writer by convention. Each registry file is owned by exactly one module, and only that module's code writes to it. Within a single module process, writes are serialised by an in-memory mutex in the write function. This is sufficient because Forge runs one instance of each module per server, and cross-node registry sharing is handled by FluxForge's sync mechanism rather than by concurrent file writes.
