# shared/notify — SparkForge Notification Client

The `notify` package is how every Forge module sends alerts and notifications. It wraps SparkForge's local HTTP API behind a simple Go interface so modules never need to know whether SparkForge is installed, what channels are configured, or what priority levels mean in terms of delivery routing. They just call `notify.Send()` and the rest is handled for them.

## Design Intent

There are two important design decisions embedded in this package that are worth understanding before using it.

The first is **graceful degradation**. If SparkForge is not installed, `notify.Send()` returns immediately without error. It does not panic, it does not log a warning on every call, and it does not require the calling module to check whether SparkForge is available. The calling module simply sends a notification and moves on. This is what makes all SparkForge integrations truly opt-in — installing SparkForge activates them automatically, and uninstalling SparkForge deactivates them just as automatically.

The second is **priority-based routing**. SparkForge routes notifications to channels based on the message priority. Modules do not decide which channels receive their messages — that is the admin's concern, configured in SparkForge's channel settings. A module sets a priority level that reflects the urgency of the event, and SparkForge handles the rest. This means a module is not tightly coupled to any specific notification channel, and an admin can change routing without touching module code.

## Priority Levels

```go
package notify

// Priority represents the urgency level of a notification.
// SparkForge routes notifications to channels based on these levels.
// Channels have a configured minimum priority — messages below that
// threshold are silently dropped for that channel.
type Priority string

const (
    // PriorityLow is for informational events that do not require action.
    // Examples: deploy started, scan scheduled, developer provisioned.
    PriorityLow Priority = "low"

    // PriorityMedium is for events that may require attention soon.
    // Examples: deploy completed, scan finished with no new findings.
    PriorityMedium Priority = "medium"

    // PriorityHigh is for events that require prompt attention.
    // Examples: deploy failed, monitor down, new high-severity finding.
    PriorityHigh Priority = "high"

    // PriorityCritical is for events that require immediate action.
    // Examples: new critical CVE, SSL cert expiring in 3 days, gateway down.
    PriorityCritical Priority = "critical"
)
```

## The Send Function

```go
// Message is the notification payload.
type Message struct {
    // Title is a short summary shown as the notification headline.
    // Keep it under 80 characters. Required.
    Title string

    // Body is the full notification text. Can be multi-line.
    // Optional — if empty, Title is used as the full message.
    Body string

    // Priority determines which SparkForge channels receive this message.
    // Required.
    Priority Priority

    // Source identifies which module is sending the notification.
    // Use your module's Name() return value. Required.
    Source string

    // Link is an optional URL to attach to the notification.
    // Useful for linking to a deploy log, scan report, or status page.
    Link string
}

// Send delivers a notification through SparkForge.
//
// If SparkForge is not installed or its API is unreachable, Send returns
// nil immediately — callers must never fail because SparkForge is absent.
//
// If SparkForge is installed but the delivery attempt itself fails (e.g.
// a downstream channel is misconfigured), Send still returns nil — SparkForge
// handles its own delivery failure logging. The calling module's responsibility
// ends at calling Send.
func Send(ctx context.Context, msg Message) error
```

## Usage Patterns

The most common pattern is sending a notification at the end of a significant operation, reflecting whether it succeeded or failed:

```go
// In SmeltForge's deploy function:
err := runDeploy(ctx, project)
if err != nil {
    notify.Send(ctx, notify.Message{
        Title:    fmt.Sprintf("Deploy failed: %s", project.ID),
        Body:     err.Error(),
        Priority: notify.PriorityHigh,
        Source:   "smeltforge",
    })
    return err
}

notify.Send(ctx, notify.Message{
    Title:    fmt.Sprintf("Deploy succeeded: %s", project.ID),
    Body:     fmt.Sprintf("Image: %s", imageRef),
    Priority: notify.PriorityMedium,
    Source:   "smeltforge",
})
```

For critical system events where immediate attention is needed:

```go
// In WatchForge when a monitor goes down:
notify.Send(ctx, notify.Message{
    Title:    fmt.Sprintf("DOWN: %s", monitor.Name),
    Body:     fmt.Sprintf("Failed %d consecutive checks. Last error: %s", failures, lastErr),
    Priority: notify.PriorityCritical,
    Source:   "watchforge",
    Link:     statusPageURL,
})
```

## The In-CLI Banner Integration

When a HIGH or CRITICAL notification is sent and SparkForge is installed, SparkForge writes the active alert to a state file that Forge Core reads before every CLI command. This is how the in-CLI alert banner (FR-010) works — `notify.Send()` at HIGH or CRITICAL priority is what activates it. The alert banner clears automatically when a RECOVERED notification is sent for the same source and event type, or when the alert is acknowledged in SparkForge.

## Alert Deduplication

SparkForge deduplicates alerts on its side — if WatchForge calls `notify.Send()` with the same monitor name and HIGH priority on every failed check, SparkForge only delivers the first one to channels until that alert resolves. Modules do not need to implement their own deduplication before calling Send. The calling module's job is simply to send a notification every time an event occurs; SparkForge decides whether to route it to channels based on whether an identical active alert already exists.
