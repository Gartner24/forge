package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gartner24/forge/smeltforge/internal/registry"
)

var pollingCmd = &cobra.Command{
	Use:   "polling",
	Short: "Manage git polling triggers",
}

var pollingEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable git polling for a project",
	RunE:  runPollingEnable,
}

var pollingDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable git polling for a project",
	RunE:  runPollingDisable,
}

var pollingInterval int

func init() {
	pollingEnableCmd.Flags().StringVar(&projectID, "project", "", "Project ID (required)")
	pollingEnableCmd.Flags().IntVar(&pollingInterval, "interval", 60, "Poll interval in seconds")
	pollingEnableCmd.MarkFlagRequired("project")

	pollingDisableCmd.Flags().StringVar(&projectID, "project", "", "Project ID (required)")
	pollingDisableCmd.MarkFlagRequired("project")

	pollingCmd.AddCommand(pollingEnableCmd)
	pollingCmd.AddCommand(pollingDisableCmd)
}

func runPollingEnable(cmd *cobra.Command, args []string) error {
	reg, err := loadRegistry()
	if err != nil {
		return cmdErr(err)
	}
	p, err := reg.Get(projectID)
	if err != nil {
		return cmdErr(err)
	}
	if p.Source.Type != "git" {
		return cmdErr(fmt.Errorf("polling only supported for git source projects"))
	}

	branch := p.Source.Branch
	if branch == "" {
		branch = "main"
	}
	p.Trigger = registry.Trigger{
		Type:     "polling",
		Interval: pollingInterval,
		Branch:   branch,
	}
	p.PollingOn = true
	if err := reg.Update(p); err != nil {
		return cmdErr(err)
	}

	printSuccess(fmt.Sprintf("polling enabled for %s (interval: %ds, branch: %s)", projectID, pollingInterval, branch))
	return nil
}

func runPollingDisable(cmd *cobra.Command, args []string) error {
	reg, err := loadRegistry()
	if err != nil {
		return cmdErr(err)
	}
	p, err := reg.Get(projectID)
	if err != nil {
		return cmdErr(err)
	}

	p.Trigger = registry.Trigger{Type: "manual"}
	p.PollingOn = false
	if err := reg.Update(p); err != nil {
		return cmdErr(err)
	}

	printSuccess(fmt.Sprintf("polling disabled for %s", projectID))
	return nil
}
