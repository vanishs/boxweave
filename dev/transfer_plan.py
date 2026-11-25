#!/usr/bin/env python3
"""Bounded offline transfer preparation and exclusive guest assembly.

No network client, automatic dispatch, retries, target overwrite or deletion.
The caller must preserve original MCP intent, receipts and readback evidence.
"""
import argparse
import base64
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
    directory = Path(output)
    directory.mkdir(mode=0o700)  # Never regenerate or replace an existing intent.
    exclusive(directory / "manifest.json", encoded(manifest))
    rows = actions(manifest)
    for row, content in zip(rows[:-1], [*chunks, encoded(manifest)]):
        args = {"box_id": box, "path": row["path"], "mode": "create",
                "content_base64": base64.b64encode(content).decode("ascii"),
                "idempotency_key": row["idempotency_key"], "work_ref": work_ref}
        exclusive(directory / (row["action"] + "-request.json"), encoded(args))
    journal = {"transfer_id": manifest["transfer_id"],
               "records": {row["action"]: {"idempotency_key": row["idempotency_key"],
                   "operation_id": None, "state": "NOT_SUBMITTED", "pages": [], "exit_code": None}
                   for row in rows}}
    exclusive(directory / "intent.json", encoded(journal))
    return {"status": "PREPARED", "transfer_id": manifest["transfer_id"],
            "parts": len(chunks), "size": len(data), "sha256": manifest["sha256"],
            "directory": str(directory), "actions": rows,
            "note": "Persist intent before dispatch; payloads contain source bytes. No calls issued."}


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


def assess(manifest, journal):
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
    status = sub.add_parser("assess")
    status.add_argument("manifest")
    status.add_argument("journal")
    join = sub.add_parser("assemble")
    join.add_argument("manifest")
    args = parser.parse_args(argv)
    try:
        if args.action == "plan":
            result = plan(args.source,args.output,args.box,args.target,args.work_ref,
                          args.transfer_ref,args.chunk_size)
        elif args.action == "assess":
            result = assess(strict_load(bounded_read(args.manifest,MAX_MANIFEST)),
                            strict_load(bounded_read(args.journal,MAX_JOURNAL)))
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
