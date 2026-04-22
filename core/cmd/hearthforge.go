package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gartner24/forge/core/internal/paths"
	"github.com/gartner24/forge/shared/audit"
	"github.com/gartner24/forge/shared/config"
	"github.com/gartner24/forge/shared/secrets"
	"github.com/spf13/cobra"
)

// ---- registry types ----

type hfProjectPreview struct {
	Enabled           bool   `json:"enabled"`
	FrontendPort      int    `json:"frontend_port"`
	BackendPort       int    `json:"backend_port,omitempty"`
	FrontendPath      string `json:"frontend_path,omitempty"`
	BackendPathPrefix string `json:"backend_path_prefix,omitempty"`
}

type hfProjectResources struct {
	CPUs   string `json:"cpus"`
	Memory string `json:"memory"`
}

type hfProject struct {
	ID            string             `json:"id"`
	Repo          string             `json:"repo"`
	DefaultBranch string             `json:"default_branch"`
	Stack         string             `json:"stack"`
	Preview       hfProjectPreview   `json:"preview"`
	Resources     hfProjectResources `json:"resources"`
	Domain        string             `json:"domain,omitempty"`
}

type hfProjectsFile struct {
	Version  int         `json:"version"`
	Projects []hfProject `json:"projects"`
}

type hfDeveloper struct {
	ID       string   `json:"id"`
	Status   string   `json:"status"`
	Projects []string `json:"projects"`
}

type hfDevsFile struct {
	Version    int           `json:"version"`
	Developers []hfDeveloper `json:"developers"`
}

type hfForgeJSON struct {
	DevBaseDomain         string `json:"dev_base_domain"`
	DevWildcardCertName   string `json:"dev_wildcard_cert_name"`
	ProxyActiveDevConfDir string `json:"proxy_active_dev_conf_dir"`
	DevWorkspaceRoot      string `json:"dev_workspace_root"`
	DevNetwork            string `json:"dev_network"`
	DefaultDevUser        string `json:"default_dev_user"`
	DefaultDevUID         int    `json:"default_dev_uid"`
	DefaultDevGID         int    `json:"default_dev_gid"`
}

// hfConfig holds all runtime paths derived from forge config and forge.json.
type hfConfig struct {
	installDir     string
	dataDir        string
	domain         string
	registryDir    string
	projectsJSON   string
	devsJSON       string
	workspacesDir  string
	keysDir        string
	sshdDir        string
	deployKeysDir  string
	gatewayAuthDir string
	proxyConfDir   string
	templatesDir   string
	devNetwork     string
	certName       string
	devUser        string
}

func loadHFConfig(cfg *config.Config) hfConfig {
	installDir := cfg.Forge.InstallDir
	if installDir == "" {
		installDir = "/opt/infra/forge"
	}
	dataDir := cfg.Forge.DataDir
	if dataDir == "" {
		dataDir = "/opt/data"
	}

	registryDir := filepath.Join(installDir, "registry")

	// Load forge.json for domain/network config.
	fj := loadForgeJSON(registryDir)

	domain := fj.DevBaseDomain
	if domain == "" {
		domain = cfg.Forge.Domain
	}
	network := fj.DevNetwork
	if network == "" {
		network = "dev-web"
	}
	certName := fj.DevWildcardCertName
	if certName == "" {
		certName = domain
	}
	wsRoot := fj.DevWorkspaceRoot
	if wsRoot == "" {
		wsRoot = filepath.Join(dataDir, "dev_workspaces")
	}
	proxyConf := fj.ProxyActiveDevConfDir
	if proxyConf == "" {
		proxyConf = filepath.Join(installDir, "proxy", "conf.d", "active")
	}
	devUser := fj.DefaultDevUser
	if devUser == "" {
		devUser = "dev"
	}

	return hfConfig{
		installDir:     installDir,
		dataDir:        dataDir,
		domain:         domain,
		registryDir:    registryDir,
		projectsJSON:   filepath.Join(registryDir, "projects.json"),
		devsJSON:       filepath.Join(registryDir, "devs.json"),
		workspacesDir:  wsRoot,
		keysDir:        filepath.Join(wsRoot, "_keys"),
		sshdDir:        filepath.Join(wsRoot, "_sshd"),
		deployKeysDir:  filepath.Join(wsRoot, "_deploy_keys"),
		gatewayAuthDir: filepath.Join(installDir, "hearthforge", "gateway", "authorized_keys"),
		proxyConfDir:   proxyConf,
		templatesDir:   filepath.Join(installDir, "templates"),
		devNetwork:     network,
		certName:       certName,
		devUser:        devUser,
	}
}

func loadForgeJSON(registryDir string) hfForgeJSON {
	var fj hfForgeJSON
	data, err := os.ReadFile(filepath.Join(registryDir, "forge.json"))
	if err == nil {
		_ = json.Unmarshal(data, &fj)
	}
	return fj
}

// ---- registry I/O ----

func readProjectsFile(path string) (hfProjectsFile, error) {
	var pf hfProjectsFile
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return hfProjectsFile{Version: 1}, nil
	}
	if err != nil {
		return pf, fmt.Errorf("reading projects.json: %w", err)
	}
	if err := json.Unmarshal(data, &pf); err != nil {
		return pf, fmt.Errorf("parsing projects.json: %w", err)
	}
	return pf, nil
}

func writeProjectsFile(path string, pf hfProjectsFile) error {
	data, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readDevsFile(path string) (hfDevsFile, error) {
	var df hfDevsFile
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return hfDevsFile{Version: 1}, nil
	}
	if err != nil {
		return df, fmt.Errorf("reading devs.json: %w", err)
	}
	if err := json.Unmarshal(data, &df); err != nil {
		return df, fmt.Errorf("parsing devs.json: %w", err)
	}
	return df, nil
}

func writeDevsFile(path string, df hfDevsFile) error {
	data, err := json.MarshalIndent(df, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ---- helpers ----

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

func validateSlug(s string) error {
	if !slugRe.MatchString(s) {
		return fmt.Errorf("%q is not a valid slug (lowercase, alphanumeric, dashes only)", s)
	}
	return nil
}

// resolvePublicKey returns the key content. If keyOrPath looks like a file path
// or ends in .pub, it reads the file. Otherwise it uses the value as-is.
func resolvePublicKey(keyOrPath string) (string, error) {
	trimmed := strings.TrimSpace(keyOrPath)
	if strings.HasSuffix(trimmed, ".pub") || strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "~/") || strings.HasPrefix(trimmed, "./") {
		expanded := trimmed
		if strings.HasPrefix(expanded, "~/") {
			home, _ := os.UserHomeDir()
			expanded = filepath.Join(home, expanded[2:])
		}
		data, err := os.ReadFile(expanded)
		if err != nil {
			return "", fmt.Errorf("reading public key file %s: %w", expanded, err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	return trimmed, nil
}

// validatePublicKey checks the key starts with a known SSH key type prefix.
func validatePublicKey(key string) error {
	validPrefixes := []string{
		"ssh-ed25519 ",
		"ssh-rsa ",
		"ecdsa-sha2-nistp256 ",
		"ecdsa-sha2-nistp384 ",
		"ecdsa-sha2-nistp521 ",
		"sk-ssh-ed25519@openssh.com ",
	}
	for _, p := range validPrefixes {
		if strings.HasPrefix(key, p) {
			return nil
		}
	}
	return fmt.Errorf("not a valid SSH public key (must start with ssh-ed25519, ssh-rsa, etc.)")
}

func promptString(prompt, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return defaultVal
	}
	line := strings.TrimSpace(scanner.Text())
	if line == "" {
		return defaultVal
	}
	return line
}

func hfAuditLog(hfc hfConfig, action, actor, detail string) {
	logPath := filepath.Join(hfc.dataDir, "logs", "hearthforge", "audit.log")
	l, err := audit.New(logPath)
	if err != nil {
		return
	}
	_ = l.Write("hearthforge", action, actor, detail)
}

// generateComposeYAML produces a docker-compose.yml for a dev container.
func generateComposeYAML(dev, project, image, network, workspaceHost, keysHost, sshdConfigPath, deployKeyDir string, frontendPort, backendPort int, cpus, memory string) string {
	var expose strings.Builder
	fmt.Fprintf(&expose, "      - \"22\"\n")
	if frontendPort > 0 {
		fmt.Fprintf(&expose, "      - \"%d\"\n", frontendPort)
	}
	if backendPort > 0 {
		fmt.Fprintf(&expose, "      - \"%d\"\n", backendPort)
	}

	var volumes strings.Builder
	fmt.Fprintf(&volumes, "      - %s:/workspace/%s\n", workspaceHost, project)
	fmt.Fprintf(&volumes, "      - %s:/etc/ssh/authorized_keys:ro\n", keysHost)
	fmt.Fprintf(&volumes, "      - %s:/etc/ssh/sshd_config:ro\n", sshdConfigPath)
	if deployKeyDir != "" {
		fmt.Fprintf(&volumes, "      - %s:/home/dev/.ssh/forge_deploy:ro\n", deployKeyDir)
	}

	memStr := memory
	if memStr == "" {
		memStr = "512m"
	}
	cpuStr := cpus
	if cpuStr == "" {
		cpuStr = "1.0"
	}

	return fmt.Sprintf(`services:
  dev:
    image: %s
    container_name: dev-%s-%s
    hostname: dev-%s-%s
    networks:
      - %s
    expose:
%s    volumes:
%s    restart: unless-stopped
    cap_drop:
      - ALL
    cap_add:
      - SYS_CHROOT
      - SETUID
      - SETGID
      - CHOWN
      - KILL
      - AUDIT_WRITE
    security_opt:
      - no-new-privileges:true
    deploy:
      resources:
        limits:
          cpus: '%s'
          memory: '%s'

networks:
  %s:
    external: true
`, image, project, dev, project, dev, network,
		expose.String(), volumes.String(),
		cpuStr, memStr, network)
}

// generateNginxVhost produces an nginx vhost config for a dev preview.
func generateNginxVhost(dev, project, hostname, certName string, frontendPort, backendPort int, backendPathPrefix string) string {
	frontendUpstream := fmt.Sprintf("dev-%s-%s:%d", project, dev, frontendPort)
	proxyCfg := fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }
    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl;
    server_name %s;

    ssl_certificate     /etc/letsencrypt/live/%s/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/%s/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;

    location / {
        proxy_pass http://%s;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
`, hostname, hostname, certName, certName, frontendUpstream)

	if backendPort > 0 && backendPathPrefix != "" {
		backendUpstream := fmt.Sprintf("dev-%s-%s:%d", project, dev, backendPort)
		proxyCfg += fmt.Sprintf(`
    location %s {
        proxy_pass http://%s;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
`, backendPathPrefix, backendUpstream)
	}

	proxyCfg += "}\n"
	return proxyCfg
}

// generateSSHSnippet produces the SSH config block for a dev.
func generateSSHSnippet(dev, project, gatewayHost, identityFile string, frontendPort, backendPort int) string {
	gwAlias := fmt.Sprintf("%s-%s-gw", dev, project)
	devAlias := fmt.Sprintf("%s-%s", dev, project)
	containerName := fmt.Sprintf("dev-%s-%s", project, dev)

	var sb strings.Builder
	fmt.Fprintf(&sb, "Host %s\n", gwAlias)
	fmt.Fprintf(&sb, "  HostName %s\n", gatewayHost)
	fmt.Fprintf(&sb, "  Port 2224\n")
	fmt.Fprintf(&sb, "  User %s-%s\n", dev, project)
	fmt.Fprintf(&sb, "  IdentityFile %s\n", identityFile)
	fmt.Fprintf(&sb, "  IdentitiesOnly yes\n")
	fmt.Fprintf(&sb, "  StrictHostKeyChecking accept-new\n")
	fmt.Fprintf(&sb, "\n")
	fmt.Fprintf(&sb, "Host %s\n", devAlias)
	fmt.Fprintf(&sb, "  HostName %s\n", containerName)
	fmt.Fprintf(&sb, "  Port 22\n")
	fmt.Fprintf(&sb, "  User dev\n")
	fmt.Fprintf(&sb, "  ProxyJump %s\n", gwAlias)
	fmt.Fprintf(&sb, "  IdentityFile %s\n", identityFile)
	fmt.Fprintf(&sb, "  IdentitiesOnly yes\n")
	fmt.Fprintf(&sb, "  StrictHostKeyChecking accept-new\n")
	if frontendPort > 0 {
		fmt.Fprintf(&sb, "  LocalForward %d localhost:%d\n", frontendPort, frontendPort)
	}
	if backendPort > 0 {
		fmt.Fprintf(&sb, "  LocalForward %d localhost:%d\n", backendPort, backendPort)
	}
	return sb.String()
}

// devImage returns the dev container image name for a stack.
func devImage(stack string) string {
	switch stack {
	case "python":
		return "forge-dev-python:latest"
	default:
		return "forge-dev-node:latest"
	}
}

// dockerRun executes docker with the given args and returns combined output.
func dockerRun(args ...string) (string, error) {
	out, err := exec.Command("docker", args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// appendAuthorizedKey appends a public key line to a file idempotently.
func appendAuthorizedKey(filePath, pubkey string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	// Check if key already present.
	existing, _ := os.ReadFile(filePath)
	for _, line := range strings.Split(string(existing), "\n") {
		if strings.TrimSpace(line) == pubkey {
			return nil // already there
		}
	}

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s\n", pubkey)
	return err
}

// overwriteFile writes content to path, creating parent dirs as needed.
func overwriteFile(path, content string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), mode)
}

// reloadProxy sends nginx -s reload to the running proxy container.
// Non-fatal on error (proxy might not be running).
func reloadProxy() {
	// Validate first.
	if out, err := exec.Command("docker", "exec", "nginx-proxy", "nginx", "-t").CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: nginx config test failed: %s\n", out)
		return
	}
	if out, err := exec.Command("docker", "exec", "nginx-proxy", "nginx", "-s", "reload").CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: nginx reload failed: %s\n", out)
	}
}

// generateDeployKey creates an ed25519 keypair, returns (privateKeyPEM, publicKeyLine, error).
func generateDeployKey(projectID string) (string, string, error) {
	tmp, err := os.MkdirTemp("", "forge-dk-")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(tmp)

	keyPath := filepath.Join(tmp, "id_ed25519")
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-C",
		fmt.Sprintf("forge-deploykey-%s", projectID), "-f", keyPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", "", fmt.Errorf("ssh-keygen: %w\n%s", err, out)
	}

	priv, err := os.ReadFile(keyPath)
	if err != nil {
		return "", "", err
	}
	pub, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return "", "", err
	}
	return string(priv), strings.TrimSpace(string(pub)), nil
}

// openHFSecrets returns the secrets store for hearthforge operations.
func openHFSecrets() (*secrets.Store, error) {
	p, err := paths.SecretsFile()
	if err != nil {
		return nil, err
	}
	return secrets.New(p)
}

// ---- sshd_config template (static) ----

const sshdConfigContent = `Port 22
Protocol 2

PermitRootLogin no
PasswordAuthentication no
KbdInteractiveAuthentication no
ChallengeResponseAuthentication no

UsePAM no

PubkeyAuthentication yes
AuthorizedKeysFile /etc/ssh/authorized_keys/%u

AllowUsers dev

X11Forwarding no
AllowAgentForwarding yes
AllowTcpForwarding yes
PermitTunnel no

Subsystem sftp internal-sftp

ClientAliveInterval 60
ClientAliveCountMax 2

PidFile /tmp/sshd.pid

LogLevel VERBOSE
`

// ---- command definitions ----

var hearthforgeCmd = &cobra.Command{
	Use:   "hearthforge",
	Short: "Manage developer environments (HearthForge)",
}

// add-project flags
var (
	hfApID     string
	hfApRepo   string
	hfApBranch string
	hfApStack  string
	hfApPort   int
	hfApDomain string
	hfApCPUs   string
	hfApMemory string
)

var hfAddProjectCmd = &cobra.Command{
	Use:   "add-project",
	Short: "Create or update a project in the registry",
	RunE:  runHFAddProject,
}

// add-dev flags
var (
	hfAdDev      string
	hfAdPubkey   string
	hfAdProject  string
	hfAdIDE      string
	hfAdRecreate bool
)

var hfAddDevCmd = &cobra.Command{
	Use:   "add-dev",
	Short: "Provision a developer environment for a project",
	RunE:  runHFAddDev,
}

var hfListDevsCmd = &cobra.Command{
	Use:   "list-devs",
	Short: "List all developers and their projects",
	RunE:  runHFListDevs,
}

// gateway-add-key flags
var (
	hfGakDev    string
	hfGakPubkey string
)

var hfGatewayAddKeyCmd = &cobra.Command{
	Use:   "gateway-add-key",
	Short: "Add or update an SSH public key for a developer",
	RunE:  runHFGatewayAddKey,
}

// delete-dev flags
var (
	hfDdDev         string
	hfDdProject     string
	hfDdAllProjects bool
	hfDdPurge       bool
	hfDdPurgeAll    bool
)

var hfDeleteDevCmd = &cobra.Command{
	Use:   "delete-dev",
	Short: "Tear down developer environments",
	RunE:  runHFDeleteDev,
}

var hfMigrateSecretsCmd = &cobra.Command{
	Use:   "migrate-secrets",
	Short: "Migrate deploy keys from disk to forge secrets",
	RunE:  runHFMigrateSecrets,
}

func init() {
	hfAddProjectCmd.Flags().StringVar(&hfApID, "id", "", "Project slug (required)")
	hfAddProjectCmd.Flags().StringVar(&hfApRepo, "repo", "", "Git repository URL (required)")
	hfAddProjectCmd.Flags().StringVar(&hfApBranch, "branch", "main", "Default branch")
	hfAddProjectCmd.Flags().StringVar(&hfApStack, "stack", "node", "Toolchain stack: node|python|mixed")
	hfAddProjectCmd.Flags().IntVar(&hfApPort, "port", 0, "Frontend dev server port")
	hfAddProjectCmd.Flags().StringVar(&hfApDomain, "domain", "", "Preview domain override")
	hfAddProjectCmd.Flags().StringVar(&hfApCPUs, "cpus", "1.0", "CPU limit")
	hfAddProjectCmd.Flags().StringVar(&hfApMemory, "memory", "512m", "Memory limit (e.g. 512m, 2g)")

	hfAddDevCmd.Flags().StringVar(&hfAdDev, "dev", "", "Developer id (required)")
	hfAddDevCmd.Flags().StringVar(&hfAdPubkey, "pubkey", "", "SSH public key string or path to .pub file (required)")
	hfAddDevCmd.Flags().StringVar(&hfAdProject, "project", "", "Project id (required)")
	hfAddDevCmd.Flags().StringVar(&hfAdIDE, "ide", "vscode", "IDE to optimise for: vscode|jetbrains|both")
	hfAddDevCmd.Flags().BoolVar(&hfAdRecreate, "recreate", false, "Tear down and recreate if environment already exists")

	hfGatewayAddKeyCmd.Flags().StringVar(&hfGakDev, "dev", "", "Developer id (required)")
	hfGatewayAddKeyCmd.Flags().StringVar(&hfGakPubkey, "pubkey", "", "Public key string or path to .pub file (reads stdin if omitted)")

	hfDeleteDevCmd.Flags().StringVar(&hfDdDev, "dev", "", "Developer id (required)")
	hfDeleteDevCmd.Flags().StringVar(&hfDdProject, "project", "", "Remove from this project only")
	hfDeleteDevCmd.Flags().BoolVar(&hfDdAllProjects, "all-projects", false, "Remove from all projects")
	hfDeleteDevCmd.Flags().BoolVar(&hfDdPurge, "purge", false, "Delete workspace (only with --project)")
	hfDeleteDevCmd.Flags().BoolVar(&hfDdPurgeAll, "purge-all", false, "Delete all workspaces (only with --all-projects)")

	hearthforgeCmd.AddCommand(hfAddProjectCmd)
	hearthforgeCmd.AddCommand(hfAddDevCmd)
	hearthforgeCmd.AddCommand(hfListDevsCmd)
	hearthforgeCmd.AddCommand(hfGatewayAddKeyCmd)
	hearthforgeCmd.AddCommand(hfDeleteDevCmd)
	hearthforgeCmd.AddCommand(hfMigrateSecretsCmd)
}

// ---- add-project ----

func runHFAddProject(cmd *cobra.Command, args []string) error {
	cfg, err := requireInit()
	if err != nil {
		return cmdErr(err)
	}
	hfc := loadHFConfig(cfg)

	id := hfApID
	repo := hfApRepo
	branch := hfApBranch
	stack := hfApStack
	port := hfApPort
	domain := hfApDomain
	cpus := hfApCPUs
	memory := hfApMemory

	if !isJSON() {
		if id == "" {
			id = promptString("Project id (slug, e.g. myapp)", "")
		}
		if repo == "" {
			repo = promptString("Repository URL (HTTPS or SSH)", "")
		}
		if branch == "main" {
			branch = promptString("Default branch", "main")
		}
		if stack == "node" {
			stack = promptString("Stack (node/python/mixed)", "node")
		}
		if port == 0 {
			portStr := promptString("Frontend dev port (e.g. 3000, enter to skip)", "")
			if portStr != "" {
				fmt.Sscanf(portStr, "%d", &port)
			}
		}
	}

	if id == "" {
		return cmdErr(fmt.Errorf("--id is required"))
	}
	if repo == "" {
		return cmdErr(fmt.Errorf("--repo is required"))
	}
	if err := validateSlug(id); err != nil {
		return cmdErr(err)
	}
	switch stack {
	case "node", "python", "mixed":
	default:
		return cmdErr(fmt.Errorf("--stack must be node, python, or mixed"))
	}

	pf, err := readProjectsFile(hfc.projectsJSON)
	if err != nil {
		return cmdErr(err)
	}

	project := hfProject{
		ID:            id,
		Repo:          repo,
		DefaultBranch: branch,
		Stack:         stack,
		Preview: hfProjectPreview{
			Enabled:           port > 0,
			FrontendPort:      port,
			BackendPathPrefix: "/api",
		},
		Resources: hfProjectResources{
			CPUs:   cpus,
			Memory: memory,
		},
		Domain: domain,
	}

	found := false
	for i, p := range pf.Projects {
		if p.ID == id {
			pf.Projects[i] = project
			found = true
			break
		}
	}
	if !found {
		pf.Projects = append(pf.Projects, project)
	}

	if err := writeProjectsFile(hfc.projectsJSON, pf); err != nil {
		return cmdErr(fmt.Errorf("saving projects.json: %w", err))
	}

	hfAuditLog(hfc, "project.added", "admin", fmt.Sprintf("project=%s repo=%s", id, repo))

	// Optionally generate deploy key (interactive only).
	if !isJSON() {
		fmt.Printf("\nGenerate a GitHub deploy key for this project? [y/N] ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() && strings.ToLower(strings.TrimSpace(scanner.Text())) == "y" {
			if err := hfGenerateDeployKey(hfc, id); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: deploy key generation failed: %v\n", err)
			}
		}
	}

	if isJSON() {
		printJSON(map[string]any{
			"ok":      true,
			"id":      id,
			"action":  map[string]bool{"created": !found, "updated": found},
		})
	} else {
		action := "created"
		if found {
			action = "updated"
		}
		fmt.Printf("Project %q %s in %s\n", id, action, hfc.projectsJSON)
		fmt.Printf("\nNext: forge hearthforge add-dev --dev <dev> --pubkey <key> --project %s\n", id)
	}
	return nil
}

func hfGenerateDeployKey(hfc hfConfig, projectID string) error {
	fmt.Printf("Generating deploy key for %q...\n", projectID)
	priv, pub, err := generateDeployKey(projectID)
	if err != nil {
		return err
	}

	store, err := openHFSecrets()
	if err != nil {
		return fmt.Errorf("opening secrets store: %w", err)
	}
	secretKey := fmt.Sprintf("hearthforge.deploykeys.%s", projectID)
	if err := store.Set(secretKey, priv); err != nil {
		return fmt.Errorf("storing deploy key: %w", err)
	}

	fmt.Printf("\nDeploy public key (add as read-only Deploy key on GitHub):\n\n%s\n\n", pub)
	fmt.Printf("Secret stored under: %s\n", secretKey)
	fmt.Printf("GitHub: Settings -> Deploy keys -> Add deploy key (Allow write access: unchecked)\n")
	return nil
}

// ---- add-dev ----

func runHFAddDev(cmd *cobra.Command, args []string) error {
	cfg, err := requireInit()
	if err != nil {
		return cmdErr(err)
	}
	hfc := loadHFConfig(cfg)

	devID := hfAdDev
	pubkeyInput := hfAdPubkey
	projectID := hfAdProject
	ide := hfAdIDE

	if !isJSON() {
		if devID == "" {
			devID = promptString("Developer id (e.g. alice)", "")
		}
		if projectID == "" {
			// Load projects and let user pick.
			pf, _ := readProjectsFile(hfc.projectsJSON)
			if len(pf.Projects) == 0 {
				return cmdErr(fmt.Errorf("no projects found -- run: forge hearthforge add-project first"))
			}
			fmt.Println("Available projects:")
			for i, p := range pf.Projects {
				fmt.Printf("  %d. %s\n", i+1, p.ID)
			}
			choice := promptString("Select project (number or id)", "")
			if choice != "" {
				var n int
				if _, scanErr := fmt.Sscanf(choice, "%d", &n); scanErr == nil && n >= 1 && n <= len(pf.Projects) {
					projectID = pf.Projects[n-1].ID
				} else {
					projectID = choice
				}
			}
		}
		if pubkeyInput == "" {
			pubkeyInput = promptString("SSH public key (paste key or file path)", "")
		}
	}

	if devID == "" {
		return cmdErr(fmt.Errorf("--dev is required"))
	}
	if projectID == "" {
		return cmdErr(fmt.Errorf("--project is required"))
	}
	if pubkeyInput == "" {
		return cmdErr(fmt.Errorf("--pubkey is required"))
	}
	if err := validateSlug(devID); err != nil {
		return cmdErr(fmt.Errorf("invalid --dev: %w", err))
	}
	if err := validateSlug(projectID); err != nil {
		return cmdErr(fmt.Errorf("invalid --project: %w", err))
	}

	pubkey, err := resolvePublicKey(pubkeyInput)
	if err != nil {
		return cmdErr(err)
	}
	if err := validatePublicKey(pubkey); err != nil {
		return cmdErr(fmt.Errorf("invalid public key: %w", err))
	}

	pf, err := readProjectsFile(hfc.projectsJSON)
	if err != nil {
		return cmdErr(err)
	}
	var project *hfProject
	for i := range pf.Projects {
		if pf.Projects[i].ID == projectID {
			project = &pf.Projects[i]
			break
		}
	}
	if project == nil {
		return cmdErr(fmt.Errorf("project %q not found -- run: forge hearthforge add-project --id %s first", projectID, projectID))
	}

	containerName := fmt.Sprintf("dev-%s-%s", projectID, devID)
	workspaceHost := filepath.Join(hfc.workspacesDir, projectID, devID)
	keysHost := filepath.Join(hfc.keysDir, projectID, devID)
	sshdConfigPath := filepath.Join(hfc.sshdDir, projectID, devID, "sshd_config")
	gatewayKeyFile := filepath.Join(hfc.gatewayAuthDir, devID+".pub")
	composeFile := filepath.Join(hfc.workspacesDir, projectID, devID, "compose.yml")
	vhostFile := filepath.Join(hfc.proxyConfDir, fmt.Sprintf("%s-%s.conf", devID, projectID))

	// Handle --recreate: stop and remove existing container.
	if hfAdRecreate {
		_, _ = dockerRun("rm", "-f", containerName)
	} else {
		// Check if already running.
		out, _ := dockerRun("ps", "--filter", "name="+containerName, "--format", "{{.Names}}")
		if strings.Contains(out, containerName) {
			if isJSON() {
				printJSON(map[string]any{"ok": true, "dev": devID, "project": projectID, "container": containerName, "status": "already-running"})
			} else {
				fmt.Printf("Container %s already running. Use --recreate to reprovision.\n", containerName)
			}
			return nil
		}
	}

	hfAuditLog(hfc, "dev.provision.started", "admin", fmt.Sprintf("dev=%s project=%s", devID, projectID))

	// Create directories.
	for _, dir := range []string{workspaceHost, keysHost, filepath.Dir(sshdConfigPath)} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return cmdErr(fmt.Errorf("creating directory %s: %w", dir, err))
		}
	}

	// Write gateway authorized_keys (canonical store).
	if err := appendAuthorizedKey(gatewayKeyFile, pubkey); err != nil {
		return cmdErr(fmt.Errorf("writing gateway authorized_keys: %w", err))
	}

	// Write container authorized_keys (keysHost/<dev>).
	containerKeyFile := filepath.Join(keysHost, "dev")
	if err := overwriteFile(containerKeyFile, pubkey+"\n", 0644); err != nil {
		return cmdErr(fmt.Errorf("writing container authorized_keys: %w", err))
	}

	// Write sshd_config.
	if err := overwriteFile(sshdConfigPath, sshdConfigContent, 0644); err != nil {
		return cmdErr(fmt.Errorf("writing sshd_config: %w", err))
	}

	// Optionally materialize deploy key from secrets.
	deployKeyDir := ""
	store, storeErr := openHFSecrets()
	if storeErr == nil {
		privKey, keyErr := store.Get(fmt.Sprintf("hearthforge.deploykeys.%s", projectID))
		if keyErr == nil {
			dkDir := filepath.Join(hfc.deployKeysDir, projectID)
			dkPath := filepath.Join(dkDir, "id_ed25519")
			if mkErr := os.MkdirAll(dkDir, 0700); mkErr == nil {
				if wErr := os.WriteFile(dkPath, []byte(privKey), 0600); wErr == nil {
					deployKeyDir = dkDir
				}
			}
		}
	}

	// Generate and write docker-compose.yml.
	image := devImage(project.Stack)
	compose := generateComposeYAML(
		devID, projectID, image, hfc.devNetwork,
		workspaceHost, keysHost, sshdConfigPath, deployKeyDir,
		project.Preview.FrontendPort, project.Preview.BackendPort,
		project.Resources.CPUs, project.Resources.Memory,
	)
	if err := overwriteFile(composeFile, compose, 0644); err != nil {
		return cmdErr(fmt.Errorf("writing compose file: %w", err))
	}

	// Start container.
	startOut, startErr := exec.Command("docker", "compose", "-f", composeFile, "-p", containerName, "up", "-d").CombinedOutput()
	if startErr != nil {
		hfAuditLog(hfc, "dev.provision.failed", "admin", fmt.Sprintf("dev=%s project=%s error=%s", devID, projectID, startOut))
		return cmdErr(fmt.Errorf("starting container: %w\n%s", startErr, startOut))
	}

	// Generate vhost config if preview enabled.
	if project.Preview.Enabled && project.Preview.FrontendPort > 0 {
		hostname := fmt.Sprintf("%s-%s.%s", devID, projectID, hfc.domain)
		vhost := generateNginxVhost(devID, projectID, hostname, hfc.certName,
			project.Preview.FrontendPort, project.Preview.BackendPort,
			project.Preview.BackendPathPrefix)
		if err := os.MkdirAll(hfc.proxyConfDir, 0755); err == nil {
			if err := os.WriteFile(vhostFile, []byte(vhost), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to write vhost config: %v\n", err)
			} else {
				reloadProxy()
			}
		}
	}

	// Update devs.json.
	df, err := readDevsFile(hfc.devsJSON)
	if err != nil {
		return cmdErr(err)
	}
	devFound := false
	for i := range df.Developers {
		if df.Developers[i].ID == devID {
			devFound = true
			if !containsString(df.Developers[i].Projects, projectID) {
				df.Developers[i].Projects = append(df.Developers[i].Projects, projectID)
			}
			df.Developers[i].Status = "active"
			break
		}
	}
	if !devFound {
		df.Developers = append(df.Developers, hfDeveloper{
			ID:       devID,
			Status:   "active",
			Projects: []string{projectID},
		})
	}
	if err := writeDevsFile(hfc.devsJSON, df); err != nil {
		return cmdErr(fmt.Errorf("updating devs.json: %w", err))
	}

	hfAuditLog(hfc, "dev.provision.completed", "admin",
		fmt.Sprintf("dev=%s project=%s container=%s", devID, projectID, containerName))

	sshSnippet := generateSSHSnippet(devID, projectID,
		"ssh."+hfc.domain, "~/.ssh/id_ed25519",
		project.Preview.FrontendPort, project.Preview.BackendPort)

	if isJSON() {
		printJSON(map[string]any{
			"ok":        true,
			"dev":       devID,
			"project":   projectID,
			"container": containerName,
			"hostname":  fmt.Sprintf("%s-%s.%s", devID, projectID, hfc.domain),
			"ssh_config": sshSnippet,
			"ide":       ide,
		})
	} else {
		fmt.Printf("Environment provisioned: %s\n\n", containerName)
		fmt.Println("SSH config snippet (add to ~/.ssh/config on developer machine):")
		fmt.Println("---")
		fmt.Println(sshSnippet)
		fmt.Println("---")
		fmt.Printf("\nDeveloper connects with:\n  ssh %s-%s\n", devID, projectID)
		if project.Preview.Enabled {
			fmt.Printf("\nPreview URL: https://%s-%s.%s\n", devID, projectID, hfc.domain)
		}
	}
	return nil
}

// ---- list-devs ----

func runHFListDevs(cmd *cobra.Command, args []string) error {
	cfg, err := requireInit()
	if err != nil {
		return cmdErr(err)
	}
	hfc := loadHFConfig(cfg)

	df, err := readDevsFile(hfc.devsJSON)
	if err != nil {
		return cmdErr(err)
	}

	if isJSON() {
		if df.Developers == nil {
			df.Developers = []hfDeveloper{}
		}
		printJSON(df.Developers)
		return nil
	}

	if len(df.Developers) == 0 {
		fmt.Println("No developers provisioned.")
		return nil
	}
	for _, d := range df.Developers {
		status := ""
		if d.Status == "disabled" {
			status = " (disabled)"
		}
		fmt.Printf("%s%s: %s\n", d.ID, status, strings.Join(d.Projects, ", "))
	}
	return nil
}

// ---- gateway-add-key ----

func runHFGatewayAddKey(cmd *cobra.Command, args []string) error {
	cfg, err := requireInit()
	if err != nil {
		return cmdErr(err)
	}
	hfc := loadHFConfig(cfg)

	devID := hfGakDev
	pubkeyInput := hfGakPubkey

	if devID == "" {
		if isJSON() {
			return cmdErr(fmt.Errorf("--dev is required"))
		}
		devID = promptString("Developer id", "")
	}
	if devID == "" {
		return cmdErr(fmt.Errorf("--dev is required"))
	}

	if pubkeyInput == "" {
		if isJSON() {
			return cmdErr(fmt.Errorf("--pubkey is required in --output json mode"))
		}
		// Read from stdin.
		fmt.Print("Paste public key (or press Enter then Ctrl-D): ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			pubkeyInput = strings.TrimSpace(scanner.Text())
		}
	}

	if pubkeyInput == "" {
		return cmdErr(fmt.Errorf("public key is required (--pubkey or stdin)"))
	}

	pubkey, err := resolvePublicKey(pubkeyInput)
	if err != nil {
		return cmdErr(err)
	}
	if err := validatePublicKey(pubkey); err != nil {
		return cmdErr(fmt.Errorf("invalid public key: %w", err))
	}

	// Append to canonical gateway store.
	gatewayKeyFile := filepath.Join(hfc.gatewayAuthDir, devID+".pub")
	if err := appendAuthorizedKey(gatewayKeyFile, pubkey); err != nil {
		return cmdErr(fmt.Errorf("writing gateway authorized_keys: %w", err))
	}

	// Update _keys/<project>/<dev>/dev for all projects the dev has.
	df, err := readDevsFile(hfc.devsJSON)
	if err != nil {
		return cmdErr(err)
	}
	var devEntry *hfDeveloper
	for i := range df.Developers {
		if df.Developers[i].ID == devID {
			devEntry = &df.Developers[i]
			break
		}
	}

	updated := []string{}
	if devEntry != nil {
		for _, projectID := range devEntry.Projects {
			containerKeyFile := filepath.Join(hfc.keysDir, projectID, devID, "dev")
			if appendErr := appendAuthorizedKey(containerKeyFile, pubkey); appendErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to update key for project %s: %v\n", projectID, appendErr)
			} else {
				updated = append(updated, projectID)
			}
		}
	}

	hfAuditLog(hfc, "gateway.key.added", "admin", fmt.Sprintf("dev=%s", devID))

	if isJSON() {
		printJSON(map[string]any{"ok": true, "dev": devID, "projects_updated": updated})
	} else {
		fmt.Printf("Key added for %s\n", devID)
		if len(updated) > 0 {
			fmt.Printf("Updated container keys for projects: %s\n", strings.Join(updated, ", "))
		}
	}
	return nil
}

// ---- delete-dev ----

func runHFDeleteDev(cmd *cobra.Command, args []string) error {
	cfg, err := requireInit()
	if err != nil {
		return cmdErr(err)
	}
	hfc := loadHFConfig(cfg)

	devID := hfDdDev
	if devID == "" {
		return cmdErr(fmt.Errorf("--dev is required"))
	}

	// Exactly one of --project or --all-projects.
	if hfDdProject == "" && !hfDdAllProjects {
		return cmdErr(fmt.Errorf("one of --project or --all-projects is required"))
	}
	if hfDdProject != "" && hfDdAllProjects {
		return cmdErr(fmt.Errorf("--project and --all-projects are mutually exclusive"))
	}
	if hfDdPurge && hfDdAllProjects {
		return cmdErr(fmt.Errorf("--purge is only valid with --project (use --purge-all with --all-projects)"))
	}
	if hfDdPurgeAll && hfDdProject != "" {
		return cmdErr(fmt.Errorf("--purge-all is only valid with --all-projects (use --purge with --project)"))
	}

	df, err := readDevsFile(hfc.devsJSON)
	if err != nil {
		return cmdErr(err)
	}

	devIdx := -1
	for i, d := range df.Developers {
		if d.ID == devID {
			devIdx = i
			break
		}
	}
	if devIdx == -1 {
		return cmdErr(fmt.Errorf("developer %q not found", devID))
	}

	var projectsToRemove []string
	if hfDdAllProjects {
		projectsToRemove = df.Developers[devIdx].Projects
	} else {
		projectsToRemove = []string{hfDdProject}
	}

	if len(projectsToRemove) == 0 {
		return cmdErr(fmt.Errorf("developer %q has no projects to remove", devID))
	}

	confirmMsg := fmt.Sprintf("Delete %s's environment in: %s?", devID, strings.Join(projectsToRemove, ", "))
	proceed, err := mustConfirm(confirmMsg)
	if err != nil {
		return cmdErr(err)
	}
	if !proceed {
		fmt.Println("Aborted.")
		return nil
	}

	removedProjects := []string{}
	for _, projectID := range projectsToRemove {
		if err := hfDeleteDevProject(hfc, devID, projectID, hfDdPurge || hfDdPurgeAll); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cleanup for %s/%s: %v\n", devID, projectID, err)
		} else {
			removedProjects = append(removedProjects, projectID)
		}
	}

	// Update devs.json.
	if hfDdAllProjects {
		if !isJSON() {
			fmt.Printf("Removing developer record for %s\n", devID)
		}
		df.Developers = append(df.Developers[:devIdx], df.Developers[devIdx+1:]...)
	} else {
		remaining := []string{}
		for _, p := range df.Developers[devIdx].Projects {
			if !containsString(removedProjects, p) {
				remaining = append(remaining, p)
			}
		}
		df.Developers[devIdx].Projects = remaining
		if len(remaining) == 0 {
			df.Developers[devIdx].Status = "disabled"
		}
	}

	if err := writeDevsFile(hfc.devsJSON, df); err != nil {
		return cmdErr(fmt.Errorf("updating devs.json: %w", err))
	}

	hfAuditLog(hfc, "dev.offboarded", "admin",
		fmt.Sprintf("dev=%s projects=%s purge=%v", devID, strings.Join(removedProjects, ","), hfDdPurge||hfDdPurgeAll))

	if isJSON() {
		printJSON(map[string]any{"ok": true, "dev": devID, "removed_projects": removedProjects})
	} else {
		fmt.Printf("Removed %s from: %s\n", devID, strings.Join(removedProjects, ", "))
	}
	return nil
}

// hfDeleteDevProject tears down one (dev, project) pair.
func hfDeleteDevProject(hfc hfConfig, devID, projectID string, purge bool) error {
	containerName := fmt.Sprintf("dev-%s-%s", projectID, devID)

	// Stop and remove container.
	_, _ = dockerRun("rm", "-f", containerName)

	// Remove vhost config.
	vhostFile := filepath.Join(hfc.proxyConfDir, fmt.Sprintf("%s-%s.conf", devID, projectID))
	_ = os.Remove(vhostFile)
	reloadProxy()

	// Remove container keys.
	keysDir := filepath.Join(hfc.keysDir, projectID, devID)
	_ = os.RemoveAll(keysDir)

	// Remove sshd config.
	sshdDir := filepath.Join(hfc.sshdDir, projectID, devID)
	_ = os.RemoveAll(sshdDir)

	// Remove compose file.
	composeFile := filepath.Join(hfc.workspacesDir, projectID, devID, "compose.yml")
	_ = os.Remove(composeFile)

	// Purge workspace if requested.
	if purge {
		workspaceDir := filepath.Join(hfc.workspacesDir, projectID, devID)
		if err := os.RemoveAll(workspaceDir); err != nil {
			return fmt.Errorf("purging workspace %s: %w", workspaceDir, err)
		}
	}

	return nil
}

// ---- migrate-secrets ----

func runHFMigrateSecrets(cmd *cobra.Command, args []string) error {
	cfg, err := requireInit()
	if err != nil {
		return cmdErr(err)
	}
	hfc := loadHFConfig(cfg)

	store, err := openHFSecrets()
	if err != nil {
		return cmdErr(fmt.Errorf("opening secrets store: %w", err))
	}

	deployKeysDir := hfc.deployKeysDir
	entries, err := os.ReadDir(deployKeysDir)
	if os.IsNotExist(err) {
		fmt.Println("No _deploy_keys directory found. Nothing to migrate.")
		return nil
	}
	if err != nil {
		return cmdErr(fmt.Errorf("reading deploy keys dir: %w", err))
	}

	migrated := 0
	skipped := 0
	var report []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectID := entry.Name()
		keyPath := filepath.Join(deployKeysDir, projectID, "id_ed25519")
		data, err := os.ReadFile(keyPath)
		if os.IsNotExist(err) {
			skipped++
			report = append(report, fmt.Sprintf("SKIP  %s  (no id_ed25519 found)", projectID))
			continue
		}
		if err != nil {
			report = append(report, fmt.Sprintf("ERROR %s  %v", projectID, err))
			continue
		}

		secretKey := fmt.Sprintf("hearthforge.deploykeys.%s", projectID)
		if err := store.Set(secretKey, string(data)); err != nil {
			report = append(report, fmt.Sprintf("ERROR %s  failed to store secret: %v", projectID, err))
			continue
		}

		// Remove plaintext key file.
		if err := os.Remove(keyPath); err != nil {
			report = append(report, fmt.Sprintf("WARN  %s  migrated but failed to delete plaintext: %v", projectID, err))
		} else {
			report = append(report, fmt.Sprintf("OK    %s  -> hearthforge.deploykeys.%s", projectID, projectID))
		}
		migrated++
	}

	hfAuditLog(hfc, "secrets.migrated", "admin",
		fmt.Sprintf("migrated=%d skipped=%d", migrated, skipped))

	if isJSON() {
		printJSON(map[string]any{"migrated": migrated, "skipped": skipped})
	} else {
		for _, line := range report {
			fmt.Println(line)
		}
		fmt.Printf("\nMigration complete: %d migrated, %d skipped.\n", migrated, skipped)
	}
	return nil
}

// ---- utils ----

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
