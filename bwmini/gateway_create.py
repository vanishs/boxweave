"""App-owned durable create receipt available to reviewed Agent gateway.

This adapter accesses ONLY the installed application SQLite file selected by
the reviewed application REST binding. No JWT/private key, JSON payload, CLI commands or
foreign path is stored in the journal.
"""
import contextlib
import os
from pathlib import Path
import sqlite3
import stat
from urllib.parse import quote

try:
    from .remote_create import CreateJournal, CreateFault, validated_request
except ImportError:
    from remote_create import CreateJournal, CreateFault, validated_request


class GatewayRESTFault(ValueError):
    pass


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
                "file:" + quote(str(self.path), safe="/") + "?mode=rw",
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
        if (existing["username"] != logical or
                existing["state"] != "submitted" or
                existing["code"] != "REST_GATE_PENDING"):
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
    # For a second invocation, consult the original durable receipt BEFORE
    # observing current Provider state. A newly created Box must not mask the
    # idempotent status of the original single dispatch.
    ledger = journal(configuration)
    digest = validated_request(original)[3]
    try:
        prior = ledger.get(username=body["username"])
    except CreateFault as exc:
        if exc.code != "CREATE_TASK_NOT_FOUND":
            raise
        prior = None
    if prior is not None:
        with ledger.store.db() as db:
            saved = db.execute(
                "SELECT payload_sha256 FROM remote_create_tasks WHERE task_id=?",
                (prior["task_id"],)).fetchone()
        if saved is None or saved["payload_sha256"] != digest:
            raise GatewayRESTFault("CREATE_NAME_CONFLICT")
        if (not isinstance(rpc_id,str) or
                rpc_id != "bw-create-"+prior["task_id"] or
                prior["code"] != "REST_GATE_PENDING"):
            return prior
    # HTTP pre-reserved durable tasks can expose definitive local
    # no-POST refusals without misrepresenting them as UNKNOWN. The
    # journal CAS proves the original arguments and pending dispatch phase.
    def refused_before_post(code):
        if (isinstance(rpc_id,str) and rpc_id.startswith("bw-create-")):
            task_id=rpc_id[len("bw-create-"):]
            if ledger.fail_before_dispatch(task_id,code,
                                          username=body["username"],digest=digest):
                return ledger.get(task_id=task_id)
        raise GatewayRESTFault(code)

    # Confirm the exact target name is not already occupied before recording
    # the non-replayable write intent. A failed/partial read is NEVER absence.
    try:
        rest = rest_wire.Client(
            {k: configuration[k] for k in ("url","token_file")})
        try:
            rest.request("GET", "/v1/containers/"+body["username"], timeout=20)
        except rest_wire.WireFault as error:
            # The centralized adapter returns *standard* gRPC NotFound / HTTP
            # 404 only for validated typed Provider replies or the exact v0.128
            # compatibility signature. A bare HTTP 404 can be a missing API route
            # and must not license an irreversible create.
            known_missing = (
                error.grpc_code is rest_wire.rest_status.GRPCCode.NOT_FOUND
                and error.http_status == 404
                and error.status_origin in ("grpc", "compatibility"))
            if not known_missing:
                raise GatewayRESTFault("REST_CREATE_PREFLIGHT_UNVERIFIED") from None
            if error.status_origin == "compatibility":
                # In upstream v0.128 GetContainer ignores peer list failures.
                # The legacy 500 text alone is therefore not proof that this
                # Box is absent on every backend. Independently check the full
                # REST inventory, including unreachable peers and pending boxes,
                # before allowing a permanent create intent.
                try:
                    inventory = rest.request("GET", "/v1/containers", timeout=20)
                    rest_routes.validate_read_result("list_containers", inventory)
                    if (not isinstance(inventory.get("unreachableBackends"), list)
                            or "totalCount" not in inventory):
                        raise rest_routes.RouteFault("REST_INVENTORY_PARTIAL")
                    names = [
                        rest_routes.logical_name(row["username"])
                        for row in inventory["containers"]
                    ]
                    if len(names) != len(set(names)):
                        raise rest_routes.RouteFault("REST_INVENTORY_PARTIAL")
                except (rest_wire.WireFault, rest_routes.RouteFault,
                        KeyError, TypeError, ValueError):
                    raise GatewayRESTFault(
                        "REST_CREATE_PREFLIGHT_UNVERIFIED") from None
                if body["username"] in names:
                    raise GatewayRESTFault("REST_RESOURCE_EXISTS")
        else:
            raise GatewayRESTFault("REST_RESOURCE_EXISTS")
    except GatewayRESTFault as error:
        return refused_before_post(str(error))
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
    except Exception as error:
        # No dispatch intent, let alone REST POST, has occurred in this
        # invocation. Existing claim contention is not a Provider outcome.
        if getattr(error, "code", None) == "OPERATION_EXISTS":
            return ledger.get(task_id=task)
        code = ("COORDINATION_SCOPE_UNVERIFIED" if
                getattr(error, "code", None) == "COORDINATION_SCOPE_UNVERIFIED"
                else "REST_CREATE_PREDISPATCH_UNVERIFIED")
        if ledger.fail_before_dispatch(task,code,
                                       username=receipt["username"],
                                       digest=original_digest):
            return ledger.get(task_id=task)
        ledger.freeze(task)
        raise GatewayRESTFault("REST_CREATE_PREDISPATCH_UNVERIFIED") from None
    try:
        # After this durable dispatch marker, a crash is necessarily
        # ambiguous. Never classify a lost worker as a no-POST failure.
        claims.dispatch_intent(task)
        if not ledger.begin_dispatch(task, original_digest):
            return ledger.get(task_id=task)
    except Exception:
        ledger.freeze(task)
        try:
            claims.unknown(task)
        except Exception:
            pass
        raise GatewayRESTFault("REST_CREATE_PREDISPATCH_UNVERIFIED") from None
    try:
        response = rest.request(method, route, body, timeout=45)
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
