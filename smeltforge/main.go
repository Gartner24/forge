package main

import (
	"os"

	"github.com/gartner24/forge/smeltforge/cmd"
)

func main() {
	// When started with no arguments (e.g. via `forge start smeltforge`),
	// default to daemon mode.
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "serve")
	}
	cmd.Execute()
}
