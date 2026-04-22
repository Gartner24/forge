package cmd

import (
	"fmt"

	"github.com/gartner24/forge/core/internal/state"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop <module>",
	Short: "Stop a running Forge module",
	Args:  cobra.ExactArgs(1),
	RunE:  runStop,
}

func runStop(cmd *cobra.Command, args []string) error {
	module := args[0]

	if _, err := requireInit(); err != nil {
		return cmdErr(err)
	}

	s, err := state.Get(module)
	if err != nil {
		return cmdErr(fmt.Errorf("reading state: %w", err))
	}
	if s == nil || !state.IsAlive(s) {
		state.Delete(module) // clean up stale state if present
		return cmdErr(fmt.Errorf("%s is not running", module))
	}

	if err := stopModule(module, s); err != nil {
		return cmdErr(fmt.Errorf("stopping %s: %w", module, err))
	}
	state.Delete(module)

	printSuccess(fmt.Sprintf("%s stopped", module))
	return nil
}
