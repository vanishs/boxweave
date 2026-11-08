# End-to-end fault simulation: LAN Hello World, failed delete, nested fix, release

**State: DESIGN SIMULATION / NOT EXECUTED.** No v1 runtime, live guest, native packet trace, installer or release was run for this scenario. Labels `v1.0.0` (first working release) and `v1.0.1` (its hypothetical bug-fix release) are illustrative; the current source-only VERSION 1.0.0 is not itself a released implementation.

**Invariant:** Complete this loop with the existing **20 MCP tools**, ordinary guest software, a restricted host-local maintenance CLI, a disposable external LAN probe, repository/CI tests and an independent Verifier. Do not add a Caddy, arbitrary host log tool, embedded process supervisor, remote upgrade tool, cross-layer orchestration or a new MCP API to solve a test-harness problem.

## Actors and evidence

- **L0**: a real host already installed with an actual verified first BoxWeave build; a separate operator can run host-local `boxweave status/doctor/upgrade`. The remotely connected Agent has **MCP tools only**, not arbitrary L0 host shell/logs.
- **LAN witness**: a test machine **different from L0**, on the intended LAN, observing actual HTTP response and source/destination endpoint. HTTP over a host loopback bind is not reachable from this witness.
- **Test Box A**: a disposable L0-owned Ubuntu guest running a simple Hello World service. It is not a privileged host and has no BoxWeave guest Agent.
- **Dev Box D**: a separate disposable L0-owned guest with `nesting=true`, in which the Agent builds and installs a candidate **independent L1 BoxWeave**. L1 only manages L1-created children. Do not install the candidate on L0 or assume a working L0 bug automatically disappears.
- **Independent Verifier**: separately checks the exact candidate binary/source SHA and actual L1/L2 native state, including the original negative case, before any formal release. It need not be a production-resident service.

Minimum evidence per operation: exact source/build/version, installed Incus daemon/extensions, native Box UUID/incarnation and project/layer, actor role, client idempotency key, Tracking/Operation/native operation IDs, phase/status, safe error summary, observed Box state and owned peer/forward state, verifier location/time, and independent native postcondition. Tokens and guest data are not copied into diagnostics.

## Main path: first release creates reachable LAN HTTP service

| Step | Action | Required observation |
|---|---|---|
| 1 | Operator installs verified first version; MCP client connects on its operator-proxied HTTPS endpoint (the BoxWeave listener itself is loopback HTTP). | `host_status` reports actual product VERSION/build SHA, Incus version/API capabilities and layer; no claim of installation solely from a GitHub Release. |
| 2 | Agent invokes `box_create` with a **persisted reusable idempotency key**; `operation_get(wait_ms)` observes terminal creation/readiness, `box_get` reads the exact UUID. | Native RUNNING **and** guest Exec readiness are independently observed; only created resources are BoxWeave-owned. |
| 3 | Via `box_exec`/guest file apply, set up and start a Hello World web server as an **ordinary guest-native service** (e.g. guest `systemd` unit), not a process whose lifetime relies on an unfinished MCP Exec stream. | The install/start Exec exits with observed code; guest service stays running and listens on a guest-reachable interface/port. A guest `curl` tests the app before publishing. An image lacking the required guest service manager is reported as capability-limited rather than silently supervised by BoxWeave. |
| 4 | Agent calls `port_forward_create` with exact Box UUID, guest port, TCP, explicit **L0 LAN interface IP and host port**, admin authority, public-bind confirmation and its own idempotency key. | A default `127.0.0.1` host bind would **not** satisfy this test. Validate the native Incus proxy-device/network-forward backend and actual bound IP/port; check foreign collisions and stored native claim. |
| 5 | From a different machine on the intended LAN, send HTTP to `http://<L0_LAN_IP>:<host_port>` and verify the Hello World body. | True PASS requires the **external LAN witness** plus native owned forward readback, not only localhost/guest success. If host firewall, interface reachability or LAN routing blocks access, classify the responsible layer. Do not silently edit host-wide firewall settings or claim the service is reachable. |
| 6 | Agent requests `box_delete` for exact UUID with standard explicit destructive confirmation and a new reusable idempotency key. | The delete confirmation must disclose that **BoxWeave-owned dependent native forwards/peer grants are included**. Cleanup is an internally journaled sequence, not an extra public cleanup API. |

## Injected delete failure and recovery

**Illustrative fault only:** `box_delete` begins, removes or changes a native forward, then native instance deletion fails (e.g. a still-present device, dependency, unexpected upstream error, or an application logic bug). No specific Incus error string or bug root cause is assumed until actual evidence exists.

The operation result must disclose a safe structured summary in **existing** `operation_get`: `tracking_id`, `operation_id`, service/build SHA, `phase` (such as `FORWARD_CLEANUP`, `PEER_CLEANUP`, `INSTANCE_DELETE`, `POST_VERIFY`), `error_class`, sanitized native error classification/message, `native_operation_id` when present, last observed instance and dependency states, evidence coverage, whether native effects may have occurred, and the next **safe** diagnostic/reconciliation action. Do **not** expose arbitrary host logs, raw Incus bodies or secret-bearing guest stdout.

- **Known failure with Box still present:** `operation_get` reports `FAILED` and `partial_effects=true` only when the causal native result and relevant state are known; otherwise the operation remains `UNKNOWN`. `PARTIAL` describes an **acceptance outcome**, not an invented operation terminal state. Agent queries `box_get` **and** `port_forward_list`; the forward might already be gone even while the Box remains. Keep a durable journal/tombstone and exact ownership evidence.
- **Response lost but deletion may have succeeded:** use the same idempotency key to recover the **original** receipt without re-dispatch. `box_get` showing not found is not by itself proof of complete cleanup; independently check owned forward/peer claims and original native effect. Preserve UNKNOWN while ambiguous. A confirmed BoxWeave-owned native orphan forward may be cleaned through the **existing** `port_forward_delete`, even if its Box no longer exists, but only after native forward identity and durable Box tombstone ownership are independently verified; never clean arbitrary network entries.
- **Bug prevents safe resolution:** preserve the original failing Box and operation/forward evidence. A remote MCP Agent can inspect the sanitized result and the public project source. If a native stack trace or private host logs are essential, an **authorized local operator** uses protected `boxweave doctor`/journal and provides a redacted report by an approved out-of-band method. Never grant MCP generic host-shell/log access to paper over the gap.
- **Repair semantics:** a duplicate `box_delete` with the original idempotency key **must not issue another delete**. After the original is resolved to a known failure/partial state and native resources are independently re-read, authorized reconciliation may finish safe remaining steps, or a **new** `box_delete` with a **new** idempotency key can be admitted if locks/identity permit. Do not free an UNKNOWN lock or force-delete foreign resources to make a test pass.

### Required Box-delete cleanup contract

At admission freeze exact Box UUID/incarnation, allowlisted associated owned forward and directional peer identities, and the intended cleanup scope. Acquire a stable lock on the Box and its owned dependent claims; prevent concurrent `box_update`, `port_forward_create`, `box_peer_set`, `box_stop` races that would invalidate the snapshot.

Use a durable, provider-aware sequence: inspect ownership and external dependencies; remove or prove removal of owned forwards; revoke/reconcile owned peer grants; request native instance deletion (including required stop policy); verify native instance absence **and** no leftover BoxWeave-owned exposure; only then mark the Box tombstone terminal and release locks. Native Incus **proxy devices** and **network forwards** have different lifecycles; don't assume removing the instance automatically removes a separate network-forward object.

If any step fails after an earlier step succeeded, report which side effect is verified and what remains. Do not claim a single atomic Incus+SQLite transaction, silently recreate removed forwards, or abandon the original journal after a failed delete. On success, prevent old idempotency/ownership claims attaching to a same-name/IP new Box.

## Repair and test candidate before release

1. Preserve the failing operation, original source SHA/build, and a reproducible failure case. Agent develops the **source code fix in disposable Dev Box D** using guest Git/Exec/file tools; capture focused Go tests for observed cleanup order and error classification.
2. Build a **candidate** version using the existing version scheme: when the first released version is `1.0.0`, a delete bug fix is `1.0.1`. Record candidate source SHA, exact binary digest and dependency manifest. **Candidate artifacts are installable from a local/repository/test source before a GitHub Release exists**; do not require the unreleased version to have a formal release URL.
3. In a disposable lab, first reproduce the original bug with the known-bad installed baseline where safe. Then create a clean **nesting-enabled Box** under an authorized outer controller and install candidate `v1.0.1` **inside that Box** as its independent L1 host. The original L0 installation stays on its first version; no silent production upgrade.
4. Because inner L1 MCP only listens to **its own** `127.0.0.1`, a remote outer Agent cannot directly reach it by addressing the L1 guest's LAN/bridge IP. Use a **temporary guest-local MCP test client** launched by ordinary authorized outer `box_exec` inside Dev Box D, which calls L1 localhost MCP using independently bootstrapped L1 credentials. Keep credentials out of Exec argv, logs, stdout and receipts; do not add parent-child MCP proxy or privileged cross-layer socket.
5. L1 candidate creates a new owned Box B, sets up guest-managed Hello World, forwards an explicit L1 host port and successfully deletes B with cleanup readback. Independent Verifier runs the original reproduction and other affected 20-tool/CLI negative cases against **these exact candidate bytes**, including an injected mid-delete failure and restart recovery.
6. To prove the **same-LAN** path to B while L1 is nested, compose **ordinary separately authorized native forwards per layer**: L0 host LAN port -> Dev Box D's exposed interface/port -> L1's locally managed port -> Box B's service. Inspect both forwarding receipts. L1-local or D-local success is not evidence of LAN reachability. If a layer's network cannot implement this, report BLOCKED/UNSUPPORTED, not automatic routing delegation.
7. If L1 creates another nesting-enabled Box and installs the same candidate as L2, repeat the core lifecycle + delete regression where capabilities permit. Distinguish mandatory P0 nesting acceptance from environments unable to support L2; do not substitute mock outcomes for native evidence.
8. Once the independent test evidence is bound to the exact candidate commit and binary digest, **CI** may automatically create its immutable formal `v1.0.1` tag/Release under the owner's standing 14B policy. A generic green unit suite or an unreviewed JSON `PASS` file is insufficient. Inadequate native evidence blocks **release**, not just a later acceptance badge.
9. Releasing candidate `v1.0.1` **does not repair L0's original live failure**. Its local administrator must explicitly upgrade L0 to the released version and then reconcile/complete cleanup of the preserved original Box, using original operation evidence and a separately authorized action. Keep the first installed version and the new candidate distinct.

## Adversarial variants (one primary scenario is not enough)

| Variant | Fault injected | What must be observed |
|---|---|---|
| V-01 | `box_create` committed natively but MCP response lost | Identical idempotency key yields original IDs; no duplicate Box. |
| V-02 | Guest service Exec completed but server dies | Guest-native service persists independently of MCP Exec; `box_get` alone cannot prove HTTP liveness. |
| V-03 | Host forward is bound only to loopback | Guest and L0-local HTTP may succeed; **external LAN witness fails**. Do not mark LAN PASS. |
| V-04 | Incorrect guest listen address or host firewall blocks traffic | Correctly separate Guest, native forward, host bind, host firewall and LAN-route diagnosis; no surprise foreign firewall change. |
| V-05 | Forward cleanup succeeded, native Box delete failed | Box remains, forward absent; operation reports partial native effects and retains evidence/lock. |
| V-06 | Native Box is gone, cleanup response or SQLite commit lost | Reconcile original UNKNOWN; check forward remnants, UUID reuse and tombstone before releasing claims. |
| V-07 | Concurrent `port_forward_create` / `box_peer_set` / `box_delete` | Serial admission and stable native identity prevent orphan exposure and foreign deletion. |
| V-08 | Incus rejects delete due to dependency or protected instance | Structured error + observed native state; no force/delete of foreign resources. |
| V-09 | Token revoked while deletion queued vs after dispatch | Before-dispatch blocked; already/possibly dispatched original reconciled, no imagined rollback. |
| V-10 | Candidate builds but L1 nested Incus/kernel fails to start | No release PASS, preserve exact environment/version evidence; rerun on capable authorized isolated host. |
| V-11 | Outer Agent attempts to connect to L1 loopback via the bridge IP | Expected failure; use guest-local authorized client or operator-configured tunnel, never a new parent management API. |
| V-12 | Nested Box B reachable from inside D but not physical LAN | Check L0/L1 individual native forward and host interface bind; no implicit cross-layer publish. |
| V-13 | Candidate tests pass but binary hash changes before tag creation | Reject release; rebuild and repeat exact-artifact verification. |
| V-14 | L1 fix passes but outer L0 remains old with its original failing Box | Valid release candidate may be published; old L0 and its operation remain unchanged until explicit local upgrade/reconciliation. |
| V-15 | Original delete partial-forward removal plus failed recovery worker restart | Retain original IDs, step evidence and safe resource locks; no blind retry or silent resource loss. |

## Reporting and gates

A completed report must state **SIMULATED, T, I, N or V** per case; exact installed source SHA/build hash; which layer/probe machine supplied evidence; actual `box_get`, `port_forward_list` and Operation statuses; whether guest HTTP was measured independently from another LAN machine; whether an old bug was actually reproduced before verifying candidate; precise retained UNKNOWN resources; and CI/Release tag digest linkage.

**Formal-release gate:** verified original regression and corrected native delete cleanup, mandatory Box/MCP/port/nesting tests, independent exact-artifact evidence, and immutable CI publication on a new VERSION. Mark incomplete network/nesting or unsupported host conditions BLOCKED, never PASS. No automatic L0 host upgrade, no installer stable-pointer movement, no source-only release.

See [operations](../design/OPERATIONS.md), [native networking](../design/NETWORK.md), [nesting](../design/NESTING.md), [Agent workflow](../development/AGENT_WORKFLOW.md), [release policy](../process/VERSIONING.md), [acceptance matrix](ACCEPTANCE.md).
