"""Minimal same-host restore authorization for the BoxWeave Agent MCP relay.

The gateway never reads private-key contents. Operator-owned exact approvals
are required before passing an approved host path to upstream's existing
restore_box_backup(age_identity_file) implementation.
"""
import json
import os
from pathlib import Path
import re
import stat

POLICY = Path("/etc/boxweave/restore-policy.json")
KEY_DIR = Path("/home/boxweave/boxweave/.restore-keys")
IDENTITY = re.compile(r"[a-z][a-z0-9-]{0,62}\Z")
BACKUP = re.compile(r"[A-Za-z0-9][A-Za-z0-9_.-]{0,127}\Z")
KEY = re.compile(r"[A-Za-z0-9][A-Za-z0-9_.-]{0,63}\Z")
MAX_APPROVALS = 16

# Fixed errors, safe for disclosure. Never include host paths or secrets.
DENIED = "RESTORE_APPROVAL_REQUIRED"
UNSAFE = "RESTORE_KEY_OR_POLICY_UNSAFE"
INVALID = "RESTORE_ARGUMENT_INVALID"


def _owned(path, *, owner, group, directory=False, private=False):
    try:
        meta = path.lstat()
    except OSError:
        raise ValueError(UNSAFE) from None
    mode = meta.st_mode
    if (meta.st_uid != owner or (not directory and meta.st_nlink != 1)
            or not (stat.S_ISDIR(mode) if directory else stat.S_ISREG(mode))):
        raise ValueError(UNSAFE)
    if private:
        # root:boxweave private dir 0750, key/policy 0640 or 0440.
        if (meta.st_gid != group or (mode & (0o027 if directory else 0o037))
                or ((mode & 0o050) != 0o050 if directory else not (mode & 0o040))):
            raise ValueError(UNSAFE)
    elif mode & 0o022:
        raise ValueError(UNSAFE)
    return meta


def _valid_tuple(entry):
    return (isinstance(entry, dict) and set(entry) == {"box", "backup", "as", "key_id"}
            and isinstance(entry["box"], str) and bool(IDENTITY.fullmatch(entry["box"]))
            and isinstance(entry["as"], str) and bool(IDENTITY.fullmatch(entry["as"]))
            and isinstance(entry["backup"], str) and bool(BACKUP.fullmatch(entry["backup"]))
            and isinstance(entry["key_id"], str) and bool(KEY.fullmatch(entry["key_id"])))


def read_policy(path=POLICY, *, owner=0, group=None):
    group = os.getegid() if group is None else group
    _owned(path.parent, owner=owner, group=group, directory=True)
    meta = _owned(path, owner=owner, group=group, private=True)
    if meta.st_size > 8192:
        raise ValueError(UNSAFE)
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, ValueError, UnicodeError):
        raise ValueError(UNSAFE) from None
    if (not isinstance(data, dict) or set(data) != {"version", "allowed_restores"}
            or type(data["version"]) is not int or data["version"] != 1
            or not isinstance(data["allowed_restores"], list)
            or len(data["allowed_restores"]) > MAX_APPROVALS
            or any(not _valid_tuple(v) for v in data["allowed_restores"])):
        raise ValueError(UNSAFE)
    found = [tuple(x[k] for k in ("box", "backup", "as", "key_id"))
             for x in data["allowed_restores"]]
    if len(set(found)) != len(found):
        raise ValueError(UNSAFE)
    return frozenset(found)


def approve(args, *, policy_path=POLICY, key_dir=KEY_DIR, owner=0, group=None):
    """Return upstream args only if exact admin authorization and key metadata pass.

    No private-key content is opened by this gateway function.
    """
    if (not isinstance(args, dict)
            or set(args) - {"box", "backup", "as", "key_id", "confirm", "force"}
            or not all(isinstance(args.get(k), str) for k in
                       ("box", "backup", "as", "key_id", "confirm"))
            or not IDENTITY.fullmatch(args["box"])
            or not IDENTITY.fullmatch(args["as"])
            or not BACKUP.fullmatch(args["backup"])
            or not KEY.fullmatch(args["key_id"])
            or args["confirm"] != args["backup"]
            or type(args.get("force", False)) is not bool
            or args.get("force") is True):
        raise ValueError(INVALID)
    group = os.getegid() if group is None else group
    approved = read_policy(policy_path, owner=owner, group=group)
    fields = (args["box"], args["backup"], args["as"], args["key_id"])
    if fields not in approved:
        raise ValueError(DENIED)
    _owned(key_dir, owner=owner, group=group, directory=True, private=True)
    path = key_dir / args["key_id"]
    meta = _owned(path, owner=owner, group=group, private=True)
    if not 1 <= meta.st_size <= 8192 or not os.access(path, os.R_OK):
        raise ValueError(UNSAFE)
    return {"box": args["box"], "backup": args["backup"], "as": args["as"],
            "force": False, "age_identity_file": str(path)}


def catalog_tool(tool):
    """Expose a key alias, never an arbitrary local private-key path."""
    schema = tool.get("inputSchema")
    if not isinstance(schema, dict) or not isinstance(schema.get("properties"), dict):
        # Unsupported upstream schema must not be advertised as safe.
        return None
    if not {"box", "backup", "age_identity_file"}.issubset(schema["properties"]):
        return None
    # Keep only the accepted upstream shape; unrecognized upstream fields
    # must not be advertised as accepted by this policy-gated proxy.
    props = {k: schema["properties"][k] for k in ("box", "backup", "as", "force")
             if k in schema["properties"]}
    props["key_id"] = {"type": "string", "description":
                       "Admin-approved basename in the private host restore-key directory; never a path or key bytes."}
    props["confirm"] = {"type": "string", "description":
                        "Repeat the exact approved backup ID to confirm a one-shot restore."}
    props["as"] = {"type": "string", "description":
                   "Explicit admin-approved new target Box. Required; no implicit target."}
    props["force"] = {"type": "boolean", "description":
                      "Not supported by this guarded Agent route. Use a separately authorized operator workflow."}
    adjusted = dict(tool)
    adjusted["inputSchema"] = dict(schema, properties=props, required=[
        "box", "backup", "as", "key_id", "confirm"
    ], additionalProperties=False)
    adjusted["description"] = (
        "Explicitly authorized same-host whole-Box restore. Requires a root-owned "
        "exact approval and protected local key alias. No arbitrary file paths, "
        "no replacement, no automatic retry after uncertain outcomes.")
    return adjusted

