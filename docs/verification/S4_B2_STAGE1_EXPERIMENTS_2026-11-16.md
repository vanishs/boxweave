# S4-B2 Step 1 — additional knowledge and native experiments (2026-11-16 PT)

**Stage policy:** the owner requested (1) targeted research/experiments, (2) concentrated B2 product implementation, (3) independent acceptance. This record is **STEP1_IN_PROGRESS**, never `S4_B2_PASS`. Only Draft PR #153; do not start B3, expose Peer, merge or release.

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

**Follow-up still required in Step 1**: deliberately failed per-NIC kernel reload/fanout with independent nftables and packet poststate; ACK loss, restart and durable quarantine; native unauthorized restricted TLS client permission matrix; Scriptlet model/read-only preflight and rollback dry-run; external IPv4/IPv6 TCP/UDP denial and positive control; native lease/MAC/source spoof under concurrent changes. Do not falsely mark an experiment PASS until its actual GitHub run, SHA and raw logs are reviewed.

## Important official upstream contract

- [Incus network ACL guide](https://linuxcontainers.org/incus/docs/main/howto/network_acls/): `drop`/`reject` outrank `allow`, ACLs on bridges need **per-NIC** attachment for intra-bridge isolation, and rule update/ETag must not be assumed equal to per-NIC kernel convergence.
- [Incus authorization](https://linuxcontainers.org/incus/docs/main/authorization/): routing `authorization.client.tls-restricted=scriptlet` removes the automatic TLS certificate Project restriction; any migration must explicitly preserve old-client Project allow/deny behavior. **No host-specific global Scriptlet approval was given** and no actual daemon-global change is authorized in this step.

## Step 2 implementation prerequisites

Only after experiments settle the actual native identity/lease and ACL failure boundaries: connect owned bridge/UUID/NIC/IP/MAC checks; ordered claims; pre-release revoke+kernel denial; `UNKNOWN` quarantine and fenced recovery; bounded ETag retries; a reviewed Scriptlet privilege transition with explicit host approval. Still no public Peer grant until the installed gate.

## Step 3 acceptance prerequisites

Separate reviewer of original native operation/ACL/kernel/packet/SQLite evidence, installed restricted HTTP MCP/OAuth, failure cases, and exact-final-HEAD quality+native gates; retain `S4_B2_PARTIAL` until all mandatory evidence exists.
