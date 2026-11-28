#!/usr/bin/env python3
"""Bounded offline transfer preparation and exclusive guest assembly.

No network client, automatic dispatch, retries, target overwrite or deletion.
The caller must preserve original MCP intent, receipts and readback evidence.
"""
import argparse
import base64
import copy
import hashlib
import json
import os
import re
import stat
import sys
import secrets
import uuid
from pathlib import Path, PurePosixPath

MAX_BYTES = 4 * 1024 * 1024
MAX_CHUNK = 40960  # Keep generated Base64 request JSON below a 64 KiB read.
MAX_PARTS = 128
MAX_MANIFEST = 65536
MAX_JOURNAL = 16 * 1024 * 1024
MAX_JOURNAL_TOTAL = 64 * 1024 * 1024
MAX_EVENTS = 1024


class TransferError(ValueError):
    pass


def digest(data):
    return hashlib.sha256(data).hexdigest()


def encoded(value):
    return (json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=True,
                       allow_nan=False) + "\n").encode("ascii")


def unique(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise TransferError("Duplicate JSON key")
        result[key] = value
    return result


def strict_load(data):
    return json.loads(data, object_pairs_hook=unique,
                      parse_constant=lambda value: (_ for _ in ()).throw(TransferError(value)))


def bounded_read(path, limit):
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, "rb") as stream:
        before = os.fstat(stream.fileno())
        if not stat.S_ISREG(before.st_mode) or before.st_size > limit:
            raise TransferError("Expected bounded regular file")
        data = stream.read(limit + 1)
        after = os.fstat(stream.fileno())
        if (len(data) > limit or len(data) != before.st_size
                or (before.st_size, before.st_mtime_ns, before.st_ctime_ns)
                != (after.st_size, after.st_mtime_ns, after.st_ctime_ns)):
            raise TransferError("File changed while reading or exceeds bound")
        return data


def exclusive(path, data):
    with Path(path).open("xb") as stream:
        stream.write(data)
        stream.flush()
        os.fsync(stream.fileno())


def target_path(value):
    if not isinstance(value, str) or not value.startswith("/") or len(value) > 512:
        raise TransferError("Expected exact absolute guest path")
    path = PurePosixPath(value)
    if (path.as_posix() != value or value.startswith("//") or not path.name
            or ".." in path.parts or len(path.name.encode("utf-8")) > 120
            or any(ord(c) < 32 or ord(c) == 127 for c in value)):
        raise TransferError("Invalid guest target path")
    return path


def check_ref(value):
    if not isinstance(value, str) or not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}", value):
        raise TransferError("Expected bounded explicit work/transfer reference")
    return value


def box_id(value):
    try:
        if str(uuid.UUID(value)) != value:
            raise ValueError()
    except (ValueError, TypeError, AttributeError) as exc:
        raise TransferError("Expected exact canonical Box UUID") from exc
    return value


def check_digest(value):
    if not isinstance(value, str) or not re.fullmatch(r"[0-9a-f]{64}", value):
        raise TransferError("Invalid SHA256")


def key(tid, action):
    return "bwtr-" + tid + "-" + action


def artifact_path(target, tid, action):
    path = target_path(target)
    return str(path.parent / ("." + path.name + ".bwtr-" + tid + "." + action))


def make_manifest(data, box, target, work_ref, transfer_ref, chunk_size):
    box_id(box)
    target_path(target)
    check_ref(work_ref)
    check_ref(transfer_ref)
    if type(chunk_size) is not int or not 1 <= chunk_size <= MAX_CHUNK:
        raise TransferError("Invalid chunk size")
    if not 1 <= len(data) <= MAX_BYTES:
        raise TransferError("Expected nonempty source up to 4 MiB")
    chunks = [data[i:i+chunk_size] for i in range(0, len(data), chunk_size)]
    if len(chunks) > MAX_PARTS:
        raise TransferError("Too many parts")
    core = {"schema": 1, "box_id": box, "target": target, "work_ref": work_ref,
            "transfer_ref": transfer_ref, "size": len(data), "sha256": digest(data),
            "chunk_size": chunk_size,
            "parts": [{"index": i, "size": len(part), "sha256": digest(part)}
                      for i, part in enumerate(chunks)]}
    tid = digest(encoded(core))[:32]
    manifest = dict(core, transfer_id=tid)
    validate_manifest(manifest)
    return manifest, chunks


def validate_manifest(value):
    fields = {"schema", "box_id", "target", "work_ref", "transfer_ref", "size", "sha256",
              "chunk_size", "parts", "transfer_id"}
    if not isinstance(value, dict) or set(value) != fields or type(value["schema"]) is not int or value["schema"] != 1:
        raise TransferError("Invalid manifest schema")
    box_id(value["box_id"])
    target_path(value["target"])
    check_ref(value["work_ref"])
    check_ref(value["transfer_ref"])
    check_digest(value["sha256"])
    size, chunk = value["size"], value["chunk_size"]
    if type(size) is not int or not 1 <= size <= MAX_BYTES or type(chunk) is not int or not 1 <= chunk <= MAX_CHUNK:
        raise TransferError("Invalid source or chunk size")
    parts = value["parts"]
    expected_count = (size + chunk - 1) // chunk
    if not isinstance(parts, list) or not 1 <= len(parts) == expected_count <= MAX_PARTS:
        raise TransferError("Invalid part count")
    for i, part in enumerate(parts):
        if (not isinstance(part, dict) or set(part) != {"index", "size", "sha256"}
                or type(part["index"]) is not int or part["index"] != i
                or type(part["size"]) is not int
                or part["size"] != min(chunk, size - i*chunk)):
            raise TransferError("Part order or size mismatch")
        check_digest(part["sha256"])
    core = {k: v for k, v in value.items() if k != "transfer_id"}
    if value["transfer_id"] != digest(encoded(core))[:32]:
        raise TransferError("Manifest identity mismatch")
    if len(encoded(value)) > MAX_MANIFEST:
        raise TransferError("Manifest exceeds guest create limit")
    return value


def actions(manifest):
    m = validate_manifest(manifest)
    tid = m["transfer_id"]
    result = []
    for part in m["parts"]:
        name = "part-%04d" % part["index"]
        result.append(dict(action=name, path=artifact_path(m["target"], tid, name),
                           idempotency_key=key(tid, name), size=part["size"],
                           sha256=part["sha256"]))
    raw = encoded(m)
    result.append(dict(action="manifest", path=artifact_path(m["target"], tid, "manifest"),
                       idempotency_key=key(tid, "manifest"), size=len(raw), sha256=digest(raw)))
    result.append(dict(action="assemble", path=m["target"], idempotency_key=key(tid, "assemble"),
                       size=m["size"], sha256=m["sha256"]))
    return result


def plan(source, output, box, target, work_ref, transfer_ref, chunk_size=MAX_CHUNK):
    data = bounded_read(source, MAX_BYTES)
    manifest, chunks = make_manifest(data, box, target, work_ref, transfer_ref, chunk_size)
    directory = Path(output).absolute()
    if directory.parent.resolve(strict=True) != directory.parent:
        raise TransferError("Bundle parent must be a canonical directory")
    directory.mkdir(mode=0o700)  # Never regenerate or replace an existing intent.
    exclusive(directory / "manifest.json", encoded(manifest))
    rows = actions(manifest)
    for row, content in zip(rows[:-1], [*chunks, encoded(manifest)]):
        args = {"box_id": box, "path": row["path"], "mode": "create",
                "content_base64": base64.b64encode(content).decode("ascii"),
                "idempotency_key": row["idempotency_key"], "work_ref": work_ref}
        exclusive(directory / (row["action"] + "-request.json"), encoded(args))
    journal = directory / "journal"
    journal.mkdir(mode=0o700)
    publish_event(journal, 0, {"kind": "init", "bundle": str(directory),
                               "manifest_sha256": digest(encoded(manifest)),
                               "transfer_id": manifest["transfer_id"]})
    sync_directory(directory)
    sync_directory(directory.parent)
    return {"status": "PREPARED", "transfer_id": manifest["transfer_id"],
            "parts": len(chunks), "size": len(data), "sha256": manifest["sha256"],
            "directory": str(directory), "actions": rows,
            "note": "Use begin on this canonical bundle before every dispatch; payloads contain source bytes. No calls issued."}


def verify_pages(pages, row, box):
    if not isinstance(pages, list) or not 1 <= len(pages) <= 256:
        raise TransferError("Missing or excessive readback pages")
    whole = hashlib.sha256()
    offset = 0
    for index, page in enumerate(pages):
        if (not isinstance(page, dict) or page.get("box_id") != box
                or page.get("path") != row["path"] or type(page.get("size")) is not int
                or page["size"] != row["size"] or type(page.get("offset")) is not int
                or page["offset"] != offset):
            raise TransferError("Readback target, size or offset mismatch")
        value = page.get("data_base64")
        if not isinstance(value, str) or len(value) > 87384:
            raise TransferError("Invalid readback payload")
        data = base64.b64decode(value, validate=True)
        if not 1 <= len(data) <= 65536 or digest(data) != page.get("page_sha256"):
            raise TransferError("Readback page size or digest mismatch")
        offset += len(data)
        last = index == len(pages)-1
        if (page.get("ended") is not last or page.get("whole_file") is not (len(pages)==1)
                or offset > row["size"] or (not last and offset == row["size"])):
            raise TransferError("Readback pagination mismatch")
        whole.update(data)
    if offset != row["size"] or whole.hexdigest() != row["sha256"]:
        raise TransferError("Independent whole-file readback mismatch")
    return whole.hexdigest()


def _assess_snapshot(manifest, journal):
    rows = actions(manifest)
    if (not isinstance(journal, dict) or set(journal) != {"transfer_id", "records"}
            or journal["transfer_id"] != manifest["transfer_id"]
            or not isinstance(journal["records"], dict)
            or set(journal["records"]) - {r["action"] for r in rows}):
        raise TransferError("Journal does not match transfer")
    results = []
    receipts = set()
    for row in rows:
        record = journal["records"].get(row["action"])
        status = "RECONCILE"
        reason = "Missing original intent/receipt; do not replay"
        if record is not None:
            if (not isinstance(record, dict)
                    or set(record) != {"idempotency_key","operation_id","state","pages","exit_code"}
                    or record["idempotency_key"] != row["idempotency_key"]
                    or not isinstance(record["pages"], list)):
                raise TransferError("Journal record identity mismatch")
            op = record["operation_id"]
            if op is not None:
                if not isinstance(op, str) or not re.fullmatch(r"[0-9a-f]{32}", op) or op in receipts:
                    raise TransferError("Invalid or reused original Operation ID")
                receipts.add(op)
            state = record["state"]
            if state == "NOT_SUBMITTED":
                if op is not None or record["pages"] or record["exit_code"] is not None:
                    raise TransferError("Not-submitted record contradicts original evidence")
                status, reason = "READY", "Explicit original not-submitted intent; caller must check authority"
            elif state == "SUCCEEDED" and op is not None:
                if row["action"] == "assemble" and (type(record["exit_code"]) is not int or record["exit_code"] != 0):
                    status, reason = "RECONCILE", "Assembly needs original observed guest exit zero"
                elif row["action"] != "assemble" and record["exit_code"] is not None:
                    raise TransferError("File create must not invent a guest exit code")
                elif not record["pages"]:
                    status, reason = "READBACK_REQUIRED", "Read original target; do not recreate"
                else:
                    try:
                        verify_pages(record["pages"], row, manifest["box_id"])
                    except (TransferError, ValueError) as exc:
                        status, reason = "RECONCILE", str(exc)
                    else:
                        status, reason = "VERIFIED", "Original success and supplied independent bytes match"
            else:
                # Unknown/missing/failed/nonterminal acknowledgements are never
                # converted into permission to retry, even if bytes exist.
                reason = "Inspect original Operation or reconcile unknown acknowledgement; do not replay"
        results.append(dict(action=row["action"], status=status, reason=reason))
    blocked = [r for r in results if r["status"] == "RECONCILE"]
    reads = [r for r in results if r["status"] == "READBACK_REQUIRED"]
    if blocked:
        status, next_action = "RECONCILE", blocked[0]["action"]
    elif reads:
        status, next_action = "READBACK_REQUIRED", reads[0]["action"]
    elif all(r["status"] == "VERIFIED" for r in results):
        status, next_action = "VERIFIED_READBACK", None
    else:
        status, next_action = "READY", next(r["action"] for r in results if r["status"] == "READY")
    # A submitted assembly without verified prerequisite receipts is not a
    # reason to send missing parts or recreate the final file.
    if results[-1]["status"] != "READY" and any(r["status"] != "VERIFIED" for r in results[:-1]):
        status, next_action = "RECONCILE", "assemble"
    return {"status": status, "next_action": next_action, "actions": results,
            "note": "Caller-supplied receipts checked for consistency only. No automatic calls, retry, or authority."}


def initial_journal(manifest):
    return {"transfer_id": manifest["transfer_id"],
            "records": {row["action"]: {"idempotency_key": row["idempotency_key"],
                "operation_id": None, "state": "NOT_SUBMITTED", "pages": [], "exit_code": None}
                for row in actions(manifest)}}


def sync_directory(path):
    fd = os.open(path, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


def publish_event(directory, expected, event):
    """Exclusive local CAS, fsync before returning a dispatch intent.

    If publication/acknowledgement is uncertain, inspect the canonical
    history; a published dispatch event is never a second permission.
    """
    if type(expected) is not int or not 0 <= expected < MAX_EVENTS:
        raise TransferError("Invalid or exhausted journal revision")
    raw = encoded(dict(event, revision=expected + 1))
    if len(raw) > MAX_JOURNAL:
        raise TransferError("Journal event exceeds bound")
    fd = os.open(directory, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    temp = None
    try:
        name = ".pending-" + secrets.token_hex(16)
        out = os.open(name, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW,
                      0o600, dir_fd=fd)
        temp = name
        with os.fdopen(out, "wb") as stream:
            stream.write(raw)
            stream.flush()
            os.fsync(stream.fileno())
        destination = f"{expected+1:08d}.json"
        os.link(temp, destination, src_dir_fd=fd, dst_dir_fd=fd, follow_symlinks=False)
        os.fsync(fd)
    finally:
        if temp is not None:
            os.unlink(temp, dir_fd=fd)
        os.close(fd)


def request_for(bundle, manifest, action):
    rows = {row["action"]: row for row in actions(manifest)}
    if action not in rows:
        raise TransferError("Unknown action")
    row = rows[action]
    if action == "assemble":
        request = {"box_id": manifest["box_id"], "work_ref": manifest["work_ref"],
                   "idempotency_key": row["idempotency_key"],
                   "manifest_path": rows["manifest"]["path"], "target": row["path"]}
        return request, digest(encoded(request))
    path = bundle / (action + "-request.json")
    raw = bounded_read(path, MAX_MANIFEST)
    request = strict_load(raw)
    if not isinstance(request, dict) or set(request) != {
            "box_id", "path", "mode", "content_base64", "idempotency_key", "work_ref"}:
        raise TransferError("Invalid prepared request")
    expected = {"box_id": manifest["box_id"], "path": row["path"], "mode": "create",
                "idempotency_key": row["idempotency_key"], "work_ref": manifest["work_ref"]}
    if any(request.get(k) != v for k, v in expected.items()) or raw != encoded(request):
        raise TransferError("Prepared request identity changed")
    if not isinstance(request["content_base64"], str):
        raise TransferError("Invalid prepared payload")
    content = base64.b64decode(request["content_base64"], validate=True)
    if len(content) != row["size"] or digest(content) != row["sha256"]:
        raise TransferError("Prepared payload does not match manifest")
    return request, digest(raw)


def apply_journal_event(bundle, manifest, current, event):
    if not isinstance(event, dict):
        raise TransferError("Invalid journal event")
    action = event.get("action")
    if action not in current["records"]:
        raise TransferError("Unknown journal action")
    record = current["records"][action]
    kind = event.get("kind")
    if kind == "dispatch":
        if set(event) != {"kind", "action", "request_sha256", "revision"}:
            raise TransferError("Invalid dispatch record")
        status = _assess_snapshot(manifest, current)
        if status["status"] != "READY" or status["next_action"] != action:
            raise TransferError("Dispatch is not next READY action; reconcile original intent")
        _, expected = request_for(bundle, manifest, action)
        if event["request_sha256"] != expected:
            raise TransferError("Dispatch request identity changed")
        record["state"] = "DISPATCHING"
    elif kind == "receipt":
        if set(event) != {"kind", "action", "record", "revision"}:
            raise TransferError("Invalid receipt event")
        new = event["record"]
        if not isinstance(new, dict) or set(new) != set(record):
            raise TransferError("Invalid receipt record")
        if new["idempotency_key"] != record["idempotency_key"]:
            raise TransferError("Original idempotency key changed")
        if record["state"] == "NOT_SUBMITTED":
            raise TransferError("Receipt requires durable pre-dispatch intent")
        state = new["state"]
        if not isinstance(state, str) or not re.fullmatch(r"[A-Z][A-Z_]{0,31}", state) or state in ("NOT_SUBMITTED", "DISPATCHING"):
            raise TransferError("Receipt cannot reset dispatch state")
        op = new["operation_id"]
        if op is None and (state != "UNKNOWN" or new["pages"] or new["exit_code"] is not None):
            raise TransferError("Missing original receipt; retain UNKNOWN")
        if record["operation_id"] is not None and op != record["operation_id"]:
            raise TransferError("Original Operation ID cannot change")
        if record["state"] in ("SUCCEEDED", "FAILED", "CANCELLED", "REJECTED", "TIMED_OUT") and state != record["state"]:
            raise TransferError("Terminal Operation state cannot change")
        if record["exit_code"] is not None and new["exit_code"] != record["exit_code"]:
            raise TransferError("Original guest exit code cannot change")
        if not isinstance(new["pages"], list) or (state != "SUCCEEDED" and
                (new["pages"] or new["exit_code"] is not None)):
            raise TransferError("Readback requires observed success")
        if new["exit_code"] is not None and (action != "assemble" or type(new["exit_code"]) is not int):
            raise TransferError("Invalid original assembly exit code")
        prior = _assess_snapshot(manifest, current)
        if next(row for row in prior["actions"] if row["action"] == action)["status"] == "VERIFIED":
            raise TransferError("Verified original evidence is immutable")
        current["records"][action] = copy.deepcopy(new)
    else:
        raise TransferError("Unknown journal event kind")
    # Strict consistency checks, even for a recorded refusal or unknown.
    _assess_snapshot(manifest, current)
    return current


def load_bundle(bundle):
    bundle = Path(bundle).absolute()
    if bundle.resolve(strict=True) != bundle or not bundle.is_dir():
        raise TransferError("Expected canonical original bundle directory")
    raw = bounded_read(bundle / "manifest.json", MAX_MANIFEST)
    manifest = validate_manifest(strict_load(raw))
    if raw != encoded(manifest):
        raise TransferError("Manifest canonical bytes changed")
    directory = bundle / "journal"
    if directory.is_symlink() or not directory.is_dir():
        raise TransferError("Canonical journal missing; legacy snapshots cannot establish READY")
    paths = sorted(p for p in directory.iterdir() if re.fullmatch(r"[0-9]{8}\.json", p.name))
    if not 1 <= len(paths) <= MAX_EVENTS:
        raise TransferError("Missing or excessive journal history")
    current = initial_journal(manifest)
    total = 0
    for number, path in enumerate(paths, 1):
        if path.name != f"{number:08d}.json":
            raise TransferError("Journal revision gap; preserve and reconcile")
        data = bounded_read(path, MAX_JOURNAL)
        total += len(data)
        if total > MAX_JOURNAL_TOTAL:
            raise TransferError("Journal history exceeds bound; preserve and reconcile")
        event = strict_load(data)
        if not isinstance(event, dict) or type(event.get("revision")) is not int or event["revision"] != number or encoded(event) != data:
            raise TransferError("Invalid immutable journal revision")
        if number == 1:
            if event != {"kind": "init", "bundle": str(bundle),
                         "manifest_sha256": digest(raw), "transfer_id": manifest["transfer_id"],
                         "revision": 1}:
                raise TransferError("Journal does not bind original bundle and manifest")
        else:
            apply_journal_event(bundle, manifest, current, event)
    return bundle, manifest, directory, current, len(paths), total


def assess(bundle):
    _, manifest, _, current, revision, _ = load_bundle(bundle)
    result = _assess_snapshot(manifest, current)
    for row in result["actions"]:
        if row["status"] == "READY" and (result["status"] != "READY" or row["action"] != result["next_action"]):
            row.update(status="WAITING", reason="Not the next dispatchable action; use canonical begin")
    return dict(result, journal_revision=revision)


def append_event(bundle, expected, event):
    bundle, manifest, directory, current, revision, total = load_bundle(bundle)
    if type(expected) is not int or expected != revision:
        raise TransferError("Journal revision conflict; inspect original history")
    candidate = dict(event, revision=revision+1)
    apply_journal_event(bundle, manifest, current, candidate)
    if total + len(encoded(candidate)) > MAX_JOURNAL_TOTAL:
        raise TransferError("Journal history exceeds bound; preserve and reconcile")
    publish_event(directory, revision, event)
    return bundle, manifest, revision+1


def begin(bundle, expected, action, *, emit_request=False):
    original, manifest, _, _, revision, _ = load_bundle(bundle)
    if revision != expected:
        raise TransferError("Journal revision conflict; inspect original history")
    request, request_sha = request_for(original, manifest, action)
    _, _, revision = append_event(original, expected,
                                  {"kind": "dispatch", "action": action, "request_sha256": request_sha})
    # Only this newly published successful claim is the next dispatch intent;
    # status/resume never returns a replacement permission.
    result = {"status": "DISPATCH_INTENT_RECORDED", "action": action,
              "journal_revision": revision, "request_sha256": request_sha,
              "idempotency_key": request["idempotency_key"], "box_id": manifest["box_id"],
              "work_ref": manifest["work_ref"],
              "note": "Caller may send this exact request once only with existing authority. If interrupted or result uncertain, reconcile original intent; never begin again."}
    if action == "assemble":
        result["manifest_path"] = request["manifest_path"]
        result["target"] = request["target"]
    else:
        result["request_file"] = str(original / (action + "-request.json"))
    if emit_request:
        # Return the exact canonical ASCII bytes, not a second payload
        # representation or another dispatch authorization. The caller
        # checks the digest and uses only this fresh successful response.
        raw = encoded(request)
        if digest(raw) != request_sha:
            raise TransferError("Prepared request changed; reconcile original claim")
        result["request_json"] = raw.decode("ascii")
    return result


def record_receipt(bundle, expected, action, receipt):
    original, _, revision = append_event(bundle, expected, {"kind": "receipt", "action": action, "record": receipt})
    return dict(assess(original), recorded_revision=revision)



def advance(bundle, expected, action, receipt, next_action):
    """Append one actual receipt, then explicitly claim the adjacent action.

    Two independent durable CAS events, not a transaction or dispatcher.
    Failure between them leaves READY; failure after the next claim leaves
    RECONCILE. Never return a previously published claim on resume.
    """
    original, manifest, _, _, revision, _ = load_bundle(bundle)
    names = [row["action"] for row in actions(manifest)]
    if revision != expected or action not in names[:-1] or names[names.index(action)+1] != next_action:
        raise TransferError("Expected current revision and exact adjacent next action")
    recorded = record_receipt(original, expected, action, receipt)
    if recorded["status"] != "READY" or recorded["next_action"] != next_action:
        return {"status": "ADVANCE_NOT_READY", "recorded": recorded, "claim": None,
                "note": "Receipt preserved; no next dispatch intent. Reconcile original evidence."}
    claim = begin(original, expected+1, next_action, emit_request=True)
    return {"status": "NEXT_INTENT_RECORDED", "recorded_revision": expected+1,
            "claim": claim,
            "note": "Two separate durable revisions. Only this fresh claim may be sent once; no automatic dispatch."}


def record_assembly_pages(bundle, expected, operation_id, exit_code, count):
    """Compose the assembly receipt from bounded original MCP page files.

    Staging page JSON is a caller-controlled evidence transfer. Do not
    substitute guest-local target reads for independent MCP readback.
    """
    original, manifest, _, _, revision, _ = load_bundle(bundle)
    if (type(expected) is not int or revision != expected
            or type(count) is not int or not 1 <= count <= 256
            or type(exit_code) is not int or exit_code != 0
            or not isinstance(operation_id, str) or not re.fullmatch(r"[0-9a-f]{32}", operation_id)):
        raise TransferError("Expected exact revision, original successful assembly and bounded page count")
    names = [f"assembly-page-{i:04d}.json" for i in range(count)]
    present = sorted(p.name for p in original.iterdir()
                     if re.fullmatch(r"assembly-page-[0-9]{4}\.json", p.name))
    if present != names:
        raise TransferError("Assembly evidence page set differs; preserve and reconcile")
    pages = [strict_load(bounded_read(original / name, MAX_MANIFEST)) for name in names]
    row = actions(manifest)[-1]
    # Validate all independent bytes before publishing any final receipt.
    verify_pages(pages, row, manifest["box_id"])
    receipt = {"idempotency_key": row["idempotency_key"], "operation_id": operation_id,
               "state": "SUCCEEDED", "pages": pages, "exit_code": exit_code}
    return record_receipt(original, expected, "assemble", receipt)


def assemble(manifest_path):
    raw = bounded_read(manifest_path, MAX_MANIFEST)
    manifest = validate_manifest(strict_load(raw))
    rows = actions(manifest)
    path = Path(manifest_path)
    expected = Path(rows[-2]["path"])
    if path != expected or raw != encoded(manifest):
        raise TransferError("Manifest path or canonical bytes mismatch")
    target = Path(manifest["target"])
    parent = target.parent
    if parent.resolve(strict=True) != parent:
        raise TransferError("Target parent must be canonical, not a symlink")
    # All temporary/part/final names are in the same reviewed guest directory.
    # The caller must ensure no hostile concurrent modification of that guest.
    dirfd = os.open(parent, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    temp_name = None
    try:
        try:
            os.stat(target.name, dir_fd=dirfd, follow_symlinks=False)
        except FileNotFoundError:
            pass
        else:
            raise TransferError("Final target already exists; read/reconcile, never overwrite")
        proposed_temp = ".bwtr-assembly-" + secrets.token_hex(16)
        fd = os.open(proposed_temp, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW,
                     0o600, dir_fd=dirfd)
        temp_name = proposed_temp
        combined = hashlib.sha256()
        size = 0
        with os.fdopen(fd, "wb") as output:
            for row in rows[:-2]:
                part_name = Path(row["path"]).name
                part_fd = os.open(part_name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=dirfd)
                with os.fdopen(part_fd, "rb") as stream:
                    before = os.fstat(stream.fileno())
                    if not stat.S_ISREG(before.st_mode) or before.st_size != row["size"]:
                        raise TransferError("Missing or invalid part size")
                    data = stream.read(row["size"] + 1)
                    after = os.fstat(stream.fileno())
                    if (len(data) != row["size"] or digest(data) != row["sha256"]
                            or (before.st_size, before.st_mtime_ns, before.st_ctime_ns)
                            != (after.st_size, after.st_mtime_ns, after.st_ctime_ns)):
                        raise TransferError("Part changed, reordered or digest mismatch")
                output.write(data)
                combined.update(data)
                size += len(data)
            if size != manifest["size"] or combined.hexdigest() != manifest["sha256"]:
                raise TransferError("Whole assembly digest mismatch")
            output.flush()
            os.fsync(output.fileno())
        # Exclusive hard-link publication; no unsafe rename/replace fallback.
        os.link(temp_name, target.name, src_dir_fd=dirfd, dst_dir_fd=dirfd, follow_symlinks=False)
        os.fsync(dirfd)
    finally:
        if temp_name is not None:
            os.unlink(temp_name, dir_fd=dirfd)
        os.close(dirfd)
    return {"status": "ASSEMBLED_READBACK_REQUIRED", "path": str(target),
            "size": size, "sha256": combined.hexdigest(),
            "note": "No remote readback or native crash-durability proof; preserve all original parts and receipts."}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="action", required=True)
    prepare = sub.add_parser("plan")
    for name in ("source","output","box","target","work-ref","transfer-ref"):
        prepare.add_argument("--"+name, required=True)
    prepare.add_argument("--chunk-size", type=int, default=MAX_CHUNK)
    status = sub.add_parser("assess", help="Read only the latest canonical bundle history")
    status.add_argument("bundle")
    start = sub.add_parser("begin", help="Durably record exact pre-dispatch intent, never dispatch")
    start.add_argument("bundle")
    start.add_argument("--expect", type=int, required=True)
    start.add_argument("--action", dest="target_action", required=True)
    start.add_argument("--emit-request", action="store_true", help="Return exact canonical request JSON with the fresh claim")
    receipt = sub.add_parser("record", help="Append original receipt and independent readback")
    receipt.add_argument("bundle")
    receipt.add_argument("--expect", type=int, required=True)
    receipt.add_argument("--action", dest="target_action", required=True)
    receipt.add_argument("--receipt", required=True)
    step = sub.add_parser("advance", help="Record previous receipt and claim explicit next action; no dispatch")
    step.add_argument("bundle")
    step.add_argument("--expect", type=int, required=True)
    step.add_argument("--action", dest="target_action", required=True)
    step.add_argument("--receipt", required=True)
    step.add_argument("--next-action", required=True)
    pages = sub.add_parser("record-assembly-pages", help="Record original assembly receipt from staged MCP page JSON")
    pages.add_argument("bundle")
    pages.add_argument("--expect", type=int, required=True)
    pages.add_argument("--operation-id", required=True)
    pages.add_argument("--exit-code", type=int, required=True)
    pages.add_argument("--count", type=int, required=True)
    join = sub.add_parser("assemble")
    join.add_argument("manifest")
    args = parser.parse_args(argv)
    try:
        if args.action == "plan":
            result = plan(args.source,args.output,args.box,args.target,args.work_ref,
                          args.transfer_ref,args.chunk_size)
        elif args.action == "assess":
            result = assess(args.bundle)
        elif args.action == "begin":
            result = begin(args.bundle, args.expect, args.target_action, emit_request=args.emit_request)
        elif args.action == "record":
            result = record_receipt(args.bundle, args.expect, args.target_action,
                                    strict_load(bounded_read(args.receipt, MAX_JOURNAL)))
        elif args.action == "advance":
            result = advance(args.bundle, args.expect, args.target_action,
                             strict_load(bounded_read(args.receipt, MAX_JOURNAL)), args.next_action)
        elif args.action == "record-assembly-pages":
            result = record_assembly_pages(args.bundle, args.expect, args.operation_id,
                                           args.exit_code, args.count)
        else:
            result = assemble(args.manifest)
    except (OSError, ValueError, TypeError, KeyError) as exc:
        print(json.dumps({"status": "REVIEW_REQUIRED", "reason": str(exc),
                          "note": "Preserve original intent and effects; never blindly replay."}), file=sys.stderr)
        return 2
    print(json.dumps(result, sort_keys=True, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
