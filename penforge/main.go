package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/gartner24/forge/penforge/cmd"
)

func main() {
	// When invoked as a daemon (no subcommand args), start the background services.
	if len(os.Args) == 1 {
		runDaemon()
		return
	}
	cmd.Execute()
}

func runDaemon() {
	mod := NewModule()
	if err := mod.Start(); err != nil {
		os.Stderr.WriteString("penforge: failed to start: " + err.Error() + "\n")
		os.Exit(1)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	mod.Stop()
}
