"""Pure lifecycle configuration contract shared by Provider and worker."""
import base64
import re
from .domain import Fault

RESIZE = {"cpu": ("1", "2", "3", "4", "5", "6", "7", "8"),
          "memory": ("1GB", "2GB", "4GB", "8GB"),
          "disk": ("8GB", "10GB", "20GB", "25GB", "30GB", "40GB", "50GB", "100GB")}

def validate_config(c, protected):
    if c is None:
        return
    keys = {"allowed_names", "cpu", "memory", "disk", "ssh_public_key", "readiness_timeout"}
    if not isinstance(c, dict) or set(c) != keys:
        raise Fault("LIFECYCLE_CONFIG_INVALID", 422)
    names = c["allowed_names"]
    if (not isinstance(names, list) or not 1 <= len(names) <= 32
            or any(not isinstance(n, str) or re.fullmatch(r"[a-z][a-z0-9-]{0,62}", n) is None
                   or n.startswith("containarium-core-") or n in protected for n in names)
            or len(set(names)) != len(names)):
        raise Fault("LIFECYCLE_CONFIG_INVALID", 422)
    if (c["cpu"] not in RESIZE["cpu"]
            or c["memory"] not in RESIZE["memory"]
            or c["disk"] not in RESIZE["disk"]
            or type(c["readiness_timeout"]) is not int or not 1 <= c["readiness_timeout"] <= 1800):
        raise Fault("LIFECYCLE_CONFIG_INVALID", 422)
    try:
        key = c["ssh_public_key"]
        if not isinstance(key, str) or len(key) > 1024 or re.fullmatch(
                r"ssh-ed25519 [A-Za-z0-9+/]+={0,2}(?: [A-Za-z0-9@._-]{1,80})?", key) is None:
            raise ValueError()
        blob = base64.b64decode(key.split()[1], validate=True)
        if len(blob) != 51 or blob[:19] != b"\x00\x00\x00\x0bssh-ed25519\x00\x00\x00\x20":
            raise ValueError()
    except (ValueError, TypeError):
        raise Fault("MANAGED_PUBLIC_KEY_REQUIRED", 422) from None
