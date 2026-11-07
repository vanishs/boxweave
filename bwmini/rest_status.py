"""Normalize Containarium REST failures to standard gRPC/HTTP status semantics.

This module owns provider-specific compatibility patches. It never retries an
operation, claims a write had no effect, or converts a successful HTTP response
(including DebugContainer's state="missing") into an error.

The public semantic vocabulary is the canonical gRPC status code set (0..16)
and grpc-gateway's HTTPStatusFromCode mapping; no BoxWeave-invented provider
error codes are exposed by this adapter.
"""
from dataclasses import dataclass
from enum import IntEnum
import json
import re


class GRPCCode(IntEnum):
    OK = 0
    CANCELLED = 1
    UNKNOWN = 2
    INVALID_ARGUMENT = 3
    DEADLINE_EXCEEDED = 4
    NOT_FOUND = 5
    ALREADY_EXISTS = 6
    PERMISSION_DENIED = 7
    RESOURCE_EXHAUSTED = 8
    FAILED_PRECONDITION = 9
    ABORTED = 10
    OUT_OF_RANGE = 11
    UNIMPLEMENTED = 12
    INTERNAL = 13
    UNAVAILABLE = 14
    DATA_LOSS = 15
    UNAUTHENTICATED = 16


# Mirrors grpc-gateway runtime.HTTPStatusFromCode, including nonstandard 499.
_HTTP = {
    GRPCCode.CANCELLED: 499,
    GRPCCode.UNKNOWN: 500,
    GRPCCode.INVALID_ARGUMENT: 400,
    GRPCCode.DEADLINE_EXCEEDED: 504,
    GRPCCode.NOT_FOUND: 404,
    GRPCCode.ALREADY_EXISTS: 409,
    GRPCCode.PERMISSION_DENIED: 403,
    GRPCCode.RESOURCE_EXHAUSTED: 429,
    GRPCCode.FAILED_PRECONDITION: 400,
    GRPCCode.ABORTED: 409,
    GRPCCode.OUT_OF_RANGE: 400,
    GRPCCode.UNIMPLEMENTED: 501,
    GRPCCode.INTERNAL: 500,
    GRPCCode.UNAVAILABLE: 503,
    GRPCCode.DATA_LOSS: 500,
    GRPCCode.UNAUTHENTICATED: 401,
}
_GRPC_NAMES = {"".join(s.capitalize() for s in item.name.split("_")): item
               for item in GRPCCode if item is not GRPCCode.OK}
_GRPC_LINE = re.compile(r"rpc error: code = ([A-Za-z]+) desc = ([\s\S]*)\Z")
_GET = re.compile(r"/v1/containers/([a-z][a-z0-9-]{0,62})\Z")
# Unambiguous HTTP authentication/availability statuses; 400, 404, 409, 500
# do NOT identify a unique gRPC code without a trustworthy gRPC envelope.
_HTTP_UNAMBIGUOUS = {
    401: GRPCCode.UNAUTHENTICATED,
    403: GRPCCode.PERMISSION_DENIED,
    429: GRPCCode.RESOURCE_EXHAUSTED,
    501: GRPCCode.UNIMPLEMENTED,
    503: GRPCCode.UNAVAILABLE,
    504: GRPCCode.DEADLINE_EXCEEDED,
}
_MAX_ERROR_BYTES = 4096


@dataclass(frozen=True)
class NormalizedError:
    grpc_code: GRPCCode | None
    http_status: int
    upstream_http_status: int
    # "grpc": validated typed gRPC reply; "compatibility": exact upstream
    # defect; "http": HTTP-only mapping; "unclassified": ambiguous response.
    origin: str

    @property
    def typed(self):
        return self.origin in ("grpc", "compatibility")


def http_status_for(code):
    """Return grpc-gateway's canonical HTTP status, never a custom number."""
    return _HTTP[GRPCCode(code)]


def _unique(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("duplicate JSON key")
        result[key] = value
    return result


def _envelope(raw, http_status):
    if not isinstance(raw, bytes) or len(raw) > _MAX_ERROR_BYTES:
        return None
    try:
        result = json.loads(raw, object_pairs_hook=_unique)
    except (ValueError, UnicodeError):
        return None
    if (not isinstance(result, dict)
            or not {"code", "error", "message"} <= set(result)
            or set(result) - {"code", "error", "message", "details"}
            or type(result["code"]) is not int or result["code"] != http_status
            or not isinstance(result["error"], str)
            or not isinstance(result["message"], str)):
        return None
    return result


def _grpc_envelope(payload, http_status):
    if payload is None:
        return None
    match = _GRPC_LINE.fullmatch(payload["error"])
    if match is None or payload["message"] != match.group(2):
        return None
    code = _GRPC_NAMES.get(match.group(1))
    if code is None or http_status_for(code) != http_status:
        return None
    return code


def _legacy_missing_v0128(payload, method, route, upstream_http):
    """The v0.128.0 confirmed GetContainer 500 Unknown -> NotFound defect.

    Strict *whole-body*, route, target and project binding prevents unrelated
    backend, route, peer, authorization or storage errors becoming absence.
    The upstream fix belongs in Containarium; see BoxWeave issue #107.
    """
    target = _GET.fullmatch(route)
    if (upstream_http != 500 or method != "GET" or target is None
            or payload is None
            or set(payload) != {"error", "code", "message"}):
        return False
    detail = (
        'failed to get container: failed to get container: '
        'Failed to fetch instance "' + target.group(1) +
        '-container" in project "default": Instance not found'
    )
    return (payload["message"] == detail
            and payload["error"] == "rpc error: code = Unknown desc = " + detail)


def normalize(method, route, upstream_http_status, raw_body):
    """Classify a single HTTP error. Uncertain semantics remain unclassified.

    Only the verified v0.128.0 GET mismatch is patched to a different status.
    Other HTTP 500 failures, even if containing "not found", remain failures.
    HTTP 404 by itself is not sufficient proof for a *create* preflight,
    because the route itself may be absent on a different Provider version.
    """
    if type(upstream_http_status) is not int or not 400 <= upstream_http_status <= 599:
        raise ValueError("expected HTTP error status")
    payload = _envelope(raw_body, upstream_http_status)
    if _legacy_missing_v0128(payload, method, route, upstream_http_status):
        return NormalizedError(GRPCCode.NOT_FOUND, 404, 500, "compatibility")
    grpc_code = _grpc_envelope(payload, upstream_http_status)
    if grpc_code is not None:
        return NormalizedError(grpc_code, http_status_for(grpc_code),
                               upstream_http_status, "grpc")
    code = _HTTP_UNAMBIGUOUS.get(upstream_http_status)
    return NormalizedError(code, upstream_http_status,
                           upstream_http_status,
                           "http" if code is not None else "unclassified")
