"""Small value contracts, with no web or subprocess dependencies."""
from dataclasses import dataclass, asdict
import hashlib
import json
import re
from pathlib import PurePosixPath

class Fault(Exception):
    def __init__(self, code, status=409):
        self.code, self.status = code, status
        super().__init__(code)

@dataclass(frozen=True)
class Target:
    provider: str
    backend: str
    project: str
    logical: str
    instance: str
    created: str
    user: str
    hostname: str

    def __post_init__(self):
        if not all(isinstance(v, str) and v and len(v) <= 200 for v in asdict(self).values()):
            raise Fault("INVALID_TARGET", 422)
        if not re.fullmatch(r"[a-z][a-z0-9-]{0,62}", self.logical):
            raise Fault("INVALID_TARGET", 422)

    def data(self):
        return asdict(self)

    def digest(self):
        return hashlib.sha256(json.dumps(self.data(), sort_keys=True).encode()).hexdigest()

def workspace(value, user):
    if not isinstance(value, str) or not value.startswith("/"):
        raise Fault("INVALID_WORKSPACE", 422)
    path = PurePosixPath(value)
    root = PurePosixPath("/home") / user
    if ".." in path.parts or any(ord(c) < 32 for c in value) or not path.is_relative_to(root):
        raise Fault("INVALID_WORKSPACE", 422)
    return str(path)

def name(value):
    if not isinstance(value, str) or not 1 <= len(value.strip()) <= 80 or any(ord(c) < 32 for c in value):
        raise Fault("INVALID_NAME", 422)
    return value.strip()
