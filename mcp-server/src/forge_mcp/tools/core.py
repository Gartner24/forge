"""Core module tools: module management and secrets."""

from fastmcp import FastMCP

from forge_mcp.tools.utils import fmt, run_forge


def register_tools(mcp: FastMCP) -> None:
    """Register Forge Core tools."""

    @mcp.tool
    def forge_status() -> str:
        """Return the name, version, and status (running/stopped/error) of every
        installed Forge module. Call this first to see what modules are available
        before issuing module-specific commands."""
        return fmt(run_forge("status"))

    @mcp.tool
    def forge_install(module: str) -> str:
        """Install a Forge module on this server.

        module: one of smeltforge, fluxforge, watchforge, sparkforge,
                hearthforge, penforge.

        Installs the module binary and registers it with Forge Core. The module
        starts automatically after installation."""
        return fmt(run_forge("install", module))

    @mcp.tool
    def forge_uninstall(module: str) -> str:
        """Uninstall a Forge module. Prompts for confirmation unless --yes is passed.

        module: the module name to remove (e.g. watchforge)."""
        return fmt(run_forge("uninstall", module))

    @mcp.tool
    def forge_update(module: str) -> str:
        """Update a Forge module to the latest version.

        module: the module name to update."""
        return fmt(run_forge("update", module))

    @mcp.tool
    def forge_secrets_set(key: str, value: str) -> str:
        """Store an encrypted secret. The value is age-encrypted before writing
        to disk — never stored as plaintext.

        key: dot-namespaced key, e.g. smeltforge.myapp.DATABASE_URL
        value: the secret value to encrypt and store."""
        return fmt(run_forge("secrets", "set", key, value))

    @mcp.tool
    def forge_secrets_get(key: str) -> str:
        """Retrieve and decrypt a secret by key.

        key: the secret key to retrieve, e.g. smeltforge.myapp.DATABASE_URL."""
        return fmt(run_forge("secrets", "get", key))

    @mcp.tool
    def forge_secrets_list() -> str:
        """List all stored secret key names. Values are never displayed — only
        key names are returned. Use forge_secrets_get to retrieve a value."""
        return fmt(run_forge("secrets", "list"))

    @mcp.tool
    def forge_secrets_delete(key: str) -> str:
        """Delete a secret by key.

        key: the secret key to delete."""
        return fmt(run_forge("secrets", "delete", key))
