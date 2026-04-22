package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gartner24/forge/smeltforge/internal/registry"
)

var (
	addSource   string
	addRepo     string
	addBranch   string
	addImage    string
	addPath     string
	addDomain   string
	addPort     int
	addStrategy string
	addWatch    bool
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Register a project for deployment",
	RunE:  runAdd,
}

func init() {
	addCmd.Flags().StringVar(&projectID, "project", "", "Project ID (required)")
	addCmd.Flags().StringVar(&addSource, "source", "", "Source type: git, registry, local")
	addCmd.Flags().StringVar(&addRepo, "repo", "", "Git repo URL")
	addCmd.Flags().StringVar(&addBranch, "branch", "main", "Git branch")
	addCmd.Flags().StringVar(&addImage, "image", "", "Docker image (for registry source)")
	addCmd.Flags().StringVar(&addPath, "path", "", "Local path (for local source)")
	addCmd.Flags().StringVar(&addDomain, "domain", "", "Public domain")
	addCmd.Flags().IntVar(&addPort, "port", 0, "Container port")
	addCmd.Flags().StringVar(&addStrategy, "strategy", "stop-start", "Deploy strategy: stop-start, blue-green")
	addCmd.Flags().BoolVar(&addWatch, "watch", false, "Auto-register WatchForge monitor")
}

func runAdd(cmd *cobra.Command, args []string) error {
	sc := bufio.NewScanner(os.Stdin)

	id := projectID
	if id == "" {
		id = prompt(sc, "Project ID")
	}
	if id == "" {
		return cmdErr(fmt.Errorf("project ID is required"))
	}

	src := addSource
	if src == "" {
		src = prompt(sc, "Source type (git/registry/local)")
	}
	if src != "git" && src != "registry" && src != "local" {
		return cmdErr(fmt.Errorf("invalid source type %q -- must be git, registry, or local", src))
	}

	var source registry.Source
	source.Type = src

	switch src {
	case "git":
		source.Repo = addRepo
		if source.Repo == "" {
			source.Repo = prompt(sc, "Git repo URL")
		}
		source.Branch = addBranch
		if source.Branch == "" {
			source.Branch = "main"
		}
	case "registry":
		source.Image = addImage
		if source.Image == "" {
			source.Image = prompt(sc, "Docker image")
		}
	case "local":
		source.Path = addPath
		if source.Path == "" {
			source.Path = prompt(sc, "Local path")
		}
	}

	domain := addDomain
	if domain == "" {
		domain = prompt(sc, "Domain")
	}
	if domain == "" {
		return cmdErr(fmt.Errorf("domain is required"))
	}

	port := addPort
	if port == 0 {
		portStr := prompt(sc, "Container port")
		p, err := strconv.Atoi(strings.TrimSpace(portStr))
		if err != nil || p == 0 {
			return cmdErr(fmt.Errorf("invalid port %q", portStr))
		}
		port = p
	}

	strategy := addStrategy
	if strategy == "" {
		strategy = "stop-start"
	}

	p := &registry.Project{
		ID:       id,
		Source:   source,
		Domain:   domain,
		Port:     port,
		Strategy: strategy,
		HealthCheck: registry.HealthCheck{
			Path:     "/health",
			Timeout:  30,
			Interval: 2,
		},
		Trigger: registry.Trigger{Type: "manual"},
		Watch:   addWatch,
	}

	reg, err := loadRegistry()
	if err != nil {
		return cmdErr(err)
	}
	if err := reg.Add(p); err != nil {
		return cmdErr(err)
	}

	if addWatch {
		fmt.Printf("Note: --watch requires WatchForge. Monitor not registered.\n")
	}

	if isJSON() {
		printJSON(p)
	} else {
		fmt.Printf("project %q added (strategy: %s, domain: %s)\n", id, strategy, domain)
	}
	return nil
}

func prompt(sc *bufio.Scanner, label string) string {
	fmt.Printf("%s: ", label)
	sc.Scan()
	return strings.TrimSpace(sc.Text())
}
