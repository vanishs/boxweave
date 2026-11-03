"""Read-only, on-demand diagnostic of BoxWeave's smallest upstream MCP contract.

No operation is replayed, no raw schema or arguments leave this module.
The SHA-256 describes a sampled advertised subset, NOT a release guarantee.
"""
import hashlib
import json
import secrets

# Specific fields actually used by the BoxWeave gateway / application.
CORE = {
    "list_containers": {},
    "get_container": {"username": "string"},
    "connect": {"exec": "string"},
    "create_container": {"username": "string", "ssh_keys": "array"},
    "debug_container": {"username": "string"},
}
LIMIT = 256


def read_tools(send, reply):
    """Bounded read-only tools/list, using an already initialized MCP stream."""
    tools, names, cursors = {}, set(), set()
    cursor = None
    for _ in range(8):
        ident = "boxweave-compat-" + secrets.token_hex(8)
        params = {} if cursor is None else {"cursor": cursor}
        send({"jsonrpc": "2.0", "id": ident, "method": "tools/list", "params": params})
        result = reply(ident, False)
        value = result.get("result")
        if not isinstance(value, dict) or not isinstance(value.get("tools"), list):
            raise ValueError("COMPAT_DISCOVERY_INVALID")
        for item in value["tools"]:
            if (not isinstance(item, dict) or not isinstance(item.get("name"), str)
                    or item["name"] in names or len(names) >= LIMIT):
                raise ValueError("COMPAT_DISCOVERY_INVALID")
            name = item["name"]
            names.add(name)
            if name in CORE:
                tools[name] = item
        cursor = value.get("nextCursor")
        if cursor is None:
            return tools
        if not isinstance(cursor, str) or not cursor or cursor in cursors:
            raise ValueError("COMPAT_DISCOVERY_INVALID")
        cursors.add(cursor)
    raise ValueError("COMPAT_DISCOVERY_LIMIT")


def analyze(tools, *, allowed=None):
    """Expose only static known tool names/field names/types and a digest."""
    if not isinstance(tools, dict):
        return unavailable()
    checks, sample, missing_tools, changed = [], [], [], []
    for name, required in CORE.items():
        if allowed is not None and name not in allowed:
            checks.append({"tool": name, "state": "not_checked"})
            continue
        value = tools.get(name)
        if value is None:
            checks.append({"tool": name, "state": "missing"})
            missing_tools.append(name)
            sample.append((name, "missing"))
            continue
        if not isinstance(value, dict):
            return unavailable()
        schema = value.get("inputSchema")
        if not isinstance(schema, dict) or schema.get("type") != "object":
            checks.append({"tool": name, "state": "schema_changed", "fields": ["inputSchema"]})
            changed.append(name)
            sample.append((name, "invalid"))
            continue
        props = schema.get("properties")
        if not isinstance(props, dict):
            props = {} if name == "list_containers" else None
        if props is None:
            checks.append({"tool": name, "state": "schema_changed", "fields": ["properties"]})
            changed.append(name)
            sample.append((name, "invalid"))
            continue
        diff = [key for key, kind in required.items()
                if not isinstance(props.get(key), dict) or props[key].get("type") != kind]
        required_now = schema.get("required", [])
        allowed_required = {
            "list_containers": set(),
            "get_container": {"username"},
            "connect": {"box", "username"},
            "create_container": {"username", "ssh_keys"},
            "debug_container": {"username"},
        }[name]
        if not isinstance(required_now, list) or any(not isinstance(k, str) for k in required_now):
            diff.append("required")
            required_now = []
        if any(k not in allowed_required for k in required_now):
            diff.append("new_required_field")
        if name == "connect" and not any(
                isinstance(props.get(key), dict) and props[key].get("type") == "string"
                for key in ("box", "username")):
            diff.append("box|username")
        if diff:
            checks.append({"tool": name, "state": "schema_changed", "fields": diff})
            changed.append(name)
        else:
            checks.append({"tool": name, "state": "matching_subset"})
        selected = sorted((k, (props[k].get("type") if isinstance(props.get(k), dict) else None))
                          for k in (*required, *(("box", "username") if name == "connect" else ())))
        sample.append((name, selected, sorted(required_now)))
    if changed:
        state, code = "schema_changed", "MCP_COMPAT_SCHEMA_CHANGED"
    elif missing_tools:
        state, code = "tool_missing", "MCP_COMPAT_TOOL_MISSING"
    else:
        state, code = "compatible_subset", "MCP_COMPAT_SUBSET_MATCH"
    if not sample:
        return {"source": "compat", "state": "not_verified", "code": "MCP_COMPAT_NOT_AUTHORIZED",
                "checks": checks, "fingerprint_sha256": None,
                "upstream_version": "not_verified", "coverage": "read_only_subset"}
    signature = hashlib.sha256(json.dumps(sample, sort_keys=True,
                                          separators=(",", ":")).encode()).hexdigest()
    return {"source": "compat", "state": state, "code": code, "checks": checks,
            "fingerprint_sha256": signature, "upstream_version": "not_verified",
            "coverage": "read_only_subset",
            "response_contract": "not_verified",
            "next_action": ("Inspect the exact upstream MCP tool schema before retrying."
                            if state != "compatible_subset"
                            else "Compare gateway event codes; tool output and permissions were not tested.")}


def unavailable():
    return {"source": "compat", "state": "not_verified",
            "code": "UPSTREAM_DISCOVERY_UNAVAILABLE", "checks": [],
            "fingerprint_sha256": None, "upstream_version": "not_verified",
            "coverage": "read_only_subset", "response_contract": "not_verified"}

