# Threat Model and Security Posture

This document defines trust boundaries, attacker models, and non-negotiable security rules for HearthForge.

## Actors

- **Admin** — manages the VPS, proxy stack, HearthForge, registries, TLS, and networks.
- **Developer** — has access only to assigned dev containers and repositories.
- **External attacker** — attempts exploitation via exposed services (proxy/gateway) or via compromised developer credentials.

## Trust Boundaries

1. **Internet boundary** — publicly exposed services: Nginx/Caddy proxy (80/443) and SSH gateway (port 2224).
2. **Host boundary** — developers must not receive host shell access. All developer work occurs inside containers.
3. **Network boundary** — production network `web` is separate from dev network `dev-web`. Dev containers must not be attached to `web`.
4. **Docker daemon boundary** — any access to the Docker daemon is equivalent to root on host. Developers must never have access to the Docker socket or membership in the Docker group.

## Non-Negotiable Rules

- No `--privileged` containers
- Do not mount `/var/run/docker.sock` into any dev container
- Dev containers must run as non-root
- Drop Linux capabilities by default (`cap_drop: [ALL]`)
- Use resource limits for dev containers (CPU and memory)
- Dev containers attach only to `dev-web`
- Proxy is the only public HTTP/HTTPS entry point
- Gateway is the only public SSH entry point
- Sensitive secrets (Cloudflare tokens, deploy keys) are not committed to Git — use `forge secrets`

## SSH Key and Identity Security

- Developer authentication is only via SSH public keys
- Keys are administered by HearthForge:
  - stored as public keys under `gateway/authorized_keys/`
  - mapped to developer identity in `registry/devs.json`
- Host keys for the gateway are persistent and stored under `gateway/keys/`

## Data Handling

- Workspaces are stored under `/opt/data/dev_workspaces/`
- Backups are stored under `/opt/data/backups/`
- Logs (proxy/gateway) are stored under `/opt/data/logs/`
- Dev containers must not have host mounts outside their workspace

## Audit Logging Requirements

At minimum, record:
- Timestamp
- Developer identity
- Source IP
- Project target
- Container target
- Session start/end (or at least session start)
- Failed auth attempts

Audit logs must be append-only (enforced by permissions) and rotated.

## Residual Risk Notes

Containers are not equivalent to hardened VMs. The mitigation strategy is:
- Strong isolation and least privilege inside containers
- Strict prohibition of Docker daemon access
- Separate dev/prod networks
- Minimal host exposure surface
- Audit logs and key management
