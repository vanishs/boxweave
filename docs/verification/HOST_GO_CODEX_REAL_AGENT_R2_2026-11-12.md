# HOST_GO Round 2 — actual Codex MCP lifecycle, candidate restart and independent snapshot V

**Verdict (2026-11-12 PT): TC-013 actual development-Agent discovery/authorization/read PASS; Round 2 overall BLOCKED, Round 1 overall BLOCKED, Round 3 NOT_STARTED.** Source under test was 0cf474fbd212c92702591ec85f75e7bea58547ee, with later documentation-only PR #133 commits. Exact native candidate executable SHA256: 021127cf723bfbe96c3668965b1e60dcc82856e5e3d41e09e84982568b5c1e85.

## Authority, controller and client

The owner approved Codex CLI v0.167.2 as the actual developer Agent and locally signed in to ChatGPT. A nonroot bwcodex OS user, without Incus socket or sudo membership, configured private authenticated v1 Streamable HTTP MCP at 127.0.0.1:18666/mcp. The installed test candidate was boxweave-hgcodex-l0-0cf.service, isolated Project user-1002 and original SQLite /home/boxweave/hgcodex-0cf/ledger.sqlite. Original boxweave.service stayed active. Preexisting L1 bw-s25-l1a UUID 21767890-6d46-4867-889d-0069c1e4ac0d was never touched.

The owner executed a local one-time scoped script to issue a two-hour temporary Admin token in a mode 0600 private bwcodex file; no token value went to this chat, GitHub or an Agent prompt. After successful cleanup, the official token-revoke command revoked it and the local file was removed.

An initial actual Codex mutation attempt with approval policy NEVER was denied: "MCP tool call requires approval, but approval policy is never". Original SQLite/native showed no effect from that request. No bypass of this guard and no original-key retry occurred. Subsequent sessions used official on-request approval with independent Auto-review, while retaining read-only filesystem sandbox, product Admin checks and restricted native Project. Actual mutation Tool logs showed started/completed. Each mutation was invoked once with a fresh original idempotency key; initial ADMITTED receipt was not misreported as terminal SUCCEEDED.

## Actual model-driven native lifecycle (same exact UUID)

New, originally absent Guest: hgcdx-r2-a UUID/incarnation 589a900e-3fb2-4a22-b02c-7de6fff5521f, nesting=false and Internet=false.

| Codex MCP Tool | Original BoxWeave Operation ID | Original Incus Operation ID | Original final state and observed native effect |
| --- | --- | --- | --- |
| box_create | efaff82fec24ec6543ea29d50a2c2305 | 4e0394a5-ed34-4359-a389-49920cd1d4d4 | SUCCEEDED; created STOPPED |
| box_start | bbffaae107fa5767d4c6bedfd8099b53 | d2bd04c6-b09f-4405-aa32-3387e9c9cace | SUCCEEDED; RUNNING |
| box_stop | 3b23f63bedfc3715cc2d4c1bded3f2e7 | 43a34bb4-f8e9-4fec-b76a-b90ba25ff855 | SUCCEEDED; STOPPED |
| box_delete | 25479b7bb0eef4beccced8992a08f2ef | f7838d63-30f6-40bc-936e-78d9ca6d4cc6 | SUCCEEDED; native ABSENT and owned registry DELETED |

Every original persisted work_ref and agent_ref=codex-cli-01672, original UUID/incarnation/native ID, and ordered durable events. New writes had distinct idempotency keys and were never replayed. Both original and candidate services stayed active at the final postflight; original native project inventory contained only unchanged foreign bw-s25-l1a RUNNING. Original SQLite contained 42 SUCCEEDED and zero UNKNOWN, up from 40 SUCCEEDED before Stop/Delete.

## Fault result: DO NOT confuse terminal restart with in-flight recovery

A guarded one-shot in-flight SIGKILL monitor ([exact source log](evidence/hgr2-faultwatch.txt), SHA256 5d277d1d4bd1d9d5645582953f6b4763e9ae3bdd7eef2d6636855f3721a4e060) was armed for the brand-new Codex box_start and observed its original bbffaae107fa5767d4c6bedfd8099b53 in DISPATCHING on 2026-11-13T02:30:29Z. That state was not permitted by its safe trigger rule, so it **did not kill**. The operation completed SUCCEEDED. This subcase is **IN_FLIGHT_SIGKILL_NOT_EXERCISED**, not PASS. No attempt to retroactively kill or replay that original was made.

Source code intentionally marks in-flight DISPATCHING/VERIFYING as UNKNOWN after a process restart (PROCESS_RESTART_OR_LOST_ACK), never autonomously redispatching a possible original effect. The mandatory in-flight lost ACK/UNKNOWN independent recovery combination therefore remains a formal BLOCKER, not a waived case.

A *separate and genuine* candidate-only SIGKILL ran after **all 40 original operations were already terminal**. Primary boxweave.service stayed active. The candidate transient systemd service was restarted with exact same binary, original SQLite, private Project and localhost MCP port (prior PID 86665, restart PID 93956). The original Create and Start ID/Incus ID/state remained unchanged and SUCCEEDED; native Guest remained RUNNING; unauthenticated MCP HTTP returned 401. A **new real Codex CLI session** then reconnected through v1 MCP and performed actual operation_get on the same two original IDs and box_get for the original UUID: SUCCEEDED, SUCCEEDED, Running; no mutation/replay. This proves terminal persistence and actual Agent reconnect, **not** mid-operation crash recovery.

## Independent separate-Box verifier

The source Box produced read-only selected SQLite/native snapshots without bearer tokens. A different Box, boxweave-test-ser2, received the exact bytes and independently ran [verification source](evidence/hgr2_verify.py) with fail-closed assertions over all original IDs, four distinct native IDs, exact UUID, agent/work attribution, original events, foreign unchanged UUID, owned DELETED, 40->42 SUCCEEDED and zero UNKNOWN. Its verdict was PASS_SCOPED_SNAPSHOT_CROSSCHECK.

| Evidence | SHA256 |
| --- | --- |
| [Before-cleanup read-only snapshot](evidence/hgr2-precleanup.json) | d46e0d60d10381d92758a7df4f610d0ce033a56b4f3011fab3dd6b5fb2a8c548 |
| [After-cleanup read-only snapshot](evidence/hgr2-postcleanup.json) | e64023b9f825b67877cd18afe0ce6986500d56fb67c4d38badc8a19be0c4f2a1 |
| [Separate-Box verdict](evidence/hgr2-independent-review.json) | e0c76f419159504c3a3f37e682e9aca39251414877f09b0699447e33c3f68bbd |

**Verifier limitation:** this is separate-Box cross-verification of independently produced read-only native/SQLite exports, not a separate live native socket login. It does not prove in-flight recovery or all BH/TC acceptance rows.

## Case-level disposition

- **TC-013 PASS** against its explicit S2.5 criterion: actual intended Agent authenticates, discovers and calls real v1 Tool, observes read result; optional work_ref/agent_ref arguments were confirmed persisted. Extra native lifecycle and real post-restart reconnect proved; no unsupported MCP Tasks claim.
- **TC-011/TC-012/TC-014 and BH-007/008/012/015/016 remain PARTIAL/BLOCKED** where mandatory two-real-Agent, pending-work disconnect, in-flight ACK/crash/UNKNOWN and full independent recovery/rollback matrix was not witnessed in this run. Earlier original SDK fault CI remains separate evidence.
- **Round 1 still BLOCKED** on documented full BH-009/BH-015 L1 foreign-Project packet/parent-socket/install-fault crosswalk. **Round 2 still BLOCKED** on remaining P0 fault/recovery and independent TC/BH signoff; **Round 3 NOT_STARTED**. No overall LAB_PASS, HOST_GO, physical-host deployment, unrelated Incus or original historic UNKNOWN change, v1 merge or release.
- Temporary Admin token revoked and local file deleted after the new Guest was fully deleted. A new documentation-only HEAD must be checked by its own GitHub Actions; green Actions do not waive absent live negative cases.
