package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gartner24/forge/shared/registry"
)

const trivyImage = "aquasec/trivy:latest"

type trivyEngine struct {
	version string
}

func NewTrivy() Engine {
	return &trivyEngine{version: trivyImage}
}

func (e *trivyEngine) Name() string    { return "trivy" }
func (e *trivyEngine) Version() string { return e.version }

func (e *trivyEngine) Pull(ctx context.Context) error {
	cli, err := dockerClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	return pullImage(ctx, cli, trivyImage)
}

func (e *trivyEngine) Run(ctx context.Context, target registry.ScanTarget, networkID string) ([]Finding, error) {
	cli, err := dockerClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	// Trivy scans container images. The target URL is treated as the image reference
	// when it does not have an HTTP/HTTPS scheme.
	imageRef := target.URL
	if strings.HasPrefix(imageRef, "http://") || strings.HasPrefix(imageRef, "https://") {
		return nil, nil // not an image reference, skip
	}

	cmd := []string{
		"image",
		"--format", "json",
		"--quiet",
		imageRef,
	}

	out, err := runContainer(ctx, cli, containerOpts{
		image:     trivyImage,
		cmd:       cmd,
		networkID: networkID,
		name:      fmt.Sprintf("penforge-trivy-%s", containerSuffix()),
	})
	if err != nil {
		return nil, fmt.Errorf("trivy: %w", err)
	}

	return parseTrivyOutput(out, target.URL)
}

type trivyReport struct {
	Results []trivyResult `json:"Results"`
}

type trivyResult struct {
	Target          string          `json:"Target"`
	Vulnerabilities []trivyVuln     `json:"Vulnerabilities,omitempty"`
}

type trivyVuln struct {
	VulnerabilityID  string `json:"VulnerabilityID"`
	PkgName          string `json:"PkgName"`
	InstalledVersion string `json:"InstalledVersion"`
	FixedVersion     string `json:"FixedVersion,omitempty"`
	Severity         string `json:"Severity"`
	Title            string `json:"Title,omitempty"`
	Description      string `json:"Description,omitempty"`
}

func parseTrivyOutput(data []byte, targetURL string) ([]Finding, error) {
	var report trivyReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}

	var findings []Finding
	for _, result := range report.Results {
		for _, v := range result.Vulnerabilities {
			name := v.Title
			if name == "" {
				name = fmt.Sprintf("%s in %s@%s", v.VulnerabilityID, v.PkgName, v.InstalledVersion)
			}
			remediation := ""
			if v.FixedVersion != "" {
				remediation = fmt.Sprintf("Upgrade %s to %s", v.PkgName, v.FixedVersion)
			}
			findings = append(findings, Finding{
				Engine:      "trivy",
				TargetURL:   targetURL,
				Severity:    normalizeSeverity(v.Severity),
				Name:        name,
				Description: v.Description,
				CVE:         v.VulnerabilityID,
				Remediation: remediation,
				Host:        result.Target,
			})
		}
	}
	return findings, nil
}
