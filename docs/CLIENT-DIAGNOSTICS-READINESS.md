# Independent client diagnostics: development preparation

Research date: 2026-10-22 (America/Los_Angeles).
Source baseline: `b2c38cd08ce08b5adb7c89ed18955b586fca2df9`, BoxWeave 0.2.0.
Status: **implemented in 0.3.0.** `boxweave client --diagnose` generates the
diagnostic command; `--diagnose` / `-Diagnose` run it in the shipped
`clients/posix/setup.py` and `clients/windows/setup.ps1`, covered by
`tests/test_diagnostics.py` and `tests/test_windows_diagnostics.py`. The
2026-10-22 research text below is kept as design background; where shipped
behavior differs from the proposal, the shipped behavior wins and is noted
inline.

## Recommended first increment

Provide a diagnostic entry point independent of enrollment. Reuse the existing
Windows PowerShell and POSIX client artifacts, with an early `--diagnose` /
`-Diagnose` branch that cannot reach setup mutations. Add `boxweave client
--diagnose` as a server-side command generator, preserving existing setup CLI
behavior. (Both command names shipped in 0.3.0.)

The generated command runs on the client, using its already enrolled transport
identity and pinned host key. It checks the connection and separately observes
the supported agent registration. It never repairs registration or enrollment.
It must also run from a saved local artifact and public descriptor when server
command generation is unavailable. A trusted saved descriptor must contain the
same minimum public connection fields; it contains no JWT or private-key text.

Use the current two single-file client assets for the MVP. Do not add remote
Python-module imports to a downloaded one-file script. Share probe functions
between setup and diagnosis within each artifact; use the same fixtures and
result contract across languages. A future shared package is a separate decision.

The generator executes on Ubuntu; diagnosis executes in native Windows,
Windows-to-WSL transport, macOS, or the actual WSL CLI runtime. A successful server
probe cannot validate the client's environment or agent configuration.

## Research scope and actual results

The user authorized MCP/local-shell testing for this investigation. The test scope
was announced before execution: existing test container, non-root launcher,
temporary protocol fixtures, no enrollment, new containers, service or route
changes, ten-minute research test budget. Local fixture use was explicitly
authorized; workload development remains remote.

| Test | Actual result | What it establishes |
| --- | --- | --- |
| Platform MCP inventory | Six containers; call succeeded | Current outer MCP read-only call works |
| `connect` identity on `boxweave-test` | exit 0; `boxweave-test`, `boxweave-test-container` | Target execution identity |
| Runtime observation | exit 0; Python 3.12.3, UID 1001, installed launcher and CLI present | Suitable existing non-root protocol test target |
| Inner MCP session | exit 0; server `containarium-mcp-server` 0.114.2; negotiated 2024-11-05; one tools page, 101 tools; list_containers succeeded | Actual installed inner MCP authentication and read-only execution |
| Windows configuration regression | 2 tests passed, exit 0 under Windows PowerShell 5.1 | Existing isolated atomic writer/adapter fixtures work |
| Same regressions under PowerShell 7.6.5 | 2 tests passed, exit 0 | Same fixtures work in this runtime |
| POSIX probe fault fixture | Five cases completed, runner exit 0 | Current checkout's probe behavior, not macOS enrollment |
| Native probe fault fixture | Five cases completed under PowerShell 5.1.26100.5651 and 7.6.5, runner exit 0 | Actual extracted probe functions against a local fake peer |

The inner protocol test started `/usr/local/bin/boxweave-mcp` as the verified user,
sent initialize/initialized, bounded tools/list pagination and list_containers.
It returned selected metadata, never raw container inventory, backend stderr,
JWTs or private keys. Its subprocess was terminated after testing. Connect may
authorize its managed public key on a plain provider daemon; this test is not a
claim of zero provider-side writes. No host SSH credential was used.

POSIX fixtures ran in a temporary directory inside `boxweave-test` and cleaned it
on exit. They executed the current local `probe` function transmitted in memory,
not a claim that the installed scripts match this checkout. Native fixtures are
retained under ignored `outputs/client-diagnostics-research/`; they load named
functions through PowerShell AST and never execute setup's top-level body.

`/usr/local/libexec/boxweave` in the tested container exposed only `cli.py` among
top-level Python modules. Do not assume the new checkout's modules are installed
there. No source sync or server update was performed.

## Reproduced behavior and implications

All three tested probe environments produced the same acceptance decisions:

| Fixture | Current result | Required diagnostic behavior |
| --- | --- | --- |
| Valid handshake/list/read-only result | Accepted | passed |
| Required tool appears on page two | Rejected as tool unavailable | Follow nextCursor within budgets |
| Peer negotiates 2025-06-18 | Rejected as unsupported | Accept only an explicit tested version set; otherwise unsupported |
| tools/call returns empty object | Accepted | Reject malformed tool result; do not claim authenticated tool success |
| tools/call returns isError=true | Rejected | provider_call_failed; do not print raw provider payload |

A fixture runner's exit 0 means the experiment completed; it does not mean the
observed empty-result acceptance is correct. No claim is made that the live
provider returns that malformed response. Protocol rejection is a compatibility
limit, not an authentication failure or evidence that all newer versions work.

Additional source findings:

- `clients/posix/setup.py:probe` and Windows `Test-BoxWeaveMcp` duplicate protocol
  handling and lack tools/list pagination and explicit CallToolResult validation.
- POSIX `send` uses blocking writes/flush outside its read timeout. Windows
  StandardInput WriteLine/Flush is also synchronous. A declared handshake timeout
  alone does not establish a total wall-clock bound.
- POSIX uses a bounded queue; Windows discards stderr asynchronously. Preserve
  output bounds and sanitized errors while adding stage visibility.
- Setup probes run after enrollment; calling setup is not a diagnostic shortcut.
- Existing server `doctor --api` is unauthenticated reachability, not client health.
- Current setup transports enforce pinned host checking, identity selection and
  batch authentication. Diagnosis should preserve that trust model.

## Input and configuration contract

Retain existing setup defaults. The server generator needs a new explicit action,
not a change that makes ordinary `boxweave client` run network probes. Existing
public descriptors/hashes and generated setup commands must remain valid.

Diagnostic input contains OS, runtime, transport, agent, endpoint, launcher and
trusted host identity; paths may be overridden explicitly in the selected runtime.
Use existing enrolled key and known-host files. Do not require a container public
key merely to list containers. Do not print the complete descriptor or command.
Select the WSL distribution explicitly; never choose another running distro as a
fallback. No WSL distro or package installation is part of diagnosis.

Separate three evidence sources:

1. Expected connection: trusted public descriptor or explicit reviewed profile.
2. Observed agent registration: read only the selected MCP entry and approved
   configuration fields; do not dump unrelated environment or server entries.
3. Connection attempt: run the diagnostic's validated OpenSSH command, not an
   arbitrary executable/remote command loaded from agent configuration.

For Codex, observe the supported global TOML entry and effective configured home;
for Cursor, observe mcp.json. Windows lacks a built-in general TOML parser: support
only the existing generated subset initially, with fixtures for spacing/comments,
and report other valid TOML spellings as unsupported rather than corrupt. POSIX
can use tomllib. Do not infer full effective agent configuration across all
project overlays. Claude CLI-owned registration inspection must be bounded and
parsed through an explicitly verified interface; where unavailable, report that
stage incomplete while allowing a separate connection result. Never treat a raw
CLI dump as safe diagnostic output.

Shipped registration coverage (0.3.0): Codex and ZCode entries are compared
field-by-field against the generated subset (ZCode also checks the stdio type
and the `features.mcp` flag); Cursor observes `mcp.json`; Claude Code's
CLI-owned registration reports `REGISTRATION_UNSUPPORTED` while still allowing
an independent connection result.

Preflight may stat the existing private-key path for presence/type/permissions;
the diagnostic must never open/hash/export its contents. OpenSSH uses that file
through its normal credential interface. Missing/unusable keys yield incomplete
or authentication failure; no keygen, enrollment or known_hosts rewriting.

## Stages and evidence

| Stage ID | Observation | Safe output |
| --- | --- | --- |
| input | Public profile is valid and internally consistent | profile ID and error code |
| runtime | Required runtime/OpenSSH and selected WSL distro available | observed version/runtime identifiers |
| registration | Selected supported MCP entry matches intended connection | present/matches/unsupported, no raw argv/env |
| trust | Required pin exists and matches trusted descriptor | matched/missing/mismatch |
| transport | Validated SSH process started or failed | process exit and narrow error category |
| initialize | MCP response negotiated a supported version | protocol and bounded server identity |
| discovery | Required read-only tool discovered | page count and tool presence |
| provider_call | list_containers returns valid successful CallToolResult | passed/denied/failed, no inventory |
| guidance | Expected instruction file/block observed | present/manual/unverified, not file contents |
| agent_ui | GUI recognized MCP and loaded instructions | unverified in automated MVP |

Transport authentication is proven by subsequent protocol/tool success, not by
SSH process creation. SSH often exits 255 for several causes. Do not claim an
exact authentication-vs-network-vs-host-key diagnosis without specific evidence.
Initially return transport_failed where certainty is insufficient. Optional
bounded stderr classification must discard raw lines and ignore unrecognized
text; messages from the network are untrusted. No raw verbose-log export.

Use BatchMode, pinned StrictHostKeyChecking and existing identity. Do not execute
unreviewed ssh_config ProxyCommand/LocalCommand/Match exec hooks by default. A
controlled config-free SSH invocation is preferred for generated direct profiles;
custom proxy/config requirements should be an explicitly unsupported MVP profile,
not silently executed or bypassed. Confirm actual Windows/WSL OpenSSH handling
with native fixtures before selecting the config-free syntax.

Do not use connect, create_container, token operations, route actions, shell
commands on the Ubuntu host or backend service probes as default tool checks.
list_containers is a specific provider contract; future providers need their own
explicit read-only tool adapter. Tool annotations alone are not authorization.

## Result schema and exit behavior

Proposed JSON example (illustrative, not an actual test receipt):

```json
{
  "schema_version": 1,
  "operation": "client_diagnose",
  "overall": "passed",
  "connection": "passed",
  "registration": "matched",
  "guidance": "present",
  "agent_ui": "unverified",
  "runtime": "windows-native",
  "transport": "wsl",
  "protocol_version": "2024-11-05",
  "tool_pages": 1,
  "stages": [
    {"id": "provider_call", "status": "passed", "code": "OK", "elapsed_ms": 18}
  ],
  "process_exit_code": null,
  "cleanup": "completed"
}
```

Stage statuses: passed, failed, incomplete. (The shipped diagnostic emits only
these three; skipped/unverified remain reserved and unused.) Overall passed
means every required automated stage passed; it never includes GUI verification.
Optional GUI/manual guidance observations must be clearly labeled. Registration
failure cannot be hidden by a separate connection success.

Exit 0: required automated checks passed. Exit 1: observed failure (invalid
response, mismatch, timeout, denied tool call or failed transport). Exit 2: invalid
input, missing prerequisite or unsupported/incomplete inspection. Unknown state
is not success. Invalid JSON input belongs to input/exit 2; invalid peer JSON
belongs to protocol/exit 1. Exit 130: operator interruption — POSIX `setup.py`
only; the Windows script has no dedicated interruption exit. On Windows,
argument-binding failures such as an out-of-range `-Timeout` are rejected by
PowerShell parameter validation and exit 1 rather than 2. Define precedence
as failed before incomplete before passed. Keep existing server status/doctor
exit semantics unchanged.

Use stable codes: INPUT_INVALID, RUNTIME_MISSING, PROFILE_UNSUPPORTED,
REGISTRATION_MISSING, REGISTRATION_MISMATCH, REGISTRATION_UNSUPPORTED,
PIN_MISSING, PIN_MISMATCH, TRANSPORT_FAILED, PROTOCOL_UNSUPPORTED,
PROTOCOL_INVALID, TOOL_UNAVAILABLE, PROVIDER_CALL_FAILED, RESPONSE_LIMIT,
DEADLINE_EXCEEDED and CLEANUP_INCOMPLETE, plus the shipped IDENTITY_MISSING,
WSL_PREFLIGHT_FAILED, INPUT_UNAVAILABLE, INTERRUPTED, FILE_PRESENT,
MANUAL_VERIFICATION_REQUIRED and REPORT_WRITE_FAILED. Remediation text maps from
codes;

Export only an explicit report path when requested, using a private atomic write
and a symlink/reparse-point check. JSON stdout contains exactly one object;
progress belongs on stderr. Reports omit user homes, full endpoint/argv, tokens,
container inventory and raw policy/configuration contents by default.

## Protocol and process requirements

- For the MVP, keep 2024-11-05 as the proven version. Add another revision only
  after versioned fixtures and a real compatible peer test. The client supports
  an explicit allowlist; unsupported negotiation produces a precise result.
- Validate JSON-RPC version, response ID/type, exclusive result/error shape,
  initialized ordering, capabilities, tools list and successful tool result shape.
  CallToolResult must have content as an array of valid bounded blocks; absent
  isError may mean false, but an empty result object is invalid. Do not require
  nonempty inventory/text: an empty inventory is valid. Unknown bounded extension
  fields can be ignored without copying them into reports.
- Continue nextCursor discovery until the required tool is found or no cursor
  remains. Treat cursors as opaque; detect repeats and bound total pages/bytes.
  Diagnostic discovery may stop on finding the target; complete inventory is not
  required. Do not persist cursors across sessions.
- Suggested initial budgets: 60-second total process budget including cleanup,
  15-second SSH connection limit, 2 MiB per protocol line, 8 MiB total response
  bytes, 16 discovery pages, bounded notification count and 4 KiB output strings.
  These are proposed defaults requiring fixtures, not measured requirements.
  Shipped: `--timeout` (default 30, range 1–300) bounds protocol I/O only;
  cleanup adds at most a few seconds and there is no fixed total process budget.
- Use a monotonic deadline; include writes, reads and cleanup. Guard the complete
  child process externally so a blocked pipe cannot defeat the deadline.
  Reserve a cleanup window within that total; do not reset it per request.
- Retain graceful stdin-close/termination, then bounded process-tree cleanup.
  Windows PowerShell 5.1 needs compatible process-tree handling; killing wsl.exe
  alone does not prove the remote launcher exited. Report cleanup uncertainty;
  do not claim that all remote descendants were cancelled.
- Reject unsupported inbound client requests with bounded JSON-RPC errors.
  No shell interpolation of paths, remote output or configuration strings.

## Files and implementation work packages

| Package | Likely changed files | Done when |
| --- | --- | --- |
| 1. Contract and probe extraction | both client setup files; client/MCP fixtures | Probe-only path has no setup side effects; same result fixtures pass |
| 2. Protocol robustness | same probe functions | Pagination, result validation and total deadlines verified |
| 3. Generator and assets | server/clients.py, install.sh, provenance handling | --diagnose command pins source/hash; existing setup commands unchanged |
| 4. Registration/guidance observations | client adapters | Supported forms compare safely; ambiguous forms yield incomplete |
| 5. Real client acceptance and docs | CLIENTS.md, CLI.md, troubleshooting, tests | Native Windows, real WSL bridge, WSL CLI and macOS evidence recorded |

No new generic gateway or registered per-box MCP server is needed. Avoid changing
server/mcp_client.py smoke semantics during initial extraction unless its own
tests justify it. Keep enrollment probes' success/failure behavior explicit and
update their fixtures when rejecting malformed responses.

Estimate: 1 day contract/extraction, 1-2 days protocol/process fixes, 1 day
generator/packaging, 1-2 days configuration observations and tests, plus 1-2 days
real-client acceptance. Total roughly 5-8 developer days; unavailable macOS/WSL
test access or agent-specific inspection contracts can extend this. This is a
planning estimate, not a fixed delivery promise.

## Acceptance matrix

Automated fixtures must cover valid/empty inventory, multi-page and repeated
cursors, unavailable tool, permission denial, malformed/empty result, invalid
JSON, wrong JSON-RPC ID/version, unsupported protocol, notification flood,
oversized output, blocked stdin, early stream close and nonzero process exit.
Test bounded cleanup, UTF-8/non-ASCII paths, spaces/quotes, CRLF, PowerShell 5.1
argv serialization, pin preservation, missing identity, incomplete config and
report redaction. Assert enrollment/config/key files are unchanged.

Use existing tests/test_clients.py, tests/test_mcp_client.py,
tests/test_windows_config.py and tests/test_windows_wsl.py where relevant; new
diagnostic tests should assert behavior rather than source-text patterns.
Run workload tests in the assigned remote LXC. Windows-native fixtures belong
on Windows; shim tests do not establish a real WSL installation result.

Release evidence must distinguish these independent claims:

- Controlled peer fixtures passed.
- Installed inner MCP passed (verified during this research).
- Real Windows/native transport passed (not tested here).
- Real Windows/WSL bridge and WSL CLI passed (not tested here).
- Real macOS enrollment-independent diagnostic passed (not tested here).
- Agent GUI recognition/guidance loading observed (not tested here).

Current research is sufficient to start contract/probe development. It is not
sufficient to publish universal client compatibility. No further broad platform
research or container provisioning is a prerequisite for the first increment.

## Official references

- [MCP lifecycle and version negotiation](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle):
  explicit version support and initialized ordering; versioned reference, not a
  claim that this revision is latest.
- [MCP pagination](https://modelcontextprotocol.io/specification/2025-06-18/server/utilities/pagination):
  tools/list cursor continuation and opaque cursor handling.
- [OpenSSH configuration manual](https://man.openbsd.org/ssh_config):
  BatchMode, trust/configuration controls and executable configuration hooks.
  Installed Windows/WSL behavior still needs its own compatibility tests.
