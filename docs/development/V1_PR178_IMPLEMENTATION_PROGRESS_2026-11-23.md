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
- Full **`make verify`** `a48605f3d69a494dec80fb14b73208dd`: exit **0**; format 831 tracked files, YAML 47 files, Go vet/mod, all-repo race including state, Python MCP/uninstall/stable gate/raw operation witness PASS. This validates source-level regression only; there was **no candidate install, real peer packet test or Host reboot** on the new code.
- Earlier absent CGO/make/git failures were **dev-Box prerequisite issues**, rectified locally, not product regressions; read Guest exit independently from Operation SUCCEEDED.

**Handoff to #177:** Keep unified Doctor/Recover in Draft until #178 has implemented and independently verified the final network/helper/Operation state model; then replace historical RC13 hard-coded expectations with exact final code and versioned schema. Preserve existing #177 guided-uninstall work.
