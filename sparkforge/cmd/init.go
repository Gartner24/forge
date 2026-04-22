package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/gartner24/forge/shared/secrets"
	"github.com/gartner24/forge/sparkforge/internal/model"
	"github.com/gartner24/forge/sparkforge/internal/paths"
	"github.com/gartner24/forge/sparkforge/internal/registry"
	"github.com/spf13/cobra"
)

const (
	gotifyContainerName = "forge-gotify"
	gotifyImage         = "gotify/server:latest"
	gotifyPort          = "7779"
	gotifyAdminUser     = "admin"
	gotifyAdminPass     = "admin"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise SparkForge and set up Gotify",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	if err := paths.EnsureDirs(); err != nil {
		return cmdErr(fmt.Errorf("creating data dirs: %w", err))
	}

	reg, err := registry.New()
	if err != nil {
		return cmdErr(err)
	}

	channels, err := reg.List()
	if err != nil {
		return cmdErr(err)
	}
	for _, ch := range channels {
		if ch.Type == model.ChannelTypeGotify {
			printSuccess("sparkforge already initialised (Gotify channel exists)")
			return nil
		}
	}

	fmt.Println("Setting up SparkForge...")

	if err := setupGotify(reg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Gotify setup failed (%v). Mobile push notifications will be unavailable.\n", err)
		fmt.Println("SparkForge initialised without Gotify. Re-run 'sparkforge init' after fixing Docker to enable mobile push.")
	} else {
		fmt.Println("Gotify started. Install the Gotify app and connect to http://localhost:7779 to receive push notifications.")
	}

	printSuccess("sparkforge init complete")
	return nil
}

func setupGotify(reg *registry.Registry) error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker not found in PATH")
	}

	// Check if container already running.
	out, _ := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", gotifyContainerName).Output()
	if string(out) != "true\n" {
		gotifyDataDir, err := paths.GotifyDataDir()
		if err != nil {
			return err
		}
		runArgs := []string{
			"run", "-d",
			"--name", gotifyContainerName,
			"-p", gotifyPort + ":80",
			"-v", gotifyDataDir + ":/app/data",
			"--restart", "unless-stopped",
			gotifyImage,
		}
		if out, err := exec.Command("docker", runArgs...).CombinedOutput(); err != nil {
			// Try pulling first.
			exec.Command("docker", "pull", gotifyImage).Run()
			if out2, err2 := exec.Command("docker", runArgs...).CombinedOutput(); err2 != nil {
				return fmt.Errorf("docker run failed: %w\n%s", err2, string(out2))
			}
			_ = out
		}
	}

	// Wait for Gotify to be ready.
	gotifyURL := "http://localhost:" + gotifyPort
	if err := waitForGotify(gotifyURL, 15*time.Second); err != nil {
		return fmt.Errorf("waiting for gotify: %w", err)
	}

	// Create a Forge application in Gotify.
	appToken, err := createGotifyApp(gotifyURL)
	if err != nil {
		return fmt.Errorf("creating gotify app: %w", err)
	}

	// Store token in secrets.
	secretsPath, err := paths.SecretsFile()
	if err != nil {
		return err
	}
	sec, err := secrets.New(secretsPath)
	if err != nil {
		return err
	}

	// Add default gotify channel.
	ch := model.Channel{
		Type:        model.ChannelTypeGotify,
		Name:        "gotify-mobile",
		Enabled:     true,
		PriorityMin: model.PriorityLow,
		Config: model.ChannelConfig{
			GotifyURL: gotifyURL,
		},
	}
	added, err := reg.Add(ch)
	if err != nil {
		return fmt.Errorf("adding gotify channel: %w", err)
	}

	if err := sec.Set(fmt.Sprintf("sparkforge.channels.%s.gotify_token", added.ID), appToken); err != nil {
		return fmt.Errorf("storing gotify token: %w", err)
	}

	return nil
}

func waitForGotify(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("gotify did not become ready within %s", timeout)
}

func createGotifyApp(gotifyURL string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"name":        "Forge",
		"description": "Forge notification orchestration",
	})
	req, err := http.NewRequest("POST", gotifyURL+"/application", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(gotifyAdminUser, gotifyAdminPass)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("POST /application: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return "", fmt.Errorf("gotify returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	token, ok := result["token"].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("no token in gotify response")
	}
	return token, nil
}
