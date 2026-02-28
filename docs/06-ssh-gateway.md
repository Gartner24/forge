# SSH gateway (Rust jump host)

This document describes the Rust SSH gateway and why it acts as a jump host rather than replacing sshd in dev containers.


## Why jump host mode

VS Code / Cursor Remote-SSH expects:
- a normal SSH server endpoint
- SFTP support
- ability to install and run a remote server component

To satisfy this, dev containers run OpenSSH (`sshd`). The gateway enforces authentication and routing but does not terminate developer sessions as a shell provider.

## High-level behavior

- Developers SSH to the gateway only.
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

