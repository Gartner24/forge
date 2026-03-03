use std::fs::{self, OpenOptions};
use std::net::SocketAddr;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::sync::Arc;

use anyhow::{Context, Result};
use russh::keys::{load_secret_key, parse_public_key_base64, PrivateKey, PublicKey};
use russh::server::{Auth, Handler, Server};
use russh::MethodSet;
use serde::Deserialize;
use tokio::sync::RwLock;
use tracing::{error, info, warn};

#[derive(Debug, Deserialize, Clone)]
struct GatewayConfig {
    #[serde(default)]
    server: ServerConfig,
    #[serde(default)]
    paths: PathsConfig,
}

#[derive(Debug, Deserialize, Clone)]
struct ServerConfig {
    /// Listen address for the gateway, e.g. "0.0.0.0:2224".
    #[serde(default = "ServerConfig::default_listen_addr")]
    listen_addr: String,

    /// Path to the SSH host private key.
    #[serde(default = "ServerConfig::default_host_key_path")]
    host_key_path: String,
}

impl Default for ServerConfig {
    fn default() -> Self {
        Self {
            listen_addr: Self::default_listen_addr(),
            host_key_path: Self::default_host_key_path(),
        }
    }
}

impl ServerConfig {
    fn default_listen_addr() -> String {
        "0.0.0.0:2224".to_string()
    }

    fn default_host_key_path() -> String {
        "/opt/infra/forge/gateway/keys/ssh_host_ed25519_key".to_string()
    }
}

#[derive(Debug, Deserialize, Clone)]
struct PathsConfig {
    /// Path to registry devs.json.
    #[serde(default = "PathsConfig::default_devs_json")]
    devs_json: String,

    /// Directory with per-developer authorized keys files.
    #[serde(default = "PathsConfig::default_authorized_keys_dir")]
    authorized_keys_dir: String,

    /// Directory where audit logs are written.
    #[serde(default = "PathsConfig::default_audit_log_dir")]
    audit_log_dir: String,
}

impl Default for PathsConfig {
    fn default() -> Self {
        Self {
            devs_json: Self::default_devs_json(),
            authorized_keys_dir: Self::default_authorized_keys_dir(),
            audit_log_dir: Self::default_audit_log_dir(),
        }
    }
}

impl PathsConfig {
    fn default_devs_json() -> String {
        "/opt/infra/forge/registry/devs.json".to_string()
    }

    fn default_authorized_keys_dir() -> String {
        "/opt/infra/forge/gateway/authorized_keys".to_string()
    }

    fn default_audit_log_dir() -> String {
        "/opt/infra/forge/gateway/logs".to_string()
    }
}

#[derive(Debug, Deserialize)]
struct DeveloperEntry {
    id: String,
    #[serde(default)]
    projects: Vec<String>,
    #[serde(default)]
    status: String,
}

/// In-memory view of allowed projects per developer.
#[derive(Debug, Default)]
struct AccessState {
    dev_projects: Vec<DeveloperEntry>,
}

impl AccessState {
    fn is_project_allowed(&self, dev_id: &str, project_id: &str) -> bool {
        self.dev_projects.iter().any(|d| {
            d.id == dev_id
                && d.status == "active"
                && d.projects.iter().any(|p| p == project_id)
        })
    }
}

/// Shared state for all SSH sessions.
#[derive(Clone)]
struct GatewayState {
    cfg: GatewayConfig,
    access: Arc<RwLock<AccessState>>,
}

impl GatewayState {
    async fn reload_access(&self) -> Result<()> {
        let path = PathBuf::from(&self.cfg.paths.devs_json);
        let text = fs::read_to_string(&path)
            .with_context(|| format!("failed to read devs.json from {}", path.display()))?;
        let json: serde_json::Value =
            serde_json::from_str(&text).context("failed to parse devs.json as JSON")?;

        let developers = json
            .get("developers")
            .and_then(|v| v.as_array())
            .cloned()
            .unwrap_or_default();

        let mut entries = Vec::new();
        for d in developers {
            let id = d
                .get("id")
                .and_then(|v| v.as_str())
                .unwrap_or_default()
                .to_string();
            let status = d
                .get("status")
                .and_then(|v| v.as_str())
                .unwrap_or("active")
                .to_string();
            let projects = d
                .get("projects")
                .and_then(|v| v.as_array())
                .map(|arr| {
                    arr.iter()
                        .filter_map(|p| p.as_str().map(|s| s.to_string()))
                        .collect::<Vec<_>>()
                })
                .unwrap_or_default();
            entries.push(DeveloperEntry {
                id,
                projects,
                status,
            });
        }

        let mut guard = self.access.write().await;
        guard.dev_projects = entries;
        Ok(())
    }

    fn authorized_keys_path(&self, dev_id: &str) -> PathBuf {
        Path::new(&self.cfg.paths.authorized_keys_dir)
            .join(format!("{}.pub", dev_id))
    }

    fn audit_log_path(&self) -> PathBuf {
        let dir = Path::new(&self.cfg.paths.audit_log_dir);
        let _ = fs::create_dir_all(dir);
        dir.join("audit.log")
    }

    fn log_audit(
        &self,
        peer: Option<SocketAddr>,
        dev_id: &str,
        project_id: &str,
        result: &str,
        reason: &str,
    ) {
        let ts = chrono::Utc::now().to_rfc3339();
        let line = format!(
            "{} peer={} dev={} project={} result={} reason={}\n",
            ts,
            peer
                .map(|p| p.to_string())
                .unwrap_or_else(|| "-".to_string()),
            dev_id,
            project_id,
            result,
            reason
        );

        let path = self.audit_log_path();
        if let Err(e) = OpenOptions::new()
            .create(true)
            .append(true)
            .open(&path)
            .and_then(|mut f| {
                use std::io::Write;
                f.write_all(line.as_bytes())
            })
        {
            error!("failed to write audit log {}: {e}", path.display());
        }
    }
}

/// Simple helper to load or generate the host key.
fn load_or_generate_host_key(path: &Path) -> Result<PrivateKey> {
    if !path.exists() {
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent)
                .with_context(|| format!("failed to create host key dir {}", parent.display()))?;
        }
        // Use ssh-keygen to generate an ed25519 host key, to keep format compatible
        // with standard OpenSSH tools.
        let status = Command::new("ssh-keygen")
            .arg("-t")
            .arg("ed25519")
            .arg("-N")
            .arg("")
            .arg("-f")
            .arg(path)
            .status()
            .context("failed to run ssh-keygen")?;
        if !status.success() {
            anyhow::bail!("ssh-keygen failed with status {status}");
        }
    }

    let key = load_secret_key(path, None)
        .with_context(|| format!("failed to load host key from {}", path.display()))?;
    Ok(key)
}

/// Match a public key against lines in authorized_keys for a developer.
fn key_allowed_for_dev(path: &Path, offered: &PublicKey) -> Result<bool> {
    let data = match fs::read_to_string(path) {
        Ok(s) => s,
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(false),
        Err(e) => return Err(e.into()),
    };

    for line in data.lines() {
        let line = line.trim();
        if line.is_empty() || line.starts_with('#') {
            continue;
        }

        // authorized_keys format: "type base64 [comment]".
        // We only care about the *base64* field for matching, ignore type and any trailing comment.
        let mut parts = line.split_whitespace();
        let _key_type = match parts.next() {
            Some(_) => (),
            None => continue,
        };
        let base64 = match parts.next() {
            Some(s) => s,
            None => continue,
        };

        if let Ok(pk) = parse_public_key_base64(base64) {
            // Compare only cryptographic key material. PublicKey equality in the
            // underlying ssh-key crate also includes comment metadata, which may
            // differ between parsed authorized_keys entries and offered client keys.
            if pk.key_data() == offered.key_data() {
                return Ok(true);
            }
        }
    }

    Ok(false)
}

struct GatewayServer {
    state: GatewayState,
}

struct GatewayHandler {
    state: GatewayState,
    peer: Option<SocketAddr>,
    dev_id: Option<String>,
    project_id: Option<String>,
}

impl russh::server::Server for GatewayServer {
    type Handler = GatewayHandler;

    fn new_client(&mut self, peer_addr: Option<SocketAddr>) -> Self::Handler {
        GatewayHandler {
            state: self.state.clone(),
            peer: peer_addr,
            dev_id: None,
            project_id: None,
        }
    }
}

impl Handler for GatewayHandler {
    type Error = anyhow::Error;

    async fn auth_publickey(
        &mut self,
        user: &str,
        key: &PublicKey,
    ) -> Result<Auth, Self::Error> {
        // Username encodes dev-project pair: <dev>-<project>.
        let parts: Vec<&str> = user.splitn(2, '-').collect();
        if parts.len() != 2 {
            warn!("invalid username format '{}', expected <dev>-<project>", user);
            if let Some(peer) = self.peer {
                self.state.log_audit(
                    Some(peer),
                    user,
                    "-",
                    "rejected",
                    "invalid-username-format",
                );
            }
            return Ok(Auth::Reject { proceed_with_methods: None });
        }

        let dev_id = parts[0].to_string();
        let project_id = parts[1].to_string();

        // Check dev+project membership.
        {
            let access = self.state.access.read().await;
            if !access.is_project_allowed(&dev_id, &project_id) {
                warn!(
                    "access denied: dev '{}' not allowed for project '{}'",
                    dev_id, project_id
                );
                if let Some(peer) = self.peer {
                    self.state.log_audit(
                        Some(peer),
                        &dev_id,
                        &project_id,
                        "rejected",
                        "project-not-allowed",
                    );
                }
                return Ok(Auth::Reject { proceed_with_methods: None });
            }
        }

        // Check key against authorized_keys/<dev>.pub.
        let auth_path = self.state.authorized_keys_path(&dev_id);
        let key_ok = key_allowed_for_dev(&auth_path, key).unwrap_or(false);
        if !key_ok {
            warn!(
                "public key not authorized for dev '{}', file {}",
                dev_id,
                auth_path.display()
            );
            if let Some(peer) = self.peer {
                self.state.log_audit(
                    Some(peer),
                    &dev_id,
                    &project_id,
                    "rejected",
                    "publickey-not-authorized",
                );
            }
            return Ok(Auth::Reject { proceed_with_methods: None });
        }

        info!(
            "authentication accepted for dev={} project={}",
            dev_id, project_id
        );

        self.dev_id = Some(dev_id.clone());
        self.project_id = Some(project_id.clone());
        if let Some(peer) = self.peer {
            self.state
                .log_audit(Some(peer), &dev_id, &project_id, "accepted", "ok");
        }

        Ok(Auth::Accept)
    }
}

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter("info")
        .with_target(false)
        .init();

    // Load config from gateway.toml if present, otherwise use defaults.
    let cfg_path = PathBuf::from("/opt/infra/forge/registry/gateway.toml");
    let cfg: GatewayConfig = if cfg_path.exists() {
        let text = fs::read_to_string(&cfg_path)
            .with_context(|| format!("failed to read {}", cfg_path.display()))?;
        toml::from_str(&text)
            .with_context(|| format!("failed to parse {}", cfg_path.display()))?
    } else {
        warn!(
            "no gateway.toml found at {}, using built-in defaults",
            cfg_path.display()
        );
        GatewayConfig {
            server: ServerConfig::default(),
            paths: PathsConfig::default(),
        }
    };

    let state = GatewayState {
        cfg: cfg.clone(),
        access: Arc::new(RwLock::new(AccessState::default())),
    };
    state
        .reload_access()
        .await
        .context("failed to load initial devs.json")?;

    // Prepare host key.
    let host_key_path = PathBuf::from(&cfg.server.host_key_path);
    let host_key = load_or_generate_host_key(&host_key_path)?;

    let mut ssh_cfg = russh::server::Config::default();
    ssh_cfg.keys.push(host_key);
    ssh_cfg.auth_rejection_time = std::time::Duration::from_secs(1);
    ssh_cfg.auth_rejection_time_initial = Some(std::time::Duration::from_millis(500));
    // Restrict to publickey only.
    let mut methods = MethodSet::empty();
    methods.push(russh::MethodKind::PublicKey);
    ssh_cfg.methods = methods;

    let ssh_cfg = Arc::new(ssh_cfg);

    let listen_addr = cfg.server.listen_addr.clone();
    info!("starting gateway on {}", listen_addr);

    let mut server = GatewayServer { state };

    server
        .run_on_address(ssh_cfg, listen_addr.as_str())
        .await
        .context("gateway server exited")?;

    Ok(())
}

