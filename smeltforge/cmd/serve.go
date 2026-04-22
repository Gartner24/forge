package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/gartner24/forge/smeltforge/internal/deploy"
	"github.com/gartner24/forge/smeltforge/internal/server"
)

var serveAddr string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the SmeltForge daemon (webhook listener, git poller)",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().StringVar(&serveAddr, "addr", ":9320", "HTTP listen address for webhook/CI endpoints")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
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

	store, err := loadSecrets()
	if err != nil {
		return cmdErr(err)
	}

	deployFn := func(ctx context.Context, projectID, trigger string) (*deploy.Result, error) {
		return engine.Deploy(ctx, projectID, trigger)
	}
	secretLookup := func(key string) (string, error) {
		return store.Get(key)
	}

	srv := server.New(reg, deployFn, secretLookup)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go srv.RunWorker(ctx)
	go srv.RunPoller(ctx)

	httpSrv := &http.Server{Addr: serveAddr, Handler: srv}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		cancel()
		httpSrv.Shutdown(context.Background())
	}()

	fmt.Printf("smeltforge listening on %s\n", serveAddr)
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return cmdErr(err)
	}
	return nil
}
