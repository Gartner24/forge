package cmd

import (
	"fmt"

	"github.com/gartner24/forge/sparkforge/internal/api"
	"github.com/gartner24/forge/sparkforge/internal/paths"
	"github.com/gartner24/forge/sparkforge/internal/router"
	"github.com/spf13/cobra"
)

func runServe(cmd *cobra.Command, args []string) error {
	if err := paths.EnsureDirs(); err != nil {
		return cmdErr(fmt.Errorf("ensuring data dirs: %w", err))
	}

	r, err := router.New()
	if err != nil {
		return cmdErr(fmt.Errorf("initialising router: %w", err))
	}

	srv, err := api.New(r)
	if err != nil {
		return cmdErr(fmt.Errorf("initialising API server: %w", err))
	}

	if err := srv.Start(); err != nil {
		return cmdErr(fmt.Errorf("API server: %w", err))
	}
	return nil
}
