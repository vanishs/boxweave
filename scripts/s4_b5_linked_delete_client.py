"""B5 disposable installed-MCP acceptance client. Only loopback, no auth bypass."""
import os
import sys
import re
from mcp_testkit import LoopbackMCP, MCPToolError

assert os.environ.get("GITHUB_ACTIONS") == "true"
assert os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
assert os.environ.get("B5_SCOPE") == "b5-disposable-20261119"
uuid = os.environ["B5_UUID"]
assert re.fullmatch(r"[0-9a-f-]{36}", uuid)
args = {"box_id": uuid, "idempotency_key": "b5-linked-peer-native-delete-001", "work_ref": "b5-linked-peer-native"}
with LoopbackMCP(token=os.environ["B5_TOKEN"], allow_mutations=True) as mcp:
    if sys.argv[1] == "admit":
        rec = mcp.call("box_delete", args).structured
        assert rec["action"] == "box_delete"
        assert re.fullmatch(r"[0-9a-f]{32}", rec["operation_id"])
        print("S4_B5_LINKED_PARENT=" + rec["operation_id"], flush=True)
    elif sys.argv[1] == "verify":
        parent = os.environ["B5_PARENT"]
        op = mcp.poll_operation(parent, seconds=100)
        assert op["state"] == "SUCCEEDED" and op.get("native_operation_id"), op
        duplicate = mcp.call("box_delete", args).structured
        assert duplicate["operation_id"] == parent and duplicate["duplicate"]
        print("S4_B5_LINKED_PARENT_MCP_REPLAY=PASS", flush=True)
    else:
        raise AssertionError("unexpected test phase")
