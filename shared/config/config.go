package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config is the full structure of ~/.forge/config.toml
type Config struct {
	Forge   ForgeConfig             `toml:"forge"`
	Modules map[string]ModuleConfig `toml:"modules"`
}

// ForgeConfig holds global forge settings.
type ForgeConfig struct {
	Domain     string `toml:"domain"`
	DataDir    string `toml:"data_dir"`
	InstallDir string `toml:"install_dir"`
	Version    string `toml:"version"`
	LogLevel   string `toml:"log_level"`
}

// ModuleConfig holds per-module settings.
type ModuleConfig struct {
	Enabled bool   `toml:"enabled"`
	APIAddr string `toml:"api_addr"`
	DataDir string `toml:"data_dir"`
}

// Load reads ~/.forge/config.toml and returns a parsed Config.
// If the file does not exist, it returns an empty Config with no errors.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	return LoadFrom(path)
}

// LoadFrom reads config from an explicit path. Returns an empty Config if the file does not exist.
func LoadFrom(path string) (*Config, error) {
	cfg := &Config{}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	return cfg, nil
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".forge", "config.toml"), nil
}
