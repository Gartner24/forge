# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in Forge, please report it responsibly. Do **not** open a public GitHub issue.

**Contact:** open a [GitHub Security Advisory](https://github.com/<user>/forge/security/advisories/new) (private disclosure).

Include:
- Description of the vulnerability
- Steps to reproduce
- Affected module(s) and version(s)
- Potential impact

You will receive an acknowledgement within 48 hours and a resolution timeline within 7 days.

## Security Model

### What Forge Protects

- Developer SSH keys are stored in the gateway's canonical `authorized_keys` store and never exposed to containers
- Secrets are encrypted at rest using age encryption
- All inter-node traffic in FluxForge is encrypted by WireGuard
- PenForge scan engines are isolated in Docker containers with no access to internal networks
- Audit logs are append-only and cannot be modified by module code

### Known Limitations

- Containers are not equivalent to hardened VMs. The mitigation strategy is strong isolation, least privilege, and no Docker socket access — not VM-level guarantees.
- FluxForge mesh security depends on the controller node not being compromised. Treat the controller node as the most sensitive node in your infrastructure.
- PenForge is a tool pointed at your own infrastructure. Scope enforcement prevents scanning arbitrary targets, but admins must register targets carefully.

### Non-Negotiable Security Rules

These rules are enforced across the entire suite and cannot be configured away:

- No `--privileged` containers
- No Docker socket mounts into any container
- Dev containers run as non-root
- `cap_drop: [ALL]` on all dev containers
- Resource limits on all dev containers
- Dev containers attach only to `dev-web` network, never `web`
- PenForge scan scope is enforced before every scan — no overrides at runtime
- All audit logs are append-only

## Supported Versions

Only the latest release of each module receives security fixes. Patch releases are issued for critical vulnerabilities.

## Disclosure Timeline

| Day | Action |
|---|---|
| 0 | Report received, acknowledgement sent |
| 7 | Resolution timeline communicated |
| 30 | Fix developed and tested |
| 45 | Fix released, advisory published |

Critical vulnerabilities (CVSS 9.0+) are patched and released within 7 days.
