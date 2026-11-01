# Agent diagnostics (D3/D4, 0.25.0 source)

[Approved design](design/DIAGNOSTICS_D3_D4_MASTER.md) | [Implementation/test plan](design/DIAGNOSTICS_D3_D4_IMPLEMENTATION.md).

The existing MCP offers `boxweave_diagnose`: gateway recent `{"source":"gateway","recent":5}`, gateway trace `{"source":"gateway","trace_id":"32-hex-id"}`, app trace `{"source":"app","trace_id":"32-hex-id"}`, or typed app receipt `{"source":"app","operation_kind":"environment","operation_id":"EXACT-ID"}`. Use it after errors, timeouts and uncertain writes, but **never blindly replay** a possibly completed mutation. Missing coverage is `not_verified`, not proof that an operation did not happen. MCP clients are not guaranteed to invoke diagnosis automatically.

## Gateway storage

An optional root-owned, not group/other-writable `/etc/boxweave/diagnostics.json` controls defaults:

```json
{"enabled":true,"auto_diagnose":true,"retention_days":7,"max_storage_mb":50}
```

Retention range is 1–90 days and capacity 1–512 MiB. The private SQLite journal is `/home/boxweave/boxweave/.diagnostics/mcp.sqlite3`, mode 0600 under owner-only directories. Only time, tool name, trace ID, classified outcome/code and elapsed milliseconds are recorded. No raw arguments, stdout/stderr, token, path contents or project logs are stored. Record failures must not alter an ordinary MCP operation.

## D4: explicit per-Box deep diagnostic grant

**Absent policy means DENY**. The administrator may create root-owned (0644 or stricter) `/etc/boxweave/project-diagnostics.json` with *real observed* Box identity. Illustrative only:

```json
{"version":1,"boxes":{"devbox":{"instance_name":"devbox-container","created":"EXACT_GET_CONTAINER_CREATED_AT","checks":["processes","ports","service","logs"],"services":["myapp.service"]}}}
```

The D4 query `{"box":"devbox","check":"ports"}` or `{"box":"devbox","check":"logs","service":"myapp.service"}` must match both the explicit policy and a fresh Provider observation plus managed SSH trust. Processes and listening TCP are summarized by counts; a service returns a bounded state; log output is classified into counts and never returned verbatim. No arbitrary command, filesystem crawl, unrestricted log query, privileged read, repair or automatic installation is possible through D4. Existing `connect(exec)` operator privileges are **independent** and not narrowed by this policy. Allow the tool names `boxweave_diagnose` and `boxweave_project_diagnose` in the optional host-wide MCP tool policy if it is enabled.

## Privacy, reporting and fallback

Ordinary users receive a private explanation/issue draft; authorized BoxWeave developers may use the same D3/D4 depth to investigate source and tests. Never automatically publish an issue or diagnostic attachment. First prepare a draft; **the user personally examines its exact fields and attachments for sensitive information; then separately approves final external submission**. If MCP is unavailable, use the existing independently authorized host `boxweave doctor` and private `boxweave report`. Real SSH/HTTP/Provider/D4 acceptance is unverified until specifically tested on an authorized Box.
