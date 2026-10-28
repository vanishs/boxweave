# TASK-003a — trusted existing-identity native Codex registration (0.2.1)

Date: 2026-10-28 America/Los_Angeles. Scope SRC-053, actual evidence SRC-054. Child slice of TASK-003, selected after TASK-011. REQ-001/002/003/016/017 and AC-001/002/003/016/017 only within the scope below; not whole TASK-003/005 or release acceptance.

## Delivered scope and trust boundary

One fixed profile codex/windows/native and one stable server name boxweave. The operator supplies an already-authorized SSH identity, reviewed host pin/known_hosts and exact installed Codex CLI version. This is registration with an existing identity, not remote identity enrollment, credential provisioning, WSL/ZCode setup or a universal installer. AgentHome is explicit; the utility never chooses or edits the personal profile automatically. The operator must choose a private AgentHome and user-private TEMP location. Cache/staging/backup permissions inherit their parents; custom Windows ACLs are not independently attested by this slice.

Trusted entry clients/windows/bootstrap-native.ps1 verifies the two payload digests from an operator-delivered descriptor before executing either. Verified bytes are copied into a unique task cache before execution, avoiding a second read of unverified original paths. Reparse paths, unexpected payloads, oversized files, invalid schema/profile/hash or mismatch fail closed. The bootstrap itself and descriptor must arrive through an independently trusted operator channel; downloading a hash with an untrusted payload is not authentication. source_id=sha256:<register payload digest> identifies reviewed immutable bytes, not a fabricated Git-release signature. New payload/digest pairs require a new operator trust decision. Do not newline-convert verified payloads.

The descriptor example is intentionally invalid until completed. It carries public host/profile/hash/version/path metadata, never private-key/JWT contents. A local private-key path is not private-key material; SSH alone uses the existing key. Neither keys nor client sign-in files are copied into a package, box or another runtime.

## Registration behavior

register-native.ps1 verifies exact CLI version, local/reparse-safe paths and a real strict-SSH initialize/initialized/tools-list/read-only call using the separately verified probe before publication. It then validates snapshot and candidate config through actual codex mcp list --json in a private staged AgentHome. No model request, server enrollment or tool mutation occurs.

Fresh selected home gets a registration. Existing conflicting transport is preserved unless -Replace is explicitly selected and forwarded by the bootstrap. Repeating the utility's canonical registration is byte-idempotent. Replace updates only the selected server's transport fields; preserves other servers, model settings, comments, non-transport fields and nested tool approval tables. Obsolete selected transport environment/header/OAuth tables are removed deliberately; backups remain private. Inline/quoted selected-table forms or unsupported/multiline forms fail closed rather than risking a broad edit. Candidate must be parsed and recognized by the real CLI before replacing config.

The current observed codex-cli 0.165.0-alpha.1.2 mcp add replacement lost startup_timeout_sec and a nested tools.connect approval table in an isolated fixture. This is a version-specific observation, not a universal CLI claim. The product therefore uses a conservative targeted edit plus read-only CLI validation rather than delegating replacement to mcp add.

A named local mutex serializes this utility per home; a pre-publication hash detects observed external changes. File.Replace publishes with an original-byte backup, or File.Move creates an absent config. This is not atomic compare-and-exchange against non-cooperating external writers: keep other config editors idle, and inspect backups after uncertain publication. No automatic whole-profile/host rollback. Stage cleanup checks absolute parent boundary and unique task-owned basename before recursive removal.

Receipt separates registration=verified_cli, connection=passed, enabled state, agent_ui=unverified and overall=incomplete. Preserved disabled configuration is not silently enabled. Local exit0 means the requested registration/observation finished, not GUI loading or inference success. No token-bearing diagnostics are printed.

## Probe repairs

The native probe now compares base64 host pins case-sensitively. It rejects empty tool content, blank text and error results instead of treating them as authenticated success. Read-only empty inventory remains valid when expressed as a nonempty supported result. SSH stdout and child JSON receipts use explicit UTF-8; redirected PowerShell5.1 subprocesses previously exposed a protocol-decoding difference. The Framework lacks StandardInputEncoding, so that unsupported property was removed during validation; ASCII lifecycle requests use the normal redirected writer. No host trust bypass or console/registry persistence was introduced.

## Actual verification

Product source authored via the existing BoxWeave MCP connect inside the assigned non-root project LXC, /home/boxweave-tenant-dev/rewrite-v2-phase4-40a600a. The built-in tool connection initially returned Transport closed for discovery/diagnosis. A fresh session through the same configured Windows native SSH restricted /usr/local/bin/boxweave-mcp launcher succeeded: MCP0.120.1, protocol2024-11-05, tools-list109 and actual list_containers. An ignored local MCP client used tools/call connect to verify boxweave-tenant-dev / boxweave-tenant-dev-container, exit0, then conduct remote development. It did not use a host shell, change registration or create a new fixed server per box.

Windows-native client verification necessarily ran in actual Windows PowerShell5.1 and installed Codex CLI, with isolated homes under ignored .runtime/native-registration/run-05. Actual source transfer matches [manifest](native-registration-source-hashes.json); the trusted test descriptor was constructed from those reviewed source bytes and the already trusted host pin. It is a test package, not proof of a released publisher identity.

| Check/command | Actual outcome |
| --- | --- |
| Windows powershell.exe -NoProfile -ExecutionPolicy Bypass -File tests/windows_native_registration.ps1 -ConnectionDescriptor <reviewed public descriptor> -TestRoot <fresh path> | exit0; all checks in [captured result](native-registration-results.json) passed |
| Native profile | real Codex CLI recognition; actual Windows SSH/MCP read; Unicode/space AgentHome; idempotence |
| Preservation | declined conflict byte-unchanged; selected replacement preserves other server/model/timeout/tool approvals; original-byte backup verified |
| Failure paths | inline form unchanged; tampered executable blocked before marker or AgentHome creation; boolean schema refused; case-only pin change PIN_MISMATCH; malformed/empty read-only result refused |
| Existing native tests/windows_connection.ps1 | exit0, AST and three no-network preflight cases |
| Project LXC .venv/bin/python -m unittest discover -s tests -v | 41 tests OK, exit0, 11.755 seconds |
| Personal config preservation | default personal config digest unchanged; no registration writes outside explicitly selected test homes |

Tested codex-cli0.165.0-alpha.1.2, Windows PowerShell5.1, native Windows OpenSSH and actual outer MCP0.120.1. No other client version, WSL profile, ZCode, model inference or GUI support is inferred. UI application source is unchanged; only package version metadata aligns to 0.2.1, so no new browser/production-deployment result is claimed from this client-only slice.

## Reproduction and cooperative GUI check

Start with the documented public descriptor example. Supply an already trusted full host public key, existing identity/known_hosts paths and reviewed installed CLI version, then trusted payload SHA256 values and matching source_id. Deliver bootstrap and descriptor through the authorized operator channel. From an actual Windows terminal:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File clients/windows/bootstrap-native.ps1 -Descriptor C:/reviewed/trusted.json -PayloadDirectory C:/reviewed/payloads -AgentHome C:/reviewed/isolated-codex-home
```

For an existing conflicting target, only add -Replace after reviewing the exact selected server transport and private backup boundary. Do not blindly point the test package at the personal home. Automated tests require a fresh TestRoot and do not copy auth.json or perform sign-in. Test homes/backups remain private and retained for inspection; no key contents or credentials are in Git.

CLI recognition is not an Agent/TUI/desktop loading observation. A later cooperative test should load the retained profile with the client's supported home/config selection, sign in through its normal UI if required (never copy sign-in tokens), inspect /mcp in the CLI TUI, and verify read-only tool availability. Actual GUI and end-to-end selected-profile observations stay pending. No interactive client was launched automatically here. Official [OpenAI Docs MCP guidance](https://learn.chatgpt.com/docs/extend/mcp?surface=cli) describes CLI configuration and /mcp; this test used installed-runtime behavior rather than assuming registration means loading.

No new box, route, public listener, host setting or real personal registration was changed. Existing demo endpoint remains as before, not a new mapped native client. No user action was required to make this scoped CLI test pass. Stop TASK-003a; next suggestion TASK-003b for supported bootstrap identity provisioning/actual client UI evidence, before treating TASK-003/005 as complete or automatically starting dependent TASK-006.