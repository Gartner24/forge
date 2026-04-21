"""Subprocess wrapper for the forge CLI."""

import json
import subprocess
from typing import Any


def run_forge(*args: str) -> dict[str, Any]:
    """Run a forge CLI command with --output json and return parsed output.

    Returns a dict with the parsed JSON on success, or {"error": "..."} on failure.
    """
    cmd = ["forge", *args, "--output", "json"]
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=60)
    except FileNotFoundError:
        return {"error": "forge CLI not found. Ensure forge is installed and on PATH."}
    except subprocess.TimeoutExpired:
        return {"error": f"Command timed out: {' '.join(cmd)}"}

    if result.returncode != 0:
        stderr = result.stderr.strip()
        stdout = result.stdout.strip()
        msg = stderr or stdout or f"forge exited with code {result.returncode}"
        return {"error": msg}

    if not result.stdout.strip():
        return {"ok": True}

    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError:
        return {"error": f"Non-JSON output: {result.stdout.strip()[:500]}"}


def fmt(data: dict[str, Any]) -> str:
    """Format a forge response as a readable string for the agent."""
    if "error" in data:
        return f"Error: {data['error']}"
    return json.dumps(data, indent=2)
