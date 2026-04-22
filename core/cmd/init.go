package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/gartner24/forge/core/internal/paths"
	sharedconfig "github.com/gartner24/forge/shared/config"
	"github.com/gartner24/forge/shared/secrets"
	"github.com/spf13/cobra"
)

var initDomain string

// defaultDataDir and defaultInstallDir are the canonical paths written by forge init.
// They are vars (not consts) so tests can override them with temp directories.
var defaultDataDir = "/opt/data"
var defaultInstallDir = "/opt/infra/forge"

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise Forge on this server",
	Long:  "Creates ~/.forge/, generates the age identity key, and writes config.toml.",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().StringVar(&initDomain, "domain", "", "Base domain for dev environments (required)")
	initCmd.MarkFlagRequired("domain")
}

func runInit(cmd *cobra.Command, args []string) error {
	cfgPath, err := paths.ConfigFile()
	if err != nil {
		return cmdErr(err)
	}

	// Idempotent: report current state if already initialised.
	existing, _ := sharedconfig.Load()
	if existing != nil && existing.Forge.Domain != "" {
		printSuccess(fmt.Sprintf("forge is already initialised (domain: %s)", existing.Forge.Domain))
		return nil
	}

	forgeDir := filepath.Dir(cfgPath)
	dataDir := defaultDataDir
	installDir := defaultInstallDir
	runDir := filepath.Join(forgeDir, "run")

	dirsToCreate := []string{forgeDir, runDir, dataDir, installDir, "/opt/data/logs"}
	for _, dir := range dirsToCreate {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return cmdErr(fmt.Errorf("creating %s: %w", dir, err))
		}
	}

	// Set ownership of /opt/data and /opt/infra/forge to the current user.
	if u, err := user.Current(); err == nil {
		uid, _ := strconv.Atoi(u.Uid)
		gid, _ := strconv.Atoi(u.Gid)
		for _, dir := range []string{dataDir, installDir, "/opt/data/logs"} {
			_ = os.Chown(dir, uid, gid)
		}
	}

	// Initialise the secrets store (generates age key if absent).
	secretsPath, err := paths.SecretsFile()
	if err != nil {
		return cmdErr(err)
	}
	if _, err := secrets.New(secretsPath); err != nil {
		return cmdErr(fmt.Errorf("initialising secrets store: %w", err))
	}

	cfg := sharedconfig.Config{
		Forge: sharedconfig.ForgeConfig{
			Domain:     initDomain,
			DataDir:    dataDir,
			InstallDir: installDir,
			Version:    version,
			LogLevel:   "info",
		},
	}

	if err := writeConfig(&cfg, cfgPath); err != nil {
		return cmdErr(fmt.Errorf("writing config: %w", err))
	}

	printSuccess(fmt.Sprintf("forge initialised (domain: %s, config: %s)", initDomain, cfgPath))
	return nil
}

// writeConfig encodes cfg as TOML and writes it to path.
func writeConfig(cfg *sharedconfig.Config, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
