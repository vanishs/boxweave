"""BoxWeave reacceptance: test-owned, least-privilege executable evidence engine.

Source/runtime catalog and evidence validation can run as a normal user.  The
engine NEVER creates or deletes provider objects and NEVER executes arbitrary
commands from a supplied test plan.  Live mutation and exact Box ownership stay
with the established BoxWeave MCP/test-box controller.
"""
from __future__ import annotations

import argparse
import ast
import datetime as dt
import fnmatch
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import shlex
import subprocess
import sys
import tempfile
import time
from urllib.error import HTTPError, URLError
from urllib.request import build_opener, ProxyHandler

SCHEMA = 1
MAX_RECORD = 4194304
LIMIT = 65536
STATUSES = frozenset(("PASS", "FAIL", "PARTIAL", "BLOCKED", "SKIPPED", "UNKNOWN"))
MUTATING = ("create", "delete", "stop", "start", "resize", "enable", "disable",
            "update", "confirm", "restore", "run", "execute", "submit", "revoke",
            "set_", "grant", "rotate", "reboot", "upgrade", "refresh", "patch")
SAFE_MCP_TOOLS = {
    "boxweave_info", "list_containers", "get_container", "get_metrics",
    "list_backends", "list_routers", "list_routes",
}
SAFE_HOST_ACTIONS = {"version", "status", "doctor", "provenance", "setup", "box"}
SECRET = re.compile(r"(?i)(authorization:\s*bearer\s+|access_token[=:]\s*|"
                    r"(?:password|private_key|client_secret)[=:]\s*)(\S+)")


class AcceptanceFault(ValueError):
    pass


def now():
    return dt.datetime.now(dt.timezone.utc).isoformat(timespec="milliseconds")


def canonical(value):
    return json.dumps(value, sort_keys=True, ensure_ascii=True,
                      separators=(",", ":")).encode("utf-8")


def digest(value):
    return hashlib.sha256(value if isinstance(value, bytes) else canonical(value)).hexdigest()


def private_mkdir(path):
    path = Path(path)
    if path.is_symlink():
        raise AcceptanceFault("UNSAFE_OUTPUT_DIRECTORY")
    path.mkdir(parents=True, exist_ok=True, mode=0o700)
    if path.stat().st_mode & 0o077:
        raise AcceptanceFault("OUTPUT_DIRECTORY_NOT_PRIVATE")
    return path


def unique_file(path, data):
    p = Path(path)
    if p.is_symlink():
        raise AcceptanceFault("OUTPUT_SYMLINK_REFUSED")
    raw = data if isinstance(data, bytes) else canonical(data) + b"\n"
    if len(raw) > MAX_RECORD:
        raise AcceptanceFault("EVIDENCE_TOO_LARGE")
    fd = os.open(p, os.O_WRONLY | os.O_CREAT | os.O_EXCL | getattr(os, "O_NOFOLLOW", 0), 0o600)
    try:
        with os.fdopen(fd, "wb") as out:
            out.write(raw)
            out.flush()
            os.fsync(out.fileno())
    except BaseException:
        raise
    return digest(raw)


def load(path):
    path = Path(path)
    if path.is_symlink() or path.stat().st_size > MAX_RECORD:
        raise AcceptanceFault("UNSAFE_EVIDENCE_INPUT")
    return json.loads(path.read_text(encoding="utf-8"))


def sanitize(data):
    if isinstance(data, str):
        return SECRET.sub(lambda m: m.group(1) + "[REDACTED]", data)
    if isinstance(data, dict):
        return {key: ("[REDACTED]" if any(tag in key.lower()
                 for tag in ("token", "secret", "password", "private_key")) else sanitize(value))
                for key, value in data.items()}
    if isinstance(data, list):
        return [sanitize(value) for value in data]
    return data


def capture(argv, *, timeout=8, cwd=None, input_text=None, limit=LIMIT):
    """No shell, bounded output. Timeout returns UNKNOWN, never a retry."""
    if not isinstance(argv, list) or not argv or any(
            not isinstance(x, str) or not x or len(x) > 4096 for x in argv):
        raise AcceptanceFault("INVALID_EXEC_VECTOR")
    start = time.monotonic()
    try:
        p = subprocess.run(argv, cwd=cwd, input=input_text, text=True,
                           stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                           timeout=min(max(timeout, 1), 120), check=False)
        state, rc = "completed", p.returncode
        output, errors = p.stdout, p.stderr
    except subprocess.TimeoutExpired as exc:
        state, rc = "UNKNOWN", None
        output = exc.stdout or b""
        errors = exc.stderr or b""
    except OSError as exc:
        state, rc, output, errors = "unavailable", None, "", type(exc).__name__
    if isinstance(output, bytes):
        output = output.decode("utf-8", "replace")
    if isinstance(errors, bytes):
        errors = errors.decode("utf-8", "replace")
    return {"argv": argv, "execution": state, "exit_code": rc,
            "stdout_sha256": digest(output.encode()), "stderr_sha256": digest(errors.encode()),
            "stdout": sanitize(output[:limit]), "stderr": sanitize(errors[:limit]),
            "truncated": len(output) > limit or len(errors) > limit,
            "elapsed_ms": round((time.monotonic() - start) * 1000)}


def ast_app_choices(source):
    tree = ast.parse(Path(source).read_text())
    for node in ast.walk(tree):
        if (isinstance(node, ast.Call) and isinstance(node.func, ast.Attribute)
                and node.func.attr == "add_argument" and node.args
                and isinstance(node.args[0], ast.Constant) and node.args[0].value == "action"):
            for kw in node.keywords:
                if kw.arg == "choices":
                    return sorted(set(ast.literal_eval(kw.value)))
    raise AcceptanceFault("APP_ACTIONS_UNAVAILABLE")


def ast_host_actions(source):
    tree = ast.parse(Path(source).read_text())
    names = set()
    for node in ast.walk(tree):
        if isinstance(node, ast.Call) and isinstance(node.func, ast.Attribute) and node.func.attr == "add_parser":
            if node.args and isinstance(node.args[0], ast.Constant) and isinstance(node.args[0].value, str):
                if (isinstance(node.func.value, ast.Name)
                        and node.func.value.id == "sub"):
                    names.add(node.args[0].value)
    # Include explicitly intercepted early dispatch even if omitted from argparse.
    for node in ast.walk(tree):
        if isinstance(node, ast.Compare) and len(node.ops) == 1 and isinstance(node.ops[0], ast.Eq):
            for operand in (node.left, *node.comparators):
                if isinstance(operand, ast.Constant) and isinstance(operand.value, str):
                    if operand.value in ("selftest", "upstream", "job", "http", "task", "client"):
                        names.add(operand.value)
    return sorted(names)


def ast_host_leaves(source):
    tree = ast.parse(Path(source).read_text())
    result = set()
    for node in ast.walk(tree):
        if not (isinstance(node, ast.Call) and isinstance(node.func, ast.Attribute)
                and node.func.attr == "add_parser" and node.args
                and isinstance(node.func.value, ast.Name)
                and node.func.value.id == "box_sub"):
            continue
        arg = node.args[0]
        if isinstance(arg, ast.Constant) and isinstance(arg.value, str):
            result.add("box " + arg.value)
        elif isinstance(arg, ast.Name):
            for loop in ast.walk(tree):
                if (isinstance(loop, ast.For) and isinstance(loop.target, ast.Name)
                        and loop.target.id == arg.id
                        and node in list(ast.walk(loop))):
                    try:
                        result.update("box " + x for x in ast.literal_eval(loop.iter))
                    except (ValueError, TypeError):
                        pass
    return sorted(result)


def feature_catalogue(root):
    path = Path(root) / "docs/verification/REACCEPTANCE_FUNCTIONAL_COVERAGE_BASELINE.md"
    if not path.exists():
        path = Path(root) / "bwmini/reacceptance_data/REACCEPTANCE_FUNCTIONAL_COVERAGE_BASELINE.md"
    if not path.is_file():
        raise AcceptanceFault("FEATURE_BASELINE_NOT_BUNDLED_OR_AVAILABLE")
    features = {}
    for line in path.read_text().splitlines():
        match = re.match(r"^\|\s*((?:F|X)\d{2})\s*\|\s*\*\*([^*]+)\*\*", line)
        if match:
            features[match.group(1)] = match.group(2)
    if len(features) != 49 or not all(k in features for k in ("F01", "F39", "X01", "X10")):
        raise AcceptanceFault("FEATURE_BASELINE_INCOMPLETE_OR_DRIFTED")
    return features


def source_web_routes(source):
    tree = ast.parse(Path(source).read_text())
    found = set()
    for node in ast.walk(tree):
        if not isinstance(node, ast.Call) or not isinstance(node.func, ast.Attribute):
            continue
        if node.func.attr not in ("get", "post", "put", "delete", "route"):
            continue
        if not node.args or not isinstance(node.args[0], ast.Constant) or not isinstance(node.args[0].value, str):
            continue
        if node.func.attr == "route":
            methods = ["UNKNOWN"]
            for kw in node.keywords:
                if kw.arg == "methods":
                    try:
                        methods = ast.literal_eval(kw.value)
                    except (ValueError, TypeError):
                        pass
        else:
            methods = [node.func.attr.upper()]
        for m in methods:
            found.add(str(m).upper() + " " + node.args[0].value)
    return sorted(found)


def parse_proto_routes(root):
    """Source declaration index; never claims the HTTP server serves the route."""
    root = Path(root)
    result = []
    if not root.is_dir():
        return result
    for p in sorted(root.rglob("*.proto")):
        text = p.read_text(errors="replace")
        # Method braces contain annotations; stop before next RPC.
        for match in re.finditer(r"\brpc\s+([A-Za-z][A-Za-z0-9_]*)\s*\([^)]*\)\s*returns\s*\([^)]*\)\s*\{(.*?)\n\s*\}", text, re.S):
            rpc, body = match.group(1), match.group(2)
            for item in re.finditer(r"\b(get|post|put|delete|patch):\s*\"([^\"]+)\"", body):
                result.append({"rpc": rpc, "method": item.group(1).upper(),
                               "path": item.group(2), "file": p.relative_to(root).as_posix()})
    return result


def repo_source(root):
    root = Path(root)
    for x in (root / "bwmini/host_cli.py", root / "bwmini/__main__.py"):
        if not x.exists():
            raise AcceptanceFault("BOXWEAVE_SOURCE_MISSING")
    source = {"host_cli": ast_host_actions(root / "bwmini/host_cli.py"),
              "host_cli_leaves": ast_host_leaves(root / "bwmini/host_cli.py"),
              "app_cli": ast_app_choices(root / "bwmini/__main__.py"),
              "web_api": source_web_routes(root / "bwmini/web.py"),
              "remote_api": source_web_routes(root / "bwmini/remote_mcp.py")}
    try:
        s = subprocess.run(["git", "-C", str(root), "rev-parse", "HEAD"],
                           capture_output=True, text=True, timeout=5)
        sha = s.stdout.strip() if s.returncode == 0 else "unknown"
    except OSError:
        sha = "unknown"
    # Distribution ZIP includes a signed-source-bound package manifest but no .git.
    if not re.fullmatch(r"[0-9a-f]{40}", sha):
        manifest = root / "manifest.json"
        if manifest.is_file():
            try:
                packaged = load(manifest)
                declared = packaged.get("source", {})
                if (isinstance(declared, dict) and
                        re.fullmatch(r"[0-9a-f]{40}", str(declared.get("commit", "")))):
                    sha = declared["commit"]
            except (OSError, ValueError, TypeError):
                pass
    return sha, source



def source_worktree_state(root):
    """Do not present dirty source code as exact Git HEAD or package bytes."""
    root = Path(root)
    manifest_path = root / "manifest.json"
    if manifest_path.is_file():
        try:
            manifest = load(manifest_path)
            files = manifest.get("files")
            version = manifest.get("source", {}).get("state")
            critical = ("bwmini/reacceptance.py", "bwmini/host_cli.py",
                        "bwmini/reacceptance_data/REACCEPTANCE_DEPENDENCIES.json",
                        "bwmini/reacceptance_data/REACCEPTANCE_FUNCTIONAL_COVERAGE_BASELINE.md")
            if (version == "clean" and isinstance(files, dict) and
                    all(name in files and (root / name).is_file()
                        and digest((root / name).read_bytes()) == files[name]
                        for name in critical)):
                return "clean"
            return "modified_or_unverified_package"
        except (OSError, KeyError, ValueError, TypeError):
            return "modified_or_unverified_package"
    try:
        result = subprocess.run(
            ["git", "-C", str(root), "status", "--porcelain=v1",
             "--untracked-files=normal"],
            capture_output=True, text=True, timeout=5)
        if result.returncode != 0:
            return "unverified"
        return "clean" if not result.stdout.strip() else "modified"
    except OSError:
        return "unverified"



def load_dependencies(root):
    path = Path(root) / "docs/verification/REACCEPTANCE_DEPENDENCIES.json"
    if not path.exists():
        path = Path(root) / "bwmini/reacceptance_data/REACCEPTANCE_DEPENDENCIES.json"
    if not path.is_file():
        raise AcceptanceFault("REACCEPTANCE_DEPENDENCIES_NOT_BUNDLED_OR_AVAILABLE")
    x = load(path)
    if x.get("schema_version") != 1 or not isinstance(x.get("rules"), list) or not x["rules"]:
        raise AcceptanceFault("REACCEPTANCE_DEPENDENCIES_INVALID")
    return x


def features_for(action, rules):
    tags = set()
    for rule in rules:
        if fnmatch.fnmatchcase(action, rule.get("action", "")):
            tags.update(rule.get("features", []))
    return sorted(tags) or ["UNMAPPED"]


def impacted(paths, rules):
    paths = list(paths)
    return sorted({f for rule in rules for f in rule.get("features", [])
                   if any(fnmatch.fnmatchcase(path, glob)
                          for glob in rule.get("source_globs", []) for path in paths)})


def mcp_session(argv, calls, timeout=12):
    """Session uses actual JSON-RPC stdio, not imported Python tool declarations."""
    if not isinstance(argv, list) or not argv or any(not isinstance(x, str) for x in argv):
        raise AcceptanceFault("BAD_MCP_LAUNCH")
    if any(any(s in x for s in ("\n", "\r", "\0")) for x in argv):
        raise AcceptanceFault("BAD_MCP_LAUNCH")
    # Each request has its own ID, but one upstream stdio process and protocol state.
    requests = [
        {"jsonrpc": "2.0", "id": "racc-init", "method": "initialize",
         "params": {"protocolVersion": "2025-06-18", "capabilities": {},
                    "clientInfo": {"name": "boxweave-reacceptance", "version": "1"}}},
        {"jsonrpc": "2.0", "method": "notifications/initialized"},
        *calls,
    ]
    record = capture(argv, input_text="\n".join(json.dumps(x) for x in requests) + "\n",
                     timeout=timeout, limit=2097152)
    messages = []
    for line in record["stdout"].splitlines():
        try:
            obj = json.loads(line)
            if isinstance(obj, dict) and ("result" in obj or "error" in obj):
                messages.append(obj)
        except ValueError:
            continue
    return {"process": record, "messages": messages}


def catalogue(root, *, host_command=None, mcp_command=None, upstream=None,
              previous=None, changed=None):
    root = Path(root).resolve()
    source_sha, source = repo_source(root)
    runtime = {}
    if host_command is not None:
        if not isinstance(host_command, list) or not host_command:
            raise AcceptanceFault("HOST_COMMAND_INVALID")
        provenance = capture(host_command + ["provenance"])
        if provenance["exit_code"] == 0:
            try:
                installed = json.loads(provenance["stdout"])
                runtime["host_provenance"] = {
                    "version": installed.get("version"), "source": installed.get("source")}
                runtime["host_source_matches_checkout"] = installed.get("source") == source_sha
            except ValueError:
                runtime["host_provenance"] = {"state": "malformed"}
        else:
            runtime["host_provenance"] = {"state": "unavailable"}
        h = capture(host_command + ["--help"])
        runtime["host_cli_help"] = h
        if h["exit_code"] == 0:
            line = next((line for line in h["stdout"].splitlines()
                         if "{" in line and "}" in line), "")
            matched = re.search(r"\{([^}]+)\}", line)
            displayed = set(matched.group(1).split(",")) if matched else set()
            runtime["host_advertised"] = sorted(displayed)
            runtime["host_hidden_source"] = sorted(set(source["host_cli"]) - displayed)
        else:
            runtime["host_advertised"] = []
            runtime["host_hidden_source"] = sorted(source["host_cli"])
    if mcp_command is not None:
        result = mcp_session(mcp_command, [
            {"jsonrpc": "2.0", "id": "racc-tools", "method": "tools/list"}])
        runtime["stdio_mcp"] = result
        for message in result["messages"]:
            if message.get("id") == "racc-tools" and isinstance(message.get("result"), dict):
                runtime["mcp_tools"] = sorted(
                    t["name"] for t in message["result"].get("tools", [])
                    if isinstance(t, dict) and isinstance(t.get("name"), str))
                runtime["mcp_schemas"] = {t["name"]: t.get("inputSchema")
                                          for t in message["result"].get("tools", [])
                                          if isinstance(t, dict) and isinstance(t.get("name"), str)}
                break
    upstream_info = {"state": "not_inspected", "declared_proto_routes": []}
    if upstream:
        u = Path(upstream)
        upstream_info = {"state": "source_only", "reference": str(u),
                         "declared_proto_routes": parse_proto_routes(u / "proto/containarium/v1")}
        v = subprocess.run(["git", "-C", str(u), "rev-parse", "HEAD"],
                           text=True, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
                           timeout=5)
        upstream_info["sha"] = v.stdout.strip() if v.returncode == 0 else "unknown"
    rules = load_dependencies(root).get("rules", [])
    actual = {
        "host_cli": source["host_cli"], "host_cli_leaves": source["host_cli_leaves"],
        "app_cli": source["app_cli"],
        "web_api": source["web_api"], "remote_api": source["remote_api"],
        "stdio_mcp": runtime.get("mcp_tools", []),
    }
    actions = []
    for interface, names in actual.items():
        for name in names:
            key = interface + ":" + name
            actions.append({"surface": interface, "name": name, "key": key,
                            "features": features_for(key, rules),
                            "class": "read_only" if is_read_only(interface, name) else "potential_write",
                            "status": "NOT_RUN"})
    delta = {}
    if previous:
        prior = load(previous)
        past = {x["key"] for x in prior.get("actions", [])}
        current = {x["key"] for x in actions}
        delta = {"added": sorted(current - past), "removed": sorted(past - current)}
    return {"schema_version": SCHEMA, "created_utc": now(), "source_sha": source_sha,
            "source_state": source_worktree_state(root),
            "source": source, "runtime": runtime, "upstream": upstream_info,
            "feature_catalogue": feature_catalogue(root),
            "impacted_features": impacted(changed or [], rules), "delta": delta,
            "actions": actions, "coverage_scope": "catalogue_only_NO_PASS",
            "missing_runtime_contexts": [s for s in ("stdio_mcp", "host_cli_help",
                "authenticated_http_mcp", "authenticated_web", "external_client")
                if s not in runtime]}


def is_read_only(surface, name):
    if surface == "stdio_mcp":
        return name in SAFE_MCP_TOOLS
    if surface == "host_cli":
        return name in SAFE_HOST_ACTIONS
    if surface == "host_cli_leaves":
        return name in ("box inspect", "box logs")
    if surface in ("web_api", "remote_api"):
        return name.split(" ", 1)[0] == "GET"
    if surface == "app_cli":
        return name in ("capabilities", "projects", "receipts", "history", "environments",
                        "environment-receipts", "environment-receipt", "job-receipts",
                        "diagnose", "access-memberships", "access-receipts", "access-receipt")
    return False



def new_run(directory, inventory):
    directory = Path(directory)
    if directory.exists():
        raise AcceptanceFault("RUN_EXISTS_REFUSE_OVERWRITE")
    private_mkdir(directory)
    private_mkdir(directory / "cases")
    unique_file(directory / "inventory.json", inventory)
    unique_file(directory / "run.json", {
        "schema_version": SCHEMA, "started_utc": now(),
        "run_id": "racc-" + secrets.token_hex(8),
        "source_sha": inventory["source_sha"],
        "source_state": inventory.get("source_state", "unverified"),
        "upstream_sha": inventory["upstream"].get("sha", "unverified"),
        "role": "single_agent_until_separately_verified",
        "authorization": "read_only_default",
        "scope": inventory["missing_runtime_contexts"],
    })
    return directory


def record_attempt(directory, action, category, result, *,
                   observer=None, actor="same_agent", target=None, expected=None):
    """Write immutable original attempt, NEVER mark a PASS here."""
    directory = Path(directory)
    manifest = load(directory / "inventory.json")
    actions = {x["key"]: x for x in manifest["actions"]}
    if action not in actions:
        raise AcceptanceFault("ACTION_NOT_IN_PINNED_INVENTORY")
    if category not in ("positive", "invalid", "denial"):
        raise AcceptanceFault("INVALID_CASE_CATEGORY")
    if not isinstance(result, dict) or not isinstance(result.get("execution"), str):
        raise AcceptanceFault("MISSING_ORIGINAL_EXECUTION")
    if result.get("execution") == "completed":
        if type(result.get("exit_code")) is not int:
            raise AcceptanceFault("MISSING_ORIGINAL_EXIT")
        if not ("stdout" in result or "response" in result):
            raise AcceptanceFault("MISSING_ORIGINAL_RESPONSE")
    if actor not in ("same_agent", "finder", "fixer", "independent_runner", "verifier"):
        raise AcceptanceFault("INVALID_ACTOR")
    if observer is not None:
        if not isinstance(observer, dict) or not observer.get("source"):
            raise AcceptanceFault("INVALID_ORACLE")
    out = {"schema_version": SCHEMA, "action": action, "category": category,
           "started_utc": now(), "actor": actor, "target": target,
           "expected": expected, "observed": sanitize(result),
           "oracle": sanitize(observer), "asserted_pass": False}
    if "stdout" in result:
        out["raw_stdout_sha256"] = digest(str(result["stdout"]).encode())
    if "response" in result:
        out["raw_response_sha256"] = digest(canonical(result["response"]))
    name = "case-" + secrets.token_hex(12) + ".json"
    unique_file(directory / "cases" / name, out)
    return name


def safe_capture(directory, host_command=None, mcp_command=None):
    """Default safe lane: a few real read-only installed actions, never a write."""
    directory = Path(directory)
    inventory = load(directory / "inventory.json")
    existing = {x["key"] for x in inventory["actions"]}
    outputs = []
    if host_command and "host_cli:version" in existing:
        result = capture(host_command + ["version"])
        outputs.append(record_attempt(directory, "host_cli:version", "positive", result,
                                      observer={"source": "installed_version_stdout",
                                                "value": result.get("stdout", "").strip()},
                                      expected={"exit_code": 0, "contains": "BoxWeave"}))
    if host_command and "host_cli:doctor" in existing:
        result = capture(host_command + ["doctor", "--mcp"], timeout=25)
        category = "positive" if result["exit_code"] == 0 else "denial"
        outputs.append(record_attempt(directory, "host_cli:doctor", category, result,
                                      observer={"source": "host_doctor_response",
                                                "independent": False,
                                                "note": "not independent Provider proof"},
                                      expected={"exit_code": result["exit_code"]}))
    if mcp_command:
        for tool, args in (("boxweave_info", {"topic": "overview"}),
                           ("list_containers", {})):
            key = "stdio_mcp:" + tool
            if key not in existing:
                continue
            result = mcp_session(mcp_command, [{
                "jsonrpc": "2.0", "id": "case-call",
                "method": "tools/call",
                "params": {"name": tool, "arguments": args}
            }], timeout=20)
            response = next((msg for msg in result["messages"]
                             if msg.get("id") == "case-call"), None)
            obs = {"execution": result["process"]["execution"],
                   "exit_code": result["process"]["exit_code"],
                   "response": response, "stderr": result["process"]["stderr"],
                   "stdout": result["process"]["stdout"][:LIMIT]}
            outputs.append(record_attempt(directory, key, "positive", obs,
                                          observer={"source": "actual_stdio_protocol",
                                                    "independent": False,
                                                    "note": "read-only response only"},
                                          expected={"rpc_is_error": False}))
    return outputs


def validate_attempt(case, known_actions):
    if case.get("schema_version") != SCHEMA or case.get("action") not in known_actions:
        return "INVALID_CASE_OR_UNKNOWN_ACTION"
    if case.get("category") not in ("positive", "invalid", "denial"):
        return "INVALID_CATEGORY"
    data = case.get("observed")
    if not isinstance(data, dict):
        return "NO_ORIGINAL_DATA"
    if data.get("execution") != "completed":
        return ("UNKNOWN_OR_UNAVAILABLE" if data.get("execution") in
                ("UNKNOWN", "unavailable") else "INVALID_EXECUTION")
    if type(data.get("exit_code")) is not int or not ("stdout" in data or "response" in data):
        return "ORIGINAL_EXIT_OR_OUTPUT_MISSING"
    if "stdout" in data and digest(str(data["stdout"]).encode()) != case.get("raw_stdout_sha256"):
        return "STDOUT_DIGEST_MISMATCH"
    if "response" in data and digest(canonical(data["response"])) != case.get("raw_response_sha256"):
        return "RESPONSE_DIGEST_MISMATCH"
    if case.get("asserted_pass"):
        return "UNTRUSTED_ASSERTED_PASS"
    expectation = case.get("expected")
    if not isinstance(expectation, dict):
        return "EXPECTED_POSTCONDITION_MISSING"
    if "exit_code" in expectation and data["exit_code"] != expectation["exit_code"]:
        return "EXIT_CODE_MISMATCH"
    if "rpc_is_error" in expectation:
        reply = data.get("response") or {}
        if not isinstance(reply, dict) or not ("result" in reply or "error" in reply):
            return "RPC_RESPONSE_MISSING"
        actual = bool(reply.get("error") or
                      (isinstance(reply.get("result"), dict) and
                       reply["result"].get("isError")))
        if actual != expectation["rpc_is_error"]:
            return "RPC_SEMANTIC_MISMATCH"
    if "contains" in expectation:
        if not isinstance(expectation["contains"], str) or not expectation["contains"]:
            return "UNSAFE_EXPECTED_SUBSTRING"
        if expectation["contains"] not in data.get("stdout", ""):
            return "EXPECTED_OUTPUT_MISSING"
    if "value_equals" in expectation and data.get("response") != expectation["value_equals"]:
        return "EXPECTED_RESPONSE_MISMATCH"
    oracle = case.get("oracle")
    if not isinstance(oracle, dict) or not isinstance(oracle.get("source"), str):
        return "NO_ORACLE"
    return None


def classify_action(row, cases):
    if not cases:
        return {"status": "BLOCKED", "reason": "NOT_INVOKED", "attempts": 0}
    faults = [c["failure"] for c in cases if c.get("failure")]
    if any(c["case"].get("observed", {}).get("execution") == "UNKNOWN" for c in cases):
        return {"status": "UNKNOWN", "reason": "DISPATCH_UNCERTAIN_NO_RETRY",
                "attempts": len(cases)}
    if faults:
        return {"status": "FAIL", "reason": sorted(set(faults))[0],
                "attempts": len(cases)}
    dimensions = {c["case"]["category"] for c in cases}
    if "positive" not in dimensions:
        return {"status": "PARTIAL", "reason": "NO_FUNCTIONAL_POSITIVE",
                "attempts": len(cases)}
    missing = {"invalid", "denial"} - dimensions
    if missing:
        return {"status": "PARTIAL", "reason": "MISSING_BOUNDARIES_" + "_".join(sorted(missing)),
                "attempts": len(cases)}
    if row["class"] == "read_only":
        # An exit-zero flag with no semantic assertion is not a functional
        # acceptance, even if a source or catalog lists the action.
        positives = [c["case"] for c in cases if c["case"]["category"] == "positive"]
        if any(not any(k in c.get("expected", {}) for k in (
                "contains", "rpc_is_error", "value_equals")) for c in positives):
            return {"status": "PARTIAL", "reason": "POSITIVE_SEMANTIC_ORACLE_MISSING",
                    "attempts": len(cases)}
    if row["class"] != "read_only":
        positives = [c["case"] for c in cases if c["case"]["category"] == "positive"]
        denied = [c["case"] for c in cases if c["case"]["category"] == "denial"]
        if any(not isinstance(c.get("target"), dict) or
               not all(c["target"].get(k) for k in ("logical", "provider_uuid", "created_at"))
               for c in positives + denied):
            return {"status": "PARTIAL", "reason": "TARGET_IMMUTABLE_IDENTITY_MISSING",
                    "attempts": len(cases)}
        if any(not c["oracle"].get("independent") or
               "effects_before" not in c["oracle"] or
               "effects_after" not in c["oracle"]
               for c in positives + denied):
            return {"status": "PARTIAL", "reason": "NO_INDEPENDENT_EFFECT_ORACLE",
                    "attempts": len(cases)}
        if any(c["oracle"]["effects_after"] != c["oracle"]["effects_before"]
               for c in denied):
            return {"status": "FAIL", "reason": "DENIAL_CAUSED_FORBIDDEN_EFFECT",
                    "attempts": len(cases)}
        if any(not c["oracle"].get("cleanup_verified") for c in positives):
            return {"status": "PARTIAL", "reason": "CLEANUP_UNVERIFIED",
                    "attempts": len(cases)}
    return {"status": "PASS", "reason": "COMPLETE_CASE_TRIAD_AND_ORACLE",
            "attempts": len(cases)}


def verify_run(directory):
    directory = Path(directory)
    inv = load(directory / "inventory.json")
    run = load(directory / "run.json")
    if inv.get("schema_version") != SCHEMA or run.get("schema_version") != SCHEMA:
        raise AcceptanceFault("UNKNOWN_EVIDENCE_SCHEMA")
    if (inv.get("source_sha") != run.get("source_sha")
            or inv.get("source_state", "unverified") != run.get("source_state", "unverified")):
        raise AcceptanceFault("EVIDENCE_SOURCE_CHANGED")
    actions = {r["key"]: r for r in inv["actions"]}
    case_refs = {}
    findings = []
    for path in sorted((directory / "cases").glob("case-*.json")):
        case = load(path)
        error = validate_attempt(case, actions)
        row = {"file": path.name, "case": case}
        if error:
            row["failure"] = error
            findings.append({"action": case.get("action"), "case": path.name,
                             "kind": "HARNESS_OR_CONTRACT", "code": error})
        case_refs.setdefault(case.get("action"), []).append(row)
    verdicts = []
    for row in inv["actions"]:
        state = classify_action(row, case_refs.get(row["key"], []))
        verdicts.append({**row, **state})
    counts = {status: sum(x["status"] == status for x in verdicts)
              for status in ("PASS", "FAIL", "PARTIAL", "BLOCKED", "UNKNOWN")}
    if not verdicts:
        counts["EMPTY_CATALOG"] = 1
    return {"schema_version": SCHEMA, "run_id": run["run_id"],
            "source_sha": run["source_sha"],
            "source_state": run.get("source_state", "unverified"), "counts": counts,
            "verdicts": verdicts, "findings": findings,
            "verification_mode": "same_agent_until_separately_verified",
            "machine_proven": "source/runtime per-action evidence only; not full system acceptance"}


def negative_controls():
    """Seeded impossible PASS and missing-claim controls must be rejected."""
    row = {"key": "host_cli:version", "class": "read_only"}
    case = {"schema_version": 1, "action": "host_cli:version",
            "category": "positive", "asserted_pass": False,
            "observed": {"execution": "completed", "exit_code": 0, "stdout": "real"},
            "raw_stdout_sha256": digest(b"real"),
            "expected": {"exit_code": 0},
            "oracle": {"source": "actual_version"}}
    bad = dict(case)
    bad["asserted_pass"] = True
    if validate_attempt(bad, {row["key"]: row}) != "UNTRUSTED_ASSERTED_PASS":
        raise AcceptanceFault("HARNESS_ACCEPTED_FALSE_PASS")
    wrong = dict(case)
    wrong["raw_stdout_sha256"] = "0" * 64
    if validate_attempt(wrong, {row["key"]: row}) != "STDOUT_DIGEST_MISMATCH":
        raise AcceptanceFault("HARNESS_IGNORED_TAMPERED_LOG")
    no_oracle = dict(case)
    no_oracle["oracle"] = None
    if validate_attempt(no_oracle, {row["key"]: row}) != "NO_ORACLE":
        raise AcceptanceFault("HARNESS_ACCEPTED_NO_ORACLE")
    if classify_action(row, [{"case": case}])["status"] == "PASS":
        raise AcceptanceFault("HARNESS_ACCEPTED_INCOMPLETE_TRIAD")
    mut = {"key": "stdio_mcp:create_container", "class": "potential_write"}
    if classify_action(mut, [{"case": case, "failure": None}])["status"] == "PASS":
        raise AcceptanceFault("HARNESS_ACCEPTED_UNVERIFIED_MUTATION")
    return {"status": "DETECTED", "controls": ["fake_pass", "altered_log",
            "missing_oracle", "partial_triads", "unverified_mutation"]}


def report_markdown(verdict, feature_catalogue=None):
    lines = ["# BoxWeave reacceptance machine evidence report", "",
             "**Run:** " + verdict["run_id"], "**Source:** " + verdict["source_sha"],
             "**Verification:** " + verdict["verification_mode"],
             "**Source state:** " + verdict.get("source_state", "unverified"),
             "", "This is a **per-action evidence record**, not whole-project PASS.",
             "A zero-failure source test and a catalog alone do not prove installed effects.",
             "", "## Exact counts", "",
             "| Classification | Actions |", "| --- | ---: |"]
    for k, v in verdict["counts"].items():
        lines.append("| " + k + " | " + str(v) + " |")
    lines.extend(["", "## Every named feature and exact action evidence", "",
                  "| Feature | Name | Mapped actions | Action PASS | Functional acceptance |",
                  "| --- | --- | ---: | ---: | --- |"])
    feature_catalogue = feature_catalogue or {}
    for feature, name in sorted(feature_catalogue.items()):
        relevant = [row for row in verdict["verdicts"] if feature in row["features"]]
        passes = sum(row["status"] == "PASS" for row in relevant)
        # Stateful combinations and external actor verification are not inferred from leaf counts.
        outcome = "PARTIAL (missing sequence/independent feature oracle)" if relevant else "BLOCKED (unmapped)"
        lines.append("| " + feature + " | " + name.replace("|", "\\\\|") +
                     " | " + str(len(relevant)) + " | " + str(passes) + " | " + outcome + " |")
    lines.extend(["", "## Every discovered action and its evidence", "",
                  "| Entry | Features | Result | Reason | Attempts |",
                  "| --- | --- | --- | --- | ---: |"])
    for row in verdict["verdicts"]:
        lines.append("| " + " | ".join((row["key"].replace("|", "\\|"),
                      ", ".join(row["features"]), row["status"],
                      row["reason"], str(row["attempts"]))) + " |")
    lines.extend(["", "## Findings requiring follow-up", ""])
    for f in verdict["findings"]:
        lines.append("- " + f["kind"] + ": " + str(f["action"]) + " / " +
                     str(f["code"]) + " (evidence " + str(f["case"]) + ")")
    if not verdict["findings"]:
        lines.append("No result-integrity failures observed; PARTIAL/BLOCKED are still open.")
    return "\n".join(lines) + "\n"


def seal_run(directory):
    directory = Path(directory)
    if (directory / "sealed.json").exists():
        raise AcceptanceFault("SEALED_REFUSE_OVERWRITE")
    cases = sorted((directory / "cases").glob("case-*.json"))
    files = [directory / "inventory.json", directory / "run.json", *cases]
    result = verify_run(directory)
    seal = {"schema_version": SCHEMA, "created_utc": now(),
            "files": {str(p.relative_to(directory)): digest(p.read_bytes()) for p in files},
            "evidence_root_sha256": digest({str(p.relative_to(directory)): digest(p.read_bytes())
                                           for p in files}),
            "verdict_sha256": digest(result)}
    unique_file(directory / "sealed.json", seal)
    unique_file(directory / "verdict.json", result)
    unique_file(directory / "report-actions.md", report_markdown(result,
        load(directory / "inventory.json").get("feature_catalogue", {})).encode())
    issues = [{"kind": ("HARNESS_OR_BEHAVIOR_FAILURE" if x["status"] == "FAIL"
               else "DISPATCH_UNKNOWN" if x["status"] == "UNKNOWN" else "COVERAGE_GAP"),
               "action": x["key"], "status": x["status"], "reason": x["reason"],
               "next_agent": "inspect original case and exact observed runtime; never replay unknown"}
              for x in result["verdicts"] if x["status"] in ("FAIL", "UNKNOWN", "PARTIAL", "BLOCKED")]
    unique_file(directory / "next-agent.json", issues)
    return seal


def verify_seal(directory):
    directory = Path(directory)
    seal = load(directory / "sealed.json")
    if seal.get("schema_version") != SCHEMA:
        raise AcceptanceFault("SEAL_SCHEMA_UNSUPPORTED")
    expected = seal["files"]
    for name, sha in expected.items():
        if Path(name).is_absolute() or ".." in Path(name).parts:
            raise AcceptanceFault("SEAL_PATH_TRAVERSAL")
        if digest((directory / name).read_bytes()) != sha:
            raise AcceptanceFault("SEALED_FILE_TAMPERED_" + name.replace("/", "_"))
    if {str(p.relative_to(directory)) for p in (directory / "cases").glob("case-*.json")} != {
            x for x in expected if x.startswith("cases/")}:
        raise AcceptanceFault("UNSEALED_CASE_INJECTION")
    if digest(expected) != seal["evidence_root_sha256"]:
        raise AcceptanceFault("SEAL_ROOT_MISMATCH")
    current = verify_run(directory)
    if digest(current) != seal["verdict_sha256"]:
        raise AcceptanceFault("VERDICT_MISMATCH")
    if current != load(directory / "verdict.json"):
        raise AcceptanceFault("VERDICT_RECORD_MODIFIED")
    return {"status": "VERIFIED", "evidence_root_sha256": seal["evidence_root_sha256"],
            "counts": current["counts"], "independent": False}



def bootstrap_readonly(root, out, *, host_command=None, mcp_command=None,
                       upstream=None, previous=None, changed=None):
    """One bounded discovery + safe-call handoff; intentionally not sealed.

    Agents must add genuinely independent negative/positive/mutation evidence
    after bootstrap and only then seal. UNKNOWN never triggers a retry.
    """
    manifest = catalogue(root, host_command=host_command,
                         mcp_command=mcp_command, upstream=upstream,
                         previous=previous, changed=changed)
    directory = new_run(out, manifest)
    attempts = safe_capture(directory, host_command, mcp_command)
    state = verify_run(directory)
    return {"run": str(directory), "source_sha": manifest["source_sha"],
            "discovered": len(manifest["actions"]),
            "named_features": len(manifest["feature_catalogue"]),
            "attempts": len(attempts), "counts": state["counts"],
            "missing_runtime_contexts": manifest["missing_runtime_contexts"],
            "status": "UNSEALED_REQUIRE_INDEPENDENT_CASES",
            "next": "record real additional cases, then inspect/seal/verify/report"}



def trusted_installed_root_mcp(path):
    # A genuine installed gateway is root-owned/read-only in its release
    # directory. Never allow arbitrary user-supplied Python through sudo.
    p = Path(path)
    if not re.fullmatch(
            r"/var/lib/boxweave-default/releases/0[.]\d+[.]\d+-[a-f0-9]{40}/mcp_gate[.]py",
            str(p)):
        return False
    if p.is_symlink() or not p.is_file() or p.resolve() != p:
        return False
    for entry in (p, p.parent, p.parent.parent, p.parent.parent.parent):
        if entry.is_symlink():
            return False
        st = entry.stat()
        if st.st_uid != 0 or st.st_mode & 0o022:
            return False
    return True


def argv_file(path, mode):
    data = load(path)
    if not isinstance(data, list) or not data or not all(
            isinstance(x, str) and 0 < len(x) < 4096 for x in data):
        raise AcceptanceFault("BAD_COMMAND_VECTOR")
    if any(any(ch in arg for ch in ("\\0", "\\n", "\\r")) for arg in data):
        raise AcceptanceFault("COMMAND_INJECTION_REFUSED")
    executable = Path(data[0]).name
    python = lambda x: re.fullmatch(r"python(?:3(?:\\.\\d+)?)?", Path(x).name) is not None
    if mode == "host":
        if executable == "boxweave" and len(data) == 1:
            return data
        if python(data[0]) and data[1:] == ["-m", "bwmini.host_cli"]:
            return data
    elif mode == "mcp":
        if (len(data) == 4 and data[:2] == ["sudo", "-n"]
                and python(data[2]) and trusted_installed_root_mcp(data[3])):
            return data
        if (python(data[0]) and len(data) == 2
                and Path(data[1]).name == "mcp_gate.py"
                and Path(data[1]).is_absolute()):
            return data
        if python(data[0]) and data[1:] == ["-m", "bwmini.mcp_gate"]:
            return data
        if (data[:4] == ["sudo", "-n", "-u", "boxweave"] and len(data) == 6
                and python(data[4]) and Path(data[5]).is_absolute()
                and Path(data[5]).name == "mcp_gate.py"):
            return data
    raise AcceptanceFault("EXECUTABLE_NOT_ALLOWLISTED_FOR_" + mode.upper())


def upstream_proof_review(root, child, declaration):
    """Only review existing declarations; no implicit Makefile fallback."""
    text = Path(declaration).read_text(encoding="utf-8")
    if len(text) > 65536:
        raise AcceptanceFault("PROOF_DECLARATION_TOO_LARGE")
    if not isinstance(child, str) or not re.fullmatch(r"[a-z][a-z0-9-]{2,62}-container", child):
        raise AcceptanceFault("PROOF_CHILD_NAME_INVALID")
    lines = [line.rstrip() for line in text.splitlines() if line.strip() and
             not line.lstrip().startswith("#")]
    start = [i for i, line in enumerate(lines) if line.startswith("test:")]
    if len(start) != 1:
        raise AcceptanceFault("PROOF_REQUIRES_EXPLICIT_TEST_DECLARATION")
    i = start[0]
    candidates = []
    initial = lines[i].split(":", 1)[1].strip()
    if initial:
        candidates.append(initial.strip("'\""))
    else:
        for line in lines[i+1:]:
            if not line.startswith(" "):
                break
            m = re.match(r"\s+(?:-\s*)?run:\s*(.+)$", line)
            if m:
                candidates.append(m.group(1).strip().strip("'\""))
    if not candidates or len(candidates) > 32:
        raise AcceptanceFault("PROOF_TEST_STEPS_UNREVIEWABLE")
    # Refuse shell operators and command substitutions; the generator is root in child.
    allowed = re.compile(r"(?:python3?|/usr/bin/python3)\s+-m\s+unittest\s+discover\s+-s\s+tests(?:\s+-v)?\Z")
    for cmd in candidates:
        if not allowed.fullmatch(cmd):
            raise AcceptanceFault("PROOF_REQUIRES_MANUAL_ROOT_COMMAND_REVIEW")
    return {"state": "REVIEWABLE_DECLARATION_NOT_EXECUTED", "child": child,
            "declaration_sha256": digest(text.encode()),
            "commands": candidates,
            "checkout_root": str(Path(root).resolve()),
            "root_command_execution": True,
            "requires": ["dedicated_nested_provider", "independent_child_identity",
                         "root_operator_confirmation", "approved_incustestbox"]}


def candidate_manifest(package, source_sha, inventory, proof):
    package = Path(package)
    if not package.is_file() or package.stat().st_size > 200 * 1024 * 1024:
        raise AcceptanceFault("CANDIDATE_PACKAGE_INVALID")
    if not re.fullmatch(r"[0-9a-f]{40}", source_sha):
        raise AcceptanceFault("CANDIDATE_SOURCE_INVALID")
    verdict = verify_run(proof)
    if verdict["source_sha"] != source_sha:
        raise AcceptanceFault("CANDIDATE_EVIDENCE_SOURCE_MISMATCH")
    if verdict.get("source_state") != "clean":
        raise AcceptanceFault("CANDIDATE_EVIDENCE_WORKTREE_NOT_CLEAN")
    required = load(inventory).get("required_actions", [])
    if not isinstance(required, list) or not required or any(
            not isinstance(name, str) or not name for name in required):
        raise AcceptanceFault("CANDIDATE_REQUIRED_GATES_EMPTY_OR_INVALID")
    by_action = {x["key"]: x for x in verdict["verdicts"]}
    for action in required:
        if action not in by_action or by_action[action]["status"] != "PASS":
            raise AcceptanceFault("CANDIDATE_REQUIRED_ACTION_NOT_VERIFIED_" + action)
    # A caller-supplied SHA or matching filename is not evidence that the
    # archive actually contains the tested source. Verify full package member
    # digests and the package's own pinned source state before attesting.
    from bwmini import distribution
    try:
        package_hash = digest(package.read_bytes())
        manifest = distribution.verify(package, package_hash)
    except (distribution.InstallError, OSError, ValueError, KeyError, TypeError,
            __import__("zipfile").BadZipFile) as exc:
        raise AcceptanceFault("CANDIDATE_PACKAGE_CONTENT_INVALID") from exc
    if (manifest["source"] != {"commit": source_sha, "state": "clean"}
            or any(x.startswith("bwmini/reacceptance") for x in manifest["files"])):
        raise AcceptanceFault("CANDIDATE_PACKAGE_SOURCE_OR_BUNDLED_REGISTRY_MISMATCH")
    return {"schema_version": SCHEMA, "package_sha256": package_hash,
            "source_sha": source_sha, "verification_run_id": verdict["run_id"],
            "required_actions": required, "required_actions_passed": True,
            "isolation": "MUST_BE_CONFIRMED_BY_SEPARATE_TEST_RUN",
            "publish": False, "created_utc": now()}


def verify_candidate(attestation, package, release_manifest=None):
    a = load(attestation)
    if digest(Path(package).read_bytes()) != a.get("package_sha256"):
        raise AcceptanceFault("CANDIDATE_BYTES_CHANGED")
    if release_manifest:
        r = load(release_manifest)
        if r.get("source") != a.get("source_sha"):
            raise AcceptanceFault("RELEASE_SOURCE_NOT_VERIFIED_CANDIDATE")
        if r.get("files", {}).get("boxweave.zip") != a.get("package_sha256"):
            raise AcceptanceFault("RELEASE_PACKAGE_NOT_VERIFIED_CANDIDATE")
    return {"state": "BYTE_EQUIVALENCE_VERIFIED",
            "candidate_source": a["source_sha"],
            "release_manifest_checked": bool(release_manifest),
            "installed_black_box": "separate_verifier_required"}



def compare_ingresses(data):
    """Compare ALREADY OBSERVED external effects; never invokes native writes."""
    if not isinstance(data, dict) or not isinstance(data.get("observations"), list):
        raise AcceptanceFault("CROSS_INGRESS_OBSERVATIONS_REQUIRED")
    observations = data["observations"]
    if len(observations) < 2 or len(observations) > 12:
        raise AcceptanceFault("CROSS_INGRESS_NEEDS_TWO_REAL_CHANNELS")
    keys = ("logical", "provider_uuid", "created_at")
    targets = set()
    semantics = set()
    effects = set()
    for item in observations:
        if not isinstance(item, dict):
            raise AcceptanceFault("INGRESS_OBSERVATION_MALFORMED")
        if not all(item.get(k) for k in ("interface", "case_id", "original_response_sha256",
                                        "response_meaning", "target")):
            raise AcceptanceFault("INGRESS_ORIGINAL_EVIDENCE_REQUIRED")
        if not re.fullmatch(r"[a-f0-9]{64}", str(item["original_response_sha256"])):
            raise AcceptanceFault("INGRESS_RESPONSE_HASH_INVALID")
        t = item["target"]
        if not isinstance(t, dict) or not all(t.get(k) for k in keys):
            raise AcceptanceFault("INGRESS_IMMUTABLE_TARGET_REQUIRED")
        targets.add(tuple(str(t[k]) for k in keys))
        semantics.add(item["response_meaning"])
        if item.get("mutation"):
            oracle = item.get("oracle")
            if not (isinstance(oracle, dict) and oracle.get("independent")
                    and type(oracle.get("effects_after")) is int
                    and type(oracle.get("effects_before")) is int):
                raise AcceptanceFault("INGRESS_EFFECT_ORACLE_REQUIRED")
            effects.add(oracle["effects_after"] - oracle["effects_before"])
    if len(targets) != 1:
        return {"status": "INCOMPARABLE", "reason": "TARGET_INCARNATION_CHANGED",
                "channels": len(observations)}
    if len({item["interface"] for item in observations}) < 2:
        raise AcceptanceFault("INGRESS_NEEDS_DISTINCT_INTERFACES")
    if len(semantics) != 1 or len(effects) > 1:
        return {"status": "SUSPECT", "reason": "ERROR_OR_EFFECT_PARITY_DIFFERENCE",
                "meanings": sorted(semantics), "effect_deltas": sorted(effects),
                "channels": len(observations),
                "root_cause": "UNDETERMINED_REQUIRES_PINNED_NATIVE_PROBE"}
    return {"status": "COMPARABLE_OBSERVATIONS", "reason": "MEANING_AND_EFFECT_PARITY",
            "channels": len(observations), "meaning": next(iter(semantics)),
            "root_cause": "NOT_ATTESTED_BY_COMPARISON_ALONE"}



def print_json(value):
    print(json.dumps(value, sort_keys=True, indent=2))


def main(argv=None):
    p = argparse.ArgumentParser(prog="python -m bwmini.reacceptance",
        description="Read-only-default source/runtime inventory, black-box evidence and candidate gates.")
    sub = p.add_subparsers(dest="command", required=True)
    inv = sub.add_parser("inventory", help="Discover source, installed CLI/MCP and pinned upstream contracts")
    inv.add_argument("--root", default=".")
    inv.add_argument("--host-argv-json")
    inv.add_argument("--mcp-argv-json")
    inv.add_argument("--upstream-tree")
    inv.add_argument("--previous")
    inv.add_argument("--changed", action="append", default=[])
    inv.add_argument("--out", required=True)
    auto = sub.add_parser("bootstrap-readonly",
                          help="One-step runtime discovery plus allowlisted read-only handoff")
    auto.add_argument("--root", default=".")
    auto.add_argument("--host-argv-json")
    auto.add_argument("--mcp-argv-json")
    auto.add_argument("--upstream-tree")
    auto.add_argument("--previous")
    auto.add_argument("--changed", action="append", default=[])
    auto.add_argument("--out", required=True)
    begin = sub.add_parser("begin", help="Initialize immutable private run from inventory")
    begin.add_argument("--inventory", required=True)
    begin.add_argument("--out", required=True)
    safe = sub.add_parser("run-safe", help="Execute allowlisted read-only installed probes only")
    safe.add_argument("--run", required=True)
    safe.add_argument("--host-argv-json")
    safe.add_argument("--mcp-argv-json")
    rec = sub.add_parser("record", help="Import independent original execution evidence, no commands run")
    rec.add_argument("--run", required=True)
    rec.add_argument("--action", required=True)
    rec.add_argument("--category", required=True, choices=("positive", "invalid", "denial"))
    rec.add_argument("--input", required=True)
    for name in ("inspect", "seal", "verify", "report"):
        t = sub.add_parser(name)
        t.add_argument("--run", required=True)
    sub.add_parser("self-test", help="Run intentionally false-PASS and tamper negative controls")
    pr = sub.add_parser("proof-review", help="Review known-safe root-in-child upstream Proof Pack declaration")
    pr.add_argument("--checkout", required=True)
    pr.add_argument("--declaration", required=True)
    pr.add_argument("--child", required=True)
    cmp = sub.add_parser("compare-ingresses", help="Compare observed error/effect semantics with exact target identity")
    cmp.add_argument("--observations", required=True)
    cp = sub.add_parser("candidate-attest")
    cp.add_argument("--package", required=True)
    cp.add_argument("--source", required=True)
    cp.add_argument("--inventory", required=True)
    cp.add_argument("--run", required=True)
    cp.add_argument("--out", required=True)
    cv = sub.add_parser("candidate-verify")
    cv.add_argument("--attestation", required=True)
    cv.add_argument("--package", required=True)
    cv.add_argument("--release-manifest")
    args = p.parse_args(argv)
    try:
        if args.command == "inventory":
            value = catalogue(args.root,
                host_command=argv_file(args.host_argv_json, "host") if args.host_argv_json else None,
                mcp_command=argv_file(args.mcp_argv_json, "mcp") if args.mcp_argv_json else None,
                upstream=args.upstream_tree, previous=args.previous, changed=args.changed)
            out = Path(args.out)
            private_mkdir(out.parent)
            unique_file(out, value)
            print_json({"written": str(out), "actions": len(value["actions"]),
                        "runtime_contexts_missing": value["missing_runtime_contexts"],
                        "diff": value["delta"]})
        elif args.command == "bootstrap-readonly":
            print_json(bootstrap_readonly(
                args.root, args.out,
                host_command=argv_file(args.host_argv_json, "host") if args.host_argv_json else None,
                mcp_command=argv_file(args.mcp_argv_json, "mcp") if args.mcp_argv_json else None,
                upstream=args.upstream_tree, previous=args.previous, changed=args.changed))
        elif args.command == "begin":
            target = new_run(args.out, load(args.inventory))
            print_json({"run": str(target), "state": "inventory_pinned_NO_PASS"})
        elif args.command == "run-safe":
            rows = safe_capture(args.run,
                host_command=argv_file(args.host_argv_json, "host") if args.host_argv_json else None,
                mcp_command=argv_file(args.mcp_argv_json, "mcp") if args.mcp_argv_json else None)
            print_json({"attempts": rows, "status": "RECORDED_NOT_PASSED"})
        elif args.command == "record":
            e = load(args.input)
            file = record_attempt(args.run, args.action, args.category,
                e.get("observed"), observer=e.get("oracle"), actor=e.get("actor", "same_agent"),
                target=e.get("target"), expected=e.get("expected"))
            print_json({"evidence": file, "status": "RECORDED_UNVERIFIED"})
        elif args.command == "inspect":
            v = verify_run(args.run)
            print_json({"counts": v["counts"], "findings": v["findings"]})
        elif args.command == "seal":
            print_json({"sealed": seal_run(args.run)["evidence_root_sha256"]})
        elif args.command == "verify":
            print_json(verify_seal(args.run))
        elif args.command == "report":
            print(report_markdown(verify_run(args.run),
                load(Path(args.run) / "inventory.json").get("feature_catalogue", {})), end="")
        elif args.command == "self-test":
            print_json(negative_controls())
        elif args.command == "proof-review":
            print_json(upstream_proof_review(args.checkout, args.child, args.declaration))
        elif args.command == "compare-ingresses":
            print_json(compare_ingresses(load(args.observations)))
        elif args.command == "candidate-attest":
            att = candidate_manifest(args.package, args.source, args.inventory, args.run)
            private_mkdir(Path(args.out).parent)
            unique_file(args.out, att)
            print_json({"attestation": args.out, "source": att["source_sha"],
                        "candidate_sha256": att["package_sha256"]})
        elif args.command == "candidate-verify":
            print_json(verify_candidate(args.attestation, args.package, args.release_manifest))
        return 0
    except (AcceptanceFault, ValueError, OSError, KeyError, TypeError) as exc:
        print_json({"status": "FAIL_CLOSED", "code": str(exc)})
        return 2


if __name__ == "__main__":
    sys.exit(main())
