# Secrets Store

Forge Core ships a built-in encrypted secrets store. It is the only sanctioned place to store credentials, tokens, API keys, and other sensitive values in a Forge installation. All modules read secrets through `shared/secrets` — they never access this file directly.

## Storage Format

Secrets are stored in a single age-encrypted file at `~/.forge/secrets.age`. The encryption key is derived from the server's host identity. The file is never written in plaintext at any point — not as a temp file, not in swap.

Key names follow a dot-separated namespace convention:

```
smeltforge.hemis.DB_URL
smeltforge.hemis.webhook_secret
hearthforge.hemis.deploy_key
sparkforge.gotify.app_token
fluxforge.controller.wireguard_key
```

The format is `<module>.<scope>.<name>`. Values can be arbitrary strings, including multiline values such as private keys.

## CLI

```bash
# Store a secret
forge secrets set smeltforge.hemis.DB_URL "postgres://..."

# Retrieve a secret (prints to stdout)
forge secrets get smeltforge.hemis.DB_URL

# List all key names (values are never shown)
forge secrets list

# List keys for a specific module
forge secrets list --prefix smeltforge.hemis

# Delete a secret
forge secrets delete smeltforge.hemis.DB_URL

# Sync to all mesh nodes (requires FluxForge)
forge secrets set sparkforge.smtp.password "..." --sync
```

## Namespace Rules

- Modules only read and write under their own namespace prefix.
- Modules never reach into another module's namespace.
- Registry files (`projects.json`, `devs.json`, etc.) never store secret values — only key names as references.
- Environment variables are never used as an intermediate storage step.

## Encryption Details

- Algorithm: age (https://age-encryption.org), X25519 key exchange
- Key derivation: from a per-server identity generated on first `forge init`
- File permissions: `600` (owner read/write only)
- The master identity key is stored at `~/.forge/identity.age`

## Integrity

On every read, the age library verifies the MAC of the ciphertext. A corrupted secrets file is rejected immediately with an error — partial or silently wrong reads are not possible. If the file is corrupted, the admin must restore from backup and no module will start that depends on the missing secrets.

## Backup

The secrets file must be backed up alongside `~/.forge/identity.age`. Both files are required to decrypt. Backing up one without the other is useless.

```bash
# Backup both required files
cp ~/.forge/secrets.age /your/backup/path/
cp ~/.forge/identity.age /your/backup/path/
```

## Rotation

To rotate a secret, simply overwrite it:

```bash
forge secrets set smeltforge.hemis.DB_URL "postgres://new-connection-string"
```

The old value is destroyed immediately. If the secret was synced across mesh nodes via `--sync`, run the same command with `--sync` again to propagate the new value.
