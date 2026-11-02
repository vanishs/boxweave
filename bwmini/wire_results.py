"""Pure, bounded Provider MCP result contracts: no transport, auth or I/O."""
import re
from .domain import Fault

def text(result):
    return "\n".join(b["text"] for b in result["content"])

def execution(result):
    structured = result.get("structuredContent")
    if isinstance(structured, dict):
        code, out = structured.get("exit_code"), structured.get("stdout")
        if type(code) is int and isinstance(out, str) and len(out) <= 4096:
            return code, out
        raise Fault("EXECUTION_FORMAT_UNSUPPORTED")
    # Only the observed platform envelope, never markers parsed out of guest text.
    match = re.fullmatch(r"exit_code: (-?\d+)\n\n--- stdout ---\n([\s\S]{0,4096})", text(result))
    if match is None:
        raise Fault("EXECUTION_FORMAT_UNSUPPORTED")
    return int(match[1]), match[2]

