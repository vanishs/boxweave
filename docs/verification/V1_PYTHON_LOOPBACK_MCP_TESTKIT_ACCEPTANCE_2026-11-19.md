# BoxWeave v1 — Python loopback MCP test kit: source and installed HTTP acceptance (2026-11-19)

**Status:** Developer tooling IMPLEMENTED and verified on disposable isolated Box; **not** an additional public MCP Tool, published release, real ChatGPT Agent, OAuth browser acceptance or physical Incus/Root signoff. Sole unmerged [PR #153](https://github.com/vanishs/boxweave/pull/153). The tested Python library/CLI/test/example source was committed by `ef3cc969f86bd1fc455e0b1f802ad88f2c04a74a`; subsequent CI/Makefile commits integrate the same source with the required quality gate.

## Files / integration

- `scripts/mcp_testkit.py` — standard-library-only, pinned MCP Streamable HTTP 2025-11-25 **numeric-loopback-only** client; Bearer token via environment/0600 file; initialize/initialized, tools/list, tools/call, JSON or finite bounded SSE, strict ID/HTTP response and max-byte checks. No remote host, redirect, URL credentials or automatic retry. Mutation and unannotated Tool calls are blocked by default.
- `scripts/mcp_lab.py` — CLI `inspect`, read-only `smoke`, explicit `call`, read-only `recover`; JSON reports contain no arguments, raw outputs, tokens, command lines or private proofs. Mutation requires both `--allow-mutations` and `--ack-disposable` (only for test fixtures).
- `scripts/test_mcp_testkit.py` — **15 standalone Python unittest cases**, including handshake, token/refusal, 2025 wire error, SSE, safe Tool policies, pagination, UNKNOWN recovery, single discarded read response, remote/redirect rejection, report redaction and CLI.
- `scripts/examples/mcp_activity_smoke.py` — short importable example a future Agent can copy and extend without rebuilding the transport.
- `docs/development/V1_PYTHON_LOOPBACK_MCP_TESTKIT_2026-11-19.md` — full usage, safe boundaries and reference contracts.
- `Makefile` adds `make test-mcp-python` and makes it part of `make verify`; `.github/workflows/quality-gate.yml` runs Python unit tests in the **full** CI branch, not expensive docs-only path.

## Evidence (disposable Box, not shared L0)

Owned disposable `bw-python-mcp-kit-1119`, Box UUID `dccc0b00-6b85-4f19-840f-60998354a183`, `work_ref=mcp-python-testkit-20261119`.

1. **Mock protocol and Python compile PASS** — `python3 -m unittest discover -s scripts -p 'test_mcp_testkit.py' -q` (15 cases) and `python3 -m py_compile` for library, CLI, example from exact repository checkout `ef3cc969`.
2. **Installed local nonroot Schema v5 PASS** — built the actual Go candidate `./cmd/boxweave` with Go 1.26.9; created an isolated fresh SQLite Schema v5 database, unprivileged service account and test-only Bearer. `serve` bound to **127.0.0.1:18673** with **nonexistent Incus socket**. The Python kit used authentic HTTP MCP JSON-RPC against that independent service:
   - `inspect`: **18** MCP Tools, protocol `2025-11-25` — PASS.
   - `smoke --expected-tools 18 --expected-schema 5 --require-activity`: **7** checks — PASS.
   - Python example Activity `overview/issues`: **4** checks — PASS.
   - `recover --work-ref sandbox/fresh`: safe, empty, read-only manifest with `safe_to_retry_unknown=false` — PASS.
   - `FaultInjectingMCP`: discard one fully processed **read** reply, then read `issues` successfully — PASS.
   - Combined witness: `LOCAL_PYTHON_MCP_REAL_HTTP_SCHEMA_V5_ALL=PASS`.
3. The local service process and temporary Bearer-secret file were removed by the shell EXIT trap. This is **real HTTP MCP/protocol wiring**, not a native Incus guest/Root mutation or browser OAuth exercise.
4. [Source quality gate #44611911319](https://github.com/vanishs/boxweave/actions/runs/44611911319) and [amd64/arm64 builds #44611911435](https://github.com/vanishs/boxweave/actions/runs/44611911435) both **PASS** on code `ef3cc969`. CI integration of Python unittest runner was added afterward; check its own exact SHA CI record rather than projecting previous PASS.

## Mandatory limitations / security

An MCP response is **not native operation completion**. The Python harness does not implement OAuth DCR/refresh/revoke, host-visible Incus packet verification, a real ChatGPT/Agent plan or full MCP SDK compliance. UNKNOWN never becomes a safe retry decision. `recover_work` is a bounded read-only SQLite/Operation summary; it cannot restore an Agent conversation or prove absent unjournaled effects. Fault injection deliberately applies only to **read** replies. The official MCP Go SDK probes, production authorization and native acceptance remain independent mandatory evidence.

**Schema compatibility:** the connected physical `boxweave L0` is still Schema v4 at commit `bc1e8447`, while this test created an isolated fresh Schema v5 database. Do not replace, migrate, reset or reuse the existing controller, OAuth credentials or active ChatGPT connection without distinct explicit approval.

[Tool guide](../development/V1_PYTHON_LOOPBACK_MCP_TESTKIT_2026-11-19.md) · [P1 OAuth/100k history/independent Actor verification](V1_ACTIVITY_P1_SECURITY_HISTORY_RECOVERY_2026-11-19.md).
