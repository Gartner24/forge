package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gartner24/forge/smeltforge/internal/deploy"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Roll back a project to its previous image",
	RunE:  runRollback,
}

func init() {
	rollbackCmd.Flags().StringVar(&projectID, "project", "", "Project ID (required)")
	rollbackCmd.MarkFlagRequired("project")
}

func runRollback(cmd *cobra.Command, args []string) error {
	reg, err := loadRegistry()
	if err != nil {
		return cmdErr(err)
	}

	d, err := newDocker()
	if err != nil {
		return cmdErr(err)
	}
	defer d.Close()

	engine, err := deploy.NewEngine(reg, d, newCaddy())
	if err != nil {
		return cmdErr(err)
	}

	fmt.Printf("Rolling back %s...\n", projectID)
	if err := engine.Rollback(context.Background(), projectID); err != nil {
		return cmdErr(err)
	}

	printSuccess(fmt.Sprintf("rolled back %s", projectID))
	return nil
}
