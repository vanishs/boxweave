"""Read-only inventory of the Platform MCP surface seen by the app launcher.

This is capability *evidence*, not authorization, execution, a registry of all
upstream tools or proof that the separately enrolled direct Agent gate sees the
same scopes. Never issue tools/call from this module.
"""
import hashlib
import json
import re
from .domain import Fault

# Tools invoked by the current application adapters or deployment checks.
# Reviewed at the source baseline; this is not a permission/allowlist.
APP_TOOLS = frozenset({
    "get_container", "list_containers", "connect", "create_container",
    "start_container", "stop_container", "delete_container", "resize_container",
    "network_tenant_capabilities", "network_guard_status", "get_network_policy",
    "patch_tenant_access", "list_passthrough_routes", "add_passthrough_route",
    "delete_passthrough_route", "get_system_info",
})

# Advisory only, never used to authorize calls. Keep separate from enforcement.
SENSITIVE_TOOLS = frozenset({
    "get_secret", "set_secret", "delete_secret", "refresh_secrets",
    "revoke_token", "mint_gateway_token", "upgrade_backend",
    "restore_backup", "set_ssh_keys", "terminate_user_sessions",
    "kms_migrate_to_envelope", "security_remediate",
})

TOOL_NAME = re.compile(r"[a-z][a-z0-9_]{0,127}\Z")


def classify(catalogue):
    """Validate the discovered *app-side* catalogue and expose safe metadata."""
    if not isinstance(catalogue, dict) or len(catalogue) > 512:
        raise Fault("MCP_CATALOG_INVALID")
    entries = []
    for name, raw in sorted(catalogue.items()):
        if not isinstance(name, str) or not TOOL_NAME.fullmatch(name):
            raise Fault("MCP_CATALOG_INVALID")
        if not isinstance(raw, dict) or raw.get("name") != name:
            raise Fault("MCP_CATALOG_INVALID")
        schema = raw.get("inputSchema")
        # An unexpected schema is explicitly unsupported, not a callable tool.
        compatible = isinstance(schema, dict) and schema.get("type") == "object"
        digest = (hashlib.sha256(json.dumps(schema, sort_keys=True,
                                             separators=(",", ":"), ensure_ascii=True).encode()).hexdigest()
                  if compatible else None)
        entries.append({
            "name": name,
            "source": "platform",
            "app_mapping": "typed_adapter" if name in APP_TOOLS else "not_integrated",
            "schema": "object_advertised" if compatible else "unverified",
            "schema_sha256": digest,
            "risk_hint": "sensitive_review" if name in SENSITIVE_TOOLS else "unclassified",
            "execution": "not_tested",
        })
    return {
        "source": "platform", "state": "advertised_to_app_launcher",
        "actor_boundary": "application_provider",
        "authorization": "not_established_by_discovery",
        "direct_agent_gateway": "not_checked",
        "count": len(entries), "tools": entries,
    }


def snapshot(provider):
    """Discover names/schema only, through the existing bounded app MCP client."""
    if getattr(provider, "mode", None) != "mcp":
        return {
            "source": "platform", "state": "not_checked",
            "reason": "live_provider_required",
            "actor_boundary": "application_provider",
            "authorization": "not_established_by_discovery",
            "direct_agent_gateway": "not_checked", "count": 0, "tools": [],
        }
    with provider.session() as client:
        catalogue = client.discover()
    return classify(catalogue)
