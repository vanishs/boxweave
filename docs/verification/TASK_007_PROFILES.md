# TASK-007 — four selected Windows profiles, 0.6.0

SRC-066 authorizes implementation; SRC-070 records the operator's exact WSL command and green ZCode desktop status, following the operator's manual trust of only the dedicated workspace. SRC-071 is final automated evidence; SRC-072 authorizes continuing remaining phase5 work in dependency order. Covers the enrollment/diagnostic/UI subset of REQ/AC-001/002/003/017, not model or release acceptance.

## Delivered

Operator-trusted schema2 descriptor, explicit Windows runtime/executable and agent version, native/explicit WSL transport, local/WSL identity preparation, public-only restricted enrollment reuse, strict pin/ACL/path gates, bounded protocol discovery/read and independently classified registration. Native/WSL null devices differ; paths remain separate argv values. The stdlib helper requires Python3.11+ on Windows and does not install it. Existing PowerShell native path stays available. Source was authored in assigned non-root development LXC and synchronized by exact hashes.

Codex TOML merge preserves unrelated settings/servers and nested tool approvals; conflicts need explicit replacement, unsupported table/transport forms stop. ZCode uses only selected workspace mcp.servers, private data root and version-specific app-server NDJSON mcp/list. The observed core needs its installed public builtin catalog passed explicitly; this is not a model credential. PROGRAMDATA/ALLUSERSPROFILE must survive environment isolation for Windows OpenSSH. API-key environment variables do not pass. ZCode's workspace effective command/args must match before its MCP process launches; a generic connected server under the same name cannot certify another transport. No model prompt/session-send occurs. See [client instructions](../../clients/README.md).

## Actual evidence

Non-root assigned LXC: Python3.12.3, full unittest discovery95 tests29.282s exit0; TypeScript/Vite build0.6.0 exit0. Thirteen profile cases include config/approval preservation, conflict/idempotence, explicit distro/POSIX/Unicode/null argv, wrong pin, real child-process EOF/deadline/oversized output/cleanup, strict MCP versus explicitly versioned ZCode NDJSON, duplicate JSON and effective-registration mismatch. These cases are fixtures, not Windows support proof.

Windows actual Python3.12.14, native OpenSSH, Ubuntu WSL user [redacted-admin]/hostname PC, /usr/bin/ssh OpenSSH8.9p1; Codex CLI0.165.0-alpha.1.2; ZCode desktop3.16.3 and bundled core0.17.0. Fresh WSL identity stayed inside /home/[redacted-admin]/bw-v2-task007-中文, modes700/600, public half authorized as exact non-root bw-v2-mcp-nested with the reviewed restricted authorizer. Private identity/auth/JWT contents were never read/copied. Existing dedicated native v3 identity reused normally. No distro installation/fallback, global client/SSH changes, new public route or model request.

Run selected Python with scripts/windows_profile_check.py --descriptors <four reviewed private JSON paths> --result <private result>. Final actual exit0; result retained under .runtime/task007/final-matrix.json. Every row below completed initialize/initialized, paged tools/list101 and actual read-only list_containers against bw-v2-mcp-nested via the already approved18447 endpoint. Actual agent registration/loading is a separate column.

| Windows profile | Real transport/read | Agent loading | Separate UI observation |
| --- | --- | --- | --- |
| Codex/native | passed | mcp list JSON exact enabled transport | Actual TUI /mcp: boxweave connected2 tools |
| Codex/Ubuntu WSL | passed | mcp list JSON exact enabled transport | Actual TUI /mcp: boxweave connected2 tools |
| ZCode/native | passed | Actual core mcp/list connected101 | Desktop workspace row exact native SSH/18447, green indicator |
| ZCode/Ubuntu WSL | passed | Actual core mcp/list connected101 | Operator SRC-070 reports exact --distribution Ubuntu --exec /usr/bin/ssh, strict pin, POSIX paths,18447/backend and green “connected and available” |

Each profile also passed repeat registration unchanged, diagnosis config-byte preservation, tampered payload, changed Agent version and wrong-pin-before-publication checks. Both WSL profiles rejected an uninstalled named distro without installation or fallback. Dedicated isolated config folders include Unicode and spaces. Public expected source hashes were selected from the reviewed checkout by the operator test driver, not accepted as authenticity evidence from an untrusted download.

Codex UI observations used transient command/argv and two read-only tool filters in the already signed-in isolated acceptance home, no sign-in-file copying or model prompt. Filters are client presentation, not backend least privilege. ZCode native and WSL UI used sequential transport configurations in the same operator-trusted test workspace; no second trust prompt was bypassed. After final regression the dedicated native workspace was restored to its native test transport; the separate WSL profile remains registered in its own isolated directory. Daily global entries were not changed. The initial desktop check was stopped with physical Escape and resumed manually by the operator, not counted as an automated WSL observation.

## Limits and recovery

Machine output deliberately keeps agent_ui=unverified/overall=incomplete; separate observations above complete TASK-007 connection/recognition scope. Model choice/access/inference, clean installation, reboot/autostart, packet isolation and all-feature release remain TASK-008/012/013. Agent startup may update its private runtime cache/database; diagnosis does not enroll keys, repair settings or mutate remote resources. Local owned process-tree termination is not proof of remote cancellation. Windows ACL gates and explicit runtime are required; no claim for arbitrary Python/Agent versions or WSL-hosted Agents. Quiesce other config editors; backups are for manual review, not unconditional whole-client rollback.

TASK-007 is complete in this selected scope. Retained lifecycle hostkey-reconciliation failure remains an infrastructure risk; no host/provider repair was performed. Official context: [ZCode MCP](https://zcode.z.ai/en/docs/mcp-services), [OpenAI MCP](https://developers.openai.com/codex/mcp/); observed installed contracts and tests, not documentation alone, establish this matrix.
