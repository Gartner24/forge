package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gartner24/forge/smeltforge/internal/docker"
	"github.com/gartner24/forge/smeltforge/internal/registry"
)

var (
	logsLines int
	logsSince string
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show container logs for a project",
	RunE:  runLogs,
}

func init() {
	logsCmd.Flags().StringVar(&projectID, "project", "", "Project ID (required)")
	logsCmd.Flags().IntVar(&logsLines, "lines", 100, "Number of lines to show")
	logsCmd.Flags().StringVar(&logsSince, "since", "", "Show logs from last duration (e.g. 1h)")
	logsCmd.MarkFlagRequired("project")
}

func runLogs(cmd *cobra.Command, args []string) error {
	reg, err := loadRegistry()
	if err != nil {
		return cmdErr(err)
	}

	p, err := reg.Get(projectID)
	if err != nil {
		return cmdErr(err)
	}

	containerName := activeContainerName(p)

	d, err := newDocker()
	if err != nil {
		return cmdErr(err)
	}
	defer d.Close()

	tail := fmt.Sprintf("%d", logsLines)
	output, err := d.Logs(context.Background(), containerName, tail, logsSince)
	if err != nil {
		return cmdErr(err)
	}

	fmt.Print(output)
	return nil
}

// activeContainerName returns the name of the currently-running container for a project.
func activeContainerName(p *registry.Project) string {
	if p.Strategy == "blue-green" && p.State.Color != "" {
		return "smeltforge-" + p.ID + "-" + p.State.Color
	}
	return docker.ContainerName(p.ID)
}
