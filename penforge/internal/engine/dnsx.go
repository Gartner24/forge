package engine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gartner24/forge/shared/registry"
)

const dnsxImage = "projectdiscovery/dnsx:latest"

type dnsxEngine struct {
	version string
}

func NewDnsx() Engine {
	return &dnsxEngine{version: dnsxImage}
}

func (e *dnsxEngine) Name() string    { return "dnsx" }
func (e *dnsxEngine) Version() string { return e.version }

func (e *dnsxEngine) Pull(ctx context.Context) error {
	cli, err := dockerClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	return pullImage(ctx, cli, dnsxImage)
}

func (e *dnsxEngine) Run(ctx context.Context, target registry.ScanTarget, networkID string) ([]Finding, error) {
	cli, err := dockerClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	domains := domainScope(target)
	if len(domains) == 0 {
		return nil, nil // no domains in scope, skip
	}

	var allFindings []Finding
	for _, domain := range domains {
		cmd := []string{
			"-d", domain,
			"-json",
			"-silent",
			"-a", "-aaaa", "-cname", "-mx", "-ns", "-txt", "-ptr",
		}

		out, err := runContainer(ctx, cli, containerOpts{
			image:     dnsxImage,
			cmd:       cmd,
			networkID: networkID,
			name:      fmt.Sprintf("penforge-dnsx-%s", containerSuffix()),
		})
		if err != nil {
			return nil, fmt.Errorf("dnsx: %w", err)
		}

		findings, err := parseDnsxOutput(out, target.URL)
		if err != nil {
			return nil, fmt.Errorf("dnsx parse: %w", err)
		}
		allFindings = append(allFindings, findings...)
	}

	return allFindings, nil
}

type dnsxResult struct {
	Host     string   `json:"host"`
	A        []string `json:"a,omitempty"`
	AAAA     []string `json:"aaaa,omitempty"`
	CNAME    []string `json:"cname,omitempty"`
	MX       []string `json:"mx,omitempty"`
	NS       []string `json:"ns,omitempty"`
	TXT      []string `json:"txt,omitempty"`
	PTR      []string `json:"ptr,omitempty"`
	Resolver string   `json:"resolver,omitempty"`
}

func parseDnsxOutput(data []byte, targetURL string) ([]Finding, error) {
	var findings []Finding
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var r dnsxResult
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue
		}

		// Report CNAME chains that could indicate subdomain takeover.
		for _, cname := range r.CNAME {
			findings = append(findings, Finding{
				Engine:      "dnsx",
				TargetURL:   targetURL,
				Severity:    "info",
				Name:        fmt.Sprintf("DNS CNAME: %s -> %s", r.Host, cname),
				Host:        r.Host,
			})
		}

		// Report DNS records as info findings.
		if len(r.A) > 0 {
			findings = append(findings, Finding{
				Engine:      "dnsx",
				TargetURL:   targetURL,
				Severity:    "info",
				Name:        fmt.Sprintf("DNS A: %s -> %s", r.Host, strings.Join(r.A, ", ")),
				Host:        r.Host,
			})
		}
	}
	return findings, scanner.Err()
}

// domainScope returns only the domain entries from the target scope.
func domainScope(target registry.ScanTarget) []string {
	var domains []string
	for _, s := range target.Scope {
		// Simple heuristic: if it doesn't look like an IP, it's a domain.
		if !isIP(s) {
			domains = append(domains, s)
		}
	}
	return domains
}

func isIP(s string) bool {
	// IPv4 check: contains dots and only digits/dots.
	parts := strings.Split(s, ".")
	if len(parts) == 4 {
		allDigits := true
		for _, p := range parts {
			for _, c := range p {
				if c < '0' || c > '9' {
					allDigits = false
					break
				}
			}
		}
		if allDigits {
			return true
		}
	}
	// IPv6 check: contains colons.
	return strings.Contains(s, ":")
}
