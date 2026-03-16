# shared/secrets — Secrets Client

The `secrets` package is how every Forge module reads encrypted secrets at runtime. It provides a namespaced client that wraps the age-encrypted secrets file (`~/.forge/secrets.age`) behind a simple `Get` / `Set` / `List` / `Delete` interface so modules never need to know about the encryption format, the file location, or the key derivation scheme.

## The Namespace Convention

Every secret in Forge is stored under a namespaced key in the format `<module>.<scope>.<name>`. Namespacing prevents collisions between modules and makes it easy to list all secrets belonging to a given module or project:

```
smeltforge.hemis.DB_URL           # SmeltForge env var for project hemis
smeltforge.hemis.SECRET_KEY
smeltforge.hemis.webhook_secret   # SmeltForge webhook validation secret
hearthforge.hemis.deploy_key      # HearthForge deploy key for project hemis
sparkforge.gotify.app_token        # SparkForge Gotify API token
sparkforge.smtp.password
fluxforge.controller.wireguard_key # FluxForge controller private key
```

The namespace separator is always `.`. Module names and scope names must be lowercase and contain only alphanumeric characters and dashes. Secret names within a scope can use any casing convention that makes sense for the context (environment variable names are conventionally `UPPER_SNAKE_CASE`; internal secrets like tokens and keys are conventionally `lower_snake_case`).

Modules must only read and write secrets under their own namespace. A module must never read another module's secrets directly — if a module needs data from another module, that data should be exposed through that module's API or through a shared registry file, not by reaching into another module's secret namespace.

## The Client Interface

```go
package secrets

// Client provides namespaced access to the Forge secrets store.
// Obtain a client via New() — do not construct it directly.
type Client struct {
    // namespace is the module's root prefix, e.g. "smeltforge"
    namespace string
}

// New creates a secrets client for the given module namespace.
// The namespace is prepended to all keys automatically, so a call to
// Get("hemis.DB_URL") on a client with namespace "smeltforge" reads
// the full key "smeltforge.hemis.DB_URL".
func New(namespace string) *Client

// Get retrieves and decrypts the value for the given namespaced key.
// Returns ErrNotFound if the key does not exist.
// Returns an error if the secrets file is corrupted or the key is malformed.
func (c *Client) Get(ctx context.Context, key string) (string, error)

// Set encrypts and stores a value under the given namespaced key.
// Creates the key if it does not exist, overwrites it if it does.
// Writing triggers a re-encryption of the entire secrets file —
// this is safe but not instantaneous. Do not call Set in hot paths.
func (c *Client) Set(ctx context.Context, key, value string) error

// Delete removes a key from the secrets store.
// Returns nil if the key does not exist — deletion is idempotent.
func (c *Client) Delete(ctx context.Context, key string) error

// List returns all key names under the client's namespace, optionally
// filtered by an additional prefix. Does not return values.
// Example: client.List(ctx, "hemis.") returns all keys for project hemis.
func (c *Client) List(ctx context.Context, prefix string) ([]string, error)

// MustGet is like Get but panics if the key is not found or if an error
// occurs. Use only in module startup code where a missing secret is
// genuinely unrecoverable (e.g. reading a TLS private key). For
// optional secrets or secrets that may not yet exist, use Get.
func (c *Client) MustGet(ctx context.Context, key string) string
```

## How Modules Use It

Modules create a client once at startup and reuse it throughout their lifetime. The client is safe for concurrent use.

```go
// In SmeltForge's initialization:
secretsClient := secrets.New("smeltforge")

// Reading an environment variable for a project at deploy time:
dbURL, err := secretsClient.Get(ctx, fmt.Sprintf("%s.DB_URL", project.ID))
if errors.Is(err, secrets.ErrNotFound) {
    // This env var was not configured — skip injection
} else if err != nil {
    return fmt.Errorf("reading DB_URL for project %s: %w", project.ID, err)
}

// Storing a generated webhook secret when registering a new project:
webhookSecret := generateCryptoRandom(32)
if err := secretsClient.Set(ctx, fmt.Sprintf("%s.webhook_secret", project.ID), webhookSecret); err != nil {
    return fmt.Errorf("storing webhook secret: %w", err)
}
```

## The Sync Flag and FluxForge

When FluxForge is installed, the admin can pass `--sync` to `forge secrets set`. This replicates the secret to all mesh nodes. The `secrets` package implements sync by calling a FluxForge-specific replication endpoint after writing locally — but this is handled transparently inside `Set` when the sync flag is active. Module code never needs to check whether FluxForge is installed or call any replication API directly.

On nodes receiving a synced secret, the write comes through the standard `Set` path as well. From the module's perspective, synced and local secrets are completely identical — both are read with `Get` and neither requires special handling.

## Security Properties

A few properties of the secrets store that are worth understanding when deciding what to store here versus elsewhere.

**Encryption at rest.** The secrets file is encrypted using age (https://age-encryption.org). The encryption key is derived from the server's host identity and is only available to the process running as the Forge service user. Plaintext secrets are never written to disk at any point — not as a temp file, not in a swap file, not in a core dump.

**No secrets in registry files.** The `registry/` JSON files (`projects.json`, `devs.json`, `monitors.json`) are committed to git in their example form and are readable by anyone with access to the server's `/opt/infra/forge/` directory. They must never contain secrets. If a project configuration requires a secret (e.g. a webhook secret, a deploy key passphrase, an API token), that value lives in `forge secrets` and the registry file stores only a reference key name, not the value itself.

**No secrets in environment variables.** Modules must not read secrets via environment variables and must not write secrets to environment variables as an intermediate step. The secrets client is the only sanctioned path for accessing encrypted secrets.

**Audit logging.** The `forge secrets set` and `forge secrets delete` CLI commands write audit events through `shared/audit` automatically. Module-level calls to `secrets.Set()` do not automatically audit — if the operation is significant enough to audit, the calling module is responsible for writing the event.
