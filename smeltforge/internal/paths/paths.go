package paths

import (
	"os"
	"path/filepath"

	"github.com/gartner24/forge/shared/config"
)

func moduleDataDir() (string, error) {
	cfg, err := config.Load()
	if err == nil && cfg != nil {
		if m, ok := cfg.Modules["smeltforge"]; ok && m.DataDir != "" {
			return m.DataDir, nil
		}
		if cfg.Forge.DataDir != "" {
			return filepath.Join(cfg.Forge.DataDir, "smeltforge"), nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "/opt/data/smeltforge", nil
	}
	return filepath.Join(home, ".forge", "data", "smeltforge"), nil
}

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

func moduleLogsDir() (string, error) {
	cfg, err := config.Load()
	if err == nil && cfg != nil && cfg.Forge.DataDir != "" {
		return filepath.Join(cfg.Forge.DataDir, "logs", "smeltforge"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "/opt/data/logs/smeltforge", nil
	}
	return filepath.Join(home, ".forge", "data", "logs", "smeltforge"), nil
}

func RegistryFile() (string, error) {
	installDir, err := moduleInstallDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(installDir, "registry", "smeltforge", "projects.json"), nil
}

func AuditLog() (string, error) {
	logsDir, err := moduleLogsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(logsDir, "audit.log"), nil
}

func Workspace(projectID string) (string, error) {
	d, err := moduleDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "workspaces", projectID), nil
}

func SecretsFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".forge", "secrets.age"), nil
}

func DataDir() (string, error) {
	return moduleDataDir()
}
