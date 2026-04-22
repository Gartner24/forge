package cmd

import (
	"context"
	"fmt"
	"text/tabwriter"
	"os"

	"github.com/spf13/cobra"

	"github.com/gartner24/forge/smeltforge/internal/docker"
	"github.com/gartner24/forge/smeltforge/internal/registry"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show project status",
	RunE:  runStatus,
}

func init() {
	statusCmd.Flags().StringVar(&projectID, "project", "", "Project ID (show single project)")
}

type projectStatus struct {
	ID          string `json:"id"`
	Domain      string `json:"domain"`
	Strategy    string `json:"strategy"`
	Container   string `json:"container"`
	Image       string `json:"image"`
	State       string `json:"state"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	reg, err := loadRegistry()
	if err != nil {
		return cmdErr(err)
	}

	d, err := newDocker()
	if err != nil {
		return cmdErr(err)
	}
	defer d.Close()

	ctx := context.Background()
	var projects []*registry.Project

	if projectID != "" {
		p, err := reg.Get(projectID)
		if err != nil {
			return cmdErr(err)
		}
		projects = []*registry.Project{p}
	} else {
		projects = reg.All()
	}

	rows := make([]projectStatus, 0, len(projects))
	for _, p := range projects {
		containerName := docker.ContainerName(p.ID)
		if p.Strategy == "blue-green" && p.State.Color != "" {
			containerName = "smeltforge-" + p.ID + "-" + p.State.Color
		}

		cs, _ := d.Status(ctx, containerName)

		ps := projectStatus{
			ID:       p.ID,
			Domain:   p.Domain,
			Strategy: p.Strategy,
		}
		if cs != nil {
			ps.Container = cs.Name
			ps.Image = cs.Image
			ps.State = cs.State
		} else {
			ps.State = "not running"
		}
		rows = append(rows, ps)
	}

	if isJSON() {
		printJSON(rows)
		return nil
	}

	if len(rows) == 0 {
		fmt.Println("No projects registered. Run 'smeltforge add --project <id>' to add one.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROJECT\tDOMAIN\tSTRATEGY\tSTATE\tIMAGE")
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.ID, r.Domain, r.Strategy, r.State, r.Image)
	}
	w.Flush()
	return nil
}
