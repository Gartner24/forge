package cmd

import (
	"fmt"
	"time"

	"github.com/gartner24/forge/sparkforge/internal/deliverylog"
	"github.com/gartner24/forge/sparkforge/internal/model"
	"github.com/spf13/cobra"
)

var logsSince string

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show delivery log",
	RunE:  runLogs,
}

func init() {
	logsCmd.Flags().StringVar(&logsSince, "since", "", "Show entries since duration (e.g. 1h, 24h, 7d)")
}

func runLogs(cmd *cobra.Command, args []string) error {
	var since time.Time
	if logsSince != "" {
		d, err := parseDuration(logsSince)
		if err != nil {
			return cmdErr(fmt.Errorf("invalid --since value %q: %w", logsSince, err))
		}
		since = time.Now().Add(-d)
	}

	dl, err := deliverylog.New()
	if err != nil {
		return cmdErr(err)
	}

	records, err := dl.Read(since)
	if err != nil {
		return cmdErr(err)
	}

	if isJSON() {
		if records == nil {
			records = []model.DeliveryRecord{}
		}
		printJSON(records)
		return nil
	}

	if len(records) == 0 {
		fmt.Println("No delivery log entries.")
		return nil
	}

	w := newTabWriter()
	fmt.Fprintln(w, "TIME\tCHANNEL\tPRIORITY\tSOURCE\tTITLE\tSTATUS")
	for _, r := range records {
		errSuffix := ""
		if r.Error != "" {
			errSuffix = " (" + r.Error + ")"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s%s\n",
			r.Timestamp.Format("2006-01-02 15:04:05"),
			r.ChannelName,
			r.Priority,
			r.Source,
			r.Title,
			r.Status,
			errSuffix,
		)
	}
	w.Flush()
	return nil
}

func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("too short")
	}
	unit := s[len(s)-1]
	valueStr := s[:len(s)-1]
	var value int
	if _, err := fmt.Sscanf(valueStr, "%d", &value); err != nil {
		return 0, fmt.Errorf("expected integer prefix")
	}
	switch unit {
	case 'h':
		return time.Duration(value) * time.Hour, nil
	case 'd':
		return time.Duration(value) * 24 * time.Hour, nil
	case 'm':
		return time.Duration(value) * time.Minute, nil
	default:
		return 0, fmt.Errorf("unknown unit %q: use h, d, or m", string(unit))
	}
}
