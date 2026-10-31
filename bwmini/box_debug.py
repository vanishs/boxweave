"""Bounded, structured Platform SSH debugging for an already authorized Box.

The upstream debug_container response may include SSH journal lines, raw shell
paths and remediation strings. This module deliberately refuses to surface
those contents in Web/API responses.
"""
import json
from .domain import Fault
from .provider import text


def summary(provider, target):
    """Read only from the upstream Platform MCP, never run a guest command."""
    if getattr(provider, "mode", "") != "mcp":
        raise Fault("LIVE_DIAGNOSTIC_REQUIRES_MCP", 409)
    # The caller must validate target ownership, incarnation and running state
    # through Service.observe_target *before* invoking this helper.
    with provider.session() as client:
        tools = client.discover()
        schema = tools.get("debug_container", {}).get("inputSchema", {})
        if (not isinstance(schema, dict) or schema.get("type") != "object"
                or "username" not in schema.get("properties", {})):
            raise Fault("DEBUG_CONTRACT_UNSUPPORTED", 409)
        result = client.tool("debug_container", {"username": target.logical})
    raw = text(result)
    if len(raw) > 32768:
        raise Fault("DEBUG_CONTRACT_UNSUPPORTED", 409)
    try:
        body = json.loads(raw)
    except (ValueError, TypeError):
        raise Fault("DEBUG_CONTRACT_UNSUPPORTED", 409) from None
    if not isinstance(body, dict):
        raise Fault("DEBUG_CONTRACT_UNSUPPORTED", 409)

    # Upstream response is camelCase. Unknown/free-form cause strings are
    # deliberately not returned to Web, even when tool execution succeeds.
    state = body.get("containerState")
    if state not in ("running", "stopped", "missing"):
        state = "unverified"
    observed = {}
    for source, dest in (("hostUserExists", "host_user_exists"),
                         ("hostUserShellExists", "host_user_shell_exists")):
        value = body.get(source)
        if source in body and type(value) is not bool:
            raise Fault("DEBUG_CONTRACT_UNSUPPORTED", 409)
        observed[dest] = value if type(value) is bool else None

    return {
        "source": "platform_debug_container",
        "state": "observed",
        "container_state": state,
        "target": target.logical,
        "host_user_exists": observed["host_user_exists"],
        "host_user_shell_exists": observed["host_user_shell_exists"],
        "sensitive_details": "withheld",
        "evidence": "live_upstream_response",
        "boxweave_host_ssh_verified": False,
    }
