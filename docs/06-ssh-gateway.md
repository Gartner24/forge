# SSH gateway (Rust jump host)

This document describes the Rust SSH gateway and why it acts as a jump host rather than replacing sshd in dev containers.


## Why jump host mode

VS Code / Cursor Remote-SSH expects:
- a normal SSH server endpoint
- SFTP support
- ability to install and run a remote server component

To satisfy this, dev containers run OpenSSH (`sshd`). The gateway enforces authentication and routing but does not terminate developer sessions as a shell provider.

## High-level behavior

- Developers SSH to the gateway only (e.g. `ssh.dev.domain.com:2224`).
- Gateway authenticates via public key.
- Gateway resolves developer identity and allowed projects.
- Gateway forwards the SSH connection to the correct dev container on `dev-web` (container sshd).

Developers never receive a host shell.

## Identity and policy mapping

- Public key -> developer identity
- Developer identity -> allowed projects
- Allowed project -> container target(s)

Sources:
- `registry/devs.json` is the source of truth for developer identities and access.
- Public keys are stored under `gateway/authorized_keys/`.
- Gateway configuration (listen address, host key path, log directory) is defined in `registry/gateway.toml` on the server.

## Developer key lifecycle

Canonical store:

- For each developer id `<dev>`, the canonical SSH public keys live in:
  - `/opt/infra/forge/gateway/authorized_keys/<dev>.pub`
- Each non-empty, non-comment line in that file is treated as one allowed key for that developer.

How keys are written:

- `devctl add-dev`:
  - Prompts for (or reuses) the developer’s SSH public key.
  - Ensures the key is present in `/opt/infra/forge/gateway/authorized_keys/<dev>.pub`.
  - Writes the same key into the container’s `_keys` directory:
    - `/opt/data/dev_workspaces/_keys/<project>/<dev>/dev`
    - so the target container’s `sshd` accepts the key once the gateway forwards the connection.
- `devctl gateway-add-key`:
  - Adds additional keys for an existing developer (e.g. second laptop).
  - Appends them to the canonical gateway file.
  - Updates `_keys` for all projects that developer currently has in `registry/devs.json`.

Containers never read `gateway/authorized_keys` directly; `devctl` is responsible for keeping per-container `_keys` in sync with the gateway’s canonical view.

## Routing model

Recommended:
- One container per developer per project, each with its own sshd.
- Gateway routes to `dev-<project>-<dev>:22` on `dev-web`.

## Host keys

Gateway host keys must be persistent:
- store under `gateway/keys/`
- do not generate ephemeral host keys on every boot

## Logging

Gateway must log at minimum:
- timestamp
- source IP
- developer identity (or unknown)
- auth success/fail
- project selected (if applicable)
- container target

Store logs:
- `gateway/logs/audit.log` (append-only)
- optionally mirror to `/opt/data/logs/gateway/`

## Deployment (Docker on dev-web)

The reference deployment runs the Rust gateway as a Docker container on the `dev-web` network, alongside the global proxy.

### 1. Build the gateway image

On the Forge host, from the `gateway/` crate directory:

```bash
cd /opt/infra/forge/gateway    # adjust if repo lives elsewhere
sudo docker build -t forge-gateway:latest .
```

The provided `Dockerfile` builds `forge-gateway` in release mode and installs it as `/usr/local/bin/forge-gateway` in the image, listening on port `2224`.

### 2. Compose service (proxy stack)

In the proxy stack (e.g. `/opt/infra/proxy/compose.yml`), add a `gateway` service:

```yaml
services:
  gateway:
    image: forge-gateway:latest
    restart: unless-stopped
    networks:
      - dev-web
    ports:
      - "2224:2224"
    volumes:
      - /opt/infra/forge/registry:/opt/infra/forge/registry:ro
      - /opt/infra/forge/gateway/keys:/opt/infra/forge/gateway/keys
      - /opt/infra/forge/gateway/authorized_keys:/opt/infra/forge/gateway/authorized_keys
      - /opt/infra/forge/gateway/logs:/opt/infra/forge/gateway/logs

networks:
  dev-web:
    external: true
```

Notes:
- The container **must** attach to `dev-web` so it can reach `dev-<project>-<dev>:22` by Docker DNS.
- Port `2224` on the host is the single public SSH entrypoint for developers.

### 3. Gateway config (`gateway.toml`)

On the host, create `/opt/infra/forge/registry/gateway.toml`:

```toml
[server]
listen_addr = "0.0.0.0:2224"
host_key_path = "/opt/infra/forge/gateway/keys/ssh_host_ed25519_key"

[paths]
devs_json = "/opt/infra/forge/registry/devs.json"
authorized_keys_dir = "/opt/infra/forge/gateway/authorized_keys"
audit_log_dir = "/opt/infra/forge/gateway/logs"
```

Also ensure the directories exist:

```bash
sudo mkdir -p /opt/infra/forge/gateway/{keys,authorized_keys,logs}
```

### 4. Host key creation and permissions

Forge expects a persistent ed25519 host key for the gateway. Generate it **on the host**:

```bash
sudo ssh-keygen -t ed25519 -N '' \
  -f /opt/infra/forge/gateway/keys/ssh_host_ed25519_key

# If the gateway container runs as uid 1000 (`gateway` user), fix ownership:
sudo chown -R 1000:1000 /opt/infra/forge/gateway/keys
```

The gateway reads this key from `host_key_path` at startup; it will not attempt to regenerate it if it already exists.

### 5. Firewall and DNS

On the host:

```bash
sudo ufw allow 2224/tcp
```

In DNS:

- Create `ssh.dev.<dev_base_domain>` pointing to the VPS public IP, for example:
  - `ssh.dev.qyvos.com -> <VPS IP>`

### 6. Starting and verifying the service

From the proxy stack directory (e.g. `/opt/infra/proxy`):

```bash
sudo docker compose up -d gateway
```

Verify:

```bash
docker ps | grep gateway
sudo ss -lntp | grep 2224
sudo docker logs gateway --tail=50
```

You should see:
- a listener on `0.0.0.0:2224`, and
- log lines similar to `starting gateway on 0.0.0.0:2224` with no fatal errors.

## Operational modes

- Terminal: normal ssh client uses ProxyJump (gateway) to reach container.
- VS Code/Cursor: uses the same ProxyJump configuration.

## Access control notes

Do not allow developers to access:
- host filesystem
- docker daemon
- other developers’ containers
- production network `web`

Network separation plus gateway policy enforcement is required.

