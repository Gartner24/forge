package engine

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/gartner24/forge/shared/registry"
)

const nmapImage = "instrumentisto/nmap:latest"

type nmapEngine struct {
	version string
}

func NewNmap() Engine {
	return &nmapEngine{version: nmapImage}
}

func (e *nmapEngine) Name() string    { return "nmap" }
func (e *nmapEngine) Version() string { return e.version }

func (e *nmapEngine) Pull(ctx context.Context) error {
	cli, err := dockerClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	return pullImage(ctx, cli, nmapImage)
}

func (e *nmapEngine) Run(ctx context.Context, target registry.ScanTarget, networkID string) ([]Finding, error) {
	cli, err := dockerClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	// Determine scan target: prefer scope IPs/hosts, fall back to URL host.
	scanTargets := scopeHosts(target)

	var allFindings []Finding
	for _, host := range scanTargets {
		cmd := []string{
			"-sV", "-sC",
			"--script=vuln",
			"-oX", "-",
			host,
		}

		out, err := runContainer(ctx, cli, containerOpts{
			image:     nmapImage,
			cmd:       cmd,
			networkID: networkID,
			name:      fmt.Sprintf("penforge-nmap-%s", containerSuffix()),
		})
		if err != nil {
			return nil, fmt.Errorf("nmap: %w", err)
		}

		findings, err := parseNmapXML(out, target.URL)
		if err != nil {
			return nil, fmt.Errorf("nmap xml parse: %w", err)
		}
		allFindings = append(allFindings, findings...)
	}

	return allFindings, nil
}

// nmapRun is the root element of nmap XML output.
type nmapRun struct {
	Hosts []nmapHost `xml:"host"`
}

type nmapHost struct {
	Addresses []nmapAddress `xml:"address"`
	Ports     nmapPorts     `xml:"ports"`
}

type nmapAddress struct {
	Addr     string `xml:"addr,attr"`
	AddrType string `xml:"addrtype,attr"`
}

type nmapPorts struct {
	Ports []nmapPort `xml:"port"`
}

type nmapPort struct {
	Protocol string      `xml:"protocol,attr"`
	PortID   int         `xml:"portid,attr"`
	State    nmapState   `xml:"state"`
	Service  nmapService `xml:"service"`
	Scripts  []nmapScript `xml:"script"`
}

type nmapState struct {
	State string `xml:"state,attr"`
}

type nmapService struct {
	Name    string `xml:"name,attr"`
	Product string `xml:"product,attr"`
	Version string `xml:"version,attr"`
}

type nmapScript struct {
	ID     string `xml:"id,attr"`
	Output string `xml:"output,attr"`
}

func parseNmapXML(data []byte, targetURL string) ([]Finding, error) {
	var run nmapRun
	if err := xml.Unmarshal(data, &run); err != nil {
		return nil, err
	}

	var findings []Finding
	for _, host := range run.Hosts {
		addr := ""
		for _, a := range host.Addresses {
			if a.AddrType == "ipv4" || a.AddrType == "ipv6" {
				addr = a.Addr
				break
			}
		}
		for _, port := range host.Ports.Ports {
			if port.State.State != "open" {
				continue
			}
			svc := port.Service
			name := fmt.Sprintf("Open port %d/%s (%s)", port.PortID, port.Protocol, svc.Name)
			desc := ""
			if svc.Product != "" || svc.Version != "" {
				desc = strings.TrimSpace(svc.Product + " " + svc.Version)
			}
			findings = append(findings, Finding{
				Engine:      "nmap",
				TargetURL:   targetURL,
				Severity:    "info",
				Name:        name,
				Description: desc,
				Host:        addr,
				Port:        port.PortID,
			})

			// Escalate severity if vuln scripts produced output.
			for _, script := range port.Scripts {
				if strings.HasPrefix(script.ID, "vuln") || script.ID == "exploit" {
					findings = append(findings, Finding{
						Engine:      "nmap",
						TargetURL:   targetURL,
						Severity:    "high",
						Name:        fmt.Sprintf("Vulnerability script: %s on port %d", script.ID, port.PortID),
						Description: script.Output,
						Host:        addr,
						Port:        port.PortID,
					})
				}
			}
		}
	}
	return findings, nil
}

// scopeHosts returns the IP/hostname targets from the target scope.
func scopeHosts(target registry.ScanTarget) []string {
	if len(target.Scope) > 0 {
		return target.Scope
	}
	// Fall back to extracting host from URL.
	host := target.URL
	for _, prefix := range []string{"https://", "http://"} {
		host = strings.TrimPrefix(host, prefix)
	}
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return []string{host}
}
