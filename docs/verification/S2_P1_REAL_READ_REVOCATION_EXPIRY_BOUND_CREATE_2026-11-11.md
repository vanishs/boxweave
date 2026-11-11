# S2 P1 — delayed real native reads and bound-UUID final-receipt interruption

## Exact accepted checkpoint

- **Source** `0f7d7c32bb00bd87d58fdf0cb9ee0c1fbf2605bc`; executable SHA256 `0dfd7368be5c5b8b99b1e7d34faf20c2f24ab4cdbfaf6102bd5d2e4b88bf3b44`.
- [Official MCP SDK / actual single-use Incus 7.6](https://github.com/vanishs/boxweave/actions/runs/43000183546/job/130454094346) **SUCCESS**, [quality](https://github.com/vanishs/boxweave/actions/runs/43000191209/job/130454142623) **SUCCESS**, [S1 and Go race](https://github.com/vanishs/boxweave/actions/runs/43000191484/job/130454119857) **SUCCESS**. This is **installed nonroot candidate** evidence on an independently fresh GitHub-hosted VM, not a production or physical-owner host.
- [Earlier scoped revocation-only native PASS](https://github.com/vanishs/boxweave/actions/runs/42998888504/job/130449762182) on source `2a16211`; [earlier original bound-UUID failure native PASS](https://github.com/vanishs/boxweave/actions/runs/42999493074/job/130451790996) on source `8c561c6`. **Both are reaccepted together at exact current `0f7d7c32bb00bd87d58fdf0cb9ee0c1fbf2605bc`.**

## Real read-token race — eight independent tests

The separately compiled official MCP SDK client authenticates to the actual installed candidate. A **restricted, test-only Incus Unix-socket proxy** first receives a genuine upstream **HTTP 200** for a real Project/owned Box GET, withholds it while the actual MCP handler is in flight, and releases the native bytes only after the exact **current token** has been independently revoked in durable SQLite **or expired** (fresh seven-second credential). No fabricated SDK handler or native response. The official client must receive an error / Tool rejection without private content.

| Public MCP read | Native upstream held | Revocation | Expiration |
|---|---|---|---|
| `host_status` | Real restricted native Project GET HTTP 200 | **PASS** | **PASS** |
| `box_list` | Real UUID-owned instance GET HTTP 200 | **PASS** | **PASS** |
| `box_get` | Real exact-UUID instance GET HTTP 200 | **PASS** | **PASS** |
| `operation_output_get` | Real exact-UUID GET before original native Exec stdout read | **PASS** | **PASS** |

The original native guest Exec had first been successfully submitted via MCP with a real UUID and recorded output. Test user revoked only their freshly created **read-specific** credentials; the seed-admin credential remained valid. The independent Incus administrator confirmed the same original UUID **after all eight refused reads**, and the correct originally admitted Box was then stopped/deleted with a final empty all-Project native inventory. The full native job reported `S2_P1_FOUR_REAL_PRIVATE_READ_TOKEN_RACES_NATIVE_POSTREAD_EMPTY=PASS`.

Previously, `operation_get(wait_ms)` passed actual installed revocation + expiry during long poll. **`operation_list` still lacks a deliberately delayed in-flight token change**; its final-reply `RequireActiveTokenID` is code/Go-race tested and real installed normal/expired ingress behavior is covered, but this is **not** equivalent to a captured mid-query race. None of these tests claims a token can retroactively undo an already-dispatched native effect or guarantees a race after HTTP response commitment.

## Actual native Create bound UUID and failed final SQLite receipt

A separate new nonroot controller with its own disposable SQLite creates a stopped Ubuntu 24.04 Box on actual restricted Incus 7.6 and receives the genuine native accepted Operation ID. **Only after that controller is running**, a test-only SQLite trigger refuses the final `VERIFYING → SUCCEEDED` controller receipt; **native Create, independent postcondition and `RegisterCreatedBox` UUID bind have already succeeded**.

The official SDK observed the original `UNKNOWN` with retained native Operation ID, exact bound UUID and `HOST_LOCAL_REVIEW`. Original keyed Create replay returned the same original without a new native effect. A differently keyed same-name Create and a differently keyed exact-UUID Delete were refused; the journal held **no** second Create or Delete record. Independent administrator read the original actual Incus `volatile.uuid` and `Stopped` status unchanged, then removed **only this newly created artificial test Box**. The unresolved test journal was never force-unlocked or replayed.

Markers `S2_P1_BOUND_UNKNOWN_INDEPENDENT_NATIVE_UUID_AND_STOPPED_STATE=PASS`, `S2_P1_BOUND_UNKNOWN_DISPOSABLE_NATIVE_FIXTURE_ONLY_CLEANUP=PASS` and final `S2_NATIVE_SDK_MCP_BLACKBOX_REAL_INCUS=PASS` were present in the exact accepted job. The already accepted 24-way concurrent Store admission test covers different Key/Actor Delete attempts while original UUID is bound and still `VERIFYING`, on **Go/race** only.

**Remaining gap:** a deliberately paused, *actual installed native* `VERIFYING` interval with independently concurrent MCP Delete is **not** yet proven. A bound UUID followed by failed final durability and safe UNKNOWN/lock is a related but different real fault. Neither test is fabricated as hardware power loss.

## Failures, limitations and release boundary

- Initial new source had a **gofmt-only** quality failure, repaired using the versioned CI source format snapshot. There was no corresponding product failure. Exact accepted candidate passes all required quality, S1/race and full native.
- Remaining S2 P1/P2: dedicated real `operation_list` in-flight credential race, native overlapping bound-Create/Delete `VERIFYING` interleaving, detailed native malformed-return/stalled I/O, bounded selected-pool contract, external WAN online positive control and real physical power/storage loss. S4-owned peer/forward-dependent deletion remains **S4**, not accepted for S2. [Open register](S2_OPEN_GATES_AND_ENVIRONMENT_BLOCKERS_2026-11-10.md).
- Current phase still **S2 PARTIAL** pending full TC/A-case exit crosswalk and environmental blockers. [Draft PR #130](https://github.com/vanishs/boxweave/pull/130) remains **unmerged**; protected `v1`, shared Incus, two historic UNKNOWN original tasks and release state untouched.
