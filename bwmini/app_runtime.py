"""Application composition root shared by CLI and independent deploy checks.

No CLI argument parsing, root helper, or reverse dependency on __main__.
Preserves the existing provider selection, opt-in coordination and Store/Service
initialization semantics; performs no remote I/O itself.
"""
import json
from pathlib import Path
from .domain import Target, Fault
from .provider import Fixture, Provider
from .service import Service
from .store import Store

def read_config(path):
    p=Path(path)
    if p.is_symlink() or p.stat().st_size>65536:
        raise Fault("INVALID_CONFIG",422)
    c=json.loads(p.read_text())
    return c

def load(path, *, reader=None):
    c=(read_config if reader is None else reader)(path)
    mode=c.get("mode")
    if mode=="fixture":
        provider=Fixture(Target(**c["target"]),c.get("fixture_counter"))
    elif mode=="mcp":
        provider=Provider(c)
    else:
        raise Fault("EXPLICIT_MODE_REQUIRED",422)
    option = c.get("coordination")
    if option not in (None, {"enabled": True}, {"enabled": False}):
        raise Fault("COORDINATION_CONFIG_INVALID", 422)
    claims = None
    if option == {"enabled": True}:
        if mode != "mcp":
            raise Fault("COORDINATION_LIVE_PROVIDER_REQUIRED", 422)
        from .coordination_bridge import SharedClaims
        claims = SharedClaims()
    store=Store(c["database"])
    if getattr(provider, "rest", None) is not None:
        provider.rest.store = store
    return c,Service(store,provider,budget=c.get("operation_timeout",60),
                     shared_claims=claims)
