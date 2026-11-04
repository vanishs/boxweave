"""Conservative disk-resize preflight shared by REST and standalone MCP.

This check rejects known shrinks but is not an atomic Provider-side constraint
and cannot prove physical filesystem quota or concurrent resize semantics.
"""
import re

_MULTIPLIERS = {
    "B": 1, "KB": 10**3, "MB": 10**6, "GB": 10**9,
    "TB": 10**12, "KiB": 2**10, "MiB": 2**20,
    "GiB": 2**30, "TiB": 2**40,
}


def disk_bytes(value):
    if not isinstance(value, str) or len(value) > 32:
        raise ValueError("RESOURCE_DISK_SIZE_UNVERIFIED")
    match = re.fullmatch(r"([1-9][0-9]{0,12})(B|KB|MB|GB|TB|KiB|MiB|GiB|TiB)", value)
    if match is None:
        raise ValueError("RESOURCE_DISK_SIZE_UNVERIFIED")
    return int(match.group(1)) * _MULTIPLIERS[match.group(2)]


def require_nonshrinking_disk(requested, current):
    """Deny an unsupported size or an observed smaller target before writing.

    Provider concurrency/physical enforcement must be checked independently.
    """
    if disk_bytes(requested) < disk_bytes(current):
        raise ValueError("RESOURCE_DISK_SHRINK_REFUSED")
