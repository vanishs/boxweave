# S4-B2 Step 1 — additional knowledge and native experiments (2026-11-16 PT)

**Stage policy:** the owner requested (1) targeted research/experiments, (2) concentrated B2 product implementation, (3) independent acceptance. This record is **STEP1_PARTIAL** after native fault and IPv6-reuse proof, never `S4_B2_PASS`. Only Draft PR #153; do not start B3, expose Peer, merge or release.

## Existing independent native foundation

- [44027000902](https://github.com/vanishs/boxweave/actions/runs/44027000902) and [44027400096](https://github.com/vanishs/boxweave/actions/runs/44027400096) **SUCCESS**: dedicated `bwbr0`, restricted Project, exactly owned A/B/C static dual-stack NICs, source+target-exact S1-compatible ACL, dynamic IPv6 NDP, native ETag revoke, pre-delete denial, replacement UUID and old IPv4 denial.
- [44025118912](https://github.com/vanishs/boxweave/actions/runs/44025118912) **security failure**: a mixed-trust bridge can admit a foreign NIC spoofing an authorized IP in IPv4 and IPv6.
- [44025845932](https://github.com/vanishs/boxweave/actions/runs/44025845932) **SUCCESS evidence of danger**: a new UUID can reuse a deleted Box's former fixed IPv4 **and IPv6**. Thus fixed NIC addresses by themselves do not prevent stale authorization transfer.
- `network_peer_policy.go`, `network_peer_bridge_guard.go` and SQLite claims/fences are **unwired internal safeguards**. The successful native fixtures are not an installed BoxWeave network writer.

## Fresh, isolated experiment set

The existing dedicated-native GitHub Actions workflow is extended with these **native readback and traffic** probes. All use a newly provisioned, disposable Incus 7.6 Ubuntu runner and the source-gated Go SDK fixture `native_s4_b2_fault_lab_test.go`; none mutates L0 or shared daemon authorization:

| ID | Experiment | Positive evidence required | What it does **not** establish |
|---|---|---|---|
| N-F01 | Two ACL writers hold the same ETag; first changes the ACL description, second submits the stale original ETag | Native second PUT receives 412/precondition; original first writer ETag and body unchanged after readback | True simultaneous scheduling, kernel NIC fanout atomicity |
| N-F02 | Create a fourth **unknown** NIC consumer in the same otherwise restricted Project while A/B has a grant | Root-observed network `UsedBy` lists it and internal native ownership guard refuses it; resource then deleted | Continuous protection from host root/OOB attachment; actual restricted-cert unauthorized write |
| N-F03 | Revoke exact dual-stack A→B, delete B, reallocate *both* addresses to a new UUID, clear **IPv6** neighbor entry | Actual A→reused IPv6 packet denied after native ACL readback (existing IPv4 negative repeated) | Crash/lost ACK/partial update path; entire established-flow teardown |

**Follow-up still required in Step 1**: ACK loss, restart and durable quarantine; native unauthorized restricted TLS client permission matrix; Scriptlet model/read-only preflight and rollback dry-run; external IPv4/IPv6 TCP/UDP denial and positive control; native lease/MAC/source spoof under concurrent changes. Do not falsely mark an experiment PASS until its actual GitHub run, SHA and raw logs are reviewed.

## Step 1 original native results — 2026-11-17 UTC (no product writer)

The first new [Incus 7.6 run 44029301631](https://github.com/vanishs/boxweave/actions/runs/44029301631) was **SUCCESS** on exact source `cef31db47ceb6122d819d1e85558692b9014da64`. Its disposable job `133857081503` independently recorded:

- **N-F01:** the second writer, holding a stale original ACL ETag, received the expected 412/precondition refusal, and fresh GET/ETag/body still matched the first writer. This is a deterministic *stale-snapshot interleaving*, not proof of fully concurrent writes or native per-NIC atomicity.
- **N-F02:** a fourth real NIC in the same restricted Project appeared in root native `Network.UsedBy`, and the unexposed exact-consumer guard refused the unknown instance. The temporary consumer was deleted before revoke. A project-scoped client attempting to bypass the guard after an earlier snapshot remains untested.
- **N-F03:** after exact ACL revoke and readback, B was deleted; a different native UUID received both B's former static IPv4 **and IPv6**. Clearing A's IPv6 neighbor and sending new A→reused IPv6 packets was **DENIED**; existing IPv4 reuse denial and exact native cleanup remained PASS. This is an *ordered positive-path* mitigation, not crash/lease-quarantine completeness.

The next [fresh disposable fault run 44029634956](https://github.com/vanishs/boxweave/actions/runs/44029634956), source `193e5061be29db589d30abaeae6b84b23561e8bc`, job `133858170603`, intentionally removed the **owned C host-side veth**. Attempted ACL revoke produced the original native SDK error `Failed updating bridge NIC ACL: Failed to trigger device update ... Unknown or missing host side veth device`. Incus ACL GET **still contained the original A→B v4/v6 allow rules**; downloaded `s4b2-fanout-disposable-evidence` native nft snapshots demonstrated that B's effective NIC chains had **lost** the A→B allow rules while A retained them and C had stale rules referencing its now-missing host veth. A→B new IPv4 ping was denied/unreachable. Exact owned cleanup **PASS**. **The workflow's SUCCESS describes a diagnostic that intentionally expected the inner revoke fixture to fail, NOT a successful ACL revoke.**

The controlled repeat [44030128743](https://github.com/vanishs/boxweave/actions/runs/44030128743), source `6328dc3dc52c4b7d0196716c0a7ba0084b91a7c2`, job `133859786092`, **SUCCESS as fault diagnosis**, added an important causal control: after C's host veth was removed but **before** ACL PUT, actual A→B IPv4 packets still passed. After the failed PUT, native ACL GET still contained the same v4/v6 grant, while extracted **per-NIC nft rules** were:

| Owned NIC | After C link loss, before PUT: v4/v6 grant rule count | After failed ACL PUT: v4/v6 grant rule count |
|---|---|---|
| A | 4 / 4 | 0 / 0 |
| B | 4 / 4 | 0 / 0 |
| C (host veth missing) | 4 / 4 | 4 / 4 |

The repeat logged `S4_B2_FANOUT_CONTROL_AFTER_C_LINK_LOSS_A_TO_B_STILL_REACHABLE=PASS`, `S4_B2_FANOUT_ACL_REVOKE_SDK_EXIT=1`, `S4_B2_FANOUT_PARTIAL_KERNEL_CHANGE_AFTER_FAILED_PUT=a,b`, and `S4_B2_FANOUT_NATIVE_CLEANUP=PASS`. Native nft before/after and ACL readback were uploaded as `s4b2-fanout-disposable-evidence`, not guessed from SQLite.

**Source-exact root cause:** Incus tag [v7.6.0 `acl/driver_common.go`](https://github.com/lxc/incus/blob/v7.6.0/internal/server/network/acl/driver_common.go) persists the proposed ACL database state *before* the kernel application, and its reverter can restore the old **database** after an error. [`acl/acl_bridge.go`](https://github.com/lxc/incus/blob/v7.6.0/internal/server/network/acl/acl_bridge.go) iterates affected running NICs, `ReloadDevice` one by one, and returns on the first failure. It does **not** atomically roll back already applied NIC kernel rules. Hence **SDK error + ACL GET of original rules != all NICs running original rules**, and vice versa. Native ACL PUT is synchronous and gives no native Operation ID, so never invent one.

**Mandatory product consequence for Step 2:** on failed, timed-out, or unverified ACL PUT, keep the original durable intent `UNKNOWN`/locked, **do not release either endpoint address, detach an owned NIC, delete an involved Box or offer a new grant**, and audit every affected effective NIC/kernel state plus actual new-flow packets in an isolated recovery path. A failed revoke must not be reinterpreted as harmless because the ACL GET rolled back. Explicit active containment/fencing must be designed, implemented and independently tested; a periodic ACL GET alone is insufficient.

**Still open for Step 1 / no acceptance claim:** true live competing writers; ACK loss, native restart/recovery and SQLite quarantine; source spoof and unknown same-Project client capabilities under *restricted TLS*; external IPv4/IPv6 TCP+UDP `internet=false` controls; Scriptlet old restricted-client rights and installer rollback. The latter requires **new host-specific authorization to change global daemon settings**, even in an isolated host; no Scriptlet/global change was performed. The unreviewed `internet=false` Peer exception remains an owner semantic decision. `S4_B2_PARTIAL`; no B3, public Peer writer or S4 release.

## Important official upstream contract

- [Incus network ACL guide](https://linuxcontainers.org/incus/docs/main/howto/network_acls/): `drop`/`reject` outrank `allow`, ACLs on bridges need **per-NIC** attachment for intra-bridge isolation, and rule update/ETag must not be assumed equal to per-NIC kernel convergence.
- [Incus authorization](https://linuxcontainers.org/incus/docs/main/authorization/): routing `authorization.client.tls-restricted=scriptlet` removes the automatic TLS certificate Project restriction; any migration must explicitly preserve old-client Project allow/deny behavior. **No host-specific global Scriptlet approval was given** and no actual daemon-global change is authorized in this step.

## Step 2 implementation prerequisites

Only after experiments settle the actual native identity/lease and ACL failure boundaries: connect owned bridge/UUID/NIC/IP/MAC checks; ordered claims; pre-release revoke+kernel denial; `UNKNOWN` quarantine and fenced recovery; bounded ETag retries; a reviewed Scriptlet privilege transition with explicit host approval. Still no public Peer grant until the installed gate.

## Step 3 acceptance prerequisites

Separate reviewer of original native operation/ACL/kernel/packet/SQLite evidence, installed restricted HTTP MCP/OAuth, failure cases, and exact-final-HEAD quality+native gates; retain `S4_B2_PARTIAL` until all mandatory evidence exists.
