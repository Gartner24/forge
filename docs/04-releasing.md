# Releasing

Forge uses per-module versioning. Each module is tagged and released independently. This means SmeltForge can release v0.3.0 while FluxForge is still at v0.1.0.

## Version Format

```
<module>/v<major>.<minor>.<patch>
```

Examples:
```
core/v0.1.0
smeltforge/v0.2.1
hearthforge/v1.0.0
```

## Release Process

```bash
# 1. Make sure you are on main and up to date
git checkout main && git pull

# 2. Run all tests
just test-all

# 3. Run lint
just lint

# 4. Tag and push the release
just release smeltforge 0.2.0
# → runs: git tag smeltforge/v0.2.0
# → runs: git push origin smeltforge/v0.2.0
```

The `release.yml` GitHub Actions workflow triggers on any tag matching `*/v*.*.*` and:
1. Runs `mise install`
2. Builds the module binary
3. Creates a GitHub Release with the binary attached
4. Updates the module's changelog

## Versioning Guidelines

- `patch` — bug fixes, no API changes
- `minor` — new features, backwards-compatible
- `major` — breaking changes to CLI interface or module API

During early development (v0.x.x), minor version bumps may include breaking changes.

## Releasing shared/

Changes to `shared/` affect all modules. When releasing a shared change:

1. Release `shared` first: `just release shared 0.2.0`
2. Update all module `go.mod` files to use the new shared version
3. Release each affected module

## Changelog

Each module maintains a `CHANGELOG.md` in its directory following [Keep a Changelog](https://keepachangelog.com/) format.
