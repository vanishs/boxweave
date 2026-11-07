# v0.50.2 standardized Provider REST status adapter: scoped verification

**Date:** 2026-11-07 UTC. **Scope:** eight existing Containarium REST routes, BoxWeave PR #110. **Verdict:** source CI PASS at prior PR head; installed candidate real-write acceptance PENDING/BLOCKED.

## Source and read-only verification
- Prior PR #110 head `d7076f63c399edaf50460f55ab4f74ff64fbc00c`: [GitHub Actions run 42343870648](https://github.com/vanishs/boxweave/actions/runs/42343870648) completed successfully. Python contracts, reacceptance fixture regressions, seven synthetic browser workflows, Python 3.14 offline wheel smoke, release ZIP/standalone Gateway packaging and code-first inventory all passed. This is not native Provider write acceptance; the release-bound v0.50.2 commit requires its own passing CI.
- Normalized errors use only standard gRPC status codes 0–16 with grpc-gateway HTTP mappings. Exact, target- and project-bound upstream v0.128.0 GetContainer missing-instance 500 is rewritten as NOT_FOUND/404; `upstream_http_status=500` is retained. Other 500s remain unclassified; uncertain writes remain UNKNOWN and must not be retried automatically. Legacy create preflight requires complete inventory corroboration.
- Read-only installed host `boxweave_diagnostic_run(profile=rest_read)`: config, Provider mode/binding, REST selected, authenticated REST inventory and MCP inventory were reported passed (six checks). This tests the currently installed runtime, **not** the unpublished candidate.

## Installed write validation blocker
- A new task-owned isolated `bwrest110-1107-qa` Box was requested with nesting and one durable HTTP MCP creation intent. A separate Provider read saw the instance RUNNING, while the original creation receipt first remained `submitted` and then transitioned to `unknown` with `UPSTREAM_CREATE_OUTCOME_UNKNOWN` / upstream acknowledgement unverified. The UNKNOWN receipt is retained without automatic replay. `debug_container` reported the host account present but guest account not yet created; `connect(exec)` returned TRUST_NOT_READY. Older test Boxes also returned TRUST_OBSERVATION_STALE.
- **Real installed candidate write verdict: BLOCKED/PARTIAL**, not guest-ready PASS. Preserve and reconcile the original receipt and native instance identity before any cleanup/replay; do not bypass SSH trust, reissue the same write or alter unrelated Boxes.
- Still unverified: hash-bound candidate installation; missing/occupied/unreachable peer tests; real async POST/CREATING/guest-ready through candidate; exact one-dispatch duplicate requests; start/stop/delete/resize native readbacks; write acknowledgement loss and R-04 UNKNOWN non-replay. A distinct independent Verifier is needed to call this installed acceptance.

## Release boundaries
The current `.github/workflows/ci.yml` automatically publishes a new immutable `v0.50.2` tag and source-bound assets from a successful main-push verify job. Publication does not promote the stable installer pointer or alter the above installed verdict. Upstream typed-status follow-ups: [#107](https://github.com/vanishs/boxweave/issues/107), [#109](https://github.com/vanishs/boxweave/issues/109).
