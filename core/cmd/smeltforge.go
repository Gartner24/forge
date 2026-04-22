package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Data model (mirrors smeltforge/internal/registry, kept local to avoid
// a cross-module binary dependency on the Docker SDK).
// ---------------------------------------------------------------------------

type sfSource struct {
	Type   string `json:"type"`
	Repo   string `json:"repo,omitempty"`
	Branch string `json:"branch,omitempty"`
	Image  string `json:"image,omitempty"`
	Path   string `json:"path,omitempty"`
}

type sfHealthCheck struct {
	Path     string `json:"path"`
	Timeout  int    `json:"timeout"`
	Interval int    `json:"interval"`
}

type sfTrigger struct {
	Type     string `json:"type"`
	Interval int    `json:"interval,omitempty"`
	Branch   string `json:"branch,omitempty"`
}

type sfContainerState struct {
	ID            string `json:"id,omitempty"`
	Image         string `json:"image,omitempty"`
	Color         string `json:"color,omitempty"`
	PreviousID    string `json:"previous_id,omitempty"`
	PreviousImage string `json:"previous_image,omitempty"`
}

type sfCIToken struct {
	ID    string `json:"id"`
	Label string `json:"label,omitempty"`
}

type sfProject struct {
	ID          string           `json:"id"`
	Source      sfSource         `json:"source"`
	Domain      string           `json:"domain"`
	Port        int              `json:"port"`
	Strategy    string           `json:"strategy"`
	HealthCheck sfHealthCheck    `json:"health_check,omitempty"`
	Trigger     sfTrigger        `json:"trigger"`
	Watch       bool             `json:"watch"`
	State       sfContainerState `json:"state,omitempty"`
	CITokens    []sfCIToken      `json:"ci_tokens,omitempty"`
	PollingOn   bool             `json:"polling_on,omitempty"`
}

// ---------------------------------------------------------------------------
// Registry helpers
// ---------------------------------------------------------------------------

func sfDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".forge", "data", "smeltforge")
}

func sfRegistryPath() string {
	return filepath.Join(sfDataDir(), "registry", "projects.json")
}

func sfSecretsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".forge", "secrets.age")
}

func sfLoadProjects() ([]*sfProject, error) {
	data, err := os.ReadFile(sfRegistryPath())
	if os.IsNotExist(err) {
		return []*sfProject{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading smeltforge registry: %w", err)
	}
	var projects []*sfProject
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, fmt.Errorf("parsing smeltforge registry: %w", err)
	}
	return projects, nil
}

func sfSaveProjects(projects []*sfProject) error {
	path := sfRegistryPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func sfFindProject(projects []*sfProject, id string) (*sfProject, error) {
	for _, p := range projects {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, fmt.Errorf("project %q not found", id)
}

func sfRandHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func sfExecBinFromPath(binPath string, args []string) error {
	if isJSON() {
		args = append([]string{"--output", "json"}, args...)
	}
	return syscall.Exec(binPath, append([]string{binPath}, args...), os.Environ())
}

func sfBinPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".forge", "bin", "smeltforge")
}

// sfRunBin runs the smeltforge binary and streams output (does not replace process).
func sfRunBin(binPath string, args []string) error {
	if isJSON() {
		args = append([]string{"--output", "json"}, args...)
	}
	c := exec.Command(binPath, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}

// ---------------------------------------------------------------------------
// Command tree
// ---------------------------------------------------------------------------

var smeltforgeCmd = &cobra.Command{
	Use:   "smeltforge",
	Short: "Deploy and manage Docker applications",
}

var sfProjectID string

func init() {
	rootCmd.AddCommand(smeltforgeCmd)

	// init
	smeltforgeCmd.AddCommand(sfInitCmd)

	// add -- all flags defined inline so they appear in --help
	sfAddCmd.Flags().StringVar(&sfProjectID, "project", "", "Project ID (required)")
	sfAddCmd.Flags().StringVar(&sfAddSource, "source", "", "Source type: git, registry, local")
	sfAddCmd.Flags().StringVar(&sfAddRepo, "repo", "", "Git repo URL")
	sfAddCmd.Flags().StringVar(&sfAddBranch, "branch", "main", "Git branch")
	sfAddCmd.Flags().StringVar(&sfAddImage, "image", "", "Docker image (for registry source)")
	sfAddCmd.Flags().StringVar(&sfAddPath, "path", "", "Local path (for local source)")
	sfAddCmd.Flags().StringVar(&sfAddDomain, "domain", "", "Public domain")
	sfAddCmd.Flags().IntVar(&sfAddPort, "port", 0, "Container port")
	sfAddCmd.Flags().StringVar(&sfAddStrategy, "strategy", "stop-start", "Deploy strategy: stop-start, blue-green")
	sfAddCmd.Flags().BoolVar(&sfAddWatch, "watch", false, "Auto-register WatchForge monitor")
	smeltforgeCmd.AddCommand(sfAddCmd)

	// deploy/rollback/logs exec to smeltforge binary; DisableFlagParsing passes args through
	smeltforgeCmd.AddCommand(sfDeployCmd)
	smeltforgeCmd.AddCommand(sfRollbackCmd)
	smeltforgeCmd.AddCommand(sfLogsCmd)

	// status
	sfStatusCmd.Flags().StringVar(&sfProjectID, "project", "", "Filter to single project")
	smeltforgeCmd.AddCommand(sfStatusCmd)

	// list
	smeltforgeCmd.AddCommand(sfListCmd)

	// env
	sfEnvCmd.AddCommand(sfEnvSetCmd)
	sfEnvCmd.AddCommand(sfEnvGetCmd)
	sfEnvCmd.AddCommand(sfEnvListCmd)
	sfEnvCmd.AddCommand(sfEnvUnsetCmd)
	smeltforgeCmd.AddCommand(sfEnvCmd)

	// webhook
	sfWebhookCmd.AddCommand(sfWebhookShowCmd)
	sfWebhookCmd.AddCommand(sfWebhookRegenerateCmd)
	smeltforgeCmd.AddCommand(sfWebhookCmd)

	// token
	sfTokenCreateCmd.Flags().StringVar(&sfProjectID, "project", "", "Project ID (required)")
	sfTokenCreateCmd.Flags().String("label", "", "Human-readable label")
	sfTokenCreateCmd.MarkFlagRequired("project")
	sfTokenListCmd.Flags().StringVar(&sfProjectID, "project", "", "Project ID (required)")
	sfTokenListCmd.MarkFlagRequired("project")
	sfTokenCmd.AddCommand(sfTokenCreateCmd)
	sfTokenCmd.AddCommand(sfTokenListCmd)
	sfTokenCmd.AddCommand(sfTokenRevokeCmd)
	smeltforgeCmd.AddCommand(sfTokenCmd)

	// polling
	sfPollingEnableCmd.Flags().StringVar(&sfProjectID, "project", "", "Project ID (required)")
	sfPollingEnableCmd.Flags().Int("interval", 60, "Poll interval in seconds")
	sfPollingEnableCmd.MarkFlagRequired("project")
	sfPollingDisableCmd.Flags().StringVar(&sfProjectID, "project", "", "Project ID (required)")
	sfPollingDisableCmd.MarkFlagRequired("project")
	sfPollingCmd.AddCommand(sfPollingEnableCmd)
	sfPollingCmd.AddCommand(sfPollingDisableCmd)
	smeltforgeCmd.AddCommand(sfPollingCmd)

	// delete
	sfDeleteCmd.Flags().StringVar(&sfProjectID, "project", "", "Project ID (required)")
	sfDeleteCmd.MarkFlagRequired("project")
	smeltforgeCmd.AddCommand(sfDeleteCmd)

	// deploy-key execs to smeltforge binary; DisableFlagParsing passes args through
	sfDeployKeyCmd.AddCommand(sfDeployKeyGenerateCmd)
	sfDeployKeyCmd.AddCommand(sfDeployKeyShowCmd)
	sfDeployKeyCmd.AddCommand(sfDeployKeyRotateCmd)
	smeltforgeCmd.AddCommand(sfDeployKeyCmd)
}

// ---------------------------------------------------------------------------
// forge smeltforge init
// ---------------------------------------------------------------------------

var sfInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise SmeltForge (creates directories, starts Caddy)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		binPath := sfBinPath()
		if err := sfRunBin(binPath, []string{"init"}); err != nil {
			if os.IsNotExist(err) {
				return cmdErr(fmt.Errorf("smeltforge binary not found at %s -- install it first", binPath))
			}
			return cmdErr(err)
		}
		return nil
	},
}

// ---------------------------------------------------------------------------
// forge smeltforge add
// ---------------------------------------------------------------------------

var (
	sfAddSource   string
	sfAddRepo     string
	sfAddBranch   string
	sfAddImage    string
	sfAddPath     string
	sfAddDomain   string
	sfAddPort     int
	sfAddStrategy string
	sfAddWatch    bool
)

var sfAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Register a project for deployment",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}

		if sfProjectID == "" {
			return cmdErr(fmt.Errorf("--project is required"))
		}
		if sfAddSource == "" || sfAddDomain == "" || sfAddPort == 0 {
			// Fall through to interactive mode in the binary.
			binPath := sfBinPath()
			fwdArgs := []string{"add"}
			fwdArgs = append(fwdArgs, cmd.Flags().Args()...)
			if sfProjectID != "" {
				fwdArgs = append(fwdArgs, "--project", sfProjectID)
			}
			if sfAddSource != "" {
				fwdArgs = append(fwdArgs, "--source", sfAddSource)
			}
			if sfAddRepo != "" {
				fwdArgs = append(fwdArgs, "--repo", sfAddRepo)
			}
			if sfAddBranch != "" {
				fwdArgs = append(fwdArgs, "--branch", sfAddBranch)
			}
			if sfAddImage != "" {
				fwdArgs = append(fwdArgs, "--image", sfAddImage)
			}
			if sfAddPath != "" {
				fwdArgs = append(fwdArgs, "--path", sfAddPath)
			}
			if sfAddDomain != "" {
				fwdArgs = append(fwdArgs, "--domain", sfAddDomain)
			}
			if sfAddPort != 0 {
				fwdArgs = append(fwdArgs, "--port", strconv.Itoa(sfAddPort))
			}
			if sfAddStrategy != "" {
				fwdArgs = append(fwdArgs, "--strategy", sfAddStrategy)
			}
			if sfAddWatch {
				fwdArgs = append(fwdArgs, "--watch")
			}
			return sfRunBin(binPath, fwdArgs)
		}

		// Full flags provided: create project inline.
		if sfAddSource != "git" && sfAddSource != "registry" && sfAddSource != "local" {
			return cmdErr(fmt.Errorf("invalid source type %q -- must be git, registry, or local", sfAddSource))
		}

		src := sfSource{Type: sfAddSource, Branch: sfAddBranch}
		switch sfAddSource {
		case "git":
			src.Repo = sfAddRepo
		case "registry":
			src.Image = sfAddImage
		case "local":
			src.Path = sfAddPath
		}

		p := &sfProject{
			ID:       sfProjectID,
			Source:   src,
			Domain:   sfAddDomain,
			Port:     sfAddPort,
			Strategy: sfAddStrategy,
			HealthCheck: sfHealthCheck{
				Path:     "/health",
				Timeout:  30,
				Interval: 2,
			},
			Trigger: sfTrigger{Type: "manual"},
			Watch:   sfAddWatch,
		}

		projects, err := sfLoadProjects()
		if err != nil {
			return cmdErr(err)
		}
		for _, existing := range projects {
			if existing.ID == sfProjectID {
				return cmdErr(fmt.Errorf("project %q already exists", sfProjectID))
			}
		}
		projects = append(projects, p)
		if err := sfSaveProjects(projects); err != nil {
			return cmdErr(err)
		}

		if sfAddWatch {
			fmt.Println("Note: --watch requires WatchForge. Monitor not registered.")
		}
		if isJSON() {
			printJSON(p)
		} else {
			fmt.Printf("project %q added (strategy: %s, domain: %s)\n", sfProjectID, sfAddStrategy, sfAddDomain)
		}
		return nil
	},
}

// ---------------------------------------------------------------------------
// forge smeltforge deploy  (execs to smeltforge binary)
// ---------------------------------------------------------------------------

var sfDeployCmd = &cobra.Command{
	Use:                "deploy",
	Short:              "Deploy a project",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		binPath := sfBinPath()
		return sfExecBinFromPath(binPath, append([]string{"deploy"}, args...))
	},
}

// ---------------------------------------------------------------------------
// forge smeltforge rollback  (execs to smeltforge binary)
// ---------------------------------------------------------------------------

var sfRollbackCmd = &cobra.Command{
	Use:                "rollback",
	Short:              "Roll back a project to its previous image",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		binPath := sfBinPath()
		return sfExecBinFromPath(binPath, append([]string{"rollback"}, args...))
	},
}

// ---------------------------------------------------------------------------
// forge smeltforge status
// ---------------------------------------------------------------------------

var sfStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show project deployment status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}

		projects, err := sfLoadProjects()
		if err != nil {
			return cmdErr(err)
		}

		if sfProjectID != "" {
			p, err := sfFindProject(projects, sfProjectID)
			if err != nil {
				return cmdErr(err)
			}
			projects = []*sfProject{p}
		}

		if isJSON() {
			if projects == nil {
				projects = []*sfProject{}
			}
			printJSON(projects)
			return nil
		}

		if len(projects) == 0 {
			fmt.Println("No projects registered.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PROJECT\tDOMAIN\tSTRATEGY\tIMAGE\tCONTAINER")
		for _, p := range projects {
			img := p.State.Image
			if img == "" {
				img = "-"
			}
			cid := p.State.ID
			if cid == "" {
				cid = "-"
			} else if len(cid) > 12 {
				cid = cid[:12]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", p.ID, p.Domain, p.Strategy, img, cid)
		}
		w.Flush()
		return nil
	},
}

// ---------------------------------------------------------------------------
// forge smeltforge list
// ---------------------------------------------------------------------------

var sfListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		projects, err := sfLoadProjects()
		if err != nil {
			return cmdErr(err)
		}
		if isJSON() {
			if projects == nil {
				projects = []*sfProject{}
			}
			printJSON(projects)
			return nil
		}
		if len(projects) == 0 {
			fmt.Println("No projects registered.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PROJECT\tSOURCE\tDOMAIN\tSTRATEGY")
		for _, p := range projects {
			src := p.Source.Type
			switch p.Source.Type {
			case "git":
				src = "git:" + p.Source.Repo
			case "registry":
				src = "registry:" + p.Source.Image
			case "local":
				src = "local:" + p.Source.Path
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.ID, src, p.Domain, p.Strategy)
		}
		w.Flush()
		return nil
	},
}

// ---------------------------------------------------------------------------
// forge smeltforge logs  (execs to smeltforge binary)
// ---------------------------------------------------------------------------

var sfLogsCmd = &cobra.Command{
	Use:                "logs",
	Short:              "Show container logs for a project",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		binPath := sfBinPath()
		return sfExecBinFromPath(binPath, append([]string{"logs"}, args...))
	},
}

// ---------------------------------------------------------------------------
// forge smeltforge env
// ---------------------------------------------------------------------------

var sfEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environment variables for a project",
}

var sfEnvSetCmd = &cobra.Command{
	Use:   "set <project> <KEY> <value>",
	Short: "Set an environment variable",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		project, key, value := args[0], args[1], args[2]
		store, err := openSecretsStore()
		if err != nil {
			return cmdErr(err)
		}
		secretKey := "smeltforge." + project + "." + key
		if err := store.Set(secretKey, value); err != nil {
			return cmdErr(err)
		}
		printSuccess(fmt.Sprintf("set %s for project %s", key, project))
		return nil
	},
}

var sfEnvGetCmd = &cobra.Command{
	Use:   "get <project> <KEY>",
	Short: "Get an environment variable value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		project, key := args[0], args[1]
		store, err := openSecretsStore()
		if err != nil {
			return cmdErr(err)
		}
		val, err := store.Get("smeltforge." + project + "." + key)
		if err != nil {
			return cmdErr(err)
		}
		if isJSON() {
			printJSON(map[string]string{"key": key, "value": val})
		} else {
			fmt.Println(val)
		}
		return nil
	},
}

var sfEnvListCmd = &cobra.Command{
	Use:   "list <project>",
	Short: "List environment variable keys for a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		project := args[0]
		store, err := openSecretsStore()
		if err != nil {
			return cmdErr(err)
		}
		allKeys, err := store.List()
		if err != nil {
			return cmdErr(err)
		}
		prefix := "smeltforge." + project + "."
		var keys []string
		for _, k := range allKeys {
			if !strings.HasPrefix(k, prefix) {
				continue
			}
			short := strings.TrimPrefix(k, prefix)
			if strings.HasPrefix(short, "_") {
				continue
			}
			keys = append(keys, short)
		}
		sort.Strings(keys)
		if isJSON() {
			if keys == nil {
				keys = []string{}
			}
			printJSON(map[string]any{"project": project, "keys": keys})
			return nil
		}
		if len(keys) == 0 {
			fmt.Printf("No env vars set for project %s.\n", project)
			return nil
		}
		for _, k := range keys {
			fmt.Println(k)
		}
		return nil
	},
}

var sfEnvUnsetCmd = &cobra.Command{
	Use:   "unset <project> <KEY>",
	Short: "Remove an environment variable",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		project, key := args[0], args[1]
		store, err := openSecretsStore()
		if err != nil {
			return cmdErr(err)
		}
		if err := store.Delete("smeltforge." + project + "." + key); err != nil {
			return cmdErr(err)
		}
		printSuccess(fmt.Sprintf("unset %s for project %s", key, project))
		return nil
	},
}

// ---------------------------------------------------------------------------
// forge smeltforge webhook
// ---------------------------------------------------------------------------

var sfWebhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Manage webhook secrets",
}

var sfWebhookShowCmd = &cobra.Command{
	Use:   "show <project>",
	Short: "Print the current webhook URL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		project := args[0]
		projects, err := sfLoadProjects()
		if err != nil {
			return cmdErr(err)
		}
		p, err := sfFindProject(projects, project)
		if err != nil {
			return cmdErr(err)
		}
		store, err := openSecretsStore()
		if err != nil {
			return cmdErr(err)
		}
		secret, err := store.Get("smeltforge." + project + "._webhook")
		if err != nil {
			return cmdErr(fmt.Errorf("no webhook secret -- run: forge smeltforge webhook regenerate %s", project))
		}
		url := fmt.Sprintf("https://%s/_smeltforge/webhook/%s/%s", p.Domain, project, secret)
		if isJSON() {
			printJSON(map[string]string{"project": project, "url": url})
		} else {
			fmt.Println(url)
		}
		return nil
	},
}

var sfWebhookRegenerateCmd = &cobra.Command{
	Use:   "regenerate <project>",
	Short: "Rotate the webhook secret",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		project := args[0]
		projects, err := sfLoadProjects()
		if err != nil {
			return cmdErr(err)
		}
		if _, err := sfFindProject(projects, project); err != nil {
			return cmdErr(err)
		}
		secret := sfRandHex(32)
		store, err := openSecretsStore()
		if err != nil {
			return cmdErr(err)
		}
		if err := store.Set("smeltforge."+project+"._webhook", secret); err != nil {
			return cmdErr(err)
		}
		printSuccess(fmt.Sprintf("webhook secret regenerated for project %s", project))
		return nil
	},
}

// ---------------------------------------------------------------------------
// forge smeltforge token
// ---------------------------------------------------------------------------

var sfTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage CI deploy tokens",
}

var sfTokenCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a CI token",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		projects, err := sfLoadProjects()
		if err != nil {
			return cmdErr(err)
		}
		p, err := sfFindProject(projects, sfProjectID)
		if err != nil {
			return cmdErr(err)
		}
		label, _ := cmd.Flags().GetString("label")
		tokenID := sfRandHex(16)
		tokenValue := sfRandHex(32)
		store, err := openSecretsStore()
		if err != nil {
			return cmdErr(err)
		}
		if err := store.Set("smeltforge."+sfProjectID+"._citoken_"+tokenID, tokenValue); err != nil {
			return cmdErr(err)
		}
		p.CITokens = append(p.CITokens, sfCIToken{ID: tokenID, Label: label})
		if err := sfSaveProjects(projects); err != nil {
			return cmdErr(err)
		}
		if isJSON() {
			printJSON(map[string]string{"id": tokenID, "token": tokenValue})
		} else {
			fmt.Printf("token ID: %s\ntoken:    %s\n\nStore the token now -- it cannot be retrieved later.\n", tokenID, tokenValue)
		}
		return nil
	},
}

var sfTokenListCmd = &cobra.Command{
	Use:   "list",
	Short: "List CI tokens for a project",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		projects, err := sfLoadProjects()
		if err != nil {
			return cmdErr(err)
		}
		p, err := sfFindProject(projects, sfProjectID)
		if err != nil {
			return cmdErr(err)
		}
		tokens := p.CITokens
		if tokens == nil {
			tokens = []sfCIToken{}
		}
		if isJSON() {
			printJSON(tokens)
			return nil
		}
		if len(tokens) == 0 {
			fmt.Printf("No CI tokens for project %s.\n", sfProjectID)
			return nil
		}
		for _, t := range tokens {
			label := t.Label
			if label == "" {
				label = "(no label)"
			}
			id := t.ID
			if len(id) > 8 {
				id = id[:8] + "..."
			}
			fmt.Printf("%s  %s\n", id, label)
		}
		return nil
	},
}

var sfTokenRevokeCmd = &cobra.Command{
	Use:   "revoke <token-id>",
	Short: "Revoke a CI token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		tokenID := args[0]
		projects, err := sfLoadProjects()
		if err != nil {
			return cmdErr(err)
		}
		var ownerProject string
		for _, p := range projects {
			for _, t := range p.CITokens {
				if t.ID == tokenID {
					ownerProject = p.ID
				}
			}
		}
		if ownerProject == "" {
			return cmdErr(fmt.Errorf("token %q not found", tokenID))
		}
		store, err := openSecretsStore()
		if err != nil {
			return cmdErr(err)
		}
		_ = store.Delete("smeltforge." + ownerProject + "._citoken_" + tokenID)
		p, _ := sfFindProject(projects, ownerProject)
		out := p.CITokens[:0]
		for _, t := range p.CITokens {
			if t.ID != tokenID {
				out = append(out, t)
			}
		}
		p.CITokens = out
		if err := sfSaveProjects(projects); err != nil {
			return cmdErr(err)
		}
		short := tokenID
		if len(short) > 8 {
			short = short[:8] + "..."
		}
		printSuccess(fmt.Sprintf("token %s revoked", short))
		return nil
	},
}

// ---------------------------------------------------------------------------
// forge smeltforge polling
// ---------------------------------------------------------------------------

var sfPollingCmd = &cobra.Command{
	Use:   "polling",
	Short: "Manage git polling triggers",
}

var sfPollingEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable git polling for a project",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		interval, _ := cmd.Flags().GetInt("interval")
		projects, err := sfLoadProjects()
		if err != nil {
			return cmdErr(err)
		}
		p, err := sfFindProject(projects, sfProjectID)
		if err != nil {
			return cmdErr(err)
		}
		if p.Source.Type != "git" {
			return cmdErr(fmt.Errorf("polling only supported for git source projects"))
		}
		branch := p.Source.Branch
		if branch == "" {
			branch = "main"
		}
		p.Trigger = sfTrigger{Type: "polling", Interval: interval, Branch: branch}
		p.PollingOn = true
		if err := sfSaveProjects(projects); err != nil {
			return cmdErr(err)
		}
		printSuccess(fmt.Sprintf("polling enabled for %s (interval: %ds)", sfProjectID, interval))
		return nil
	},
}

var sfPollingDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable git polling for a project",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		projects, err := sfLoadProjects()
		if err != nil {
			return cmdErr(err)
		}
		p, err := sfFindProject(projects, sfProjectID)
		if err != nil {
			return cmdErr(err)
		}
		p.Trigger = sfTrigger{Type: "manual"}
		p.PollingOn = false
		if err := sfSaveProjects(projects); err != nil {
			return cmdErr(err)
		}
		printSuccess(fmt.Sprintf("polling disabled for %s", sfProjectID))
		return nil
	},
}

// ---------------------------------------------------------------------------
// forge smeltforge delete
// ---------------------------------------------------------------------------

var sfDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Stop and remove a project",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		proceed, err := mustConfirm(fmt.Sprintf("Stop and remove project %q?", sfProjectID))
		if err != nil {
			return cmdErr(err)
		}
		if !proceed {
			fmt.Println("Aborted.")
			return nil
		}
		// Exec to binary to stop containers + clean Caddy route, then remove from registry.
		binPath := sfBinPath()
		c := exec.Command(binPath, "delete", "--project", sfProjectID, "--yes")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

// ---------------------------------------------------------------------------
// forge smeltforge deploy-key  (execs to smeltforge binary)
// ---------------------------------------------------------------------------

var sfDeployKeyCmd = &cobra.Command{
	Use:   "deploy-key",
	Short: "Manage SSH deploy keys for git source projects",
}

var sfDeployKeyGenerateCmd = &cobra.Command{
	Use:                "generate",
	Short:              "Generate a deploy key pair",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		binPath := sfBinPath()
		return sfExecBinFromPath(binPath, append([]string{"deploy-key", "generate"}, args...))
	},
}

var sfDeployKeyShowCmd = &cobra.Command{
	Use:                "show",
	Short:              "Print the public deploy key",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		binPath := sfBinPath()
		return sfExecBinFromPath(binPath, append([]string{"deploy-key", "show"}, args...))
	},
}

var sfDeployKeyRotateCmd = &cobra.Command{
	Use:                "rotate",
	Short:              "Generate a new deploy key pair",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireInit(); err != nil {
			return cmdErr(err)
		}
		binPath := sfBinPath()
		return sfExecBinFromPath(binPath, append([]string{"deploy-key", "rotate"}, args...))
	},
}
