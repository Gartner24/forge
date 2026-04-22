package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/gartner24/forge/sparkforge/internal/dedup"
	"github.com/gartner24/forge/sparkforge/internal/model"
	"github.com/spf13/cobra"
)

var alertsCmd = &cobra.Command{
	Use:   "alerts",
	Short: "Manage active alerts",
}

var alertsFilterPriority string
var acknowledgeAll bool

var alertsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active alerts",
	RunE:  runAlertsList,
}

var alertsAcknowledgeCmd = &cobra.Command{
	Use:   "acknowledge [<id>]",
	Short: "Acknowledge (clear) an active alert",
	RunE:  runAlertsAcknowledge,
}

func init() {
	alertsListCmd.Flags().StringVar(&alertsFilterPriority, "priority", "", "Filter by minimum priority")
	alertsAcknowledgeCmd.Flags().BoolVar(&acknowledgeAll, "all", false, "Acknowledge all active alerts")
	alertsCmd.AddCommand(alertsListCmd)
	alertsCmd.AddCommand(alertsAcknowledgeCmd)
}

func runAlertsList(cmd *cobra.Command, args []string) error {
	store, err := dedup.New()
	if err != nil {
		return cmdErr(err)
	}

	alerts, err := store.List()
	if err != nil {
		return cmdErr(err)
	}

	if alertsFilterPriority != "" {
		minP, err := model.ParsePriority(alertsFilterPriority)
		if err != nil {
			return cmdErr(err)
		}
		filtered := alerts[:0]
		for _, a := range alerts {
			if a.Priority.Level() >= minP.Level() {
				filtered = append(filtered, a)
			}
		}
		alerts = filtered
	}

	if isJSON() {
		if alerts == nil {
			alerts = []model.Alert{}
		}
		printJSON(alerts)
		return nil
	}

	if len(alerts) == 0 {
		fmt.Println("No active alerts.")
		return nil
	}

	w := newTabWriter()
	fmt.Fprintln(w, "ID\tPRIORITY\tSOURCE\tEVENT_TYPE\tTITLE\tFIRED")
	for _, a := range alerts {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			a.ID, a.Priority, a.Source, a.EventType, a.Title,
			formatSince(a.FiredAt),
		)
	}
	w.Flush()
	return nil
}

func runAlertsAcknowledge(cmd *cobra.Command, args []string) error {
	store, err := dedup.New()
	if err != nil {
		return cmdErr(err)
	}

	if acknowledgeAll {
		alerts, err := store.List()
		if err != nil {
			return cmdErr(err)
		}
		for _, a := range alerts {
			if err := store.Remove(a.ID); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to acknowledge %s: %v\n", a.ID, err)
			}
		}
		printSuccess(fmt.Sprintf("acknowledged %d alert(s)", len(alerts)))
		return nil
	}

	if len(args) == 0 {
		return cmdErr(fmt.Errorf("provide an alert ID or use --all"))
	}

	if err := store.Remove(args[0]); err != nil {
		return cmdErr(err)
	}
	printSuccess(fmt.Sprintf("alert %s acknowledged", args[0]))
	return nil
}

func formatSince(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh%dm", h, m)
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
