## `docs/07-devctl.md`

```md
# devctl (admin CLI)

This document describes the admin CLI responsibilities, commands, expected prompts, and generated artifacts.

## Purpose

`devctl` is an admin-only CLI that provisions and manages developer environments.

It must:
- read project definitions from `registry/projects.json`
- manage developer identities and access in `registry/devs.json`
- generate dev container compose definitions (from templates)
- generate proxy vhost configs for dev domains (from templates)
- trigger safe proxy reloads (after validation)

## Core commands

- `devctl add-project`
- `devctl add-dev`
- `devctl list-devs`
- `devctl grant <dev> <project>`
- `devctl revoke <dev> <project>`
- `devctl disable-dev <dev>`
- `devctl delete-dev <dev> <project>`

## add-project

Inputs:
- project id (e.g. `hemis`)
- repository URL
- stack type (node/python/mixed)
- default branch
- dev ports (frontend/backend)
- optional startup commands

Output:
- updates `registry/projects.json`

## add-dev (provision)

Prompts:
- developer id (e.g. `santiago`)
- developer public key (paste or file)
- select project (1..n)
- optional: bootstrap clone (yes/no)
- optional: resource overrides (cpu/memory)

Generates:
- dev container name: `dev-<project>-<dev>`
- workspace path:
  - host: `/opt/data/dev_workspaces/<project>/<dev>/` (if bind mount)
  - container: `/workspace/<project>`
- dev hostname: `<dev>-<project>.dev.domain.com`
- proxy vhost config written to proxy live directory:
  - `/opt/infra/proxy/conf.d/active/dev/<dev>-<project>.conf` (or equivalent)
- registry updates:
  - `devs.json` developer record and access mapping
- key storage:
  - `gateway/authorized_keys/<dev>.pub` (or structured per dev)

Outputs to admin:
- DNS checklist (if wildcard not used)
- SSH config snippet for ProxyJump
- verification commands:
  - container running
  - ssh connection
  - nginx validation/reload status

## grant / revoke

- grant adds project access and provisions container if needed
- revoke removes project access and optionally stops/removes container

## disable-dev

- disables developer key at gateway mapping level
- does not necessarily delete workspaces (admin choice)

## delete-dev <dev> <project>

Removes:
- container
- workspace volume/bind mount (optional archive)
- proxy vhost config
- access mapping
- key files (optional: keep for audit)

## Safety requirements

- Validate Nginx config before reload:
  - `nginx -t` in proxy container
- Do not delete workspaces without explicit confirmation flag
- Keep audit logs immutable and retained beyond workspace deletion

