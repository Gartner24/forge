package paths

import (
	"os"
	"path/filepath"
)

func baseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".forge", "data", "penforge"), nil
}

func RegistryDir() (string, error) {
	base, err := baseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "registry"), nil
}

func TargetsFile() (string, error) {
	d, err := RegistryDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "targets.json"), nil
}

func DataDir() (string, error) {
	base, err := baseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "data"), nil
}

func FindingsFile() (string, error) {
	d, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "findings.json"), nil
}

func ScansDir() (string, error) {
	d, err := DataDir()
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
	d, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "audit.log"), nil
}

func TemplatesDir() (string, error) {
	base, err := baseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "templates", "nuclei"), nil
}

// EnsureDirs creates all required penforge data directories.
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
