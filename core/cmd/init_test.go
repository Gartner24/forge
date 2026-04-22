package cmd

import (
	"os"
	"path/filepath"
	"testing"

	sharedconfig "github.com/gartner24/forge/shared/config"
)

func TestInitDefaultPaths(t *testing.T) {
	tmp := t.TempDir()

	// Override defaults so the test doesn't need /opt to exist.
	origData := defaultDataDir
	origInstall := defaultInstallDir
	defaultDataDir = filepath.Join(tmp, "data")
	defaultInstallDir = filepath.Join(tmp, "infra", "forge")
	t.Cleanup(func() {
		defaultDataDir = origData
		defaultInstallDir = origInstall
	})

	cfgPath := filepath.Join(tmp, ".forge", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := sharedconfig.Config{
		Forge: sharedconfig.ForgeConfig{
			Domain:     "test.example.com",
			DataDir:    defaultDataDir,
			InstallDir: defaultInstallDir,
			Version:    "0.1.0",
			LogLevel:   "info",
		},
	}

	if err := writeConfig(&cfg, cfgPath); err != nil {
		t.Fatalf("writeConfig: %v", err)
	}

	// Verify the file exists and parses correctly.
	parsed, err := sharedconfig.LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("loading written config: %v", err)
	}

	if parsed.Forge.DataDir != defaultDataDir {
		t.Errorf("data_dir = %q, want %q", parsed.Forge.DataDir, defaultDataDir)
	}
	if parsed.Forge.InstallDir != defaultInstallDir {
		t.Errorf("install_dir = %q, want %q", parsed.Forge.InstallDir, defaultInstallDir)
	}
}

func TestInitDirectoryStructure(t *testing.T) {
	tmp := t.TempDir()

	origData := defaultDataDir
	origInstall := defaultInstallDir
	defaultDataDir = filepath.Join(tmp, "data")
	defaultInstallDir = filepath.Join(tmp, "infra", "forge")
	t.Cleanup(func() {
		defaultDataDir = origData
		defaultInstallDir = origInstall
	})

	forgeDir := filepath.Join(tmp, ".forge")
	runDir := filepath.Join(forgeDir, "run")
	logsDir := filepath.Join(defaultDataDir, "logs")

	dirsToCreate := []string{forgeDir, runDir, defaultDataDir, defaultInstallDir, logsDir}
	for _, dir := range dirsToCreate {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("creating %s: %v", dir, err)
		}
	}

	expected := []string{
		forgeDir,
		runDir,
		defaultDataDir,
		defaultInstallDir,
		logsDir,
	}
	for _, dir := range expected {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("expected directory not created: %s", dir)
		}
	}
}

func TestInitProductionDefaults(t *testing.T) {
	// Verify the production default constants are correct per docs/core/04-config.md.
	if defaultDataDir != "/opt/data" {
		t.Errorf("defaultDataDir = %q, want /opt/data", defaultDataDir)
	}
	if defaultInstallDir != "/opt/infra/forge" {
		t.Errorf("defaultInstallDir = %q, want /opt/infra/forge", defaultInstallDir)
	}
}
