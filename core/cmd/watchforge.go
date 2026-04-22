package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	sharedconfig "github.com/gartner24/forge/shared/config"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Data model (mirrors watchforge/internal/registry but kept local so core
// does not import the watchforge binary's packages).
// ---------------------------------------------------------------------------

type wfMonitorType string

const (
	wfTypeHTTP      wfMonitorType = "http"
	wfTypeTCP       wfMonitorType = "tcp"
	wfTypeDocker    wfMonitorType = "docker"
	wfTypeSSL       wfMonitorType = "ssl"
	wfTypeHeartbeat wfMonitorType = "heartbeat"
)

type wfMonitor struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Type           wfMonitorType `json:"type"`
	Target         string        `json:"target"`
	IntervalSec    int           `json:"interval"`
	Public         bool          `json:"public"`
	Paused         bool          `json:"paused"`
	Threshold      int           `json:"threshold"`
	ExpectedStatus int           `json:"expected_status,omitempty"`
	Contains       string        `json:"contains,omitempty"`
	TimeoutSec     int           `json:"timeout,omitempty"`
	Grace          int           `json:"grace,omitempty"`
	Token          string        `json:"token,omitempty"`
	LastPing       *time.Time    `json:"last_ping,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
}

type wfIncident struct {
	ID          string     `json:"id"`
	MonitorID   string     `json:"monitor_id"`
	MonitorName string     `json:"monitor_name"`
	OpenedAt    time.Time  `json:"opened_at"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
	Reason      string     `json:"reason"`
}

// ---------------------------------------------------------------------------
// Registry helpers
// ---------------------------------------------------------------------------

func wfDataDir(cfg *sharedconfig.Config) string {
	if m, ok := cfg.Modules["watchforge"]; ok && m.DataDir != "" {
		return m.DataDir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".forge", "data", "watchforge")
}

func wfRegistryDir(cfg *sharedconfig.Config) string {
	return filepath.Join(wfDataDir(cfg), "registry")
}

func wfMonitorsPath(cfg *sharedconfig.Config) string {
	return filepath.Join(wfRegistryDir(cfg), "monitors.json")
}

func wfIncidentsPath(cfg *sharedconfig.Config) string {
	return filepath.Join(wfRegistryDir(cfg), "incidents.json")
}

func loadMonitors(cfg *sharedconfig.Config) ([]wfMonitor, error) {
	data, err := os.ReadFile(wfMonitorsPath(cfg))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var monitors []wfMonitor
	return monitors, json.Unmarshal(data, &monitors)
}

func saveMonitors(cfg *sharedconfig.Config, monitors []wfMonitor) error {
	path := wfMonitorsPath(cfg)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(monitors, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func loadIncidents(cfg *sharedconfig.Config) ([]wfIncident, error) {
	data, err := os.ReadFile(wfIncidentsPath(cfg))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var incidents []wfIncident
	return incidents, json.Unmarshal(data, &incidents)
}

func findMonitor(monitors []wfMonitor, id string) (*wfMonitor, error) {
	for i := range monitors {
		if monitors[i].ID == id {
			return &monitors[i], nil
		}
	}
	return nil, fmt.Errorf("monitor %q not found", id)
}

// monitorStatus derives the operational status from the open incidents list.
func monitorStatus(m wfMonitor, incidents []wfIncident) string {
	if m.Paused {
		return "PAUSED"
	}
	for _, inc := range incidents {
		if inc.MonitorID == m.ID && inc.ClosedAt == nil {
			return "DOWN"
		}
	}
	return "HEALTHY"
}

func wfNewID() string {
	return wfRandHex(16, true)
}

func wfNewToken() string {
	return wfRandHex(16, false)
}

func wfRandHex(n int, uuid bool) string {
	b := make([]byte, n)
	if f, err := os.Open("/dev/urandom"); err == nil {
		_, _ = f.Read(b)
		f.Close()
	}
	if uuid {
		return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	}
	return fmt.Sprintf("%x", b)
}

// ---------------------------------------------------------------------------
// Command tree
// ---------------------------------------------------------------------------

var watchforgeCmd = &cobra.Command{
	Use:   "watchforge",
	Short: "Manage WatchForge uptime monitors",
}

func init() {
	rootCmd.AddCommand(watchforgeCmd)

	// init
	watchforgeCmd.AddCommand(wfInitCmd)

	// add
	wfAddCmd.Flags().StringVar(&wfAddType, "type", "", "Monitor type: http, tcp, docker, ssl, heartbeat (required)")
	wfAddCmd.Flags().StringVar(&wfAddName, "name", "", "Monitor name (required)")
	wfAddCmd.Flags().StringVar(&wfAddTarget, "target", "", "Target URL, host:port, or container name")
	wfAddCmd.Flags().IntVar(&wfAddInterval, "interval", 60, "Check interval in seconds")
	wfAddCmd.Flags().BoolVar(&wfAddPublic, "public", false, "Show on public status page")
	wfAddCmd.Flags().IntVar(&wfAddExpectedStatus, "expected-status", 200, "Expected HTTP status code")
	wfAddCmd.Flags().StringVar(&wfAddContains, "contains", "", "Response body must contain this string")
	wfAddCmd.Flags().IntVar(&wfAddTimeout, "timeout", 10, "Request timeout in seconds")
	wfAddCmd.Flags().IntVar(&wfAddThreshold, "threshold", 2, "Consecutive failures before DOWN alert")
	wfAddCmd.Flags().IntVar(&wfAddGrace, "grace", 3600, "Heartbeat grace period in seconds")
	_ = wfAddCmd.MarkFlagRequired("type")
	_ = wfAddCmd.MarkFlagRequired("name")
	watchforgeCmd.AddCommand(wfAddCmd)

	// list
	watchforgeCmd.AddCommand(wfListCmd)

	// status
	wfStatusCmd.Flags().StringVar(&wfStatusMonitor, "monitor", "", "Show status for a specific monitor ID")
	watchforgeCmd.AddCommand(wfStatusCmd)

	// pause
	wfPauseCmd.Flags().StringVar(&wfPauseMonitor, "monitor", "", "Monitor ID (required)")
	_ = wfPauseCmd.MarkFlagRequired("monitor")
	watchforgeCmd.AddCommand(wfPauseCmd)

	// resume
	wfResumeCmd.Flags().StringVar(&wfResumeMonitor, "monitor", "", "Monitor ID (required)")
	_ = wfResumeCmd.MarkFlagRequired("monitor")
	watchforgeCmd.AddCommand(wfResumeCmd)

	// delete
	wfDeleteCmd.Flags().StringVar(&wfDeleteMonitor, "monitor", "", "Monitor ID (required)")
	_ = wfDeleteCmd.MarkFlagRequired("monitor")
	watchforgeCmd.AddCommand(wfDeleteCmd)

	// incidents
	wfIncidentsCmd.Flags().StringVar(&wfIncidentsMonitor, "monitor", "", "Filter by monitor ID")
	wfIncidentsCmd.Flags().StringVar(&wfIncidentsSince, "since", "", "Show incidents since duration (e.g. 7d, 24h)")
	watchforgeCmd.AddCommand(wfIncidentsCmd)

	// update
	wfUpdateCmd.Flags().StringVar(&wfUpdateMonitor, "monitor", "", "Monitor ID (required)")
	wfUpdateCmd.Flags().StringVar(&wfUpdateName, "name", "", "New name")
	wfUpdateCmd.Flags().StringVar(&wfUpdateTarget, "target", "", "New target")
	wfUpdateCmd.Flags().IntVar(&wfUpdateInterval, "interval", 0, "New interval in seconds")
	wfUpdateCmd.Flags().StringVar(&wfUpdatePublic, "public", "", "Set public visibility (true/false)")
	wfUpdateCmd.Flags().IntVar(&wfUpdateThreshold, "threshold", 0, "New failure threshold")
	wfUpdateCmd.Flags().StringVar(&wfUpdateContains, "contains", "", "New body contains string")
	wfUpdateCmd.Flags().IntVar(&wfUpdateExpectedStatus, "expected-status", 0, "New expected HTTP status")
	wfUpdateCmd.Flags().IntVar(&wfUpdateTimeout, "timeout", 0, "New timeout in seconds")
	wfUpdateCmd.Flags().IntVar(&wfUpdateGrace, "grace", 0, "New grace period in seconds")
	_ = wfUpdateCmd.MarkFlagRequired("monitor")
	watchforgeCmd.AddCommand(wfUpdateCmd)

	// heartbeat-url
	wfHeartbeatURLCmd.Flags().StringVar(&wfHeartbeatURLMonitor, "monitor", "", "Monitor ID (required)")
	_ = wfHeartbeatURLCmd.MarkFlagRequired("monitor")
	watchforgeCmd.AddCommand(wfHeartbeatURLCmd)
}

// ---------------------------------------------------------------------------
// forge watchforge init
// ---------------------------------------------------------------------------

var wfInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise WatchForge data directories",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := requireInit()
		if err != nil {
			return cmdErr(err)
		}
		registryDir := wfRegistryDir(cfg)
		statusDir := filepath.Join(wfDataDir(cfg), "data", "status")
		auditDir := filepath.Join(wfDataDir(cfg), "data")
		for _, dir := range []string{registryDir, statusDir, auditDir} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return cmdErr(fmt.Errorf("creating %s: %w", dir, err))
			}
		}
		// Initialise empty registry files if absent.
		for _, path := range []string{wfMonitorsPath(cfg), wfIncidentsPath(cfg)} {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				if err := os.WriteFile(path, []byte("[]"), 0644); err != nil {
					return cmdErr(fmt.Errorf("creating %s: %w", path, err))
				}
			}
		}
		printSuccess("watchforge initialised")
		return nil
	},
}

// ---------------------------------------------------------------------------
// forge watchforge add
// ---------------------------------------------------------------------------

var (
	wfAddType           string
	wfAddName           string
	wfAddTarget         string
	wfAddInterval       int
	wfAddPublic         bool
	wfAddExpectedStatus int
	wfAddContains       string
	wfAddTimeout        int
	wfAddThreshold      int
	wfAddGrace          int
)

var wfAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a monitor",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := requireInit()
		if err != nil {
			return cmdErr(err)
		}

		mtype := wfMonitorType(wfAddType)
		switch mtype {
		case wfTypeHTTP, wfTypeTCP, wfTypeDocker, wfTypeSSL, wfTypeHeartbeat:
		default:
			return cmdErr(fmt.Errorf("unknown type %q -- valid: http, tcp, docker, ssl, heartbeat", wfAddType))
		}

		if mtype != wfTypeHeartbeat && wfAddTarget == "" {
			return cmdErr(fmt.Errorf("--target is required for type %s", mtype))
		}

		m := wfMonitor{
			ID:          wfNewID(),
			Name:        wfAddName,
			Type:        mtype,
			Target:      wfAddTarget,
			IntervalSec: wfAddInterval,
			Public:      wfAddPublic,
			Threshold:   wfAddThreshold,
			CreatedAt:   time.Now().UTC(),
		}

		switch mtype {
		case wfTypeHTTP:
			m.ExpectedStatus = wfAddExpectedStatus
			m.Contains = wfAddContains
			m.TimeoutSec = wfAddTimeout
		case wfTypeHeartbeat:
			m.Grace = wfAddGrace
			m.Token = wfNewToken()
		}

		monitors, err := loadMonitors(cfg)
		if err != nil {
			return cmdErr(fmt.Errorf("loading monitors: %w", err))
		}
		monitors = append(monitors, m)
		if err := saveMonitors(cfg, monitors); err != nil {
			return cmdErr(fmt.Errorf("saving monitors: %w", err))
		}

		if isJSON() {
			printJSON(m)
		} else {
			fmt.Printf("Monitor added: %s (%s)\n", m.Name, m.ID)
			if mtype == wfTypeHeartbeat {
				fmt.Printf("Heartbeat URL: %s\n", wfHeartbeatURL(cfg, m.ID, m.Token))
			}
		}
		return nil
	},
}

// ---------------------------------------------------------------------------
// forge watchforge list
// ---------------------------------------------------------------------------

var wfListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all monitors",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := requireInit()
		if err != nil {
			return cmdErr(err)
		}
		monitors, err := loadMonitors(cfg)
		if err != nil {
			return cmdErr(fmt.Errorf("loading monitors: %w", err))
		}
		if isJSON() {
			if monitors == nil {
				monitors = []wfMonitor{}
			}
			printJSON(monitors)
			return nil
		}
		if len(monitors) == 0 {
			fmt.Println("No monitors configured. Run 'forge watchforge add' to create one.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tTYPE\tTARGET\tINTERVAL\tPUBLIC\tPAUSED")
		for _, m := range monitors {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%ds\t%s\t%s\n",
				m.ID[:8], m.Name, m.Type, m.Target, m.IntervalSec,
				boolYN(m.Public), boolYN(m.Paused))
		}
		w.Flush()
		return nil
	},
}

// ---------------------------------------------------------------------------
// forge watchforge status
// ---------------------------------------------------------------------------

var wfStatusMonitor string

type wfStatusRow struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Interval int    `json:"interval"`
	Paused   bool   `json:"paused"`
}

var wfStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show operational status of monitors",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := requireInit()
		if err != nil {
			return cmdErr(err)
		}
		monitors, err := loadMonitors(cfg)
		if err != nil {
			return cmdErr(fmt.Errorf("loading monitors: %w", err))
		}
		incidents, err := loadIncidents(cfg)
		if err != nil {
			return cmdErr(fmt.Errorf("loading incidents: %w", err))
		}

		var rows []wfStatusRow
		for _, m := range monitors {
			if wfStatusMonitor != "" && m.ID != wfStatusMonitor {
				continue
			}
			rows = append(rows, wfStatusRow{
				ID:       m.ID,
				Name:     m.Name,
				Type:     string(m.Type),
				Status:   monitorStatus(m, incidents),
				Interval: m.IntervalSec,
				Paused:   m.Paused,
			})
		}

		if wfStatusMonitor != "" && len(rows) == 0 {
			return cmdErr(fmt.Errorf("monitor %q not found", wfStatusMonitor))
		}

		if isJSON() {
			if rows == nil {
				rows = []wfStatusRow{}
			}
			printJSON(rows)
			return nil
		}
		if len(rows) == 0 {
			fmt.Println("No monitors configured.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tTYPE\tSTATUS\tINTERVAL")
		for _, r := range rows {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%ds\n", r.ID[:8], r.Name, r.Type, r.Status, r.Interval)
		}
		w.Flush()
		return nil
	},
}

// ---------------------------------------------------------------------------
// forge watchforge pause
// ---------------------------------------------------------------------------

var wfPauseMonitor string

var wfPauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause a monitor",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := requireInit()
		if err != nil {
			return cmdErr(err)
		}
		monitors, err := loadMonitors(cfg)
		if err != nil {
			return cmdErr(fmt.Errorf("loading monitors: %w", err))
		}
		m, err := findMonitor(monitors, wfPauseMonitor)
		if err != nil {
			return cmdErr(err)
		}
		if m.Paused {
			printSuccess(fmt.Sprintf("%s is already paused", m.Name))
			return nil
		}
		m.Paused = true
		if err := saveMonitors(cfg, monitors); err != nil {
			return cmdErr(fmt.Errorf("saving monitors: %w", err))
		}
		printSuccess(fmt.Sprintf("Paused: %s", m.Name))
		return nil
	},
}

// ---------------------------------------------------------------------------
// forge watchforge resume
// ---------------------------------------------------------------------------

var wfResumeMonitor string

var wfResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume a paused monitor",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := requireInit()
		if err != nil {
			return cmdErr(err)
		}
		monitors, err := loadMonitors(cfg)
		if err != nil {
			return cmdErr(fmt.Errorf("loading monitors: %w", err))
		}
		m, err := findMonitor(monitors, wfResumeMonitor)
		if err != nil {
			return cmdErr(err)
		}
		if !m.Paused {
			printSuccess(fmt.Sprintf("%s is not paused", m.Name))
			return nil
		}
		m.Paused = false
		if err := saveMonitors(cfg, monitors); err != nil {
			return cmdErr(fmt.Errorf("saving monitors: %w", err))
		}
		printSuccess(fmt.Sprintf("Resumed: %s", m.Name))
		return nil
	},
}

// ---------------------------------------------------------------------------
// forge watchforge delete
// ---------------------------------------------------------------------------

var wfDeleteMonitor string

var wfDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a monitor",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := requireInit()
		if err != nil {
			return cmdErr(err)
		}
		monitors, err := loadMonitors(cfg)
		if err != nil {
			return cmdErr(fmt.Errorf("loading monitors: %w", err))
		}
		m, err := findMonitor(monitors, wfDeleteMonitor)
		if err != nil {
			return cmdErr(err)
		}
		deleteName := m.Name
		ok, err := mustConfirm(fmt.Sprintf("Delete monitor %q (%s)?", deleteName, m.ID))
		if err != nil {
			return cmdErr(err)
		}
		if !ok {
			fmt.Println("Aborted.")
			return nil
		}
		out := make([]wfMonitor, 0, len(monitors)-1)
		for _, mon := range monitors {
			if mon.ID != wfDeleteMonitor {
				out = append(out, mon)
			}
		}
		if err := saveMonitors(cfg, out); err != nil {
			return cmdErr(fmt.Errorf("saving monitors: %w", err))
		}
		printSuccess(fmt.Sprintf("Deleted: %s", deleteName))
		return nil
	},
}

// ---------------------------------------------------------------------------
// forge watchforge incidents
// ---------------------------------------------------------------------------

var (
	wfIncidentsMonitor string
	wfIncidentsSince   string
)

var wfIncidentsCmd = &cobra.Command{
	Use:   "incidents",
	Short: "List incidents",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := requireInit()
		if err != nil {
			return cmdErr(err)
		}
		incidents, err := loadIncidents(cfg)
		if err != nil {
			return cmdErr(fmt.Errorf("loading incidents: %w", err))
		}

		var since time.Time
		if wfIncidentsSince != "" {
			d, err := parseSinceDuration(wfIncidentsSince)
			if err != nil {
				return cmdErr(err)
			}
			since = time.Now().Add(-d)
		}

		var filtered []wfIncident
		for _, inc := range incidents {
			if wfIncidentsMonitor != "" && inc.MonitorID != wfIncidentsMonitor {
				continue
			}
			if !since.IsZero() && inc.OpenedAt.Before(since) {
				continue
			}
			filtered = append(filtered, inc)
		}

		if isJSON() {
			if filtered == nil {
				filtered = []wfIncident{}
			}
			printJSON(filtered)
			return nil
		}
		if len(filtered) == 0 {
			fmt.Println("No incidents.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "MONITOR\tOPENED\tCLOSED\tREASON")
		for _, inc := range filtered {
			closed := "-"
			if inc.ClosedAt != nil {
				closed = inc.ClosedAt.UTC().Format("2006-01-02 15:04")
			}
			reason := inc.Reason
			if len(reason) > 50 {
				reason = reason[:47] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				inc.MonitorName,
				inc.OpenedAt.UTC().Format("2006-01-02 15:04"),
				closed,
				reason,
			)
		}
		w.Flush()
		return nil
	},
}

// ---------------------------------------------------------------------------
// forge watchforge update
// ---------------------------------------------------------------------------

var (
	wfUpdateMonitor        string
	wfUpdateName           string
	wfUpdateTarget         string
	wfUpdateInterval       int
	wfUpdatePublic         string
	wfUpdateThreshold      int
	wfUpdateContains       string
	wfUpdateExpectedStatus int
	wfUpdateTimeout        int
	wfUpdateGrace          int
)

var wfUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update monitor configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := requireInit()
		if err != nil {
			return cmdErr(err)
		}
		monitors, err := loadMonitors(cfg)
		if err != nil {
			return cmdErr(fmt.Errorf("loading monitors: %w", err))
		}
		m, err := findMonitor(monitors, wfUpdateMonitor)
		if err != nil {
			return cmdErr(err)
		}

		if cmd.Flags().Changed("name") {
			m.Name = wfUpdateName
		}
		if cmd.Flags().Changed("target") {
			m.Target = wfUpdateTarget
		}
		if cmd.Flags().Changed("interval") {
			m.IntervalSec = wfUpdateInterval
		}
		if cmd.Flags().Changed("public") {
			v, err := strconv.ParseBool(wfUpdatePublic)
			if err != nil {
				return cmdErr(fmt.Errorf("--public must be true or false"))
			}
			m.Public = v
		}
		if cmd.Flags().Changed("threshold") {
			m.Threshold = wfUpdateThreshold
		}
		if cmd.Flags().Changed("contains") {
			m.Contains = wfUpdateContains
		}
		if cmd.Flags().Changed("expected-status") {
			m.ExpectedStatus = wfUpdateExpectedStatus
		}
		if cmd.Flags().Changed("timeout") {
			m.TimeoutSec = wfUpdateTimeout
		}
		if cmd.Flags().Changed("grace") {
			m.Grace = wfUpdateGrace
		}

		if err := saveMonitors(cfg, monitors); err != nil {
			return cmdErr(fmt.Errorf("saving monitors: %w", err))
		}
		if isJSON() {
			printJSON(m)
		} else {
			printSuccess(fmt.Sprintf("Updated: %s", m.Name))
		}
		return nil
	},
}

// ---------------------------------------------------------------------------
// forge watchforge heartbeat-url
// ---------------------------------------------------------------------------

var wfHeartbeatURLMonitor string

var wfHeartbeatURLCmd = &cobra.Command{
	Use:   "heartbeat-url",
	Short: "Print the heartbeat ping URL for a heartbeat monitor",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := requireInit()
		if err != nil {
			return cmdErr(err)
		}
		monitors, err := loadMonitors(cfg)
		if err != nil {
			return cmdErr(fmt.Errorf("loading monitors: %w", err))
		}
		m, err := findMonitor(monitors, wfHeartbeatURLMonitor)
		if err != nil {
			return cmdErr(err)
		}
		if m.Type != wfTypeHeartbeat {
			return cmdErr(fmt.Errorf("monitor %q is type %s, not heartbeat", m.Name, m.Type))
		}
		url := wfHeartbeatURL(cfg, m.ID, m.Token)
		if isJSON() {
			printJSON(map[string]string{"url": url, "monitor_id": m.ID})
		} else {
			fmt.Println(url)
		}
		return nil
	},
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func wfHeartbeatURL(cfg *sharedconfig.Config, id, token string) string {
	domain := cfg.Forge.Domain
	if domain == "" {
		return fmt.Sprintf("/_watchforge/heartbeat/%s/%s", id, token)
	}
	return fmt.Sprintf("https://status.%s/_watchforge/heartbeat/%s/%s", domain, id, token)
}

func boolYN(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func parseSinceDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "d") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q -- use Go duration syntax (1h, 30m) or Nd for N days", s)
	}
	return d, nil
}
