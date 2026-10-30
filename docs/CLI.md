# BoxWeave management CLI — 0.11.0

The new `boxweave.sh` is a **read-only management and diagnostic entry**. From a reviewed source checkout use `bash boxweave.sh <command>`; from a verified installed release use `bash "$HOME/boxweave-v2/current/boxweave.sh" <command>`. It uses the package-local Python interpreter when present and does not elevate privileges. All successful responses are bounded JSON.

| Command | Meaning |
| --- | --- |
| `version [--components]` | Local VERSION and optionally independently observed Incus/Containarium client versions; no server inference |
| `status [--prefix PATH]` | App release/user-services/loopback observation using distribution health; no provider probe |
| `doctor [--prefix PATH] [--mcp]` | Status and host CLI versions, confinement unverified; `--mcp` explicitly checks authenticated MCP discovery/read |
| `box inspect NAME [--project PROJECT]` | Exact Incus read-only query and selected non-secret metadata; no suffix guess |
| `client` | Four selected profile names and guide only; no enrollment or script creation |

Exit: `0` for successful observation/ready status, `1` for read failure/failed status, `2` for malformed arguments or an incomplete/missing app. A product version is **not** a verified running release. CLI health does not prove effective confinement, boot readiness, Agent GUI, packet enforcement, public TLS or inference.

Old job/selftest, provider upgrade, host-key/token and client script generation commands are not recreated as unsafe aliases. Application mutations continue through explicit `python -m bwmini` plan/confirm actions. [Scope](product/SCOPE.md), [host bootstrap](HOST_BOOTSTRAP.md), [tests](verification/REMEDIATION_TEST_QUEUE.md).
