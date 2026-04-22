package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gartner24/forge/smeltforge/internal/deploy"
)

var deployNode string

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a project",
	RunE:  runDeploy,
}

func init() {
	deployCmd.Flags().StringVar(&projectID, "project", "", "Project ID (required)")
	deployCmd.Flags().StringVar(&deployNode, "node", "", "Target mesh node IP (requires FluxForge)")
	deployCmd.MarkFlagRequired("project")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	if deployNode != "" {
		fmt.Printf("Note: --node targeting requires FluxForge. Deploying locally.\n")
	}

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

	fmt.Printf("Deploying %s...\n", projectID)
	result, err := engine.Deploy(context.Background(), projectID, "cli")
	if err != nil {
		return cmdErr(fmt.Errorf("deploy failed: %w", err))
	}

	if isJSON() {
		printJSON(map[string]any{
			"ok":       true,
			"project":  result.ProjectID,
			"image":    result.Image,
			"strategy": result.Strategy,
			"duration": result.Duration.String(),
		})
	} else {
		fmt.Printf("deployed %s (%s) in %s\n", result.ProjectID, result.Strategy, result.Duration.Round(1e6))
	}
	return nil
}
