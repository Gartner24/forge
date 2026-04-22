package paths

import (
	"os"
	"path/filepath"
)

func ForgeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".forge"), nil
}

func ConfigFile() (string, error) {
	d, err := ForgeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.toml"), nil
}

func SecretsFile() (string, error) {
	d, err := ForgeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "secrets.age"), nil
}

func RunDir() (string, error) {
	d, err := ForgeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "run"), nil
}

func BinDir() (string, error) {
	d, err := ForgeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "bin"), nil
}

func DataDir() (string, error) {
	d, err := ForgeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "data"), nil
}

func ModuleBin(module string) (string, error) {
	bin, err := BinDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(bin, module), nil
}

func ModuleStateFile(module string) (string, error) {
	run, err := RunDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(run, module+".json"), nil
}

func ModuleLogFile(module string) (string, error) {
	data, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(data, module, "forge.log"), nil
}
