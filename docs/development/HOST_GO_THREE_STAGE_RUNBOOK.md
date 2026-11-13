# HOST_GO: three user-triggered execution rounds (S2.5 BRIDGEHEAD)

**Owner-approved gate clarification, 2026-11-13 UTC:** `HOST_GO` now means
**eligible for a separately authorized installation on a dedicated physical
development host**, not "already installed and accepted on that host".
`LAB_PASS` remains signed with its originally documented bounded evidence;
`HOST_GO` is **not yet signed**. Round 3 will first use a newly isolated,
nesting-enabled **main-managed Box** running a pinned v1 and native Incus,
reachable via an authenticated remote MCP transport. The owner will configure
and explicitly authorize the new ChatGPT Agent connection before any Agent
write tests. Cloudflare Tunnel reachability alone is not OAuth, authorization,
or proof of working MCP. The separate host-specific post-install check is
`HOST_POSTFLIGHT_PASS` and is never inherited from a nested Box.

## Authoritative meaning of the gates (supersedes old host-first wording)

| Gate | Real-world meaning | What it permits |
|---|---|---|
| `ROUND1_PASS` / `ROUND2_PASS` / `LAB_PASS` | Historical bounded isolated acceptance with explicitly preserved limitations and UNKNOWN receipts | Begin isolated live-v1 Agent rehearsal only |
| `HOST_GO=PASS` | Same pinned v1 source/binary in an isolated nesting-enabled Box: actual authorized development Agent → authenticated remote MCP → original durable Operations → genuine nested Incus L1/L2, real public IPv4 HTTPS, native security/negative checks, independent verifier and final Actions all pass | **Request** approval to install this exact candidate on a named dedicated physical development host |
| `HOST_GO=BLOCKED` or `PENDING` | Any mandatory isolated live-Agent/network/security/original-operation/Verifier/CI proof missing | Do not treat the candidate as ready for physical installation |
| `HOST_POSTFLIGHT_PASS` | **After a separate exact-host approval and bounded pilot installation**: actual physical host inventory, privileges, pool/ACL/network, Agent+MCP+Incus poststate, fallback/rollback, retained data and independent verifier pass | Begin normal development use **on that specific host only** |

The physical host's exact identity, OS, native Incus configuration,
host-owned permissions, backup/rollback and any modifications require **fresh
specific user authorization**, even after `HOST_GO=PASS`. `HOST_GO` does not
permit automatic physical install, global daemon ACL change, PR merge, S5
release or production use. Physical-host evidence cannot be fabricated
from the nested Box; new real-host failures yield `HOST_POSTFLIGHT_NO_GO`
without rewriting signed, still-true LAB evidence. Public IPv6 Internet
remains mandatory at S5; simulated routed IPv6 I+N+V and genuine public
IPv4 remain mandatory at this gate.

**Preparatory permission (owner's separate 2026-11-13 request):** it is
permitted to provision *one new, individually named isolated nested Box*
under the existing `main` manager for v1/tunnel readiness **without
executing or signing Round 3**. Inventory first; do not retry prior UNKNOWN
creation receipts. No public unauthenticated MCP, fake OAuth password or
exposure of Bearer credentials. If stable tunnel / OAuth-compatible provider
or ChatGPT write-capable connection is absent, record `AGENT_PATH_BLOCKED`,
stop exposure, give the owner the single next action, and do not award
`HOST_GO`.

## Copy-ready next-Agent entry prompt

Use this same prompt for the next Agent (the user's next message chooses the stage):

> Work on `vanishs/boxweave`, Draft PR #133, without touching unrelated native Incus or historic UNKNOWN. Read `AGENTS.md`, `docs/development/HOST_GO_THREE_STAGE_RUNBOOK.md`, `docs/verification/HOST_GO_STAGE_STATUS.md` and the scoped BRIDGEHEAD acceptance cases. The user will say `完成HOST_GO 第一阶段`, `完成HOST_GO 第二阶段`, or `完成HOST_GO 第三阶段`. Execute **only the named stage** end-to-end now: recover original receipts/commit and precise source/binary, use multiple authorized isolated Boxes for real installed MCP/native negative tests and an independent Verifier, fix and retest as needed, persist checkpoints/UNKNOWN/decisions in docs and PR #133, run **same-head Actions after Box-first acceptance**, then report a truthful PASS/PARTIAL/BLOCKED in Chinese and **stop**. Do not automatically run the next stage. Do not install on a physical host without fresh approval of its exact identity and effects. Never waive mandatory safety or mistake the existing green expanded LAB for formal LAB_PASS/HOST_GO. If blocked, document it and finish unrelated safe tests for this stage; never repeat an UNKNOWN native write.

## Exact conversation commands and dispatch

The following three owner messages are the only automatic execution triggers. Match the ordinal; do not start another round from the current planning request or a prior round's successful Actions.

| Message from owner | Execute *only* this round | Required end of turn |
|---|---|---|
| `完成HOST_GO 第一阶段` (also accept `完成 HOST_GO 第一阶段`) | Round 1 — ownership/network security | Record acceptance and report results; STOP |
| `完成HOST_GO 第二阶段` (also `完成 HOST_GO 第二阶段`) | Round 2 — fault recovery/real Agent | Record acceptance and report results; STOP |
| `完成HOST_GO 第三阶段` (also `完成 HOST_GO 第三阶段`) | Round 3 — isolated live-v1 Agent gateway / final Actions | Record final HOST_GO or NO_GO; STOP |

**On each invocation:** perform that round end-to-end, not just write another plan. Resume any prior incomplete round from [live checkpoint](../verification/HOST_GO_STAGE_STATUS.md), GitHub PR/commits and original native receipts, rather than resetting state or repeating uncertain writes. A user calling the next ordinal is not permission to mark an earlier blocked security prerequisite PASS. If earlier rounds are incomplete, complete independently safe tasks within the requested round, mark dependent work BLOCKED and report the exact prerequisite; do not advance into another round autonomously. Do not start Round 2/3 at the end of Round 1/2.

### Small reading set and reality check (each round)

1. Read `AGENTS.md`, this document, [stage status](../verification/HOST_GO_STAGE_STATUS.md), [BRIDGEHEAD matrix](../verification/BRIDGEHEAD_ACCEPTANCE.md) **only the cases for this round**, [latest exact-source evidence](../verification/S25_EXACT_HEAD_FINAL_LAB_2026-11-12.md), the applicable [phase protocol](PHASE_EXECUTION_PROTOCOL.md) and recent [execution log](EXECUTION_LOG.md). Read scoped native/network/operation docs and upstream SDK references only as the subtest begins. Correct stale source HEAD or contradictions based on real code and logs.
2. Identify actual PR #133 head, source/binary/build SHA, GitHub Actions, current installed v1 version, L0/L1/L2 IDs and separate user/Project/ACL, native resources, SQLite original Operations, worktree edits and no-shared-resource boundary. Exact native Incus daemon/APT version is read from the host, **not** assumed from Go SDK. Do not confuse the live old `boxweave` plugin with the v1 candidate.
3. Record date, environment, original key/Tracking ID/BoxWeave Operation ID/native Operation ID/UUID and actor, expected result, test command/result and separate **I (installed MCP/CLI), N (native Incus/kernel/packet), V (independent verifier)** evidence. Redact credentials; append failed and successful attempts. Do not overwrite historical red or UNKNOWN evidence.
4. Prefer multiple purpose-built, individually named **disposable, isolated Boxes** for Finder, Fixer, independent Verifier and negative tests. In each, first inventory foreign resources; do exact-owned UUID cleanup after independent postcondition. A successfully compiled script, direct `incus` CLI test, SDK-only mock or successful GitHub job is **not interchangeable** with same-candidate v1 installed MCP+native+Verifier.
5. Within the round: inspect → test real Box early → reproduce negatives → fix product or tester narrowly → local `make verify`, Go race/format → retest on independent Boxes → record exact IDs/digest → commit to PR #133 (never force push or merge). **Only after those Box tests** run or review Actions for the exact branch HEAD and resolve failures; do not use Actions as the first and only testing environment. If Actions cannot faithfully reproduce an environment (actual physical host/Agent), separately record the independent host proof.
6. If a noncritical test repeatedly fails or an authority/credential is unavailable: record `BLOCKED` with root cause, original operation and safest independent next test, then **continue other safe work within that round**. A failure of a mandatory P0 prevents its gate, no fake PASS. Never repeat possibly dispatched/UNKNOWN operations, remove old receipts, adopt foreign Incus resources or silently bypass a failed prerequisite.
7. **End of each invocation:** persist status in [HOST_GO_STAGE_STATUS](../verification/HOST_GO_STAGE_STATUS.md), update the BH/TC matrix and append-only [EXECUTION_LOG](EXECUTION_LOG.md) or phase-specific incident log; commit PR #133 and add a GitHub comment linking real Actions. Reply in Chinese with `阶段判定 (PASS/PARTIAL/BLOCKED)`, installed/native/V counts and named cases, bugs fixed, native Box/cleanup state, exact code+binary digests, Actions outcomes, unresolved IDs and **one clear next-round precondition**. STOP. Never promise asynchronous work.

### Non-negotiable safe boundaries

- **Never** use the four historical UNKNOWN tasks (`7e1cbf043d1fd4ea3d9ec0093a04371e`, `f811afeeb778c422fe99b9881baa40f2`, `192a6601c54078f8eaf1437a6a3dc13b`, `a2fdf885b87a0b66fb3a17f913316c55`) as retry/delete/adoption targets. Read-only, exact-original status inspection if safe does not clear UNKNOWN.
- A general authorization to create/test managed **Boxes** does **not** authorize an unnamed, dedicated **physical** host, OS global Incus change, host daemon restart/upgrade, existing workload removal or public network exposure. Round 3 requires *fresh explicit authorization of the exact host and allowed effects*. Legacy `main` and v1 never co-install on that host.
- Keep v1's existing one-service/loopback MCP and Project isolation, exact UUID/incarnation, shared-default-Profile ingress reject and implicit/explicit default egress reject; allow Internet **only on reviewed individual owned NICs**; maintain IPv4+IPv6 source/MAC filtering, sibling/foreign denies and least privilege. No anonymous/OAuth shortcut, admin socket sharing, host shell Tool or second product architecture.
- User waived **new S2.5 disk quota pressure tests**, not truthful disclosure. A usable `dir` driver on non-ext4/XFS with absent quota proof reports `DIR_QUOTA_UNVERIFIED` and host-root disk-exhaustion risk. Do not claim hard quota. Test host space/inodes/PSI and monitor thresholds without destructive exhaustion.
- Complete S2.5 development-host readiness **only**; S3/S4/S5 features, full 20/20 release gate, public launch and stable user-data compatibility are not silently included.
- Retain current authoritative BH-001–BH-017 and TC-011–014 semantics. Reuse **already accepted S1/S2 evidence** only where source/contract/fault surface remains unchanged and a per-case exact-version crosswalk shows why it is valid. A P2 redundant permutation may be recorded as `DEFERRED/NON-BLOCKING` with explicit owner agreement **if the gate contract is formally amended**, never marked PASS. Do not unilaterally reduce a mandatory safety/physical/Agent gate. Distinguish `TARGETED_LAB_PASS`, `LAB_PASS` and `HOST_GO` in every report.

## Round 1 — isolated native network/ownership gate

**Trigger:** `完成HOST_GO 第一阶段`. **Goal:** same-candidate **I+N+V** security boundary on several disposable real Boxes, and a verified installation/prerequisite baseline, avoiding missing direct-WAN and negative tests. Keep actual physical host untouched.

- **Map:** BH-001 lab inventory, BH-002 disposable baseline, BH-003, BH-005 auth/security subset, BH-009 **mandatory**, BH-010 native nested networking subset, BH-011, BH-015 security verifier subset, TC-011 identity/foreign subset. Use current BH and exact-head reports for existing evidence; do not retest all already accepted S1/S2 blindly.
- Build/install the **exact HEAD candidate** separately in independent disposable native L0 and nested L1 with their own Project/ACL/SQLite/tokens. Verify APT/digest and 13 public S1/S2 Tools + local CLI; 8/8 reviewed prerequisites, negative SHA/authority, owned ACL `used_by`, effective NIC and kernel rules, root Profile implicit/default egress reject and per-Box `internet=true/false`.
- Create **at least two owned sibling Guest Boxes per tested layer**, plus foreign Project/actor negative fixtures only in disposable owner-controlled Projects. Test actual positive **unproxied public IPv4 HTTPS** and **isolated, separately routed simulated IPv6-WAN HTTPS** at L0 and nested L1 with genuine packets, TLS and independent Verifier; not mocks, mere route/ACL readback or a same-bridge peer. Prove individual-NIC Internet off/on/off, bidirectional IPv4/IPv6 sibling/foreign denials, forged IPv4/IPv6/MAC and UUID/reused-name denial, and absence of parent/foreign socket access. Capture original operations, actual packets and native rules at identical binary SHA. If no public IPv6 upstream exists, label `PUBLIC_IPV6_DEFERRED_TO_S5`; this does not waive simulated IPv6 I+N+V or safety. See [owner-approved timing](../decisions/IPV6_PUBLIC_WAN_PHASED_ACCEPTANCE_2026-11-12.md).
- Exercise forbidden Viewer writes, wrong/expired/revoked tokens, bad Host/Origin/path, nonroot host preparation, forged scoped work/agent correlation; separate authenticated client and Verifier. Run real positive Guest Exec and negative output, not just `incus` CLI.
- Finder/Fixer/Verifier must be distinct in evidence: independent native packet witness on exact same candidate digest; no unilateral weakening of ingress/ACL to make an egress test green. Check Box cleanup by exact UUID, native and SQLite receipt, ensure foreign resources intact.
- **Gate:** `ROUND1_PASS` requires all Round 1 P0 ownership/safety evidence, real public IPv4, simulated routed IPv6 HTTPS and IPv4/IPv6 packet negatives with the same installed MCP I, native N and independent V. Missing any means `ROUND1_BLOCKED`; don't call BH-009 PASS. Real public IPv6 is a **deferred S5 release gate**, not an early S2.5 requirement by itself. Report and **STOP**; Round 2 needs a separate turn.

## Round 2 — failure recovery, Agent transport, independent negative replays

**Trigger:** `完成HOST_GO 第二阶段`. **Goal:** prove **no unsafe replay / no privilege transfer / independent fallback** under failure and an actual development Agent client when available, in disposable Boxes.

- **Map:** BH-004, BH-006 missing Get/List/Ready, BH-007, BH-008, BH-012, BH-013 non-quota stress, BH-014 evidence reuse, BH-015 independent fault verifier, BH-016 lab cleanup; TC-011–TC-014. Round 1 P0 remains mandatory for later HOST_GO.
- Use two genuinely separate authenticated clients and external checkpoint. Inject controlled delayed/lost native ACK, lost MCP reply **after dispatch**, duplicate key/conflicting intent, SIGKILL/controller restart and client disconnect at admission, dispatch, ACK and SQLite finalization; revoked queued action and shutdown while Exec is quiet. **Never replay original UNKNOWN writes**. Inspect original SQLite/BoxWeave/native IDs and output; distinguish `RUNNING`/`VERIFYING`/terminal/UNKNOWN and blocked client versus native offline; assert no second native effect. Independently repeat selected highest-risk cases rather than all redundant permutations.
- In owned disposable host only, test operator-local failed installation/interrupted upgrade, service stop/restart and safe rollback or fail-closed on incompatible pre-release schema; independent recovery WITHOUT the failed MCP service, retaining ledger/UNKNOWN and foreign native objects. No forced host/global Incus remediation. Use bounded slow I/O, timeouts, concurrent operations, measured resource limits, PSI and bounded retry rules. Skip new destructive disk-quota pressure tests per owner decision.
- Test the **actual intended development Agent/client** through its approved v1 authentication/transport (not the substitute official Go SDK lab Agent), real Tool discovery, authorized read and one original-key S2 mutation with exact native readback and independent Verifier. If this Agent cannot connect, investigate supported plugin/network/credential compatibility; do not exfiltrate secrets or circumvent the tool's authorization restrictions. Record `AGENT_PATH_BLOCKED`, concrete missing user action, and continue all independent tests. Don't declare TC-013 PASS from official SDK alone.
- Repeat high-priority two-Agent shared-work/foreign-work/ref spoof, token revoke/rotate/blocked-client negatives, and a complete independent original-operation recovery. Reuse only crosswalked S1/S2 accepted cases; sign off BH/TC rows individually with evidence or recorded non-blocking variance explicitly approved by owner.
- **Gate:** `ROUND2_PASS` needs P0 fault/recovery/rollback evidence and real intended-Agent integration (or an explicitly documented owner-approved change of which Agent will be the real development client). Otherwise `ROUND2_PARTIAL/BLOCKED`, no HOST_GO. Report source/IDs/cleanup and **STOP**, regardless of spare context/time.

## Round 3 — live v1 MCP in one isolated nested Box, then HOST_GO decision

**Formal trigger:** `完成HOST_GO 第三阶段`. A separately approved
preparatory Box/Tunnel provisioning request is not the formal test trigger.

1. **Isolated test admission:** confirm Round 1+2 signed LAB evidence and its
   explicit limitations; read-only inventory of the existing `main` provider,
   prior UNKNOWN receipts and foreign managed Boxes. Create **one new**
   nesting-enabled disposable main-managed Box with its own distinct logical
   name/tenant, bounded resources and no foreign resource changes. Confirm the
   original durable gateway Create status before proceeding; never retry an
   UNKNOWN native/gateway Create. Enable nesting only as explicitly requested
   and verify effective native readiness; don't assume `enable_nesting` worked.
2. **Pinned v1 and inside-only Incus:** use a reviewed precise PR #133
   source SHA and binary SHA-256, install confined v1+Incus inside this
   **new Box only**. Record actual daemon version, native prerequisites,
   Project/Profile/ACL/pool and auth. No shared parent daemon/global Scriptlet
   changes, no old `main` co-install, no silent privilege escalation.
   A potentially incompatible isolation, quota or native authorization
   failure is an explicit `BLOCKED`, not reason to relax controls.
3. **Safe remote Agent path:** keep v1 MCP loopback-only in that Box; publish
   HTTPS **only** through an intentionally authorized Cloudflare Tunnel or
   equivalent origin-authenticated transport. Require valid v1 admin/viewer
   credentials behind the tunnel. A random Tunnel URL or an OAuth "password"
   is **not** an OAuth 2.1 authorization server. If ChatGPT requires OAuth,
   use a separately reviewed compliant provider with resource metadata,
   authorization-code PKCE and correct audience/scopes. Never put secrets in
   GitHub commits, PR comments, URLs or chat. Prove public 401 and
   authorized Tool discovery independently **before** asking the owner
   to connect. Record ChatGPT plan/feature limitations as
   `AGENT_PATH_BLOCKED`, not as a fake verified connection.
4. **Owner connects and triggers:** give the owner only the validated
   HTTPS MCP URL and exact approved connection instructions; **STOP**
   until the owner confirms that the *new* v1 MCP is connected, specifies
   which Agent/client may run the test and explicitly requests Round 3
   execution. Existing `main` tools are not proof that new v1 is connected.
5. **Native Agent+security acceptance:** with the owner's new connection,
   authenticate and inspect real Tool discovery, then execute a small original
   idempotency-key v1 Guest lifecycle and realistic Agent Git/build/test
   inside an owned L2 behind an unprivileged nested L1. Capture distinct
   original BoxWeave/native Operations, actual Incus UUIDs, retained
   output and reconnect recovery. Prove IPv4 public HTTPS positive/off/on,
   routed simulated IPv6 HTTPS, parent/foreign socket denial, IPv4/IPv6/MAC
   spoof and peer/foreign negative cases, Viewer/expired/revoked denied
   writes, and fail-closed default network. An independent verifier
   rechecks effective native state and exact owned cleanup. Reuse previous
   accepted LAB faults rather than injecting destructive parent-host faults.
   Do not replay UNKNOWN originals.
6. **Final exact candidate gate:** after these real Box/Agent tests, run
   all relevant same-head Actions (quality/race, original native S2 and
   nested independent S2.5) and reconcile source/binary identity.
   Record failures and fixes with original IDs. `HOST_GO=PASS` only when
   the **new ChatGPT or owner-approved actual intended Agent**'s live remote
   MCP I + native N + independent V and all required negatives, genuine
   IPv4, simulated IPv6 and Actions succeed. Otherwise mark
   `HOST_GO=BLOCKED`; never infer PASS from an SDK simulator or a tunnel URL.
   Update BH/TC with the added Stage-3 Agent witness and append-only log,
   commit to PR #133 and report the next owner step. **STOP.**

**Separate later physical-host pilot (not a prerequisite for HOST_GO):**
after `HOST_GO=PASS`, obtain fresh explicit authorization naming the
dedicated host and allowed effects. Run read-only capacity/security/native
preflight, exact-candidate bounded installation and L0/L1/L2/Agent
postflight, real public IPv4 and independent native verifier, safe
stop/rollback and original ledger reconciliation. Only then sign
`HOST_POSTFLIGHT_PASS` for that one host. If no permission or a P0 fails,
record `HOST_POSTFLIGHT_NOT_AUTHORIZED/NO_GO` and **do not install or
continue writes**. No automatic PR merge or S5 release.

## After-round reporting contract and persistent state

The per-round record in [HOST_GO_STAGE_STATUS](../verification/HOST_GO_STAGE_STATUS.md) has **NOT_STARTED → IN_PROGRESS → PASS/PARTIAL/BLOCKED** (not self-awarded on plan creation). Update: current/ref, attempted fixtures and host scopes, all original Operation/UUID, I/N/V per BH/TC, independent signer, native cleanup, fault/fix, exact latest source/binary digest, same-head GitHub Actions, outstanding `UNKNOWN`/owner decisions and next trigger. Report **only the just-invoked round** in the final user response. A hard blocker should not erase successful partial evidence; it prevents only the unsafe dependent gate. No asynchronous continuation after responding; next round starts on the next separate user message.
