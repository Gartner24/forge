
# Threat model and security posture

This document defines trust boundaries, attacker models, and non-negotiable security rules for Forge.

## Actors

- Admin: manages the VPS, proxy stack, Forge, registries, TLS, and networks.
- Developer: has access only to assigned dev containers and repositories.
- External attacker: attempts exploitation via exposed services (proxy/gateway) or via compromised developer credentials.

## Trust boundaries

1. Internet boundary
   - Publicly exposed services: Nginx proxy (80/443) and SSH gateway (SSH port).
2. Host boundary
   - Developers must not receive host shell access.
   - All developer work occurs inside containers.
3. Network boundary
   - Production network `web` is separate from dev network `dev-web`.
   - Dev containers should not be attached to `web`.
4. Docker daemon boundary
   - Any access to the Docker daemon is equivalent to root on host.

   - Developers must never have access to Docker socket or membership in the Docker group.

## Non-negotiable rules

- No `--privileged` containers.
- Do not mount `/var/run/docker.sock` into any dev container.
- Dev containers must run as non-root.
- Drop Linux capabilities by default (`cap_drop: [ALL]`).
- Use resource limits for dev containers (CPU and memory).
- Dev containers attach only to `dev-web`.
- Proxy is the only public HTTP/HTTPS entry point.
- Gateway is the only public SSH entry point.
- Sensitive secrets (Cloudflare tokens, deploy keys) are not committed to Git.

## SSH key and identity security

- Developer authentication is only via SSH public keys.
- Keys are administered by Forge:
  - stored as public keys under `gateway/authorized_keys/`
  - mapped to developer identity in `registry/devs.json`
- Host keys for the gateway are persistent and stored under `gateway/keys/`.

## Data handling

- Workspaces are stored under `/opt/data/dev_workspaces/`.
- Backups are stored under `/opt/data/backups/`.
- Logs (proxy/gateway) are stored under `/opt/data/logs/` and gateway’s internal `logs/` as configured.
- Dev containers should not have host mounts outside their workspace.

## Audit logging requirements

At minimum record:
- timestamp
- developer identity
- source IP
- project target
- container target
- session start/end (or at least session start)
- failed auth attempts

Audit logs should be append-only (enforced by permissions) and rotated.

## Residual risk notes

Containers are not equivalent to hardened VMs. The mitigation strategy is:
- strong isolation and least privilege inside container
- strict prohibition of Docker daemon access
- separate dev/prod networks
- minimal host exposure surface
- audit logs and key management

