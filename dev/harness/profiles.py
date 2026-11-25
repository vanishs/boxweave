"""Declarative environment prerequisites; no provisioning or installation."""
import argparse

from . import environment
from .common import HarnessError, repo_path
from .records import read_json


def observe_profile(root, name):
    data = read_json(repo_path(root, "dev/profiles.json"))
    if not isinstance(data, dict) or data.get("schema_version") != 1:
        raise HarnessError("Unsupported profiles schema")
    profiles = data.get("profiles")
    if not isinstance(profiles, dict) or name not in profiles:
        raise HarnessError("Unknown environment profile")
    profile = profiles[name]
    if (not isinstance(profile, dict) or not isinstance(profile.get("require"), list)
            or any(not isinstance(item, str) or item not in environment.REQUIREMENTS for item in profile["require"])
            or type(profile.get("native_review")) is not bool or not isinstance(profile.get("purpose"), str)):
        raise HarnessError("Invalid environment profile")
    result, code = environment.run(argparse.Namespace(require=profile["require"]), root)
    if profile["native_review"]:
        result["blockers"].append("independent native capability/authority and fixture review")
        result["status"] = "REVIEW_REQUIRED"
        code = 1
    return {"profile": name, "purpose": profile["purpose"], **result}, code


def register(parser):
    parser.add_argument("name")


def run(args, root):
    return observe_profile(root, args.name)
