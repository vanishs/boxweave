"""Immutable local JSON revisions; not a distributed lock or sync protocol."""
import json
import os
import re
import tempfile
from pathlib import Path

from .checkpoint import unique_keys
from .common import HarnessError


def read_json(path):
    try:
        with Path(path).open(encoding="utf-8") as stream:
            return json.load(stream, object_pairs_hook=unique_keys,
                             parse_constant=lambda value: (_ for _ in ()).throw(ValueError(value)))
    except (OSError, ValueError) as exc:
        raise HarnessError("Cannot read strict JSON record") from exc


def latest(directory):
    directory = Path(directory)
    if not directory.is_dir() or directory.is_symlink():
        raise HarnessError("Record directory missing or unsafe")
    paths = sorted(p for p in directory.iterdir() if re.fullmatch(r"[0-9]{8}\.json", p.name))
    if not paths:
        raise HarnessError("No published revision")
    for index, path in enumerate(paths, 1):
        if path.name != f"{index:08d}.json" or path.is_symlink() or not path.is_file():
            raise HarnessError("Record history has a gap or unsafe entry")
        data = read_json(path)
        if not isinstance(data, dict) or type(data.get("revision")) is not int or data["revision"] != index:
            raise HarnessError("Revision does not match record history")
    return data


def publish(directory, expected, data):
    """Publish one complete revision with exclusive hard-link creation.

    Writers of the same revision cannot overwrite each other on a supported
    local filesystem. Remote Git histories require separate reconciliation.
    """
    directory = Path(directory)
    if type(expected) is not int or not 0 <= expected < 99999999:
        raise HarnessError("Invalid expected revision")
    if expected == 0:
        try:
            directory.mkdir(mode=0o700)
        except FileExistsError as exc:
            raise HarnessError("Record directory already exists; inspect it") from exc
    elif latest(directory)["revision"] != expected:
        raise HarnessError("Revision conflict; reload before editing")
    data = dict(data, revision=expected + 1)
    destination = directory / f"{expected + 1:08d}.json"
    fd, name = tempfile.mkstemp(prefix=".pending-", dir=directory)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as stream:
            json.dump(data, stream, sort_keys=True, indent=2, allow_nan=False)
            stream.write("\n")
            stream.flush()
            os.fsync(stream.fileno())
        try:
            os.link(name, destination)
        except FileExistsError as exc:
            raise HarnessError("Revision conflict; reload before editing") from exc
        except OSError as exc:
            raise HarnessError("Exclusive local publication unavailable") from exc
    finally:
        os.unlink(name)
    return data
