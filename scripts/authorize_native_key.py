"""Authorize only a public Ed25519 key for this non-root user's fixed MCP launcher."""
import argparse
import base64
import fcntl
import hashlib
import json
import os
from pathlib import Path
import pwd
import re
import socket
import stat
import struct
import tempfile
import time
import uuid

LAUNCHER = "/usr/local/bin/boxweave-mcp"


class Rejected(Exception):
    pass


def checked(path, directory=False, private=True):
    path = Path(path).absolute()
    for item in (path, *path.parents):
        if item.is_symlink():
            raise Rejected("UNSAFE_PATH")
    info = path.stat()
    if info.st_uid != os.geteuid() or info.st_mode & (0o077 if private else 0o022):
        raise Rejected("PATH_NOT_PRIVATE")
    if directory != stat.S_ISDIR(info.st_mode):
        raise Rejected("UNSAFE_PATH")
    if not directory and (not stat.S_ISREG(info.st_mode) or info.st_nlink != 1):
        raise Rejected("UNSAFE_PATH")
    return path


def public_key(text):
    match = re.fullmatch(r"ssh-ed25519 ([A-Za-z0-9+/]+={0,2})(?: [A-Za-z0-9@._-]{1,80})?", text.strip())
    if not match:
        raise Rejected("PUBLIC_KEY_INVALID")
    try:
        blob = base64.b64decode(match[1], validate=True)
    except ValueError:
        raise Rejected("PUBLIC_KEY_INVALID") from None
    if len(blob) != 51 or blob[:19] != struct.pack(">I", 11)+b"ssh-ed25519"+struct.pack(">I", 32):
        raise Rejected("PUBLIC_KEY_INVALID")
    return match[1], blob


def authorize(home, text, expected_user, expected_hostname):
    if os.geteuid() == 0 or pwd.getpwuid(os.geteuid()).pw_name != expected_user or socket.gethostname() != expected_hostname:
        raise Rejected("IDENTITY_MISMATCH")
    encoded, blob = public_key(text)
    home = checked(home, directory=True, private=False)
    directory = home / ".ssh"
    if not directory.exists():
        directory.mkdir(mode=0o700)
    checked(directory, directory=True)
    lockfd = os.open(directory / ".boxweave-native.lock", os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    try:
        info = os.fstat(lockfd)
        if not stat.S_ISREG(info.st_mode) or info.st_uid != os.geteuid() or info.st_nlink != 1 or info.st_mode & 0o077:
            raise Rejected("UNSAFE_LOCK")
        deadline = time.monotonic() + 10
        while True:
            try:
                fcntl.flock(lockfd, fcntl.LOCK_EX | fcntl.LOCK_NB)
                break
            except BlockingIOError:
                if time.monotonic() >= deadline:
                    raise Rejected("ENROLLMENT_BUSY")
                time.sleep(0.05)
        target = directory / "authorized_keys"
        existed = target.exists()
        old = checked(target).read_bytes() if existed else b""
        if target.is_symlink() or len(old) > 262144:
            raise Rejected("UNSAFE_PATH")
        line = 'restrict,command="'+LAUNCHER+'" ssh-ed25519 '+encoded+' boxweave-v2-native'
        lines = old.decode("utf-8").splitlines()
        identical = line in lines
        matching = [entry for entry in lines if encoded in entry.split()]
        if matching and (not identical or len(matching) != 1):
            raise Rejected("KEY_CONFLICT")
        if identical:
            return {"code":"OK","changed":False,"backup":None}
        data = old + (b"" if not old or old.endswith(b"\n") else b"\n") + line.encode()+b"\n"
        fd, temporary = tempfile.mkstemp(prefix=".boxweave-public-", dir=directory)
        try:
            with os.fdopen(fd, "wb") as stream:
                stream.write(data); stream.flush(); os.fsync(stream.fileno())
            if target.exists() != existed or (existed and checked(target).read_bytes() != old):
                raise Rejected("AUTHORIZED_KEYS_CHANGED")
            backup = None
            if existed:
                backup = directory / ("authorized_keys.boxweave-"+uuid.uuid4().hex+".bak")
                descriptor = os.open(backup, os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_NOFOLLOW, 0o600)
                with os.fdopen(descriptor, "wb") as stream:
                    stream.write(old); stream.flush(); os.fsync(stream.fileno())
            os.replace(temporary, target)
            descriptor = os.open(directory, os.O_DIRECTORY)
            try:
                os.fsync(descriptor)
            finally:
                os.close(descriptor)
        finally:
            if Path(temporary).exists():
                Path(temporary).unlink()
        fingerprint = base64.b64encode(hashlib.sha256(blob).digest()).decode().rstrip("=")
        return {"code":"OK","changed":True,"backup":str(backup) if backup else None,
                "fingerprint":"SHA256:"+fingerprint}

    finally:
        os.close(lockfd)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--public-key-file", required=True)
    parser.add_argument("--expected-user", required=True)
    parser.add_argument("--expected-hostname", required=True)
    args = parser.parse_args()
    try:
        file = Path(args.public_key_file)
        if file.is_symlink() or file.stat().st_size > 1024:
            raise Rejected("PUBLIC_KEY_INVALID")
        if not Path(LAUNCHER).is_file() or not os.access(LAUNCHER, os.X_OK):
            raise Rejected("LAUNCHER_UNAVAILABLE")
        result = authorize(Path.home(), file.read_text(), args.expected_user, args.expected_hostname)
        print(json.dumps(result))
        return 0
    except Rejected as error:
        print(json.dumps({"code":str(error),"enrollment":"not_completed"}))
    except (OSError, ValueError, UnicodeError):
        print(json.dumps({"code":"STATE_UNAVAILABLE","enrollment":"not_completed"}))
    return 2


if __name__ == "__main__":
    raise SystemExit(main())
