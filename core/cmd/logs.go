package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gartner24/forge/core/internal/paths"
	"github.com/spf13/cobra"
)

var (
	logsLines int
	logsSince string
)

var logsCmd = &cobra.Command{
	Use:   "logs <module>",
	Short: "View module logs",
	Args:  cobra.ExactArgs(1),
	RunE:  runLogs,
}

func init() {
	logsCmd.Flags().IntVar(&logsLines, "lines", 0, "Number of lines to show (0 = follow live)")
	logsCmd.Flags().StringVar(&logsSince, "since", "", "Show logs from last duration (e.g. 1h, 30m)")
}

func runLogs(cmd *cobra.Command, args []string) error {
	module := args[0]

	if _, err := requireInit(); err != nil {
		return cmdErr(err)
	}

	logPath, err := paths.ModuleLogFile(module)
	if err != nil {
		return cmdErr(err)
	}

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return cmdErr(fmt.Errorf("no log file for %s (has it been started?)", module))
	}

	var sinceTime time.Time
	if logsSince != "" {
		d, err := time.ParseDuration(logsSince)
		if err != nil {
			return cmdErr(fmt.Errorf("invalid --since value %q: expected a duration like 1h, 30m", logsSince))
		}
		sinceTime = time.Now().Add(-d)
	}

	if logsLines > 0 {
		return showLastLines(logPath, logsLines, sinceTime)
	}

	if !sinceTime.IsZero() {
		// --since without --lines: read all matching lines from start.
		return showSince(logPath, sinceTime)
	}

	// Default: live tail.
	return tailLive(logPath)
}

// showLastLines reads the entire file and prints the last n lines.
// If sinceTime is non-zero, only lines whose prefix timestamp is after that time are counted.
func showLastLines(path string, n int, since time.Time) error {
	f, err := os.Open(path)
	if err != nil {
		return cmdErr(fmt.Errorf("opening log: %w", err))
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !since.IsZero() && !lineAfter(line, since) {
			continue
		}
		lines = append(lines, line)
		// Keep a sliding window of n lines.
		if len(lines) > n {
			lines = lines[len(lines)-n:]
		}
	}
	if err := sc.Err(); err != nil {
		return cmdErr(fmt.Errorf("reading log: %w", err))
	}
	for _, l := range lines {
		fmt.Println(l)
	}
	return nil
}

// showSince reads the file from the beginning and prints lines after sinceTime.
func showSince(path string, since time.Time) error {
	f, err := os.Open(path)
	if err != nil {
		return cmdErr(fmt.Errorf("opening log: %w", err))
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if lineAfter(line, since) {
			fmt.Println(line)
		}
	}
	return sc.Err()
}

// tailLive follows the log file, printing new bytes as they arrive.
func tailLive(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return cmdErr(fmt.Errorf("opening log: %w", err))
	}
	defer f.Close()

	// Seek to end to show only new lines.
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return cmdErr(fmt.Errorf("seeking log: %w", err))
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigs)

	buf := make([]byte, 4096)
	for {
		select {
		case <-sigs:
			return nil
		default:
			n, _ := f.Read(buf)
			if n > 0 {
				os.Stdout.Write(buf[:n])
			} else {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

// lineAfter tries to parse an RFC3339 or similar timestamp at the start of a
// log line. Returns true if the line's timestamp is after t, or if no timestamp
// is detected (passes all lines through gracefully).
func lineAfter(line string, t time.Time) bool {
	if len(line) < 20 {
		return true
	}
	// Try common timestamp prefixes: RFC3339 up to 'Z' or +00:00.
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		end := len(f)
		if end > len(line) {
			continue
		}
		if ts, err := time.Parse(f, line[:end]); err == nil {
			return ts.After(t)
		}
	}
	return true // no timestamp found -- include line
}
