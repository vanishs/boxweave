"""Read-only installed application diagnostic profiles for the Agent gateway.

This is launched with a *fixed* application-owned interpreter and config by
gate_diagnostics. It never mutates a Box, runs an installer or reads token bytes
into its output. No exception or upstream response text is returned.
"""
import argparse
import json
from .app_runtime import read_config
from .domain import Fault
from .provider import Provider
from .mcp import MCP

SCHEMA_VERSION = 1
PROFILES = ("preflight", "rest_read")
CHECK_IDS = frozenset(("config", "provider_mode", "provider_rest", "runtime_binding",
                       "rest_inventory", "mcp_inventory"))
STATES = frozenset(("passed", "failed", "not_verified"))


def check(ident, state, code):
    if ident not in CHECK_IDS or state not in STATES:
        raise ValueError("PROBE_CHECK_INVALID")
    return {"id": ident, "state": state, "code": code}


def conclude(checks):
    statuses = {entry["state"] for entry in checks}
    return ("failed" if "failed" in statuses else
            "not_verified" if "not_verified" in statuses else "passed")


def run(profile, config_path, persisted_path, *, reader=read_config,
        provider_factory=Provider, mcp_factory=MCP):
    """Never return configuration, inventory, tokens, exceptions or raw errors."""
    if profile not in PROFILES:
        raise ValueError("PROBE_PROFILE_INVALID")
    checks = []
    try:
        runtime = reader(config_path)
        persistent = reader(persisted_path)
        if not isinstance(runtime, dict) or not isinstance(persistent, dict):
            raise ValueError("invalid config")
        checks.append(check("config", "passed", "CONFIG_READ_PASSED"))
    except (Fault, OSError, ValueError, TypeError, KeyError):
        return {"schema_version": SCHEMA_VERSION, "profile": profile,
                "state": "not_verified", "coverage": "app_owned_probe",
                "checks": [check("config", "not_verified", "CONFIG_READ_UNAVAILABLE")],
                "omissions": ["app_config_unavailable"]}

    if runtime.get("mode") != "mcp" or runtime.get("provider") != "containarium":
        checks.append(check("provider_mode", "failed", "PROVIDER_MODE_UNSUPPORTED"))
        return {"schema_version": SCHEMA_VERSION, "profile": profile,
                "state": conclude(checks), "coverage": "app_owned_probe",
                "checks": checks, "omissions": ["provider_not_initialized"]}
    checks.append(check("provider_mode", "passed", "PROVIDER_MODE_VALID"))
    if ((runtime.get("provider_api") is None and persistent.get("provider_api") is None)
            or (isinstance(runtime.get("provider_api"), dict)
                and runtime["provider_api"] == persistent.get("provider_api"))):
        checks.append(check("runtime_binding", "passed", "PROVIDER_BINDING_MATCHES"))
    else:
        checks.append(check("runtime_binding", "failed", "PROVIDER_BINDING_DRIFT"))

    try:
        provider = provider_factory(runtime)
    except (Fault, OSError, ValueError, TypeError, KeyError):
        checks.append(check("provider_rest", "not_verified", "PROVIDER_INIT_UNAVAILABLE"))
        return {"schema_version": SCHEMA_VERSION, "profile": profile,
                "state": conclude(checks), "coverage": "app_owned_probe",
                "checks": checks, "omissions": ["provider_probe_unavailable"]}
    if getattr(provider, "rest", None) is None:
        checks.append(check("provider_rest", "not_verified", "REST_NOT_CONFIGURED"))
    else:
        checks.append(check("provider_rest", "passed", "REST_SELECTED"))
    if profile == "rest_read":
        if getattr(provider, "rest", None) is not None:
            try:
                provider.rest.inventory()
                checks.append(check("rest_inventory", "passed", "REST_AUTHENTICATED_READ_PASSED"))
            except Fault as exc:
                code = ("REST_AUTH_REJECTED" if exc.code == "API_AUTH_REJECTED"
                        else "REST_READ_UNAVAILABLE")
                checks.append(check("rest_inventory",
                                    "failed" if code == "REST_AUTH_REJECTED" else "not_verified",
                                    code))
            except (OSError, ValueError, TypeError, KeyError):
                checks.append(check("rest_inventory", "not_verified", "REST_READ_UNAVAILABLE"))
        else:
            checks.append(check("rest_inventory", "not_verified", "REST_NOT_CONFIGURED"))
        try:
            with mcp_factory(provider.argv, 20) as client:
                client.discover()
                client.tool("list_containers", {})
            checks.append(check("mcp_inventory", "passed", "MCP_AUTHENTICATED_READ_PASSED"))
        except (Fault, OSError, ValueError, TypeError, KeyError, TimeoutError):
            checks.append(check("mcp_inventory", "not_verified", "MCP_READ_UNAVAILABLE"))
    return {"schema_version": SCHEMA_VERSION, "profile": profile,
            "state": conclude(checks), "coverage": "app_owned_probe",
            "checks": checks, "omissions": []}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--profile", choices=PROFILES, required=True)
    parser.add_argument("--config", required=True)
    parser.add_argument("--persisted", required=True)
    args = parser.parse_args(argv)
    try:
        result = run(args.profile, args.config, args.persisted)
    except Exception:
        # The caller's subprocess result is always bounded and secret-free.
        result = {"schema_version": SCHEMA_VERSION, "profile": args.profile,
                  "state": "not_verified", "coverage": "app_owned_probe",
                  "checks": [check("config", "not_verified", "PROBE_RUNTIME_UNAVAILABLE")],
                  "omissions": ["probe_exception_redacted"]}
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
