package engine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/mount"
	"github.com/gartner24/forge/shared/registry"
)

const nucleiImage = "projectdiscovery/nuclei:latest"

type nucleiEngine struct {
	version string
}

func NewNuclei() Engine {
	return &nucleiEngine{version: nucleiImage}
}

func (e *nucleiEngine) Name() string    { return "nuclei" }
func (e *nucleiEngine) Version() string { return e.version }

func (e *nucleiEngine) Pull(ctx context.Context) error {
	cli, err := dockerClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	return pullImage(ctx, cli, nucleiImage)
}

func (e *nucleiEngine) Run(ctx context.Context, target registry.ScanTarget, networkID string) ([]Finding, error) {
	cli, err := dockerClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	templatesDir, cleanup, err := tempDir("nuclei-templates")
	if err != nil {
		return nil, err
	}
	defer cleanup()

	cmd := []string{
		"-u", target.URL,
		"-jsonl",
		"-silent",
		"-no-color",
	}

	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: templatesDir,
			Target: "/custom-templates",
		},
	}

	out, err := runContainer(ctx, cli, containerOpts{
		image:     nucleiImage,
		cmd:       cmd,
		mounts:    mounts,
		networkID: networkID,
		name:      fmt.Sprintf("penforge-nuclei-%s", containerSuffix()),
	})
	if err != nil {
		return nil, fmt.Errorf("nuclei: %w", err)
	}

	return parseNucleiOutput(out, target.URL)
}

type nucleiResult struct {
	TemplateID  string         `json:"template-id"`
	Info        nucleiInfo     `json:"info"`
	Host        string         `json:"host"`
	MatchedAt   string         `json:"matched-at"`
	Description string         `json:"description,omitempty"`
	CVSS        float64        `json:"cvss-score,omitempty"`
	CVE         string         `json:"cve,omitempty"`
}

type nucleiInfo struct {
	Name        string            `json:"name"`
	Severity    string            `json:"severity"`
	Description string            `json:"description,omitempty"`
	Reference   []string          `json:"reference,omitempty"`
	Remediation string            `json:"remediation,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Classification nucleiClassify `json:"classification,omitempty"`
}

type nucleiClassify struct {
	CVEIDs []string `json:"cve-id,omitempty"`
}

func parseNucleiOutput(data []byte, targetURL string) ([]Finding, error) {
	var findings []Finding
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var r nucleiResult
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue
		}
		f := Finding{
			Engine:      "nuclei",
			TargetURL:   targetURL,
			Severity:    normalizeSeverity(r.Info.Severity),
			Name:        r.Info.Name,
			Description: r.Info.Description,
			Host:        r.Host,
			Remediation: r.Info.Remediation,
		}
		if len(r.Info.Classification.CVEIDs) > 0 {
			f.CVE = strings.Join(r.Info.Classification.CVEIDs, ", ")
		}
		findings = append(findings, f)
	}
	return findings, scanner.Err()
}

func normalizeSeverity(s string) string {
	switch strings.ToLower(s) {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	default:
		return "info"
	}
}
