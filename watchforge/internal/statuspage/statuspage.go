package statuspage

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gartner24/forge/watchforge/internal/registry"
)

// MonitorView is the resolved state of a monitor for page rendering.
type MonitorView struct {
	Monitor       registry.Monitor
	Status        string // "healthy", "down", "paused"
	ActiveReason  string
	Uptime30d     float64
}

// Generate writes index.html atomically to statusDir.
// statuses maps monitor ID to "healthy"/"down"/"paused".
func Generate(statusDir string, monitors []registry.Monitor, incidents []registry.Incident, statuses map[string]string) error {
	if err := os.MkdirAll(statusDir, 0755); err != nil {
		return fmt.Errorf("creating status dir: %w", err)
	}

	// Build active incident index
	activeReason := make(map[string]string)
	for _, inc := range incidents {
		if inc.ClosedAt == nil {
			activeReason[inc.MonitorID] = inc.Reason
		}
	}

	// Compute 30-day uptime per monitor
	window := 30 * 24 * time.Hour
	windowStart := time.Now().Add(-window)
	downtime := make(map[string]time.Duration)
	for _, inc := range incidents {
		if inc.OpenedAt.After(time.Now()) {
			continue
		}
		end := time.Now()
		if inc.ClosedAt != nil {
			end = *inc.ClosedAt
		}
		start := inc.OpenedAt
		if start.Before(windowStart) {
			start = windowStart
		}
		if end.After(windowStart) {
			downtime[inc.MonitorID] += end.Sub(start)
		}
	}

	views := make([]MonitorView, 0, len(monitors))
	for _, m := range monitors {
		if !m.Public {
			continue
		}
		status := statuses[m.ID]
		if status == "" {
			status = "healthy"
		}
		dt := downtime[m.ID]
		uptime := 100.0
		if window > 0 {
			uptime = (float64(window-dt) / float64(window)) * 100
			if uptime < 0 {
				uptime = 0
			}
		}
		views = append(views, MonitorView{
			Monitor:      m,
			Status:       status,
			ActiveReason: activeReason[m.ID],
			Uptime30d:    uptime,
		})
	}

	overall := "All Systems Operational"
	overallClass := "ok"
	for _, v := range views {
		if v.Status == "down" {
			overall = "Major Outage"
			overallClass = "down"
			break
		}
	}
	if overallClass == "ok" {
		for _, v := range views {
			if v.Status == "paused" {
				overall = "Maintenance in Progress"
				overallClass = "paused"
				break
			}
		}
	}

	// Recent closed incidents (last 90 days)
	cutoff := time.Now().Add(-90 * 24 * time.Hour)
	var history []registry.Incident
	for _, inc := range incidents {
		if inc.ClosedAt != nil && inc.ClosedAt.After(cutoff) {
			history = append(history, inc)
		}
	}

	html := buildHTML(overall, overallClass, views, history)

	tmp := filepath.Join(statusDir, "index.html.tmp")
	if err := os.WriteFile(tmp, []byte(html), 0644); err != nil {
		return fmt.Errorf("writing status page: %w", err)
	}
	return os.Rename(tmp, filepath.Join(statusDir, "index.html"))
}

func buildHTML(overall, overallClass string, views []MonitorView, history []registry.Incident) string {
	var b strings.Builder

	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>System Status</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:system-ui,sans-serif;background:#f8f9fa;color:#212529;padding:24px}
.banner{padding:16px 24px;border-radius:8px;font-size:1.1rem;font-weight:600;margin-bottom:24px}
.banner.ok{background:#d1fae5;color:#065f46}
.banner.down{background:#fee2e2;color:#991b1b}
.banner.paused{background:#fef3c7;color:#92400e}
table{width:100%;border-collapse:collapse;background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 1px 3px rgba(0,0,0,.1)}
th,td{padding:12px 16px;text-align:left;border-bottom:1px solid #e9ecef}
th{background:#f1f3f5;font-size:.85rem;text-transform:uppercase;letter-spacing:.05em;color:#6c757d}
.badge{display:inline-block;padding:2px 10px;border-radius:12px;font-size:.8rem;font-weight:600}
.badge.healthy{background:#d1fae5;color:#065f46}
.badge.down{background:#fee2e2;color:#991b1b}
.badge.paused{background:#fef3c7;color:#92400e}
h2{margin:24px 0 12px;font-size:1rem;color:#6c757d;text-transform:uppercase;letter-spacing:.05em}
.ts{color:#6c757d;font-size:.85rem}
</style>
</head>
<body>
`)

	b.WriteString(fmt.Sprintf(`<div class="banner %s">%s</div>`+"\n", overallClass, html.EscapeString(overall)))

	b.WriteString("<h2>Monitors</h2>\n")
	b.WriteString(`<table>
<thead><tr><th>Name</th><th>Type</th><th>Status</th><th>30-Day Uptime</th><th>Details</th></tr></thead>
<tbody>
`)
	if len(views) == 0 {
		b.WriteString("<tr><td colspan=\"5\">No public monitors configured.</td></tr>\n")
	}
	for _, v := range views {
		details := ""
		if v.Status == "down" && v.ActiveReason != "" {
			details = html.EscapeString(v.ActiveReason)
		}
		b.WriteString(fmt.Sprintf(
			"<tr><td>%s</td><td>%s</td><td><span class=\"badge %s\">%s</span></td><td>%.2f%%</td><td>%s</td></tr>\n",
			html.EscapeString(v.Monitor.Name),
			html.EscapeString(string(v.Monitor.Type)),
			v.Status,
			strings.ToUpper(v.Status),
			v.Uptime30d,
			details,
		))
	}
	b.WriteString("</tbody></table>\n")

	if len(history) > 0 {
		b.WriteString("<h2>Recent Incidents</h2>\n")
		b.WriteString(`<table>
<thead><tr><th>Monitor</th><th>Opened</th><th>Closed</th><th>Reason</th></tr></thead>
<tbody>
`)
		for i := len(history) - 1; i >= 0; i-- {
			inc := history[i]
			closed := "-"
			if inc.ClosedAt != nil {
				closed = inc.ClosedAt.UTC().Format("2006-01-02 15:04 UTC")
			}
			b.WriteString(fmt.Sprintf(
				"<tr><td>%s</td><td class=\"ts\">%s</td><td class=\"ts\">%s</td><td>%s</td></tr>\n",
				html.EscapeString(inc.MonitorName),
				inc.OpenedAt.UTC().Format("2006-01-02 15:04 UTC"),
				closed,
				html.EscapeString(inc.Reason),
			))
		}
		b.WriteString("</tbody></table>\n")
	}

	b.WriteString(fmt.Sprintf(`<p class="ts" style="margin-top:16px">Last updated: %s</p>`, time.Now().UTC().Format(time.RFC1123)))
	b.WriteString("\n</body>\n</html>\n")
	return b.String()
}
