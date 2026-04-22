package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

func forgeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home dir: %w", err)
	}
	return filepath.Join(home, ".forge"), nil
}

func sparkforgeDataDir() (string, error) {
	d, err := forgeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "data", "sparkforge"), nil
}

func ChannelsFile() (string, error) {
	d, err := sparkforgeDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "registry", "channels.json"), nil
}

func DeliveryLogFile() (string, error) {
	d, err := sparkforgeDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "data", "delivery.log"), nil
}

func AlertsFile() (string, error) {
	d, err := sparkforgeDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "data", "alerts.json"), nil
}

func SecretsFile() (string, error) {
	d, err := forgeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "secrets.age"), nil
}

func GotifyDataDir() (string, error) {
	d, err := sparkforgeDataDir()
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
