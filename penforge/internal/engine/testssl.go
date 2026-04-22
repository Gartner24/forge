package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/mount"
	"github.com/gartner24/forge/shared/registry"
)

const testsslImage = "drwetter/testssl.sh:latest"

type testsslEngine struct {
	version string
}

func NewTestssl() Engine {
	return &testsslEngine{version: testsslImage}
}

func (e *testsslEngine) Name() string    { return "testssl" }
func (e *testsslEngine) Version() string { return e.version }

func (e *testsslEngine) Pull(ctx context.Context) error {
	cli, err := dockerClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	return pullImage(ctx, cli, testsslImage)
}

func (e *testsslEngine) Run(ctx context.Context, target registry.ScanTarget, networkID string) ([]Finding, error) {
	cli, err := dockerClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	outputDir, cleanup, err := tempDir("testssl")
	if err != nil {
		return nil, err
	}
	defer cleanup()

	outputFile := "/output/result.json"

	cmd := []string{
		"--jsonfile", outputFile,
		"--severity", "LOW",
		"--quiet",
		"--color", "0",
		target.URL,
	}

	_, err = runContainer(ctx, cli, containerOpts{
		image: testsslImage,
		cmd:   cmd,
		mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: outputDir,
				Target: "/output",
			},
		},
		networkID: networkID,
		name:      fmt.Sprintf("penforge-testssl-%s", containerSuffix()),
	})
	if err != nil {
		return nil, fmt.Errorf("testssl: %w", err)
	}

	resultPath := filepath.Join(outputDir, "result.json")
	data, err := os.ReadFile(resultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no output file means no findings or scan skipped
		}
		return nil, fmt.Errorf("reading testssl output: %w", err)
	}

	return parseTestsslOutput(data, target.URL)
}

type testsslFinding struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Finding  string `json:"finding"`
	CVE      string `json:"cve,omitempty"`
}

func parseTestsslOutput(data []byte, targetURL string) ([]Finding, error) {
	var raw []testsslFinding
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var findings []Finding
	for _, r := range raw {
		sev := normalizeTestsslSeverity(r.Severity)
		if sev == "" {
			continue // OK or INFO-level non-finding
		}
		findings = append(findings, Finding{
			Engine:    "testssl",
			TargetURL: targetURL,
			Severity:  sev,
			Name:      fmt.Sprintf("SSL/TLS: %s", r.ID),
			Description: r.Finding,
			CVE:       r.CVE,
		})
	}
	return findings, nil
}

func normalizeTestsslSeverity(s string) string {
	switch strings.ToUpper(s) {
	case "CRITICAL":
		return "critical"
	case "HIGH":
		return "high"
	case "MEDIUM":
		return "medium"
	case "LOW":
		return "low"
	case "INFO":
		return "info"
	default:
		return ""
	}
}
