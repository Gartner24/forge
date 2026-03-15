# Project registry

The project registry defines which projects can be provisioned into developer environments and how to configure them.

Location:
- `registry/projects.json`

## Required fields (recommended)

- `id`: unique project id (lowercase, dashes allowed)
- `repo`: git repository URL
- `default_branch`: e.g. `main`
- `stack`: `node`, `python`, `mixed`
- `dev_ports`:
  - `frontend`: port number
  - `backend`: port number (optional)
- `bootstrap`:
  - `clone`: boolean default
  - `path`: workspace path inside container (default `/workspace/<project>`)

Optional fields:
- `startup`:
  - list of commands to start dev servers
- `resources`:
  - default cpu/memory limits
- `preview`:
  - whether to expose HTTP preview via proxy

## Example entries

- `hemis`:
  - mixed stack (frontend + backend)
  - ports: frontend 5173 or 3000, backend 5000
- `tiap`:
  - depends on stack
  - define ports accordingly

## Validation rules

- `id` must match naming convention
- `repo` must be a valid URL or SSH remote
- ports must be numeric and safe for internal exposure
- if `preview` is enabled, proxy template must support routing

## devs.json relationship

`registry/devs.json` references project `id` values to grant access. No access should be granted to unknown project ids.

## Change management

- Changes to `projects.json` are admin-only.
- Maintain example schemas and avoid committing secrets.

