# PR #178 — implementation evidence and architecture gates (2026-11-23)

> Status: **PARTIAL IMPLEMENTATION / NOT READY TO MERGE**. This is a candidate branch, not installed RC13 behavior. The authoritative acceptance is [Owner decision](V1_PEER_MCP_AUTOSTART_NONDISRUPTIVE_OWNER_DECISION_2026-11-23.md). Do not use this note as a substitute for Native tests, operator approval, or #177 Doctor's final schema.

## Completed code slice: new Box autostart

- `internal/native/instance_options.go`: `InstanceOptions.ForCreate()` chooses **`boot.autostart=true` only when omitted on a *new* Create**. It is not used for Update. An explicit `false` or `true` remains exactly as supplied.
- `internal/control/box_tools.go`: calls `ForCreate()` **after durable admission** but before the native Create effect and original poststate verification. This keeps the omitted-field historical admission intent unchanged for idempotent original replay. It enforces native `boot.autostart=true` via the existing typed `Apply` and `Matches` readback, not by inference from a running Box.
- Two test cases in `internal/native/instance_options_test.go` and `internal/control/create_shape_test.go` check omit/true/false, native persisted readback, missing field and mismatched false refusal.
- No automatic migration of existing Boxes, no Host reboot acceptance, and no Peer preservation acceptance: the *old Peer Root barrier still deliberately changes every consumer's boot flag to false*. This code slice **does not** satisfy the full autostart requirement under Peer.

## Audited current ownership and risk

| Stage | Current source / privilege | Safety and availability implication |
| --- | --- | --- |
| MCP admission | `internal/control/peer_tools.go`, actor/UUID/incarnation and Project/ACL ledger locked in SQLite | Creates an original but does not execute Peer |
| Root staging | `internal/install/native_peer_root_service.go`, root-owned systemd unit with fixed original and plan SHA, exclusive file creation | Requires local operator action for each distinct original; re-staging/turnover unproven |
| Shared ACL barrier | `internal/native/peer_acl_isolation.go`, all bridge consumers set `boot.autostart=false` and stopped | Interrupts C/D and user services even if only A→B is changed |
| Kernel mutation | `internal/install/native_peer_root_{grant_apply,integrated}.go`, root-only reviewed SQLite original, ETag ACL PUT, exact conntrack invalidation and NFT readback | Prior safety argument depends on full bridge stop; removing it without substitute proof is forbidden |
| Recovery | root private journal, original claim and UNKNOWN handling | No speculative replay/restore of an uncertain original |

Incus upstream [#4008](https://github.com/lxc/incus/issues/4008) documented an older bridged ACL update failure leaving NIC filters absent despite successful-looking configuration. Although closed, candidate **Incus 7.6** MUST prove kernel fail-closed across partial updates and ACK loss rather than infer safety from a ticket status.

## Alternatives to prototype in disposable nested Native environment (no winner yet)

| Candidate | Why it could protect C/D | Main hazard | Required rollback / decision |
| --- | --- | --- | --- |
| N1: per-Box (or narrowly affected per-NIC) immutable baseline deny plus separately reviewed pair-specific native ACL changes | Only A/B policy changes; C/D are neither stopped nor given an ACL update; potential preservation of C/D TCP/UDP sessions | Incus bridged NIC filter replacement may briefly remove policy; cross-endpoint partial fanout can fail open; IPv4/IPv6 revoke and stale conntrack | Prove live kernel deny during every fault point with fail-closed A/B quarantine only; otherwise **reject N1**, no silent old full-Project STOP |
| N2: root-local, fixed-Project, atomic `nftables` policy/set transaction with immutable per-Box source binding and deny guard | Independently scoped kernel policy could update A→B with no Incus shared ACL fanout to C/D | Incus owns bridge chains; ordering, NAT/conntrack, daemon restart/reconciliation, IPv6, Host privilege and state ownership can conflict | Prove durable owner/reconciliation and atomic readback while C/D maintain long-lived connections; no broad arbitrary nft MCP/root tool; otherwise **reject N2** |
| Historical RC13 shared ACL full-bridge barrier | Proven baseline deny while all live ports stopped | C/D interruption, listener loss, autostart overwritten | **Not acceptable final outcome**; retain only as untouched baseline until an independently proven replacement exists |

No fallback may report success when the requested availability and fail-closed safety cannot both be proven; use a typed `UNSUPPORTED/BLOCKED` result before any effect.

## Least-privilege MCP→Host mediation alternatives (not implemented)

- H1: one-time trusted Host provisioning of a **socket-activated, tightly bounded** root-local Peer executor. The unprivileged daemon submits only an original ID; the helper derives Project/ACL/actor intent from independently checked SQLite and kernel/native evidence, recalculates the exact digest, acquires per-Project locks, checks authorization/revocation at dispatch, journals once, and exits. Peer callers must never select shell/path/ACL/raw policy. Verify service binary/unit owner and tamper protection. Refuse incomplete or stale evidence.
- H2: one-time trusted provisioning of a **capability-restricted, fixed-entrypoint one-shot** Host helper with no public root socket, scoped to the same pinned Project and same original-verification pipeline. Capability bounding and Incus root API access are distinct: prove no arbitrary Host mutation, different Project access or confused-deputy takeover. Do not assume capability bounding automatically confines an Incus Unix socket.
- Both must implement safe initial bootstrapping and unit turnover without deleting incomplete journals, and test: wrong UUID/incarnation/digest/actor, stopped or replaced native object, stale ETag, concurrent originals, expired/revoked credentials, Host helper restart, journal missing, UNKNOWN no replay, and wrong Project. An approval policy at initial installation is **not** per-Operation owner approval. Exceptional unproven recovery remains blocked for explicit trusted local review.

## Acceptance matrix — real observations still pending

1. Baseline disposable **A/B/C/D**: independent L1 Incus 7.6, candidate daemon, ACL/nft and exact ledger, C/D HTTP listener, Forward and long-lived TCP/UDP session with uptime/process IDs.
2. Grant and revoke A→B under concurrent C/D traffic: A→B allowed/denied at correct phases; B→A, C→B and A→C denied in IPv4 + IPv6, TCP + UDP; no C/D stop/restart, loss, changed boot flags or session resets.
3. Inject faults at partial native fanout, ETag stale/update ACK loss, conntrack purge, journal PREPARE/COMPLETE, helper crash, power lifecycle and duplicate concurrent grants. Observe **actual kernel deny**, not just API/SQLite.
4. Test static IP/MAC changes, address reuse/reincarnation, owner/project drift, disconnected/forged MCP actor, root helper replacement. No cross-Project effects.
5. New Create implicit true, explicit false, Update true/false, Peer preserve all three prestate classes (unset/true/false), daemon/Incus restart and owner-authorized Host reboot. **Do not silently migrate existing Boxes.**
6. Exact final PR HEAD: Go + race, Python, installed MCP and real Native integration, proof from independent reviewer. CI on an older HEAD does not sign off a newer HEAD.

## Evidence from development Box (separate from Native acceptance)

- Disposable L0-owned dev Box: `pr178-lab-1123`, UUID `ac585d2c-37ae-4879-a2e6-3cecc338d4d7`; Create original `dd6f79f137ee230473b6fbd6be5c9e7f`; Ready original `ea1e80af9afa8d118ec534be4f4f0881`.
- Go 1.26.9, `CGO_ENABLED=1`. Targeted new tests `8b14ac1b5e4676d6bb44f4c57fc3039c`: native/control both PASS, observed Guest exit **0**.
- Full affected Go packages `881704fa30a9f639b5909553b78d9d30`: both PASS, Guest exit **0**.
- Targeted `-race` `2517ff8468eeb6c98ccddfcbec2f9e7f`: both PASS, Guest exit **0**.
- Full **`make verify`** `a48605f3d69a494dec80fb14b73208dd`: exit **0**; format 831 tracked files, YAML 47 files, Go vet/mod, all-repo race including state, Python MCP/uninstall/stable gate/raw operation witness PASS. This validates source-level regression only; the new BoxWeave candidate was **not installed**, and no Host reboot was tested. A **separate manual raw Incus 7.6 per-NIC packet experiment** is documented below; it does not exercise the candidate `box_peer_set` implementation.
- Earlier absent CGO/make/git failures were **dev-Box prerequisite issues**, rectified locally, not product regressions; read Guest exit independently from Operation SUCCEEDED.

**Handoff to #177:** Keep unified Doctor/Recover in Draft until #178 has implemented and independently verified the final network/helper/Operation state model; then replace historical RC13 hard-coded expectations with exact final code and versioned schema. Preserve existing #177 guided-uninstall work.

## Independent Native N1 feasibility experiment — real Incus 7.6, isolated only

**This is experimental feasibility evidence, NOT a production BoxWeave native acceptance.** All commands ran inside the separate L0-owned Box `pr178-lab-1123`; the physical Host, running L0 service and three earlier application Boxes were not changed. Exact installed Incus package: signed Zabbly `1:7.6-ubuntu24.04-202611072126`, signing fingerprint `4EFC590696CB15B87C73A3AD82CC8797C838DCFD`. L1 daemon initialized its own `incusbr0`. Parent AppArmor observed `unconfined`: this does not prove enforcing recursive isolation.

### Live topology and trace

| Inner name | IPv4 in test | ACL attached only to this instance | `boot.autostart` |
| --- | --- | --- | --- |
| `pr178-a` | `10.152.98.160` | `pr178-a-acl` | `true` |
| `pr178-b` | `10.152.98.190` | `pr178-b-acl` | `true` |
| `pr178-c` | `10.152.98.67` | `pr178-c-acl` | `true` |
| `pr178-d` | `10.152.98.185` | `pr178-d-acl` | `true` |

The experiment overrides the four NICs individually; every ACL `used_by` contains only its owning instance. D has a Python HTTP listener on port 8080 and persistent TCP echo listener on 9090. D's ingress rules admit C on these two ports. A separately running **C-side service** maintained repeated HTTP requests plus **one persistent TCP socket** to D; all observed counters and reconnects came from within C, not guessed from API state.

- Initial C→D HTTP returned `200` and the long-lived TCP echo monitor reported `http_ok=26, tcp_ok=26, http_fail=0, tcp_fail=0, reconnects=1` (initial connection only). This is inner Native original `c347a561f0fe6e35e832bda8aa18a8b2`, Guest exit 0.
- Before granting A→B: `curl` from A to B and from C to B both timed out, `http_ok=168, tcp_ok=168, failures=0`, original `3c80dfe406745ba3416f0a146f76e6cc`, Guest exit 0.
- Added **one rule only to B's exclusive ACL** (`ingress action=allow source=10.152.98.160 protocol=tcp destination_port=8080`); A→B HTTP became `200`, C→B remained denied, C/D counters reached `281/281` with zero failures, one initial TCP connection, all 4 instances Running and all 4 `boot.autostart=true`. Original `f1e60c8079c8df8fdaa0e6255f5cc596`, Guest exit 0.
- Negative rule test: `allow-stateless` on B's bridged ingress was **refused by Incus before mutation** (`Action "allow-stateless" is only supported on OVN networks`); B's correct original ACL remained, A→B still HTTP 200, C→B still denied; C/D counters `483/483`, zero failures, same persistent TCP socket. Original `606c7518e347b22d44179573d0d9fe1c`, Guest exit 0. **This is only input rejection; it does NOT inject a mid-PUT/ETag ACK loss or prove atomic fail-closed behavior under all kernel errors.**
- Removed the exact A→B ingress rule from B's ACL, A→B and C→B again timed out, while C/D counters reached **`876/876` with `http_fail=0, tcp_fail=0, reconnects=1`**, all 4 still Running. Original `7889b7c8f5dd93522e0359d92150eaca`, Guest exit 0. A first attempted `rule remove ingress 0` gave CLI syntax error (original `ed42ac6a91eda9214919f78be1ba71c3`, Guest exit 1) **before** any rule removal; corrected exact-key removal was verified by empty ACL ingress.

### Critical constraints before choosing N1

1. The lab intentionally set **default egress=allow** and used DHCP addresses, so **changing B alone is insufficient for BoxWeave production**. Existing production compilers explicitly deny reserved/private egress; final A→B grants would need **two narrowly targeted, independently attested** effects (A egress and B ingress), with whole-policy private egress deny, exact static IPv4/IPv6, source-MAC anti-spoof and direction/reverse/conntrack controls preserved. The prototype does not demonstrate these.
2. Live happy-path continuity is not fail-open proof. The failed `allow-stateless` attempt was rejected at policy validation, so **ACK-loss, partial nft render, dual-stack, concurrent edits, Host helper interruption, and journal UNKNOWN** remain deliberately marked **UNTESTED**.
3. Proof of C/D unaffected is scoped to the observed IPv4 HTTP and persistent TCP echo + stable native Running/autostart; no IPv6, UDP, Forward, power-cycle, process-PID or external Host watcher was measured.
4. The candidate branch still uses **one shared ACL in the actual installer and old root barrier**, and the new code has NOT been installed in this nested daemon. No claim that `box_peer_set` is automatic or nondisruptive in RC13 or #178 candidate.
5. Follow-up engineering decision: N1 is **promising for blast-radius reduction**, but needs a new bootstrap / migration design and per-original rollback proof. Do **not** short-circuit existing shared ACL safety to copy the lab's commands into the product.

### CI for the first source commit

At commit `7151831092a776a0dfd99042cd5fe8482836b56c`: `v1 S1 source checks` run `45241210248` **success**, `v1 required quality gate` run `45241210103` **success**, `v1 Linux amd64 and arm64 candidate artifacts` run `45241210101` **success**. These are *that exact commit's* CI results, not evidence that any later docs/source HEAD also passed or that Native full contract passed.


## Scoped compiler and native conntrack research (second implementation slice)

New pure compiler/test source and complete disposable Incus 7.6 dual-stack + long-lived-flow evidence: [PR #178 scoped ACL experiment](V1_PR178_SCOPED_POLICY_EXPERIMENT_2026-11-23.md). **Security finding:** per-Box DROP alone denied new A→B but left preexisting A→B TCP alive; after exact A/B conntrack purge the stream stopped while C/D's TCP/HTTP remained healthy. No production per-Box writer or Root/MCP path is enabled; full no-interruption acceptance is still BLOCKED. Clean isolated Box full `make verify` passed on second code slice; separately run exact-head CI when submitted.


## Separate pre-Incus bridge kernel guard candidate

Source files `internal/install/network_peer_scoped_guard.go` and tests compile a strict bridge-family nftables policy. A/B live grant/revoke and dual-stack packets were confirmed separately from Incus's own ACLs while the C/D long-lived HTTP/TCP monitor had zero failures. Hardened full-subnet and IPv6 link-local unicast rule tests passed. Exact evidence, original operations and remaining safety gates: [PR178 guard lab report](V1_PR178_SCOPED_KERNEL_GUARD_PROTOTYPE_2026-11-23.md). This is **only a pure compiler**: no production nft writer, Root authority, MCP hook, guard boot persistence, ownership verifier or migration. Existing shared-ACL barrier untouched; full Peer acceptance still BLOCKED. Isolated Box full `make verify` operation `c6e78141ee0be4d2a5b03588deb11320` Guest exit 0; exact-code-commit CI should be checked separately.


## Strict full-Project per-Box ACL inventory preflight

New pure read-only `NativePeerGuardValidateScopedInventory` and tests in `internal/install/network_peer_scoped_inventory*.go`: fails closed unless complete root-observed Project instance names, exact each native UUID/MAC/static IPv4/IPv6/anti-spoof state, unique per-owner ACL+ETag+UsedBy and all expected bridge consumers are consistent. The returned members can be compiled into the separate nft guard; no native write or Root/MCP integration is enabled. Unit negative cases and whole isolated Box `make verify` original `60c257449b6d1209fa808d2d940e8c72` **Guest exit 0**. Complete evidence/limits: [root inventory preflight](V1_PR178_ROOT_GUARD_READONLY_INVENTORY_2026-11-23.md). #178 remains Draft and unsafe to merge.


## Exact sysfs veth and nftables JSON kernel verification

Root-only read-only witnesses added: `network_peer_scoped_kernel_ports.go` (strict Incus host veth to live bridge `brif` mapping, foreign raw ports refused) and `network_peer_guard_kernel_readback.go` (fixed absolute `nft -j` read, complete semantic kernel JSON v1 verification; only counters/handles ignored). The guard plan SHA now v2 binds actual expected kernel JSON. Native disposable A/B/C/D tests: exact 4 running ports accepted, inserted foreign raw veth made 5 and was refused; extra nft `accept` rule refused; C→D monitor ended at 2932 HTTP/TCP successes, zero failures, all temporary rules/devices cleaned. Clean `make verify` Operation `846272014e7588a47a7caad8f37ca3ca` **guest exit 0**. [Full witness report](V1_PR178_ROOT_KERNEL_PORTS_NFT_READBACK_2026-11-23.md). No Root network writer or MCP auto dispatcher yet; #178 remains Draft.


## One-direction root nft batch and ACK ambiguity guard

Added internal package-private `network_peer_guard_transition.go` and tests: independently validates one exact four-rule dual-stack Peer direction delta, refusing any C/D or terminal bridge DROP mutation; fixed root-local nft batch runs at most once after a verified kernel preimage; ACK loss or postimage mismatch returns attempted/UNKNOWN without replay or speculative success. Disposable Incus 7.6 performed real root nft A→B grant/revoke with IPv4+IPv6 HTTP 200 and simulated committed-but-ACK-lost response; C→D ended with 248 HTTP+TCP successful, zero failures, no Box stop. No durable journal or MCP Root integration enabled. Full `make verify` original `0fc034ec1785c65dcebddfec0f5fb75d` Guest exit 0. Evidence and remaining blockers: [guarded root nft batch checkpoint](V1_PR178_ONE_DIRECTION_KERNEL_BATCH_2026-11-23.md). #178 remains Draft.


## Durable isolated Root guard-step journal

Added `network_peer_guard_journal.go` + composition and tests. One exact immutable original is persisted as `DISPATCH_UNKNOWN` (root-only 0700 directory, exclusive O_NOFOLLOW 0600 journal, file/dir fsync) **before** a single private nft batch. Only a strict independently verified kernel postimage writes `GUARD_STEP_KERNEL_VERIFIED` (not a completed Peer Operation); any ACK loss, missing poststate, corrupt journal or completion fault stays UNKNOWN, and the original cannot replay. Actual disposable Incus 7.6 IPv4/IPv6 A→B HTTP 200 and injected committed-but-ACK-lost on-disk journal verified; C→D 199 HTTP+TCP success zero failure, test-only scripts removed. Full isolated `make verify` `321f4f5ca7a5d8ef0e62b76bef03d819` Guest exit 0. [Complete evidence and blockers](V1_PR178_DURABLE_GUARD_ORIGINAL_JOURNAL_2026-11-23.md). No installed Root/MCP dispatcher or per-Box production migration exists; #178 remains Draft.


## Multi-ACL per-Box SQLite ledger and 1024 row acceptance

Shared old Peer SQLite model requires all Boxes' `network_bindings.acl` to be identical; that is incompatible with per-Box native ACLs. New independent pure name package `internal/peerscope` is shared by `state` and `install`, and new `ReadCompleteScopedNetworkAuditLedger` verifies **one exact derived ACL per Box**, full native identity and dual-stack fenced addresses, foreign and history refusal. Existing shared-ACL reader/Peer handlers are deliberately unchanged. Tests cover valid three-Box and negative partial migration, SQLite UNIQUE MAC/IP, and 1,024 row positive / 1,025th refused. Full isolated `make verify` `8db75190dc9b2f65d26e1eac9864b9b0` Guest exit 0. [Read-only multi-ACL report and migration blockers](V1_PR178_PER_BOX_SQLITE_LEDGER_DESIGN_2026-11-23.md). No production migration or MCP Root dispatch enabled; keep #178 Draft.


## Root ACL default Project provenance correction

Actual disposable Incus 7.6 cross-Project testing proved the root-created ACL and network object have native `Project=default`, while tenant references in `UsedBy` are canonically `?project=<tenant>`; the old pure inventory fixture incorrectly modeled ACL.Project as the tenant. Updated strict root-native ACL checker and expanded URL canonicalization tests without loosening foreign Project/bridge ownership checks. Targeted tests and full isolated `make verify` `165e02d1a31c8e471d6d2fd9c16e2815` Guest exit 0; temporary Project/ACL deleted. [Full native evidence](V1_PR178_ROOT_DEFAULT_ACL_NATIVE_PROVENANCE_2026-11-23.md). No production network mutation or MCP Root integration enabled.


## V2 scoped Peer original and atomic staging

Added **v2-only** `ScopedPeerOriginalIntent`, a read-only claimed-original `ReadRootScopedPeerOriginal` and UNKNOWN recovery-only checker, five exact bridge guard/per-Box ACL/per-Box UUID claims, full same-transaction SQLite/actor/token/dual-stack proof, and an **internal-only atomic StageScopedPeerOriginal**. Unrelated C/D grant policies are preserved; regrant records a durable explicit DROP-before marker and produces separate `RevokedBefore`/`RevokedAfter` views. Old v1 Peer/root API and MCP unchanged and refuse v2. Extensive grant/revoke/UNKNOWN/admission/foreign/partial/lease negative tests and full isolated `make verify` original `8e98aa38b6749aa46672aaa54dca176c` **Guest exit 0**. [Complete code/evidence/remaining issues](V1_PR178_SCOPED_V2_ORIGINAL_AND_ATOMIC_STAGE_2026-11-23.md). STILL no installed Root writer, safe shared→per-Box migration or automatic MCP dispatch; keep Draft.


## Legacy v1 verified Peer lineage compatibility for scoped v2

Root finalizer research found old CONFIRMED rows have blank `operation_id` but retain `last_verified_operation_id`, while REVOKED rows keep `operation_id=last_verified_operation_id`. New `scopedPeerVerifyHistoricalOperationTx` requires exact old successful original with canonical intent, actor, Project, pair, typed resource claims and matching kernel-verified terminal event, refusing any forged/missing/UNKNOWN provenance; distinguishes legacy from future v2 completion events. V2 Root reader and atomic stager now validate ALL siblings' verified provenance plus previous grant/revoke on pending A/B in their same SQLite transactions, and v2 regrant preserves real old revoked original. 30+ negative and real SQLite-finalizer fixture tests pass. Isolated `make verify` `db0b1bab117296dc28de5764c590854f` Guest exit 0. [Detailed evidence](V1_PR178_VERIFIED_LEGACY_PEER_LINEAGE_2026-11-23.md). Not an installed Root/MCP system, no actual network mutation. Keep Draft; #177 untouched.


## Root scoped v2 SQLite-to-native read-only policy compilation

New `NativePeerScopedRootTransitionFromEvidence` composes the already read-only v2 SQLite original and exact complete native Incus tenant instances, root-default ACL objects/UsedBy/ETags, deterministic per-Box lease ownership, full A/B/C/D active/denied policy and native bridge guard, then refuses any delta except the exact original A→B and its revocation/regrant DROP. Independently compiles every unrelated C/D ACL and requires unchanged pre/post; records each original `boot.autostart` string, including explicit false. No native/root write, no trusted kernel preimage collector, no MCP automatic mediation. Four-owner positive/19 negative `-race` cases and isolated full `make verify` `afba7e5cd074da76ed276651b0f784fa` Guest exit 0. [Read-only root native compilation report](V1_PR178_ROOT_SCOPED_SQLITE_NATIVE_PLAN_2026-11-23.md). Keep Draft and #177 untouched.
