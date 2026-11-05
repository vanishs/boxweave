"""Bounded, structured Platform SSH debugging for an already authorized Box.

The upstream debug_container response may include SSH journal lines, raw shell
paths and remediation strings. This module deliberately refuses to surface
those contents in Web/API responses.
"""
import json
from .domain import Fault
from .wire_results import text


def summary(provider, target):
    """Read Provider debug without guest commands; only return safe fields."""
    if getattr(provider, "mode", "") != "mcp":
        raise Fault("LIVE_DIAGNOSTIC_REQUIRES_MCP", 409)
    # The caller must validate target ownership, incarnation and running state
    # through Service.observe_target *before* invoking this helper.
    if (getattr(provider, "rest", None) is not None
            and getattr(provider, "debug_rest_enabled", True)):
        body = provider.rest.debug_container(target.logical)
        # A logical Box name can be recycled between the authorization
        # read and the debug GET. Reject changed incarnations before any
        # response reaches Web; this remains read-only and no retry occurs.
        current = provider.rest.container(target.logical)
        if (current.get("username") != target.logical
                or current.get("name") != target.instance
                or str(current.get("createdAt")) != target.created):
            raise Fault("TARGET_CHANGED", 409)
    else:
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
