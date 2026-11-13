# Live gate precedence — 2026-11-13

**Owner-approved new installation gate:** [pilot and release contract](../development/V1_PILOT_DEPLOY_AND_RELEASE_CONTRACT.md). `HOST_GO` is **NOT** required before a separately authorized **reversible dedicated-host pilot**; actual host-only postflight and the remaining required P0/CI/Verifier are done **after** that bounded installation. Current: `ROUND1_PASS=YES`, `ROUND2_PASS=YES`, bounded `LAB_PASS=YES`, real ChatGPT `NESTED_PILOT_CORE_PASS=YES` (scoped), `PILOT_DEPLOY_GO=CONDITIONAL_ONLY` (awaiting exact host/allowed effects and preflight), `HOST_POSTFLIGHT=NOT_STARTED`, `HOST_GO=PENDING_HOST_POSTFLIGHT_AND_REMAINING_P0`. Full per-Operation OAuth Agent evidence in [PR #133](https://github.com/vanishs/boxweave/pull/133#issuecomment-6851880060); 15/15 latest original Ops SUCCEEDED, 0 new UNKNOWN, exact native Box cleanup; remaining simulated IPv6/network-negative matrix, separately signed Verifier, exact-head native/nested Actions. No physical install or Release here. **Historical checkpoints follow unchanged and may mention superseded old gates.**

---

## 2026-11-13 — RFC 9700-complete local OAuth lab checkpoint

**Exact latest code-bearing v1 candidate:** `a8dd90c7bd0c88150e96fba954b3468887c963af`; [updated OA-001–010 acceptance report](S25_EMBEDDED_OAUTH_ACCEPTANCE_2026-11-13.md). Source-level **OAuth 2.1 / RFC 9700 public-client Refresh Token replay detection** added after discovering the preceding `f2af899` implementation only rotated token digests without retaining the consumed digest/grant relationship. Now spent digest-only rows are retained with `expires_at=0`; same-client/resource replay revokes the active grant and Access/Refresh successor atomically. Foreign-client/restricted-resource failures do not revoke an unrelated grant. Tests cover twelve concurrent duplicate refreshes, reopened SQLite replay, and injected read-only SQLite revocation rollback. No database schema migration, additional MCP Tool or public listener.

- Independent clean GitHub checkout on `boxweave-test-ser2`: full `make verify` including all five Go race packages PASS; exact code-bearing [required quality](https://github.com/vanishs/boxweave/actions/runs/43410574621) and [S1 source checks](https://github.com/vanishs/boxweave/actions/runs/43410574556) both SUCCESS.
- The owner-approved separately isolated **inner Guest** `hg-v1-oauth-1113` (UUID `3cc315ad-7e02-4249-a784-6b36570d08e5`, Incus 7.6, restricted `user-1001`) was upgraded through the formal installer to version `1.0.25-hostgo-oauth-a8dd90`; installed SHA-256 `9307011adf0587d07ce8c81ed2d293c8be11715e7dea41ead690d03d27e00a73` independently read back. The previously settled `f2af899` upgrade was formally `install-commit`-ed first, not forced; the **new** exact-prior `f2af899` rollback backup was retained.
- Final read-only postflight: service active, SQLite v2 untouched, no original application Operations, owner-scoped native `user-1001` inventory EMPTY, fictitious test OAuth public origin disabled, all old test OAuth grants inactive, **no Cloudflare Tunnel**, original outer `demo-1-container` and other existing main services preserved. No historic UNKNOWN retries, deletion, adoption or physical host installation.
- **Boundary:** this completes a bounded **local OAuth implementation and security acceptance** only. External ChatGPT OAuth browser login, actual public Quick Tunnel, installed remote authenticated original-key MCP native write/readback and independent Verifier **remain unperformed**; earlier locally generated test-token script was tool-guard blocked and not bypassed. Therefore `OA-008/009=BLOCKED`, `HOST_GO=PENDING_LIVE_AGENT`, physical `HOST_POSTFLIGHT=NOT_AUTHORIZED`, PR #133 Draft. Old Round1/2/LAB signed gates unchanged; Round 3 NOT invoked.

## 2026-11-13 — embedded OAuth implementation and independent local Box lab

**This is a pre-Round-3 OAuth implementation checkpoint, not a HOST_GO stage signoff.** Exact code-bearing commit `f2af899125eda7e6acd0b28a2b36e65eea2ac73d`; [full OA-001–010 evidence and limits](S25_EMBEDDED_OAUTH_ACCEPTANCE_2026-11-13.md).

- One canonical `/mcp`, embedded Go OAuth with RFC 9728/8414/8707/9207, DCR/PKCE S256, Argon2id consent, SQLite v2 atomic code/refresh, revocable OAuth grants and unchanged existing actor/Project authorization. No new public MCP tools, second HTTP process or Flask/main compatibility layer.
- **Independent exact-source lab:** existing nesting-enabled `boxweave-test-ser2` created only the new inner Guest `hg-v1-oauth-1113` (UUID `3cc315ad-7e02-4249-a784-6b36570d08e5`); signed Incus 7.6 / restricted project `user-1001` / new isolated ACL and native prerequisites 8/8 PASS. Native-only disposable L2 actual `print(42)` and exact-UUID cleanup PASS; NOT an OAuth MCP Agent native mutation.
- **Actual installed candidate:** version `1.0.25-hostgo-oauth-f2af899`, verified SHA-256 `8371560029213919d5d4885afb003f9101515150d5913bcc5735bbda7f20fa06`; installed via formal `upgrade` after properly committing the prior verified installer rollback backup. Current service healthy, loopback HTTP MCP 401, both RFC9728 metadata, RFC9207 flag and seven HTTP security negatives PASS. The new installer rollback backup remains intentionally retained.
- Source-level official MCP Go SDK authenticated DCR/PKCE/browser simulation + `initialize`, `tools/list` (13), `host_status`, viewer refusal, JSON/no-SSE; atomic/restart/SQLite failure and revocation negatives PASS. Source exact-head independent `make verify` Go race and [quality](https://github.com/vanishs/boxweave/actions/runs/43408420432) + [S1](https://github.com/vanishs/boxweave/actions/runs/43408420414) Actions both SUCCESS.
- The **installed earlier candidate** performed local browser-like DCR, PKCE, authorization callback and token issuance; an initial manually constructed HTTP MCP `initialize` had incorrect Accept headers (test-harness defect). A subsequent secret-bearing test invocation was blocked before execution; NO installed authenticated Agent MCP original-key mutation/I+N+V is asserted. Real ChatGPT connection through a public Cloudflare Quick Tunnel remains owner-dependent and was NOT performed.
- **Final safe postflight:** fictitious test public origin disabled; no `cloudflared` process or public tunnel, Guest native `user-1001` inventory empty, source Guest and service still running; original older services/L1 and all historic gateway UNKNOWN receipts untouched.

**Gates:** `OA-001–007=SCOPED_LOCAL_LAB` (see distinctions per ID in report), `OA-008=BLOCKED_REAL_CHATGPT`, `OA-009=BLOCKED_REMOTE_AGENT_NATIVE_I+N+V`, `OA-010=PASS_CODE_HEAD` (documentation-head Actions recheck separately). `ROUND1_PASS`, `ROUND2_PASS`, bounded `LAB_PASS` unchanged; **`HOST_GO=PENDING_LIVE_AGENT`, `HOST_POSTFLIGHT=NOT_AUTHORIZED`, PR #133 Draft, no merge or release**.

## 2026-11-13 08:20 UTC — reused existing nested Box; pinned v1 installed, Cloudflare credentials PENDING (not HOST_GO)

**Authorized topology switch:** Owner chose to reuse the *existing*
main-managed `boxweave-test-cli` Box after two uniquely named
top-level gateway Creates independently remained UNKNOWN. Read-only
`boxweave_nesting.inspect` showed `security.nesting=true` on
`boxweave-test-cli`. Both its old installed `boxweave`
SHA256 `715b2d0059b5fa8b15e340c0b0270e9fe4b6f0e5a15b5fd03234012f057796c9`
and the existing L1 `bw-s25-l1a` UUID
`21767890-6d46-4867-889d-0069c1e4ac0d` remain unchanged,
RUNNING and active. Old `user-1002` Incus Project/SQLite/service
were **not** reused.

**Isolated new inner-Guest host:** On **inner Incus 7.6 of
`boxweave-test-cli` only**, created and started one new
default-project nonprivileged Ubuntu 24.04 Guest
`hg-v1-cft-1113`, original native UUID
`c45c08e3-cb41-4354-8bf7-f1eb5651408d`,
`security.nesting=true`, 2 CPU / 2GiB memory. It is NOT a new
main-managed top-level Box or physical host; no new top-level gateway
create was attempted. Initially DNS was absent during first boot,
then DHCP + public HTTPS outbound were independently seen operational.
Fresh inside-only apt Incus package
`1:7.6-ubuntu24.04-202611072126`, running daemon Incus 7.6,
fresh `dir` pool, uid 1001 dedicated nonroot `boxweave`
restricted Incus `user-1001` Project with independently confirmed
foreign-default Project denial. Pinned Ubuntu 24.04 image aliased
inside `user-1001`.

**Exact pinned candidate identity:** Detached *clean* checkout of
Draft PR #133 `df2ba2a6125378044b4d6afb5389cb23ed2c36c0`,
Go 1.26.9/CGO1, binary stamped
`1.0.25-hostgo-isolated-pilot` with that exact source commit,
SHA256 `59dc559bb771e8dca0e16ee4d899d6162eecc735e4cc2e89cd00d0234ac6a511`.
Independently verified after native file push, after installed v1
stage and after service activation. On same exact source, local
`make verify` passed format, YAML duplicate checks,
`go mod tidy -diff`, vet, module verify, all 5 Go race packages.
No source changes to the candidate checkout.

**Actual v1 native and installed positive:** v1 native-prepare-plan
review SHA
`c8012a44dc31c0771ee9015ce45610fb1e58706cb39a3fbd066234d044962b98`
was approved/applied **inside new disposable Guest only**
for `user-1001`, pool `default`,
bridge `incusbr-1001`, new exclusive
`bw-hostgo-1113-acl`, rejecting private IPv4 and ULA IPv6,
enforcing anti-spoof/NIC ownership. `native-prereq-plan` and
exact SHA `native-prereq-verify` both 8/8 PASS, with **explicit
`DIR_POOL_HOST_ROOT_SHARED` / `DIR_QUOTA_UNVERIFIED`** warning.
First install-stage refused when source binary was not readable
from root-only directory; no install manifest/binary/unit was
present at that refusal. Moved identical byte-pinned binary to
root-owned, service-readable `/usr/local/libexec` and
repeated the *proven pre-dispatch* stage. Formal
`install-stage`, nonroot `doctor ready=true` /
`native.ready=true`, `install-activate`, active
`boxweave.service` and loopback-only
`http://127.0.0.1:18650/mcp` unauthenticated HTTP 401 PASS.
No alternate public listener or 2nd service on that Guest.

**Actual independent L2 native-only smoke, with cleanup:** Restricted
`boxweave` actor in `user-1001` created
`hg-v1-native-l2-smoke` native UUID
`c6abe197-c51c-4872-a5e4-8f0c814f7492`,
started it, executed `hello-from-hostgo-l2` and Python
`print(42)` exit 0, independently observed native RUNNING,
then STOPPED/DELETED by identical UUID and verified empty
`user-1001` instance inventory. A shell pipeline returned exit
141 due to `head` SIGPIPE **after successful native Start**;
subsequent read-only native state verified RUNNING and safe
Exec; Start was NOT redispatched. **This is native-only smoke,
not a v1 MCP Agent/Operation I+N+V PASS.**

**Cloudflare preparation, not public enablement:** Verified
Cloudflare-signed APT package `cloudflared 2026.11.0`
inside this Guest. Prepared nonroot
`cloudflared-hostgo.service` with
`--token-file /etc/cloudflared/hostgo.token` and root-owned
config directory; verified systemd syntax. The unit remains
**disabled/inactive** and its tunnel token file does NOT exist.
No Cloudflare account/domain/tunnel ID, public hostname,
route, OAuth Portal or ChatGPT connector has been supplied or
verified. Automated generation of a local admin test token
was blocked by execution permission guard; verified
`/root/hostgo-admin.secret` absent. Do not invent a token or
claim authenticated MCP positive. A Cloudflare **MCP Portal**
can act as OAuth client-facing layer while storing v1
Bearer auth upstream (official Cloudflare documentation), but
it still needs the operator's approved domain, Access policy,
named Tunnel identity and manually protected upstream token.
No one should paste secrets into chat or GitHub.

**Current truth:** LAB signed evidence remains unchanged.
`PREP_NATIVE_READY=PASS`,
`PREP_CLOUDFLARED_INSTALLED=PASS`,
`TUNNEL_PUBLIC_READY=NO`,
`AUTHENTICATED_REMOTE_MCP=NOT_TESTED`,
`INTENDED_AGENT_PATH=BLOCKED`,
`HOST_GO=PENDING_LIVE_AGENT`,
`HOST_POSTFLIGHT=NOT_AUTHORIZED`.
Formal Round 3 not invoked and no final same-head Actions
for Stage 3 claimed. Existing historical gateway UNKNOWN
task IDs `c90bcadde227381550e4528cb5b45e83`
and `428b9e4ff08eee5de9d04bbd0f75a275`
remain original UNKNOWN, never retried/deleted/adopted.
Next operator action: provide the Cloudflare-controlled
hostname / configure a named Tunnel and MCP Portal **without
sharing credentials here**, then securely provision the
Cloudflare token and separately issue the native v1 admin
token from a local trusted terminal. Only after public
auth and owner ChatGPT connection are verified may live
MCP Agent tests begin.

# HOST_GO three-round live checkpoint — do not infer PASS from this file

This is the **small mutable checkpoint** for [the user-triggered three-round runbook](../development/HOST_GO_THREE_STAGE_RUNBOOK.md). Detailed factual incidents append to [EXECUTION_LOG](../development/EXECUTION_LOG.md) and the per-ID [BH matrix](BRIDGEHEAD_ACCEPTANCE.md); never overwrite past failures. Only the currently invoked round can change to IN_PROGRESS, and evidence after the round must be committed to PR #133 before reporting it. A future Agent must read current GitHub and native state rather than trusting a stale checkpoint.

## Current gate semantics — owner's 2026-11-13 explicit correction

**ROUND1_PASS=YES, ROUND2_PASS=YES, LAB_PASS=YES (bounded historical
scope). HOST_GO=PENDING_LIVE_AGENT; HOST_POSTFLIGHT=NOT_AUTHORIZED.**
The previous physical-host-first `HOST_GO` definition was circular:
installing on the real host both required and purportedly granted the
same gate. Under the updated [Runbook](../development/HOST_GO_THREE_STAGE_RUNBOOK.md),
`HOST_GO` is **pre-install candidate eligibility** obtained by a new
separately isolated `main`-managed nesting-capable Box with the **actual
new v1 remote MCP and intended Agent**, original native Incus/SQLite
receipts, independent security verifier, real public IPv4, routed
simulated IPv6, negative cases and final source-exact Actions. The
physical host's separate post-install result is
`HOST_POSTFLIGHT_PASS/NO_GO`, never inherited from LAB or HOST_GO.

The owner separately authorized **preparatory provisioning of one new
nested disposable Box and safe MCP/Tunnel/OAuth readiness** in this
conversation; that permission **does not sign/trigger full Round 3**,
authorize the physical host, waive ChatGPT write-capability/real OAuth
requirements or permit an unauthenticated public service. The Agent
must be configured by the owner and explicitly confirmed before the
subsequent live MCP test. A 401-protected endpoint or tunnel by itself
is not `HOST_GO`.

**New isolated-box preparation attempt, 2026-11-13 07:28Z: BLOCKED,
not a successful create, not Round-3 acceptance.** Existing main
managed backend `dev` was healthy for read inventory, Incus 7.5.1,
daemon 0.128.0; main host had 4 physical CPUs and 28 committed,
so only one 2-CPU/4-GB/35-GB nested isolated Box was requested.
New logical name `bw-v1-agentgate-1113`, original stable
request_key `85d13929b727a493e8aa6a0a5995c7c1`, task_id
`c90bcadde227381550e4528cb5b45e83`, initial
`REST_GATE_PENDING` followed by **`UPSTREAM_CREATE_OUTCOME_UNKNOWN`**.
Read-only `get_container` yielded `NOT_FOUND`, `debug_container`
reported `containerState=missing`; original durable
`boxweave_create_status` remains UNKNOWN. Gateway diagnostics
showed `UPSTREAM_JSONRPC_ERROR`, not a verified no-dispatch.
A separate REST task read refused `TRUST_OR_WORKLOAD_KEY_GATE`.
**Do not repeat, delete, adopt or use the original unknown create,
or infer safety from a missing read-back.** No new Box/installed v1,
Tunnel, OAuth provider, authenticated public MCP or actual Agent
connection is verified; no public URL or password exists. Existing
owned/foreign workloads untouched. Next: independently reconcile
original gateway/native intent under authorized provider diagnostic
authority or obtain an explicitly approved independent clean Box
path; only after safe resolution perform isolated v1 and authentication
setup. `HOST_GO=PENDING_LIVE_AGENT`, formal Round 3 not invoked.

**BH-017 original physical-host verification is retained**, but is now
the mandatory `HOST_POSTFLIGHT` acceptance row rather than an early
`HOST_GO` prerequisite. The live-Agent, native network and independent
negative-case gate remains fully mandatory for `HOST_GO` in the new
Runbook Stage 3. All original failed CI evidence, unknown receipts and
prior narrower signoffs remain unchanged. No PR merge/release approved.

**Owner-approved IPv6 timing contract change (2026-11-12):** Real public IPv6 WAN is deferred to S4 rehearsal / mandatory S5 actual reacceptance; public IPv4 remains real for S2.5. An isolated routed simulated IPv6 HTTPS WAN with the exact installed MCP I + genuine native packet N + independent V, positive/off/on/off and all v4/v6/MAC spoof and peer/foreign negatives is still mandatory. The separately routed **simulated IPv6 WAN I+N+V subcase PASSED** later on 2026-11-12 (original IDs and witnesses in [S25_SIMWAN6_I_N_V](S25_SIMWAN6_I_N_V_2026-11-12.md)); **Round 1 as a whole remains BLOCKED** by other ownership/role/foreign/Agent and full independent BH-009 security gaps; historical public-v6 failures are preserved. `PUBLIC_IPV6_DEFERRED_TO_S5` is not a public-v6 PASS. This docs-only approval does not start any round. [Contract](../decisions/IPV6_PUBLIC_WAN_PHASED_ACCEPTANCE_2026-11-12.md).

## Formal 2026-11-13 Round 1 + Round 2 isolated LAB acceptance

**ROUND1_PASS = YES; ROUND2_PASS = YES; LAB_PASS = YES in the
owner-approved bounded LAB equivalent scope. ROUND3 =
READY_FOR_EXACT_HOST_APPROVAL; HOST_GO = NOT_AUTHORIZED.** This
supersedes older status rows only as of this later independent review.
It does not erase historical failed runs or UNKNOWN receipts.

- Every BH-001 through BH-016 and TC-011 through TC-014 has an
  evidence-class I/N/V/T and version/owner-reuse entry in
  [the formal 20-row crosswalk](HOST_GO_R12_FORMAL_LAB_CROSSWALK_2026-11-13.md).
  Explicit exception 1: latest GitHub ephemeral Guest direct public
  IPv4 on test-only d671d0c FAILED by 5-second timeout, not PASS.
  The owner accepted as LAB equivalent the source-Go-exact private ACL
  with genuine direct Guest HTTPS public IPv4 ON/OFF/ON and independent
  native V on an isolated second Box, combined with the earlier
  Codex/native MCP and same-current-policy installed SDK packet proof.
  The failed original ID 87e8ecf655fb82172c409ca030ce627f
  was never retried. Actual installed MCP I+N+V real public IPv4 on
  the exact approved physical-host binary is a **BH-017 hard gate**,
  never an inherited LAB claim.
- Explicit exception 2: BH-015 composites separate original native
  S2 and installed nested L0/L1 binaries from the same source, each
  independently pinned by its own real executable SHA; never falsely
  one cross-workflow byte-identical SHA. Stage-3 candidate must
  independently prove a single pinned binary for its own host.
- Owner instructed retention of previously proven model-driven Codex
  CLI TC-013 PASS rather than repeating login/write every update.
  The true physical-host Agent workflow is still BH-017.
- Latest unmodified accepted runtime/source 07084fd: five full
  Actions SUCCESS, including native Incus and independent nested
  MCP/Verifier; repair source 457c6dc was 5/5 too. The d671d0c
  direct Guest public IPv4 TEST-ONLY attempt was reverted after
  diagnosing the runner limitation; failed original evidence is
  preserved. No product/ACL security weakening.
- Genuine public IPv6 deferred to S5; routed simulated IPv6 HTTPS
  accepted, and native dir hard quota remains DIR_QUOTA_UNVERIFIED
  if not independently enforced. Existing shared Incus/unknown
  protected; exact new disposable native public4 test Project,
  network, image, ACL, Guest UUID and tokens cleaned; original
  demo-1-container and bw-s25-l1a remained running.
- **Stage 3 entry only:** exact physical host not identified, no
  new host privilege or native mutation approved. Do not invoke
  installer or physical-host discovery until the owner's fresh
  explicit exact host identity/effects/risk envelope.

## Latest owner-requested Round-1/2 three-P0-gate remediation — 2026-11-13 UTC

**Three named LAB gap groups are now LAB-SCOPED PASS, with owner-approved
controlled fault / routed IPv6 HTTPS substitutes accurately labeled.
Formal overall ROUND1_PASS, ROUND2_PASS, all-BH/TC LAB_PASS, physical
HOST_GO and Round 3 are NOT declared by these targeted results.**
This is a later evidence checkpoint, not a deletion of any earlier
failure/UNKNOWN record.

- Source-tested PR #133 code-bearing revision
  457c6dcf8f95feba64ce4d8ae936cbcf12297b92;
  same-source Actions **5/5 SUCCESS**: S1 #43372818101;
  quality #43372818066 / #43372809733; real native S2+
  hardened-preflight #43372809632; L0/L1/L2 nested MCP, real foreign
  v4/v6 packets and independent verifier #43372809750.
  Exact nested candidate binary SHA256
  f8d6487b607b5fc234ffd7664bae04e7dffb8d63ae93676b29136341e76feca2;
  signed Incus 7.6; original evidence ZIP artifact #14012879072,
  independently read SHA256
  5a5cf57506176183b2e8a41da53cbcedc8305eb38e771c631537e3121862c2c3.
- **BH-009/BH-015 network submatrix = LAB-SCOPED PASS**:
  initial internet=true routed IPv4 foreign Project penetration at
  79804ff was a real P0 defect; fixed via exact dedicated ACL
  private IPv4/ULA egress reject with one-gateway carveouts, retaining
  fe80::/10 RA/NDP. Same-binary native L1 bidirectional cross-Project
  IPv4/IPv6 packet denies and positive gateways, native socket separation,
  actor/Viewer/anti-spoof and independent original native UUID cleanup
  PASS; public IPv4 HTTPS is genuine, IPv6 WAN is **separately routed
  simulated HTTPS**, not public IPv6. Parent Incus/SQLite never exposed.
- **BH-007/008/012/015/016 fault and rollback submatrix =
  LAB-SCOPED PASS**: real original native ACK losses, SIGKILL, SQLite
  WAL/ENOSPC, recorded UNKNOWN no-replay, independent native/SQLite
  reconciliation and exact offline rollback. Controlled fault proxies
  are bounded simulation of the failure boundary, not physical power loss.
- **TC-011/012/014 two-Agent submatrix = LAB-SCOPED PASS**:
  distinct authenticated official SDK Agents, one installed native
  Exec after original lost MCP HTTP reply, new Agent recovering one
  immutable original, revoked credential 401, no replay, independent V.
  Prior actual model-driven Codex CLI original-key v1 lifecycle and
  reconnect at 0cf474f remains distinct TC-013 evidence and was **not
  rerun against the new disposable CI binary**.
- The final blocker was correctly strict native ACL exclusivity: S2
  two-controller old test reused the same ACL for user-1002 and
  user-1003 Profiles; diagnostic run #43372141160 showed two used_by
  references despite exact correct private CIDRs. Fixture at 457c6dc
  gives second controller a distinct ACL. Native final used_by now
  only the original user-1002 Profile and preflight SHA/drift negative
  checks PASS. Unrelated existing demo-1-container and historical L1
  remain intact; no shared Incus, physical host or historic UNKNOWN
  mutation. Fresh exact-host authority is still required for Round 3.
- [Full chronological failed and passing evidence, IDs, hashes, limits,
  independent cleanup and exact Actions](HOST_GO_R12_P0_NETWORK_AND_FAULT_REACCEPT_2026-11-12.md).

**Remaining formal boundary:** do not convert this targeted three-group
PASS into overall Round1/Round2 or LAB_PASS without individually closing
and recording all other outstanding BH-001–016 and TC cases in the
authoritative matrix. BH-017 actual dedicated host remains
NOT_AUTHORIZED. No automatic Round-3 invocation, v1 merge or release.

## Planning snapshot (2026-11-12, documentation only)

- Existing **tested source**: `7a01000a12114eba0fe4937af0e955bd6e46ed87` with expanded nested MCP LAB 5/5 at that revision. Latest **PR #133 documentation checkpoint**: `31d91584f07ff19135e96f086cd348bd4075f45d`, five same-head Actions SUCCESS, Draft against `v1`. On current head the candidate v1 binary digest was `ee64e247e6054ec62d3e93db4c2a25c51f369816d9b80a9faab07dde2658d3e4`. Reconfirm exact current head and artifacts on invocation; these are historical baselines, not authorization.
- [Exact-head formal matrix](S25_EXACT_HEAD_FINAL_LAB_2026-11-12.md): BH-014 individually signed; remaining BH-001–016 cases PARTIAL/BLOCKED, BH-017 NOT_AUTHORIZED, TC-011–014 not fully signed. Expanded LAB PASS is **not formal LAB_PASS**.
- `LAB_PASS=NO`; `HOST_GO=NOT_AUTHORIZED` until accepted real-host proof. Physical host exact identity has not been provided; it **must** be separately and explicitly approved in Round 3.
- Quota stress waiver remains; `DIR_QUOTA_UNVERIFIED` must be disclosed. No old `main` co-installation is needed or permitted. No production host or shared Incus mutation by the author of this planning document.
- Historical UNKNOWN: `7e1cbf043d1fd4ea3d9ec0093a04371e`, `f811afeeb778c422fe99b9881baa40f2`, `192a6601c54078f8eaf1437a6a3dc13b`, `a2fdf885b87a0b66fb3a17f913316c55` — no write retry, deletion or adoption.
- `boxweave-test-ser2` is a previously used isolated development Box; `boxweave-test-cli` contains an existing native-only L1 `bw-s25-l1a` in Project `user-1002`. Before using either, read-only inventory + native consent/ownership; never presume current availability or an empty Project.

## Latest owner-requested combined Round-1/2 reacceptance (2026-11-12 UTC)

**TWO_RETAINED_GUESTS_CLEANUP=PASS**: original successful Guest UUIDs
6b51eb00-fd33-4dfb-8d30-eadd55753a5f and
aa79c354-4019-405f-ad00-0731d1553bd6 were stopped and deleted via
new authenticated original v1 MCP intents; both native absent and
owned_boxes rows DELETED. Old service and foreign bw-s25-l1a unaffected.
**Round 1 remains BLOCKED** because BH-009/BH-015 exact-current-source
full L0+L1 anti-spoof/public-v4/independent matrix is not signed.
**Round 2 remains BLOCKED** because TC-013 actual intended ChatGPT
development Agent uses legacy Gateway and lacks an approved v1 MCP
connection, plus remaining individual BH/TC formal crosswalk.
Same prior PR head d613737 exact Actions 5/5 SUCCESS; SHA-checked
independent native artifact is bounded evidence, not formal ALL PASS.
LAB_PASS=NO; HOST_GO=NOT_AUTHORIZED; Round 3 NOT_STARTED.
[Detailed original Stop/Delete receipts, CI artifact and open gates](HOST_GO_ROUND12_CLEANUP_FOLLOWUP_2026-11-12.md).
This checkpoint adds evidence, does not erase prior failed attempts,
and is not permission to replay any native UNKNOWN.

## 2026-11-12 owner-authorized Codex + same-binary dual-layer I+N+V continuation

**Round 1 remains BLOCKED, but its current-source dual-layer packet, role, original Operation, cleanup and separate-Box Verifier submatrix is PASS.** On source 0cf474f with L0/L1 exactly the same installed binary SHA256 021127cf723bfbe96c3668965b1e60dcc82856e5e3d41e09e84982568b5c1e85, both layers independently checked 13 official MCP tools, strict unproxied public IPv4 TLS, IPv4/IPv6/MAC spoof-with-valid-controls, bidirectional sibling denial, Viewer/expired/revoked auth negatives and native same-UUID operation finality. L0 also checked foreign-owned existing L1 Box bidirectional IPv4/IPv6 denial with positive controls. Separate Box verifier rechecked exact source/binary and per-layer 12 predelete, 16 postdelete original SUCCEEDED Operations, native packet logs and safe cleanup (4/4 new owned Guests DELETED, zero new native UNKNOWN). Earlier same-SHA test cohort demonstrated separately routed simulated IPv6 WAN I+N+V at L0/L1, not real public IPv6. Remaining formal BH-009/BH-015 full current-binary/L1 foreign Project/parent-socket/install/fault crosswalk is not signed; no unilateral P0 waiver.

**Round 2 remains BLOCKED on TC-013 CODEX_LOGIN_REQUIRED.** Owner selected real Codex CLI 0.167.2; independent nonroot actor bwcodex and private loopback Bearer-env MCP client configured, but actual login status reports Not logged in, so no actual Codex model-driven Tool discovery/read/original-key write or reconnect can be claimed. Scoped native SIGKILL recovery and existing Go SDK two-Agent/rollback CI evidence remain scoped positive, not a substitute for Codex. No physical host, Round 3, historic UNKNOWN, main service or release changed. [Complete original IDs, signed packet/SQLite snapshots and remaining blockers](HOST_GO_R12_CODEX_R1_NATIVE_REACCEPT_2026-11-12.md).

## Codex login completion and real model-driven read attempt (2026-11-12 PT)

**Codex ChatGPT login = PASS.** Dedicated low-privilege bwcodex actor now returns Logged in using ChatGPT. However the BoxWeave v1 Bearer environment variable BW_V1_MCP_BEARER is absent in the actual Codex process. A genuine model-driven read-only Codex exec completed but reported the private boxweave-v1 host_status tool unavailable: **no v1 MCP Tool read occurred**. Direct v1 unauthorized request returned 401, correct fail-closed. Attempted protected Viewer credential handoff was safety-blocked; no credential bypass or native effect occurred. The previously created empty placeholder was cleaned.

**TC-013 and Round 2 remain BLOCKED: V1_MCP_BEARER_NOT_PROVISIONED.** Owner device login requirement is now satisfied, but live model-driven MCP call, original-key mutation and independent recovery are still unverified. Round 1 still has the other individually recorded mandatory case-level blockers. At prior document head d051e98, Actions 5/5 SUCCESS; new documentation commits require new exact-head Actions. Detailed evidence: HOST_GO_R12_CODEX_R1_NATIVE_REACCEPT_2026-11-12.md.

## Actual Codex Viewer read confirmed (2026-11-12 PT)

Owner issued local expiring Viewer credential. Real model-driven Codex
0.167.2 called private boxweave-v1/host_status once, completed with
mcp_ready=true and ready=true. Separate v1 candidate service journal
confirms actor 4979e4e0 initialize, tools/list and tools/call result=ok
(144ms). No native mutation or new Guest; original foreign L1 intact.
**TC-013 real Agent read subcase PASS, full TC-013 BLOCKED** until
original-key mutation, crash/recovery and independent verifier.
The previous uncredentialed Codex failure is retained, not erased.
Previous exact documentation HEAD 920d719 Actions 5/5 SUCCESS.
Report: HOST_GO_R12_CODEX_R1_NATIVE_REACCEPT_2026-11-12.md.

## Latest live Codex Round-2 lifecycle and separate-Box snapshot V (2026-11-12 PT)

**TC-013 = PASS** at the actual-Agent discovery/auth/read gate; real
Codex CLI also used on-request + Auto-review to create, start, stop and
delete one brand-new disposable native Guest with four distinct original
idempotency keys, four original BoxWeave and four native Incus Operation
IDs, original UUID and durable work/Agent references. All four original
operations settled SUCCEEDED, and the exact native Guest was removed,
original foreign bw-s25-l1a remained RUNNING, zero UNKNOWN. The locally
approved temporary Admin credential was revoked and deleted.

A guarded in-flight SIGKILL watcher saw only DISPATCHING and **did not
inject**; no in-flight crash PASS. Separately, the candidate-only
SIGKILL **after terminal** and same-binary/SQLite/Project restart
succeeded, and a new actual Codex model-driven session re-read two exact
original receipts and Box state through v1 MCP without replay. An
independent separate-Box verifier crosschecked source native/SQLite
pre/post snapshots and signed its scope as PASS_SCOPED_SNAPSHOT_CROSSCHECK,
not as independent direct Incus socket V or in-flight recovery PASS.

**Round 2 remains BLOCKED** pending TC-011/012/014 and BH fault/recovery
in-flight/lost-ACK original-case signoff. **Round 1 remains BLOCKED**
on its separate L1 foreign-Project/native parent socket and installation
fault integration gaps. Round 3 NOT_STARTED. Exact evidence:
[Real Codex Round-2 report](HOST_GO_CODEX_REAL_AGENT_R2_2026-11-12.md).

## Round scoreboard

| Trigger | Status | Cases / evidence | Exact code and native binary | Remaining blockers | User told? |
|---|---|---|---|---|---|
| 完成HOST_GO 第一阶段 | **PASS (owner-approved bounded LAB equivalent)** | BH-001/002/003/005/009/010/011/015 and TC-011; full per-case 20-row exact-version crosswalk, actual native packet+socket V, installed SDK, historical Codex | accepted source 07084fd; native original pinned SHA per LAB; second Box current-source native ACL exact CIDR | GH Guest direct public-v4 test-only d671d0c timed out; accepted separate exact-ACL native public-v4 ON/OFF/ON I/N/V composition; mandatory repeat on approved physical BH-017 | YES |
| 完成HOST_GO 第二阶段 | **PASS (owner-approved bounded LAB evidence)** | BH-004/006/007/008/012/013/014/015/016 and TC-011/012/013/014; original native ACK/UNKNOWN/rollback, lost MCP reply, actual Codex historical, two distinct SDK Agents, independent original V | accepted source 07084fd, separate native/nested per-run exact binary SHA; 5/5 original Actions PASS | No repeated Codex login required by owner; actual host Agent, host capacity and exact single binary pinned required in BH-017 | YES |
| 完成HOST_GO 第三阶段 | **READY_FOR_EXACT_HOST_APPROVAL (HOST_NOT_AUTHORIZED)** | All LAB BH-001–016 / TC-011–014 closed with documented owner-approved equivalent boundaries | actual physical source/binary/host identity unset | Need owner's fresh exact standalone host/OS/Project/pool/ACL/port/rollback authorization; until then no host discovery or installation | YES |

## Mandatory round-end entry (overwrite the active row only; preserve prior checkpoint elsewhere)

For the round that the user actually invoked, record `started_at`, `completed_at`, `stage_verdict` (PASS/PARTIAL/BLOCKED), exact git branch/commit and binary SHA, current candidate signed Incus runtime, the actual host/Box inventory and native original IDs, BH/TC statuses with I/N/V, verifier identity, original UNKNOWN and failed operation receipts, cleanup and retained Boxes, commit/PR comment/Actions URLs, and `next_ordinal_requires_new_user_turn=true`. Add an append-only incident in `EXECUTION_LOG.md` and show the owner the result. Do not self-invoke another ordinal.
## Round 1 evidence and preserved checkpoint (2026-11-12 UTC)

- **started_at:** 2026-11-12T10:06Z (conversation-local approximate); **completed_at:** 2026-11-12T10:16:53Z (last observed Box clock); **stage_verdict:** **BLOCKED**, not ROUND1_PASS, not LAB_PASS, not HOST_GO. This user invocation authorizes only Round 1.
- **PR / exact tested code:** #133, `feat/v1-s25-native-readiness-20261111`, `7c31700225497727e1f87c149827cd9051e2da0b`; native prerequisite read-only candidate built with `go1.26.9`, `-trimpath`, and embedded `main.commit=7c31700225497727e1f87c149827cd9051e2da0b`, SHA-256 `23c2ca9ee4fe8943ce86d0c0e93c5f17a2549c4659a1a13f5dae54d2055f227f`. It was **not installed** as an MCP service. This checkpoint documentation commit changes Git HEAD; it does not certify a newly rebuilt binary at that later documentation HEAD.
- **Box-first inventory:** legacy-managed isolated `boxweave-test-ser2` (Incus 7.6 APT `1:7.6-ubuntu24.04-202610301520`, current source checkout at exact test SHA, pre-existing `demo-1` untouched); separate `boxweave-test-cli` (Incus 7.6 APT `1:7.6-ubuntu24.04-202611072126`, old running v1 `8af4e080`, private Project `user-1002`, ACL `bw-s25-lab-exclusive`, existing L1 `bw-s25-l1a` native UUID `21767890-6d46-4867-889d-0069c1e4ac0d` untouched). Original tracked hook worktree changes in ser2 were not reset or committed.
- **New test Boxes blocked:** original create keys `a10f7112f1a04cc78d4dfda6b7110201` / `b10f7112f1a04cc78d4dfda6b7110202`, original task IDs `fb7b54a1f5a06da16bc7c617e5ae4b27` / `3fede508363e572babcf61e6177ea974`, names `bw-hostgo-r1-finder-1112` / `bw-hostgo-r1-verify-1112`. Both initially `REST_GATE_PENDING`, then `UPSTREAM_CREATE_OUTCOME_UNKNOWN`; original status read again remained `UNKNOWN`. Read-only get returned `NOT_FOUND`, debug reported `containerState=missing` on the backend. **Missing is not proof that a possibly dispatched Create can be safely retried.** No repeat, delete, adopt or new same-name intent; preserve both receipts.
- **Historic UNKNOWN read-only status:** `7e1cbf043d1fd4ea3d9ec0093a04371e`, `f811afeeb778c422fe99b9881baa40f2`, `192a6601c54078f8eaf1437a6a3dc13b`, `a2fdf885b87a0b66fb3a17f913316c55` all still `UPSTREAM_CREATE_OUTCOME_UNKNOWN`. No native or gateway writes to them.
- **Exact-head candidate local proof:** `make verify` PASS (format/YAML, mod tidy-diff, vet, mod verify, full CGO race); root-local `native-prereq-plan -project user-65534` returned 8 checks, exactly 1 true, `config_ready=false`, `packet_witness_required=true`. `doctor -project user-65534` refused ready (`REFUSE_ROOT_RUNTIME`, exit 1). Both read-only and **not** an installed candidate I+N+V acceptance.
- **Older installed/native bounded reference only:** on `boxweave-test-cli` old installed `main.commit=8af4e080`, loopback MCP invalid Host/Origin/path/no bearer gave 403/403/404/401. Native Project profile readback showed private ingress reject and v4/v6/MAC filtering, with explicitly per-L1 egress allow; ACL used_by was only the owned profile/L1. L1 direct public IPv4 HTTPS returned 200; L1 IPv6 had a route but HTTPS failed (000), and L0 IPv6 to an alternate public destination also failed (000). Treat as `NATIVE_WAN_ENV_BLOCKED_OR_UNDIAGNOSED` on the **older** candidate, not proof of current HEAD IPv6 correctness or a confirmed product bug. No spoofed packet / false-positive enforcement claim was made.
- **Evidence grades / matrix:** current exact source has root-local nonmutating CLI+native N partial and source T, **0 newly installed MCP I, 0 same-candidate independent verifier V**, and **0 newly signed complete I+N+V BH/TC cases**. BH-001/002/003/005/010/011/015 and TC-011 remain PARTIAL/UNVERIFIED; mandatory BH-009 is **BLOCKED** (current installed I+N+V, direct v4/v6 WAN, sibling/foreign and spoof negatives not witnessed). BH-017 physical host not authorized and not attempted. Do not relabel old observations.
- **Cleanup / scope:** no test-owned new native instance exists by verified read-only get; this is not resolution of the two UNKNOWN receipts. No ACL/Project/global Incus or existing L1 mutation, no physical-host change, no deletion. Existing four historic UNKNOWNs and foreign objects retained. Independent verifier cannot sign an absent test installation.
- **GitHub source checks prior to the reporting commit:** exact tested SHA `7c31700225497727e1f87c149827cd9051e2da0b` S1 source check run `43196663428` SUCCESS and required quality run `43196663520` SUCCESS; native BH-009 installed suite **not run for this HEAD**. After this documentation commit, re-check Actions for its new HEAD; source green never substitutes for a blocked native gate.
- **Safe resumption:** independently reconcile the two new gateway UNKNOWN outcomes (without retry/adoption), obtain two available authorized fresh disposable Boxes or isolated independent hosts, then install the same new candidate with complete owned native prerequisites; repeat exact binary I+N+V on IPv4/IPv6 public WAN, peer/foreign, spoof, tokens, and cleanup. **next_ordinal_requires_new_user_turn=true.** Round 2 may perform unrelated safe tests if separately invoked, but may not infer Round 1 PASS.
## Round 1 resumed Box-first acceptance — 2026-11-12 UTC

- **Resume requested by owner:** "请您开始" after asking to repeat Box test/fix/verify; scope remained Round 1 exclusively. **Verdict still BLOCKED**, but new native partial proofs were obtained.
- **Tested source SHA:** `e8782860f9c43a21af1193ff9cb9225f3619f81c`; installed binary SHA-256 `68a1f2b8e9b6e5b5ecbfbc88d5ec3de4549989c944b55159a828ddf0c1c31c31` reproducibly rebuilt in two separate isolated Boxes with Go1.26.9. This checkpoint's eventual docs-only commit will have a later Git SHA; no implicit binary identity rebinding.
- **Native/local:** signed Incus 7.6 on disposable `bwracc-1106-v2` and nested `boxweave-test-ser2`→owned, unprivileged L1. Two distinct restricted OS actors/Projects and dedicated ACLs; real installed `doctor ready=true` and root-local native prerequisites 8/8 with wrong-SHA refusal. Full `make verify` PASS. Current-source installed MCP HTTP 401/403/404 negative boundaries PASS; official SDK authenticated original-key writes NOT run.
- **Packets:** native v4 default deny vs per-guest allow gives real direct HTTPS=200, bidirectional v4/v6 sibling and foreign deny, forged v4/v6/MAC deny with legitimate gateway positive controls, native nft/effective-NIC witnesses. Direct public IPv6 failed also at outer L0 → `NATIVE_WAN_ENV_BLOCKED_OR_UNDIAGNOSED`, not product FAIL/PASS.
- **Verifier:** independent outer-root native poststate/installed binary readback, but no distinct-host same-version MCP I+N+V signoff. Zero newly fully accepted formal BH/TC cases; BH-009 BLOCKED.
- **Original unknowns:** two new gateway UNKNOWN original `fb7b54a1f5a06da16bc7c617e5ae4b27`, `3fede508363e572babcf61e6177ea974` still UNKNOWN; four historical UNKNOWNs untouched. No retry, adoption or deletion.
- **Native cleanup:** `ser2` siblings UUIDs `b0821421-8348-43a9-99d6-71fc5b019a33`, `c86c153c-4277-49bc-b234-d28acb061c19` and nested L1 `81b57e94-f2b5-4424-b507-c1a2813ba293` stopped/deleted with native absence readback. Other isolated host's failed-start `f35945a1-b6c5-49a5-a569-8d6b222675ac` deleted by original UUID; installed v1 service stopped/disabled. Exclusive empty test Projects/ACL and sealed evidence retained in disposable Boxes for possible safe continuation; foreign `demo-1` unchanged.
- **Detailed audit:** [Round 1 resumed report](HOST_GO_ROUND1_RESUME_2026-11-12.md); add exact final same-head Actions links after publication in PR #133 comment. **next_ordinal_requires_new_user_turn=true**.

## Round 1 additional scoped IPv6 simulator I+N+V acceptance — 2026-11-12 UTC

- **Not a new execution of Round 2 or 3:** user specifically authorized completion of simulated IPv6 WAN I+N+V within Round 1.
- Exact tested source `f2ef9670d3a99d69ec314d054d56f1602d9396f0`; candidate binary SHA-256 `be924579f64f373c9afbce9c1cb999706bf3cc468d46ef7696d0a88088a3bef2`, signed Incus 7.6 package `1:7.6-ubuntu24.04-202611072126`.
- Independent official SDK MCP/native/SQLite Verifier all **PASS** for L0 and nonprivileged nested L1, separately routed IPv6 namespace HTTPS certificate CA/SAN and per-Guest off/on control; actual original SQLite operation, UUID, native NIC and cleanup confirmed. [Complete run #43257185059](https://github.com/vanishs/boxweave/actions/runs/43257185059) SUCCESS; artifact 13963520912. Native/Box-first off→on and spoof negative and foreign cleanup independently retained.
- Prior [CI failure #43255943147](https://github.com/vanishs/boxweave/actions/runs/43255943147) due to Verifier restricted log read and [failure #43256631404](https://github.com/vanishs/boxweave/actions/runs/43256631404) due to test-client nested cleanup environment are preserved; fixed only test harness, not BoxWeave production authorization.
- **`SIMULATED_WAN_IPV6_I_N_V=PASS`**, **`PUBLIC_IPV6_REAL=DEFERRED_TO_S5`**, but **BH-009 full and ROUND1 remain BLOCKED** pending remaining I+N+V and Agent/security case gaps. LAB_PASS=NO, HOST_GO=NOT_AUTHORIZED. No physical deployment, no release, six gateway UNKNOWN original operations not replayed. All newly owned native fixture Guests and network namespace removed; foreign `demo-1` still RUNNING.
- Full [scoped evidence and original IDs](S25_SIMWAN6_I_N_V_2026-11-12.md). This pending evidence/archiving checkpoint is a later docs SHA than installed accepted binary; any new same-head Actions must be checked without relabeling tested executable bytes. **next_ordinal_requires_new_user_turn=true**.

## Round 2 user-triggered and resumed — 2026-11-12 UTC

- **Verdict: ROUND2=BLOCKED / selected installed fault, SIGKILL, ACK/SQLite and offline rollback I+N+V PASS; TC-013 actual intended Agent remains BLOCKED.** [Detailed original receipts, complete I/N/V and cleanup report](HOST_GO_ROUND2_2026-11-12.md). User separately invoked `完成HOST_GO 第二阶段` then requested continuation after the earlier response stopped; this is **only** Round 2, not an automatic transition.
- **Exact tested source** `ec40d20f49af8603c5438da84346f5d30d575174`. Disposable Actions installed candidate digest `bdfd8646ec500a6d13ad7784e4365ad639db6dfce8a0317aa4dc8ebd19167e51`. Independent Box-first `ec40d20` executable digest `1d1ba0f2d8c14d567d93073ddd6fa57fb1de323b911fb886413ec2bd23bea804`, differing **only by intended build-label ldflags**; explicit non-equivalence retained. Source Go1.26.9, signed Incus7.6. Both original/rollback native and SQLite identities reviewed.
- **Actions** tested source `ec40d20`: [nested I+N+V #43269999551](https://github.com/vanishs/boxweave/actions/runs/43269999551) SUCCESS, [native S2 lost ACK/WAL/ENOSPC/revoke #43269999281](https://github.com/vanishs/boxweave/actions/runs/43269999281) SUCCESS, S1/race [#43270011818](https://github.com/vanishs/boxweave/actions/runs/43270011818) and 2 required quality runs [#43270011928](https://github.com/vanishs/boxweave/actions/runs/43270011928) / [#43269999365](https://github.com/vanishs/boxweave/actions/runs/43269999365) all SUCCESS. Archived artifact `13968487159`, SHA-256 `2b5baa96c8823ad29b269456dec6224564cc1755df5ea85e238a19dcca8f0e6a` opened independently.
- **Box-first original L1 UUID** `f626044f-996b-45ab-8666-e676811de7ff` (`boxweave-test-ser2` / confined Project `user-1004`) and actual inner v1 service original PID `4061` → after dedicated SIGKILL `4114`, NRestarts=1, exact original SHA and HTTP 401 retained; bad upgrade SHA/no-backup rollback rejected, canary `1c5aac03e6322ade6de0505ea4dbcf14e8ac170bc4957748e699272fe244366a` intentionally upgraded with backup and operator-root-local offline rollback restored original digest and HTTP auth. 8/8 native prerequisites, separate ACL/foreign deny and root-shared `dir` warnings accepted as warnings only. **This Box made no MCP authenticated mutations**; official SDK and ledger crosswalk are separate same-source disposable Actions I+N+V, not asserted as identical executable bytes.
- **Cleanup:** exact original `f626044f-996b-45ab-8666-e676811de7ff` verified to have zero inner native guests, local service safely disabled/stopped, L1 stopped/deleted by UUID, owner `user-1004` empty. Previously created `hgr2-fault-l1` original UUID `0903b47d-5adf-4a5d-aa77-7fa117c0bb1f` was found absent in independent read-only inventory on resume and was not retried. Foreign `demo-1-container` remains RUNNING. Six original external gateway UNKNOWN records still UNKNOWN, all untouched. No physical/global daemon changes, no merge/release.
- **Case crosswalk:** BH-004/006/007/008/012/013/015/016 and TC-011/012/014 remain **PARTIAL** despite selected strong passes; BH-014 prior exact-source scoped **PASS** preserved. TC-013 is **AGENT_PATH_BLOCKED**, because the connected ChatGPT BoxWeave tool is the old gateway and not the installed v1 MCP; no approved credential-safe v1 connection. Never substitute official Go SDK for actual chosen development Agent.
- **Same-head Actions:** this doc-only reporting commit follows tested `ec40d20`; recheck new HEAD Actions separately. **Round 1 remains BLOCKED, LAB_PASS=NO, HOST_GO=NOT_AUTHORIZED, Round 3 NOT_STARTED**. **next_ordinal_requires_new_user_turn=true**, and Round 3 also requires fresh, exact physical-host approval.


## Round 1 re-invoked, exact-source 58626 local I+N+V progress — 2026-11-12

- **Trigger/round:** owner explicitly repeated `完成HOST_GO 第一阶段`; **ROUND1=BLOCKED**, `LAB_PASS=NO`, `HOST_GO=NOT_AUTHORIZED`; only Round 1 was executed, **next_ordinal_requires_new_user_turn=true**.
- **Exact tested source:** `58626acb4d3ddf11f5bcd4e9fb0fd5d7fe87095f`; the current-source binary was independently built in **three isolated Boxes** with Go 1.26.9, all SHA-256 `640cdd28d601b84ed3e1a35bdaf50322e13db46084417459d585a8d7ca1fb34f`. `make verify` on ser2 PASS, tree clean. Same source did not modify product Go since earlier simulated-IPv6 proof, but **new binary acceptance is not inferred**.
- **Current-source accepted local subproof:** nesting-enabled `boxweave-test-cli` original two official SDK MCP Create `SUCCEEDED` and real Incus UUIDs, authenticated on/off/on (original `box_update` was ON then OFF), public IPv4 HTTPS **HTTP 200/TLS verify 0**, Internet-off negative, IPv4/IPv6 sibling peer denials and self positive controls, source IPv4/IPv6 and raw-MAC spoof negatives with legitimate packet controls, Viewer/foreign UUID/token expiry/revoke denial. Separate same-host read-only verifier reconciled original SQLite/native UUID/ACL/MAC receiver log; **not full independent L0+L1 gate**. [Full evidence](HOST_GO_ROUND1_58626_REACCEPT_2026-11-12.md).
- **Two new local UNKNOWNs on outer nesting-disabled `bwracc-1106-v2`:** original application Ops `4de5c57f4f55946862794f06f210d699` and `13d66b5e290227d90c94b3dacb4407d4`; native Ops `ddddb2b6-e02f-4ff7-a4df-f744ce406e62`, `74a2082d-9b77-4065-b6ff-e425c3fa239c`; native STOPPED UUIDs `90e3fb4b-1e99-4b77-b83a-18756f53fc9e` and `db9ee003-558a-4d2e-be1e-c376613f7a29`. Original native LXC proc/cgroup permission denied; provider independently read outer `nesting=false`. **No replay, delete or adoption**.
- **Two successful yet retained owned test Guests:** CLI A original Create `e89ce4335a89e3e1590497792708cc92`/UUID `6b51eb00-fd33-4dfb-8d30-eadd55753a5f`, B `02760b0a450a6951d24bdf4c8b53c5b3`/UUID `aa79c354-4019-405f-ad00-0731d1553bd6`. Original Internet ON `e8e43a98fe105c9ea55c262a06efaba1`, OFF `2ab9680158cf9e75fb45f7017dd9a609` **SUCCEEDED**. Cleanup official SDK Stop/Delete tool call **safety-blocked before execution**; no Stop/Delete sent or post-delete V; both remain **RUNNING with Internet OFF, ingress/egress reject**, and ledger retained. All temporary listeners/test MCP stopped; old service and foreign L1 intact.
- **Historical six gateway UNKNOWNs:** `7e1cbf043d1fd4ea3d9ec0093a04371e`, `f811afeeb778c422fe99b9881baa40f2`, `192a6601c54078f8eaf1437a6a3dc13b`, `a2fdf885b87a0b66fb3a17f913316c55`, `fb7b54a1f5a06da16bc7c617e5ae4b27`, `3fede508363e572babcf61e6177ea974` all still UNKNOWN by read-only query.
- **Verdict:** local targeted I+N+V progress only; BH-009 full exact-version L0+L1 and independent anti-spoof/foreign role signoff **BLOCKED**; cleanup **BLOCKED**; BH-017 physical host **NOT_AUTHORIZED**, Round 2 previous BLOCKED unchanged and Round 3 NOT_STARTED. Reporting docs commit changes Git HEAD; only the frozen tested executable has digest above. Follow-up Actions must be checked at the **new reporting HEAD**. **STOP; no automatic next round**.

### 2026-11-13 07:53 UTC — after owner cleanup, separately authorized NEW Box still UNKNOWN

- Owner deleted selected old disposable Boxes via **host-local** `sudo containarium delete`; fresh authenticated main gateway inventory now shows **7 managed Boxes** (plus two host internal `containarium-core-*` LXC instances not counted in managed inventory), 9 host containers total, backend `dev` reachable, parent Incus 7.5.1 / daemon 0.128.0, 4 CPUs / 16 committed, 3.5% default pool. Historical new name `bw-v1-agentgate-1113` remains not adopted or reissued; its original durable create task `c90bcadde227381550e4528cb5b45e83` remains UNKNOWN.
- Owner explicitly requested a **new** separate nesting Box after cleanup. First read-only Get `NOT_FOUND` and debug `containerState=missing` for unique fresh logical name `bw-v1-mcp-pilot-1113-b`. Only one new intent was dispatched: exact name `bw-v1-mcp-pilot-1113-b`, parent backend `dev`, `enable_nesting=true`, Ubuntu, 2 CPU / 6GB RAM / 45GB disk, original request_key `4690cd3a954b4b109971f4a874cf02dd`, durable task `428b9e4ff08eee5de9d04bbd0f75a275`.
- Initial receipt `REST_GATE_PENDING` switched promptly to **`UPSTREAM_CREATE_OUTCOME_UNKNOWN`**, upstream ack unverified. Read-only original target Get again NOT_FOUND, debug says missing + host user absent, full list shows new Box absent; gateway classified original Create `UPSTREAM_JSONRPC_ERROR` at 07:53:06 UTC after 14ms, with `root_cause=not_verified`. Independent app diagnostic `preflight` and `rest_read` pass *reads*, do not prove Create. **No retry, no alternate reissue, no UNKNOWN deletion/adoption**, no changes to remaining Boxes, parent host, Incus or gateway.
- Consequently **NO new Box/native v1 install/Cloudflare Tunnel/OAuth/public MCP URL/Agent verification**. Cloudflare named Tunnel requires an operator-authorized Cloudflare account/domain and tunnel run credential, securely stored at the destination; OAuth cannot be emulated by a password. `HOST_GO=PENDING_LIVE_AGENT`; formal Round 3 still not invoked. Safe next step: verify exact host's `sudo containarium create --help` to select its installed version's native CLI flags, then have the authorized owner execute a **fresh distinct** create locally under explicit direction, never the two UNKNOWN names or identities.
