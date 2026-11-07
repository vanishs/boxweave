"""App-owned durable create receipt available to reviewed Agent gateway.

This adapter accesses ONLY the installed application SQLite file selected by
the trusted root policy. No JWT/private key, JSON payload, CLI commands or
foreign path is stored in the journal.
"""
import contextlib
import os
from pathlib import Path
import sqlite3
import stat

try:
    from .remote_create import CreateJournal, CreateFault, validated_request
    from .gateway_rest import GatewayRESTFault
except ImportError:
    from remote_create import CreateJournal, CreateFault, validated_request
    from gateway_rest import GatewayRESTFault


def checked_app_store(path):
    value = Path(path) if isinstance(path, str) else Path("/")
    if (not value.is_absolute() or ".." in value.parts
            or len(str(value)) > 1024):
        raise GatewayRESTFault("REST_STATE_DB_INVALID")
    try:
        file = value.lstat()
        directory = value.parent.lstat()
        if (not stat.S_ISREG(file.st_mode) or file.st_nlink != 1
                or file.st_uid != os.geteuid() or file.st_mode & 0o077
                or not stat.S_ISDIR(directory.st_mode)
                or directory.st_uid != os.geteuid()
                or directory.st_mode & 0o077):
            raise ValueError()
    except (OSError, ValueError):
        raise GatewayRESTFault("REST_STATE_DB_UNSAFE") from None
    return value


class AppStore:
    def __init__(self, path):
        self.path = checked_app_store(path)

    @contextlib.contextmanager
    def db(self):
        checked_app_store(str(self.path))
        try:
            db = sqlite3.connect(
                "file:" + self.path.as_posix() + "?mode=rw",
                uri=True, timeout=3)
            db.row_factory = sqlite3.Row
            db.execute("PRAGMA foreign_keys=ON")
            db.execute("PRAGMA synchronous=FULL")
            try:
                yield db
                db.commit()
            except Exception:
                db.rollback()
                raise
            finally:
                db.close()
        except sqlite3.Error:
            raise GatewayRESTFault("REST_STATE_DB_UNAVAILABLE") from None


def journal(config):
    path = config.get("state_db") if isinstance(config, dict) else None
    if path is None:
        raise GatewayRESTFault("REST_STATE_DB_NOT_SELECTED")
    return CreateJournal(AppStore(path))


def reserve(original, request_id, config):
    """Returns receipt, owner of one new dispatch, original normalized payload.

    Existing nonterminal / UNKNOWN receipt is *never replayed*. Both SSH and
    HTTP use the same Store. HTTP's pre-reservation is matched exactly by
    opaque task id and stored argument hash, not a client-supplied name alone.
    """
    logical, key, payload, digest = validated_request(original)
    store = journal(config)
    if isinstance(request_id, str) and request_id.startswith("bw-create-"):
        task_id = request_id[len("bw-create-"):]
        existing = store.get(task_id=task_id)
        if existing["username"] != logical or existing["state"] != "submitted":
            return existing, False, payload
        with store.store.db() as db:
            row = db.execute(
                "SELECT payload_sha256 FROM remote_create_tasks WHERE task_id=?",
                (task_id,)).fetchone()
            if row is None or row["payload_sha256"] != digest:
                raise GatewayRESTFault("REST_CREATE_RECEIPT_MISMATCH")
        return existing, True, payload
    receipt, fresh = store.reserve(logical, key, digest)
    return receipt, fresh, payload


def dispatch(configuration, raw_args, rpc_id, *, normalize, protected):
    """Single, durably claimed REST POST; returns original task receipt.

    No user-controlled method/endpoint/token. A CREATING response is never
    described as Guest-ready; postconditions belong to the existing app worker.
    """
    if (not protected or not isinstance(configuration, dict)
            or configuration.get("mode") != "coordinated"):
        raise GatewayRESTFault("REST_COORDINATION_REQUIRED")
    if not isinstance(raw_args, dict):
        raise GatewayRESTFault("REST_CREATE_ARGUMENT_INVALID")
    nesting = raw_args.get("enable_nesting")
    if nesting is not None and type(nesting) is not bool:
        raise GatewayRESTFault("NESTING_BOOLEAN_REQUIRED")
    if nesting is not None and (
            any(raw_args.get(k) for k in ("backend_id", "pool", "region"))
            or raw_args.get("isolation") not in (None, "", "container")
            or raw_args.get("os_type") not in (None, "", "ubuntu")):
        raise GatewayRESTFault("LOCAL_UBUNTU_NESTING_ONLY")
    original, supplied = dict(raw_args), dict(raw_args)
    supplied.pop("enable_nesting", None)
    supplied.pop("request_key", None)
    try:
        key_injected = normalize(supplied)
        from . import rest_routes, rest_wire, coordination_bridge
    except ImportError:
        import rest_routes
        import rest_wire
        import coordination_bridge
    try:
        method, route, body = rest_routes.route(
            "create_container", key_injected, async_create=True)
    except rest_routes.RouteFault as exc:
        raise GatewayRESTFault(str(exc)) from None
    # Everything above is validated before the permanent no-replay intent.
    receipt, fresh, payload = reserve(original, rpc_id, configuration)
    if not fresh:
        return receipt
    task = receipt["task_id"]
    original_digest = validated_request(original)[3]
    ledger = journal(configuration)
    try:
        ledger.intent_options(
            task, nesting=nesting,
            source=("http" if isinstance(rpc_id, str)
                    and rpc_id.startswith("bw-create-") else "ssh"))
        scope = coordination_bridge.published_scope()
        claims = coordination_bridge.SharedClaims()
        claims.acquire_environment(task, "gateway_" + task, scope,
                                   "create", receipt["username"], None, {})
        # Commit shared R-04 dispatch intent BEFORE any Provider write.
        claims.dispatch_intent(task)
        if not ledger.begin_dispatch(task, original_digest):
            return ledger.get(task_id=task)
    except Exception:
        ledger.freeze(task)
        raise GatewayRESTFault("REST_CREATE_PREDISPATCH_UNVERIFIED") from None
    try:
        response = rest_wire.Client(configuration).request(
            method, route, body, timeout=45)
        target = response.get("container")
        if (not isinstance(target, dict) or
                target.get("username") != receipt["username"] or
                target.get("state") not in (
                    "CONTAINER_STATE_CREATING", "CONTAINER_STATE_PROVISIONING",
                    "CONTAINER_STATE_RUNNING")):
            raise GatewayRESTFault("REST_CREATE_ACCEPTANCE_UNVERIFIED")
        # This is only server-side acceptance of a single async create.
        ledger.accept(task)
        return ledger.get(task_id=task)
    except Exception:
        ledger.freeze(task)
        try:
            claims.unknown(task)
        except Exception:
            pass
        return ledger.get(task_id=task)
