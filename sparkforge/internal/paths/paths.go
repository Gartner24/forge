package paths

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gartner24/forge/shared/config"
)

// moduleDataDir returns the sparkforge data directory.
// Priority: module config > global config > $HOME/.forge/data/sparkforge (test/pre-init fallback).
// Production systems always have config written by forge init.
func moduleDataDir() (string, error) {
	cfg, err := config.Load()
	if err == nil && cfg != nil {
		if m, ok := cfg.Modules["sparkforge"]; ok && m.DataDir != "" {
			return m.DataDir, nil
		}
		if cfg.Forge.DataDir != "" {
			return filepath.Join(cfg.Forge.DataDir, "sparkforge"), nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "/opt/data/sparkforge", nil
	}
	return filepath.Join(home, ".forge", "data", "sparkforge"), nil
}

// moduleInstallDir returns the forge install directory.
// Priority: global config > $HOME/.forge (test/pre-init fallback).
func moduleInstallDir() (string, error) {
	cfg, err := config.Load()
	if err == nil && cfg != nil && cfg.Forge.InstallDir != "" {
		return cfg.Forge.InstallDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "/opt/infra/forge", nil
	}
	return filepath.Join(home, ".forge"), nil
}

func ChannelsFile() (string, error) {
	installDir, err := moduleInstallDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(installDir, "registry", "sparkforge", "channels.json"), nil
}

func DeliveryLogFile() (string, error) {
	d, err := moduleDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "delivery.log"), nil
}

func AlertsFile() (string, error) {
	d, err := moduleDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "alerts.json"), nil
}

func SecretsFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home dir: %w", err)
	}
	return filepath.Join(home, ".forge", "secrets.age"), nil
}

func GotifyDataDir() (string, error) {
	d, err := moduleDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "gotify"), nil
}

func EnsureDirs() error {
	channelsFile, err := ChannelsFile()
	if err != nil {
		return err
	}
	deliveryLog, err := DeliveryLogFile()
	if err != nil {
		return err
	}
	alertsFile, err := AlertsFile()
	if err != nil {
		return err
	}
	gotifyDir, err := GotifyDataDir()
	if err != nil {
		return err
	}
	for _, p := range []string{
		filepath.Dir(channelsFile),
		filepath.Dir(deliveryLog),
		filepath.Dir(alertsFile),
		gotifyDir,
	} {
		if err := os.MkdirAll(p, 0755); err != nil {
			return fmt.Errorf("creating %s: %w", p, err)
		}
	}
	return nil
}
