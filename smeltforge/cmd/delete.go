package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gartner24/forge/smeltforge/internal/docker"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Stop and remove a project",
	RunE:  runDelete,
}

func init() {
	deleteCmd.Flags().StringVar(&projectID, "project", "", "Project ID (required)")
	deleteCmd.Flags().BoolVar(&yesFlag, "yes", false, "Skip confirmation")
	deleteCmd.MarkFlagRequired("project")
}

func runDelete(cmd *cobra.Command, args []string) error {
	reg, err := loadRegistry()
	if err != nil {
		return cmdErr(err)
	}

	p, err := reg.Get(projectID)
	if err != nil {
		return cmdErr(err)
	}

	proceed, err := mustConfirm(fmt.Sprintf("Stop and remove project %q? This cannot be undone.", projectID))
	if err != nil {
		return cmdErr(err)
	}
	if !proceed {
		fmt.Println("Aborted.")
		return nil
	}

	d, err := newDocker()
	if err != nil {
		return cmdErr(err)
	}
	defer d.Close()

	ctx := context.Background()

	// Stop containers (handle both stop-start and blue-green).
	_ = d.StopAndRemove(ctx, docker.ContainerName(p.ID))
	_ = d.StopAndRemove(ctx, docker.BlueContainerName(p.ID))
	_ = d.StopAndRemove(ctx, docker.GreenContainerName(p.ID))

	// Remove Caddy route.
	newCaddy().RemoveRoute(projectID)

	if err := reg.Delete(projectID); err != nil {
		return cmdErr(err)
	}

	fmt.Printf("project %q deleted\n", projectID)
	fmt.Printf("Note: env vars not removed. Run 'smeltforge env list %s' to clean up.\n", projectID)
	return nil
}
