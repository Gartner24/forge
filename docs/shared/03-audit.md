# shared/audit — Audit Log Writer

The `audit` package is how every Forge module writes structured audit log entries. Every significant operation in the system — provisioning a developer environment, deploying an application, running a security scan, authenticating through the SSH gateway — produces an audit record. These records are append-only, never deleted, and form the permanent operational history of a Forge installation.

## Why Structured Audit Logs Matter

Audit logs serve two distinct purposes that are easy to conflate but important to keep separate.

The first is **operational debugging** — when something goes wrong at 2am, the audit log is the first place you look to understand what happened, who triggered it, and what the system did in response. For this purpose, structured logs (JSON Lines format) are far more useful than free-text logs because you can filter them programmatically: show me all deploys for project `hemis` in the last 24 hours, or show me all failed SSH auth attempts from a specific IP.

The second is **accountability** — a permanent, tamper-resistant record of who did what. This is why audit logs are append-only at the filesystem level (mode 0644, owned by root, written by module processes that do not have permission to truncate or delete). Even if a module has a bug that tries to overwrite a log entry, the filesystem prevents it.

The `audit` package enforces both: it writes structured JSON, and it opens log files in append-only mode exclusively.

## The Event Type

```go
package audit

import "time"

// Event is a single audit log entry. Every field that is marked required
// must be populated — the Write function will return an error if they are not.
type Event struct {
    // Timestamp is when the event occurred, in UTC. Required.
    // Always populated automatically by Write() — callers do not set this.
    Timestamp time.Time `json:"timestamp"`

    // Module is the canonical name of the module writing this event.
    // Use your module's Name() return value. Required.
    // Example: "smeltforge", "hearthforge", "penforge"
    Module string `json:"module"`

    // EventType is a short identifier for the kind of event. Required.
    // Use snake_case. Be consistent — the same operation should always
    // produce the same EventType so logs can be filtered reliably.
    // Examples: "deploy.started", "deploy.completed", "deploy.failed",
    //           "dev.provisioned", "dev.offboarded", "ssh.auth.accepted",
    //           "ssh.auth.rejected", "scan.started", "scan.completed"
    EventType string `json:"event_type"`

    // Actor is the identity of whoever triggered this event. Required.
    // For admin CLI operations, use "admin". For developer SSH events,
    // use the developer id (e.g. "alice"). For automated triggers like
    // webhooks or scheduled scans, use the trigger source
    // (e.g. "webhook:github", "scheduler", "smeltforge:post-deploy-hook").
    Actor string `json:"actor"`

    // Target is the resource the event acted upon. Required.
    // Use a structured identifier that combines type and id.
    // Examples: "project:hemis", "developer:alice", "monitor:api-health",
    //           "container:dev-hemis-alice", "scan-target:app.example.com"
    Target string `json:"target"`

    // Result is the outcome of the operation. Required.
    Result Result `json:"result"`

    // Message is a human-readable description of what happened.
    // Optional but strongly recommended for Error results — include
    // enough detail that the error is understandable without context.
    Message string `json:"message,omitempty"`

    // Metadata holds event-specific structured data.
    // Optional. Use this for data that is valuable for filtering or
    // debugging but does not belong in the standard fields above.
    // Examples for a deploy event: {"image_ref": "...", "strategy": "blue-green"}
    // Examples for an SSH auth event: {"source_ip": "1.2.3.4", "key_fingerprint": "..."}
    Metadata map[string]string `json:"metadata,omitempty"`
}

// Result represents the outcome of an audited operation.
type Result string

const (
    ResultSuccess Result = "success"
    ResultFailure Result = "failure"
    ResultPartial Result = "partial" // operation completed but with non-fatal errors
)
```

## Writing Events

```go
// Write appends a single audit event to the module's log file.
//
// The log file path is read from ~/.forge/config.toml at the key
// audit.<module>.log_path. If the file does not exist, Write creates it
// with permissions 0644. The file is always opened in O_APPEND|O_WRONLY mode.
//
// Write sets Event.Timestamp automatically — callers must not set it.
//
// Returns an error if required fields are missing, if the log file cannot
// be opened, or if the write itself fails. A write failure is serious —
// callers should log it but must not silently swallow it.
func Write(ctx context.Context, event Event) error

// MustWrite is like Write but panics on error. Use only in contexts where
// an audit write failure is genuinely unrecoverable — in practice this is
// rare. Most callers should use Write and handle the error.
func MustWrite(ctx context.Context, event Event)
```

## Usage Patterns

The most important pattern is writing an event at both the start and end of significant operations — not just on failure. A start event establishes that an operation began; the end event records the outcome. Both together give you a complete picture when debugging.

```go
// In HearthForge's provision function:
audit.Write(ctx, audit.Event{
    Module:    "hearthforge",
    EventType: "dev.provision.started",
    Actor:     "admin",
    Target:    "developer:alice",
    Result:    audit.ResultSuccess,
    Message:   "Starting provisioning for alice on project hemis",
    Metadata:  map[string]string{"project": "hemis"},
})

err := provisionContainer(ctx, dev, project)
if err != nil {
    audit.Write(ctx, audit.Event{
        Module:    "hearthforge",
        EventType: "dev.provision.failed",
        Actor:     "admin",
        Target:    "developer:alice",
        Result:    audit.ResultFailure,
        Message:   err.Error(),
        Metadata:  map[string]string{"project": "hemis"},
    })
    return err
}

audit.Write(ctx, audit.Event{
    Module:    "hearthforge",
    EventType: "dev.provision.completed",
    Actor:     "admin",
    Target:    "developer:alice",
    Result:    audit.ResultSuccess,
    Metadata:  map[string]string{
        "project":   "hemis",
        "container": "dev-hemis-alice",
    },
})
```

## Log File Format and Location

Each module writes to its own log file. The path is configured per-module in `~/.forge/config.toml`:

```toml
[audit.smeltforge]
log_path = "/opt/data/logs/smeltforge/audit.log"

[audit.hearthforge]
log_path = "/opt/data/logs/hearthforge/audit.log"
```

Each line in the file is a complete, self-contained JSON object — the JSON Lines format. This makes the file both human-readable with standard tools (`tail`, `grep`, `jq`) and machine-parseable without a special reader:

```json
{"timestamp":"2026-03-14T10:23:01Z","module":"smeltforge","event_type":"deploy.completed","actor":"webhook:github","target":"project:hemis","result":"success","metadata":{"image_ref":"ghcr.io/user/hemis:abc123","strategy":"blue-green","duration_ms":"8420"}}
{"timestamp":"2026-03-14T10:31:55Z","module":"hearthforge","event_type":"ssh.auth.rejected","actor":"unknown","target":"gateway","result":"failure","message":"unknown developer identity","metadata":{"source_ip":"203.0.113.42"}}
```

## Log Rotation

When a log file reaches 100MB, the `audit` package automatically rotates it by renaming the current file to `audit.log.1` (and shifting older rotations to `.2`, `.3`, etc.) and opening a fresh `audit.log`. Rotation never interrupts in-progress writes — the rename and new-file creation are performed atomically from the perspective of the writing goroutine. Rotated files are never deleted by Forge; retention is the admin's responsibility.

## Reading Audit Logs

The audit package also provides a reader for querying log files programmatically — this is used internally by modules that need to display history (such as the HearthForge offboarding check that warns if a developer still has active sessions):

```go
// Query returns all audit events matching the filter from the given log file.
// Results are returned in chronological order.
func Query(ctx context.Context, logPath string, filter Filter) ([]Event, error)

// Filter specifies which events to return. All fields are optional — omitted
// fields match any value.
type Filter struct {
    Module    string    // match a specific module
    EventType string    // match a specific event type (prefix match supported)
    Actor     string    // match a specific actor
    Target    string    // match a specific target (prefix match supported)
    Result    Result    // match a specific result
    After     time.Time // only events after this time
    Before    time.Time // only events before this time
}
```
