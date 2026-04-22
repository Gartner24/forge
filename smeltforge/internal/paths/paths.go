package paths

import (
	"os"
	"path/filepath"
)

func forgeDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".forge", "data", "smeltforge"), nil
}

func RegistryFile() (string, error) {
	d, err := forgeDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "registry", "projects.json"), nil
}

func AuditLog() (string, error) {
	d, err := forgeDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "deploys.log"), nil
}

func Workspace(projectID string) (string, error) {
	d, err := forgeDataDir()
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
	return forgeDataDir()
}
