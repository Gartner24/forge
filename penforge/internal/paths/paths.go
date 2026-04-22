package paths

import (
	"os"
	"path/filepath"

	"github.com/gartner24/forge/shared/config"
)

func moduleDataDir() (string, error) {
	cfg, err := config.Load()
	if err == nil && cfg != nil {
		if m, ok := cfg.Modules["penforge"]; ok && m.DataDir != "" {
			return m.DataDir, nil
		}
		if cfg.Forge.DataDir != "" {
			return filepath.Join(cfg.Forge.DataDir, "penforge"), nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "/opt/data/penforge", nil
	}
	return filepath.Join(home, ".forge", "data", "penforge"), nil
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

func RegistryDir() (string, error) {
	installDir, err := moduleInstallDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(installDir, "registry", "penforge"), nil
}

func TargetsFile() (string, error) {
	d, err := RegistryDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "targets.json"), nil
}

func DataDir() (string, error) {
	return moduleDataDir()
}

func FindingsFile() (string, error) {
	d, err := moduleDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "findings.json"), nil
}

func ScansDir() (string, error) {
	d, err := moduleDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "scans"), nil
}

func ScanDir(scanID string) (string, error) {
	d, err := ScansDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, scanID), nil
}

func AuditFile() (string, error) {
	cfg, err := config.Load()
	if err == nil && cfg != nil && cfg.Forge.DataDir != "" {
		return filepath.Join(cfg.Forge.DataDir, "logs", "penforge", "audit.log"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "/opt/data/logs/penforge/audit.log", nil
	}
	return filepath.Join(home, ".forge", "data", "logs", "penforge", "audit.log"), nil
}

func TemplatesDir() (string, error) {
	d, err := moduleDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "templates", "nuclei"), nil
}

func EnsureDirs() error {
	dirs := []func() (string, error){
		RegistryDir,
		DataDir,
		ScansDir,
		TemplatesDir,
	}
	for _, fn := range dirs {
		d, err := fn()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}
