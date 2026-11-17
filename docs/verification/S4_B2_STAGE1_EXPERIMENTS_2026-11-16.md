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

**Follow-up still required in Step 1**: ACK loss, restart and durable quarantine; full existing restricted TLS client **instance exec/inherited resource** parity and reviewed installed-service rollback; external IPv4/IPv6 TCP/UDP denial and positive control; native lease/MAC/source spoof under concurrent changes. Do not falsely mark an experiment PASS until its actual GitHub run, SHA and raw logs are reviewed.

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

## P0 breakthrough: real **failed GRANT -> live unauthorized traffic**, 2026-11-17 UTC

**Highest severity native design observation of Step 1.** [Fresh disposable Incus 7.6 action #44034201810](https://github.com/vanishs/boxweave/actions/runs/44034201810) **SUCCESS as a diagnostic**, source `72f75a7c5712e2862c335bd48652b733e926d0d3`, job `133873032373`, `v1-s4-b2-grant-fault-native.yml`. Actual implementation used the *unexposed* same-package Go SDK S1-exact peer ACL compiler and dedicated managed bridge with A/B/C. It intentionally destroyed only C's host-side veth, on a **new disposable runner**. It was not a production BoxWeave grant.

1. Actual A→B IPv4 packets were **denied** at native S1 baseline. Deleting only C's host veth left A→B **still denied before the grant PUT**.
2. SDK exact ACL grant `UpdateNetworkACL` returned **error** on the first attempt: `Failed updating bridge NIC ACL: Failed to trigger device update ... Unknown or missing host side veth device`, instance `s4b2c`. This is a real native per-NIC fanout partial failure, not a mocked SQL error.
3. Independent root `incus network acl show bw-s4b2-native` after the error had **`ingress: []` and baseline S1 private egress rejects**: native database *appeared to deny* A→B.
4. Native `nft list ruleset` **after the failed PUT** independently showed A and B NICs **4 IPv4 and 4 IPv6** exact A→B allow rule occurrences each, while C had **0/0**. This matches partial kernel application followed by native ACL database rollback.
5. **Actual A→B IPv4 guest ping succeeded after the failed PUT**, logged `S4_B2_GRANT_FAULT_AFTER_SDK_FAILURE_A_TO_B=REACHABLE_FAIL_OPEN_OBSERVED`. The Actions job did **not** call this a product PASS: it succeeded because the diagnostic captured the *intentionally unsafe* state. Exact native A/B/C, ACL, Project and bridge cleanup was **PASS**, and the signed ephemeral runner was destroyed.

**This is proved fail-open**: `SDK PUT error` + `ACL GET denies` + `SQLite UNKNOWN/lock` alone **does not prevent network access already enabled in kernel**. Any design that allows applying a new grant to live source+target while an unrelated NIC may fail native reload **must be forbidden** until a real physical/SDK fail-closed barrier is proven. A recovery sweeper that only reads ACL database or retries non-atomically cannot establish safety.

**Candidate remedy requiring native test before code implementation:** a quiescent mutation barrier that places both authorized endpoints (and any other potentially affected owned participant) into a demonstrably non-forwarding state *before* the shared-ACL update, preserves static v4/v6 leases while quiesced, and permits them to resume only after native ACL/ETag readback plus effective NIC and packet denial/allow checks. If an unrelated NIC fails, keep the endpoints stopped/isolated and all address/ACL claims latched; do **not** resume until the network state has been independently reconciled. Stopping endpoints may be disruptive and is **only a candidate**; whether it prevents fail-open on reload/start and preserves leased addresses must be experimentally proved. A non-disruptive alternative would also need a provable safe barrier; default to no new Peer writes if none exists.

## Authorized native TLS Scriptlet experiment — 2026-11-17 UTC

**Explicit owner permission was granted for a disposable standalone Incus host** (and optional L0), but **only newly provisioned isolated GitHub Actions Incus 7.6 was used**. No shared L0, production cert, physical host, prior Scriptlet or daemon-global state was touched. The disposable CI workflow is [`v1-s4-b2-scriptlet-native.yml`](../../.github/workflows/v1-s4-b2-scriptlet-native.yml).

**Actual green evidence:** [run 44033052427](https://github.com/vanishs/boxweave/actions/runs/44033052427), commit `2f4fe3564c04805f2c0b5c126a625e198f68d94e`, native job `133869301570`, **SUCCESS**. A freshly installed Incus 7.6 daemon listened on local-loopback HTTPS; the job created three independent ephemeral, restricted, real client TLS certificates: `legacy` scoped to `bwlegacy`, `other` to `bwother`, and `writer` to `default`. Curl used the actual client certificate and pinned the Incus server's public key; secrets and private keys were **not uploaded**.

| Native HTTP operation | Original `tls` route | Under disposable `scriptlet` route | After reverting exactly to original route |
|---|---|---|---|
| Legacy GET existing `bwlegacy` Project | 200 | 200 | 200 |
| Legacy GET existing `bwother` Project | 403 | 403 | 403 |
| Other GET existing `bwother` Project | 200 | 200 | 200 |
| Other GET existing `bwlegacy` Project | 403 | 403 | 403 |
| Writer GET its dedicated `bw-s4b2-scriptlet` ACL | 200 | 200 | (not needed) |
| Writer GET `bw-s4b2-not-owned` ACL in same `default` Project | **200** (existing TLS project-wide privilege) | **403** (Scriptlet restricted) | **200** (original TLS behavior restored) |
| Writer PUT dedicated ACL | Not tested | **200**, independently read back through native root ACL show | Not tested |
| Writer PUT foreign same-Project ACL | Not tested | **403**, native foreign ACL unchanged | Not tested |
| Writer POST create new ACL | Not tested | 403 | Not tested |
| Writer DELETE its named ACL | Not tested | 403 | Not tested |
| Writer GET other Project objects | Not tested | 403 | Not tested |

The job then attempted to install **syntactically invalid** Scriptlet source: Incus rejected it (exit 1), the previous restricted Scriptlet rights still worked, and exact `authorization.client.tls-restricted` and `authorization.scriptlet` values were **unset/restored to their original empty settings**; old/other/writer HTTP rights were tested again. The isolated host was ephemeral. A preflight-before-change and a `trap` to unset the two global settings were installed for the authorized test only.

**Important new implementation knowledge:**

1. Upstream [Incus 7.6 `shared/scriptlet/marshal.go`](https://github.com/lxc/incus/blob/v7.6.0/shared/scriptlet/marshal.go) uses **Go JSON tags**, not exported Go field names, for `details.Certificate`. Correct Scriptlet attributes are `details.Certificate.restricted` and `details.Certificate.projects`; capitalized `Restricted` caused real `403 Invalid field "Restricted"` in earlier red probe [44032441909](https://github.com/vanishs/boxweave/actions/runs/44032441909).
2. `details.ProjectName` is the **request's project context**, not always the project identified by the resource URL. An existing Project GET `/1.0/projects/bwlegacy` can have default request context: checking it alone incorrectly denied old clients in [44032642777](https://github.com/vanishs/boxweave/actions/runs/44032642777). Explicit named-Project resource authorization (and the provider's native `object`/entitlement) must be respected.
3. Foreign `GET /1.0/instances?project=...` can return **HTTP 200 with an empty filtered list**, so `200` is not an authorization proof. Tests switched to actual existing named Project objects. That initial mistaken negative assertion failed in [44032250952](https://github.com/vanishs/boxweave/actions/runs/44032250952).
4. The **test Scriptlet only replicated the selected legacy Project-object GET authorization**, while enforcing narrow GET/PUT for the writer. This is **NOT full old-client capability parity**: real earlier restricted clients may also edit/launch/stop/exec instances or view inherited resources. Do not copy the test-only narrow legacy rule into production as if it preserves prior rights. The production Scriptlet must reproduce the documented TLS driver permission matrix including server view limits, project edit refusal, inherited-resource read-only exceptions, all-project semantics, object project and entitlements. Source-exact [Incus 7.6 `driver_tls.go`](https://github.com/lxc/incus/blob/v7.6.0/internal/server/auth/driver_tls.go) must be consulted and a larger native allow/deny matrix independently tested.

**Stage verdict:** The *authorized disposable Scriptlet switch and exact ACL writer* feasibility is now **NATIVE_TEST_PASS**; **installation-ready Scriptlet old-client compatibility is still PARTIAL**, and installed BoxWeave actor/ACK-loss/quarantine/kernel fanout recovery are not accepted. No L0 global changes occurred. B2 stays `S4_B2_PARTIAL`.

### Expanded source-derived legacy TLS *write* parity

A **separate**, fully disposable Incus 7.6 signed-Zabbly runner [44033677061](https://github.com/vanishs/boxweave/actions/runs/44033677061) **SUCCESS** on `2b76eab688947c7359e25490531db727b11a68a5`, native job `133871322553`, exercised [`v1-s4-b2-scriptlet-parity-native.yml`](../../.github/workflows/v1-s4-b2-scriptlet-parity-native.yml). This scriptlet uses the actual upstream [`authorization_objects.go`](https://github.com/lxc/incus/blob/v7.6.0/internal/server/auth/authorization_objects.go) representation `object_type:project/resource` and derives project/server/object permission branches from [`driver_tls.go`](https://github.com/lxc/incus/blob/v7.6.0/internal/server/auth/driver_tls.go), rather than relying only on request paths. Its exact ACL writer still additionally checks the authenticated fingerprint, exact canonical ACL object and exact GET/PUT path+method. These are test-only Starlark formulas **not** a production installer.

The independently observed new HTTP/native controls:

- Existing legacy cert GET owned native Profile 200; GET existing foreign Profile 403. Existing other cert shows symmetric owned 200 / foreign 403. Both preserve named Project 200/403 before, during and after route change.
- **Legacy restricted TLS client PUT its own `bwlegacy/s4b2legacy` Profile returned HTTP 200**, and an independently privileged native `incus profile show` confirmed the exact new description. The same legacy client PUT to foreign `bwother/s4b2other` Profile returned **403**, and that foreign Profile was unchanged.
- Legacy cert GET `/1.0` 200 (permitted limited server view); PUT owned Project configuration 403; POST create global Project 403. The dedicated ACL writer continued to have only the exact ACL GET/PUT, with foreign ACL PUT 403 and native unchanged.
- Invalid Scriptlet source was refused while the existing good Scriptlet remained effective; original TLS route and empty Scriptlet were restored, and the legacy profile read/foreign denial matrix passed after rollback.

**Interpretation:** Source-derived *selected* old-client **read AND write** parity is **NATIVE_TEST_PASS**, including a real instance-scope-independent Profile update, cross-Project negative and exact ACL writer. Still **not** proof of every legacy client's complete native TLS entitlement, inherited default Profile/image handling, instance Start/Exec/Files, or installed service upgrade. Those must be inventoried and tested before B2 signoff. Source exact object identity is much safer than a broad `Method=="GET"` Scriptlet fallback. This closes the previous *only-GET* experiment gap for the tested Profile endpoints but not the full installer gate.

## Important official upstream contract

- [Incus network ACL guide](https://linuxcontainers.org/incus/docs/main/howto/network_acls/): `drop`/`reject` outrank `allow`, ACLs on bridges need **per-NIC** attachment for intra-bridge isolation, and rule update/ETag must not be assumed equal to per-NIC kernel convergence.
- [Incus authorization](https://linuxcontainers.org/incus/docs/main/authorization/): routing `authorization.client.tls-restricted=scriptlet` removes the automatic TLS certificate Project restriction; any migration must explicitly preserve old-client Project allow/deny behavior. **No host-specific global Scriptlet approval was given** and no actual daemon-global change is authorized in this step.

## Step 2 implementation prerequisites

Only after experiments settle the actual native identity/lease and ACL failure boundaries: connect owned bridge/UUID/NIC/IP/MAC checks; ordered claims; pre-release revoke+kernel denial; `UNKNOWN` quarantine and fenced recovery; bounded ETag retries; a reviewed Scriptlet privilege transition with explicit host approval. Still no public Peer grant until the installed gate.

## Step 3 acceptance prerequisites

Separate reviewer of original native operation/ACL/kernel/packet/SQLite evidence, installed restricted HTTP MCP/OAuth, failure cases, and exact-final-HEAD quality+native gates; retain `S4_B2_PARTIAL` until all mandatory evidence exists.

## 2026-11-17 UTC — native quiescent daemon-restart proof, per-flow conntrack risk and Worker pre-PUT contract

**Gate remains \`S4_B2_PARTIAL\`; B3 NOT STARTED.** This section supplements, not rewrites, original failed-GRANT fanout and Scriptlet evidence above.

| Evidence | Exact source / result | Actual conclusion |
| --- | --- | --- |
| [#44042810829](https://github.com/vanishs/boxweave/actions/runs/44042810829) | \`a17f1f7e487a04ce7174ad7240b6bf730a6d6cfe\`, FAILED | Test script called nonexistent Incus 7.6 CLI \`incus waitready\` **after** stopped isolation and failed grant already proved; not a demonstrated ACL barrier failure; fixed next commit |
| [#44043398861](https://github.com/vanishs/boxweave/actions/runs/44043398861), native job \`133903143855\` | \`50ad2eacd91612f00d7ef7088440abe0bd675b74\`, SUCCESS | STOPPED endpoints with disabled autostart retained B UUID and pinned IPv4/IPv6 through native daemon restart after partial failed GRANT, v4/v6 post-restart denial, stopped-address collision refusal, recovered grant real v4/v6 allow, revoke zero v4/v6 nft accept rules, exact cleanup. **However pre-flush IPv4 post-revoke ping REACHABLE**; only test-only global conntrack flush made new v4/v6 flows denied |
| [#44044051904](https://github.com/vanishs/boxweave/actions/runs/44044051904), native job \`133905267137\` | \`bc80e0f43a3df6e0c1c24f4a31069ed8c350c4fc\`, SUCCESS | Added \`conntrack -D -f ipv4/ipv6 -s A -d B\` with original logs; commands ran and later IPv4 packet denied, but pre-delete packet was **already denied**. Hence no proof scoped conntrack flush repairs a still-live preexisting session. No production controller conntrack code or host-wide flush permitted |

**Failure mechanism and product requirement:** Incus ACL ETag/GET and even per-NIC NFT rule removal do not prove all prior conntrack-mediated flows are closed; observed native experiment once kept a revoked IPv4 flow reachable. Verify a *specific already-established flow* before/after exact scoped cleanup without blanket global flush, or choose a conservative barrier that never restarts a pair until independent packet/kernel/flow denial is established. Retain the durable claims/dual-stack address quarantines and UNKNOWN under ambiguous ACL PUT, crash, timeout or peer-dependent Box lifecycle. A disposable standalone runner's root conntrack command is **not** permission to ship root-required networking side effects into restricted BoxWeave runtime.

**Code foundation:** \`156f8252999a31059cbf9b2ea6eab107cfcd3a32\` requires typed pre-PUT \`SynchronousACLIsolation\` callback; nil refuses submission; isolation failure stays UNKNOWN and holds ACL claims; unit tests cover ordering/no ACK/conflicting admission. The unit isolation stub is a **test-only no-op**. Actual production Incus SDK STOP/autostart, exclusive lease/UUID/NIC checks, authorized Scriptlet installer, end-to-end installed OAuth/MCP and independent verifier remain **UNIMPLEMENTED/UNVERIFIED**. Before Go commit: \`gofmt\` verified; in disposable L0 Box Go1.26.9 \`go vet\`, full affected race tests, \`go mod verify\`, \`go mod tidy -diff\` and full \`make verify\` PASSED. Exact \`156f825\` quality gate [#44043538122](https://github.com/vanishs/boxweave/actions/runs/44043538122) and exact \`bc80e0f\` [#44044051970](https://github.com/vanishs/boxweave/actions/runs/44044051970) SUCCESS, not independent installed B2 acceptance.

**Clean containment:** isolated L0 development UUID \`7dff1deb-a1df-4323-963d-09ca3df9a8d2\`, create \`ce685010b2f3d72c65c180223354620a\`, stop \`fb54384ffaaccbd3f34d896f191964ba\`, delete \`0edbc5a042a514262971d2246a71a8a6\`, all SUCCEEDED and native operation IDs recorded in \`EXECUTION_LOG.md\`; final five preexisting Boxes unchanged. No shared L0 ACL/Scriptlet modification. This is **native design evidence**, not a restored native kernel guard in a deployed controller. Keep \`S4_B2_PARTIAL\`, and do not start \`box_peer_set\` B3 until an independently reviewed exact-final-HEAD B2 PASS.
