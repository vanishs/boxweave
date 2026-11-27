# Consolidation acceptance and evidence map

This is the initial crosswalk for the single pre-release consolidation PR. Complete granular test/marker mapping in P0, then update alongside each implementation slice. Retained behavior requires equivalent or stronger evidence in the single architecture; obsolete Legacy-only behavior requires explicit retirement rationale, not a fake PASS. See [current acceptance](../verification/ACCEPTANCE.md), [continuity](../verification/CONTINUITY_ACCEPTANCE.md), [original A-/BH-/TC- matrix](https://github.com/vanishs/boxweave/blob/06d27434b11997e9254822c122483c9dacba12a3/docs/verification/ACCEPTANCE.md) and [H5 evidence](PR188_H5_DELIVERY.md).

| ID | Mandatory target and suggested baseline witnesses | Stage |
| --- | --- | --- |
| C01 | Exact source format, vet, Go race, Python nonempty discovery, build, CGO SQLite real startup; native amd64/arm64 build, artifact hashes, required quality-gate, full vs docs-only classifier negative. Existing make verify, quality and release workflows. | all / P5 |
| C02 | Fresh schema, clean startup, restart persistence, old/foreign/malformed/partial schema refusal with no write/adoption, ownership and original replay/UNKNOWN across same current schema. S2 and TC-003, state tests, SQLite crash probes. | P1 |
| C03 | All 23 MCP tools from installed discovery, actual positive and denied lifecycle/update/readiness/Exec/file calls, UUID/Project and layer isolation, exclusive create and unsafe conditional writes refused; token/OAuth revoke/refresh, old official SDK interoperability, bounded output and pagination. S2/S3/S4, TC cases. | P1/P3/P5 |
| C04 | Per-Box ACL and quarantine, IPv4/enabled IPv6 default deny, autostart true/explicit false, directional grant/revoke and real TCP/UDP as supported; reverse/unrelated connections, PID/NIC/Forward and boot choices preserved. N2/N3/N4, Forward suite. | P1 |
| C05 | Partial native effect, stale ETag, ACK loss, crash/restart, exact-key replay, concurrent original, same-name different UUID, unresolved locks, independently verified recovery/no redispatch. Retained address/identity/direction/history limits and reserved revoke/Delete headroom. N4, D4, S2 fault/GC, H4. Synthetic 1024 ledger records are not real 1024 instances. | P1/P3 |
| C06 | Single fresh installation contract, observed artifact identity, scope/guard/helper health, existing compatible Incus and unrelated resources untouched, correct missing prerequisite and stale/declined review refusal. OAuth optional continuation without reinstall/password reset, existing password preservation. H3A, S2.5, N2 and forward policy. | P2 |
| C07 | Normal full retirement including owned dependencies, offline/abnormal and interrupted resume where supported, completed-data purge, stale/mixed/foreign/UNKNOWN refusal, archive integrity, no duplicate effects, clean reinstall, recognized inherited LXC policy preserved. H1/H2/H5, #182/#184/#188. Do not translate old Legacy-specific resume success into proof of new path. | P2 |
| C08 | Sole operator lifecycle and actual CLI leaves (public and internal); each retained CLI leaf positive/negative or documented fixed service path; Doctor root/restricted evidence coverage, original recover review/stale/decline, actionable redacted MCP errors and host_status capacity parity. D0-D5 and H3B/H4A/H4B. | P3 |
| C09 | Independent nested controller install and inner Box lifecycle, real guest exit, own credentials/scope, no parent socket, conditional native capabilities and quota evidence. S2.5/BH and TC. Functional nested PASS is not hostile recursive isolation proof. Separate physical reboot/LAN/public IPv6/storage evidence or owner-approved bounded substitution in #170. | P2/P5 |
| C10 | Retired production execution unreachable/absent: inspect dispatch, flags, SQL, systemd, wrapper, artifacts and current docs; old installation evidence explicitly refuses without effects; no silent fallback/reset. Each retired symbol linked to inventory/caller proof; no obsolete test-only runtime in shipped build. | P1-P5 |
| C11 | Every retired workflow/test/marker mapped to retained behavior plus equivalent new witness or explicit obsolete Legacy-only requirement. Preserve release S2/S2.5 native gates, exact signature/asset checks and required quality; test new triggers and negative calibration, not just green aggregate. | P0-P5 |
| C12 | Fresh checkout and independent code review, BDH activation/resume, learning audit and applicable Eval, no stale active references; actual reuse/outcome only if observed. Exact final CI and coverage crosswalk, no unresolved release-critical regression. #170 receives new source/contract and evidence limits. | P4/P5 |

## Active P2-P5 acceptance routing

As of the 2026-11-27 P1 development closeout, the C01-C12 contract above
is unchanged. [R5 integrated review](https://github.com/vanishs/boxweave/pull/191#issuecomment-7216191395)
on 0e903a2 confirms P1 development complete, formal acceptance deferred.
Use the [active plan](CONSOLIDATION_PLAN.md#active-delivery-plan-p2-p5-2026-11-27)
and [handoff ledger](CONSOLIDATION_PROGRESS.md#active-p2-p5-handoff).
Earlier P1 history below is retained.

| Gate | Baseline disposition | Next acceptance |
| --- | --- | --- |
| C01 | FAIL, two Install tests; quality/S1/dual arch red | P5 exact aggregate; preserve H obligation |
| C02 | P1 scoped schema7/startup/refusal evidence PASS within scope | P2/P5 affected exact installed revalidation |
| C03 | P1 installed 23-tool and PR S2 crosswalk PASS within scope | P3 navigation/contract, P5 final |
| C04 | Peer/Forward and local WAN PASS; hosted WAN UNVERIFIED | P5 independent positive or approved bounded equivalence |
| C05 | Recovery/ACK/WAL/no replay evidence PASS within scope; H partial | P3/P5 retained original and H composition |
| C06 | Fresh/negative constituent proof exists | P2 one complete installed contract |
| C07 | Normal retirement/reinstall PASS; offline Scoped missing | P2 full supported offline/normal, archive/resume/purge |
| C08 | Not accepted for consolidated target | P3 full operator contract |
| C09 | S25 functional PASS; limitations retained | P2/P5 independent nested scope; #170 physical limits |
| C10 | Runtime/DDL and old-state refusal evidenced within P1 scope | P2-P5 remaining caller/artifact inventory |
| C11 | PR S2 PASS 72 markers/23 jobs; release S2 and old N2 open | P2 R3/R4, P4 exact mapping, P5 final |
| C12 | Not started; historical BDH evidence retained | P4/P5 independent exact review and actual workflow use |

Do not use successful PR S2 as release, WAN or all-N2 proof. Actual
job 140419352217 verifies current fresh Legacy refusal and normal
Scoped retirement; full offline legacy behavior must be either explicitly
obsolete or covered by the supported Scoped equivalent, including
preservation of retained originals and foreign resources. No omission
of unique generic behavior. C06/C07 may be accepted in their own scope
without calling inherited P1 H PASS. Every mandatory unresolved
obligation blocks final P5 acceptance; development sequencing is not
an acceptance waiver. #170 remains separate; no unapproved publish.

## Active P1 recovery interpretation

Historical execution interpretation; active routing is above.

Use [R0-R5](CONSOLIDATION_PLAN.md#p1-convergence-recovery-r0-r5-active)
and the [live ledger](CONSOLIDATION_PROGRESS.md#current-p1-recovery-ledger).
This changes execution order, not C01-C12 or final acceptance. Prior
checkpoint statements are historical, not new blanket hold points.
R0 must identify the exact dependency and behavioral oracle for each
remaining mandatory gate. Historical evidence is reusable within its
proven scope; run final applicable gates on the exact new candidate.
A reversible source cutover may combine runtime/schema and minimum
consumer changes; do not retire a required behavior without equivalent
proof or explicit obsolete-contract disposition. Never merge an
unaccepted candidate. WAN is a separate named gate: earlier real Box
positive and Hosted UNVERIFIED remain distinct; no automatic equivalence
or waiver. R5 closes P1 only with mandatory P1 evidence.

Owner-approved P1/P2 boundary: full offline retirement migration belongs
to P2, with UNKNOWN archival, interruption/resume, purge and foreign
preservation obligations intact. Minimum P1 dependencies and the two
blocked core Install regressions remain P1. No acceptance waiver.

### Install legacy-oracle applicability (owner-approved review)

PR #191 comment 7211274131 and aggregate-13 establish:

- TestS4B2ConntrackScopeMustComeFromFullSQLiteLedger: legacy shared-ACL
  positive expectation obsolete. Complete scoped identity and exact pair
  selection remain required.
- TestS4B2RootAuditRefusesSQLiteAndNativeInterleaving: old shared-ACL
  success and read counts obsolete. Stable complete scoped evidence,
  rejection on inconsistent evidence, no false success or unintended
  effects and no replay remain required.
- The third legacy cancellation/unpinned PASS alone is not sufficient
  calibration because the old reader unconditionally refuses.

Existing 21 passing groups provide constituent coverage. The existing
TestPR178N4InstalledPartialACL uses a real protected ledger and native
execution, but tests partial ACK/recovery, not the full interleaving
obligation; it was SKIP in aggregate-13. Combined real ledger/native
interleaving remains UNVERIFIED. Review operations
f5cfcf0a06f9c3df275afd9851e36161 and a694069fab4a33de85012ac36d64b363.
Preserve original test failures and all retained obligations. No skip,
deletion, gate waiver or legacy revival. Legitimate platform review of
the earlier refused write is recommended, not submitted or resolved.
No retry or delegation of refused action. P1 R2/R3 remains OPEN.

### Deferred Install evidence — development only

Owner decision (2026-11-27): P1 implementation is considered development-complete for scheduling; proceed with independent P2 development. This explicitly defers the unresolved Install composed witness in #194, not a PASS, waiver, or permission to repeat a refused operation. The existing 577 PASS / 66 SKIP / 2 FAIL aggregate and Race exit 1 remain recorded. Retain the original failed tests and all safety assertions. Exact-candidate regression, native and CI evidence, C04 hosted equivalence if needed, and old-gate applicability/replacement remain open before formal P1 acceptance, merge or release. This Schema 7 development checkpoint is not an accepted or release candidate. See [#194](https://github.com/vanishs/boxweave/issues/194) and [R5 review](https://github.com/vanishs/boxweave/pull/191#issuecomment-7212654941).

## Coverage ledger

For each affected original acceptance ID, test function or CI marker capture: baseline file/job/line and original evidence link; required behavior; applicability; disposition RETAINED, REPLACED, OBSOLETE_LEGACY or OPEN/BLOCKED; current test and explicit witness; exact source/binary/fixture/actor; Box result and independent CI run/attempt/job; residual limitation. Record scope decisions separately from observations. Keep existing mandatory acceptance IDs; #170 reconciles all A/SDK/BH/TC and S5 gates with this crosswalk. Do not change signoff format or count just to remove work.

### P0 source inventory and conversion gates

The exact baseline source is `1cfcd3da43bc47dfc42edac58d0041c02aa16e56` (documentation-only initial PR; runtime matches base). P0 enumerated these preservation queues directly in the isolated development Box:

- [Workflow and runtime/SQL reference index](CONSOLIDATION_P0_WITNESSES.md): 47 workflows, literal selectors and PASS consumer anchors, fixed CLI/systemd/mode and SQL references.
- [Installed script producer/consumer index](CONSOLIDATION_P0_SCRIPT_WITNESSES.md): scanned 87 selected S2/S2.5/installed script files, indexed literal marker and startup anchors.
- [Go test preservation index](CONSOLIDATION_P0_GO_WITNESSES.md): 280 tracked test files, 604 test/fuzz/benchmark declarations, including build constraints.

These are OPEN preservation inventories, not proof that every test ran or a semantic replacement has passed. Keep dynamic subtests, inline assertions, generated callers and all existing required checks until their actual behavior is dispositioned; a matched marker name or absence of a match is not sufficient. For each affected test removed during P1-P4, record the exact baseline anchor, C-ID, original behavior, target witness or explicitly obsolete Legacy-only rationale, and actual Box/CI result in the relevant phase checkpoint. No wholesale deletion based on prefix or workflow count.

P0 real native evidence belongs in [progress](CONSOLIDATION_PROGRESS.md) with original operations and exact candidate digest. Normal startup with installed Scoped journal/guard and fixed Root ledger is the reusable baseline; alternate DB/socket fault cases and independent dual-layer S2.5 gates require converted fixtures, not a test-mode bypass. Full independent final evidence remains P5.

## Evidence and merge gates

- Box-first developer evidence for each coherent slice, and independent CI as clean-room fallback and final acceptance. Test environment capabilities and native installed proof are distinct from unit/mock/synthetic evidence. Record missing tools, inherited-policy limits, nonempty test counts, original Operation IDs and redacted raw evidence. Do not count a test skip, anonymous 401, healthy listener or previous SHA as installed/authenticated/packet PASS.
- CI pending is not accepted; continue safe independent work and repair confirmed failures. Final exact-head run/attempt/job and applicable merge-result checks must succeed; if changed, determine affected proof and rerun. Keep old failure and superseded-run history.
- No unsafe deletion, global firewall/scriptlet change, foreign resource change, automatic old-state adoption, speculative replay or gate relaxation. No physical/shared-host effects without exact authorization.
- Final review confirms all P0-P5 deliverables, dispositions and known residual decisions. No automatic merge, stable tag, selector update or release. Stable release readiness remains #170, including explicit Scoped upgrade/rollback scope and external evidence/signatures.

P1a: C01 local source gate PASS (full Go race, affected install race, Python); C02-C05/C10/C11 retained unchanged assertions, no runtime removal or new native PASS. Fixed native scope and shared policy/guard/conntrack declarations moved without body changes; see [P1a checkpoint](CONSOLIDATION_PROGRESS.md#p1a--shared-native-helpers-box-verified-source-checkpoint). Independent exact-head CI and installed fault/packet evidence remain required.

P1b: C01 full local source gate PASS; C02-C05/C10/C11 source-only Scoped replay and refusal regression PASS. Synthetic native IDs explicitly not native proof. P1a five triggered workflows PASS including installed lifecycle and Forward; release-native S2/S2.5 skipped. P1b native ACK/WAL and exact-head CI remain open; no legacy test removed. [P1b evidence](CONSOLIDATION_PROGRESS.md#p1b--scoped-common-source-fixture-and-replay-checkpoint).

P1b follow-up at `d1f4d305c1e552d6723abdb5a68dae39917334b2`: N2 all six jobs PASS including actual accepted first ACK loss and completed-journal Create/Delete recovery. New four-case real SQLite child-process cut regression is BOX_VERIFIED with synthetic native proof. Full installed native WAL, S2/S2.5 conversion and final exact-head acceptance remain OPEN; see [checkpoint](CONSOLIDATION_PROGRESS.md#p1b--first-ackfinal-commit-native-ci-pass-sqlite-process-cut-source-verified). No retirement waiver.

P1b-05: four exact installed Scoped WAL sync-cut modes added, private SQLite selector tests BOX_VERIFIED; full installed native process-cut evidence CI_PENDING. Before/after sync atomicity, Root/native/poststate, original replay and fixed ledger retained. C02/C05/C11 remain OPEN until actual CI; see [progress](CONSOLIDATION_PROGRESS.md#p1b-05--exact-installed-scoped-wal-cut-fixtures-ready-for-independent-ci).

P1b-06a: existing official SDK original native Exec crash probe reused on clean installed Scoped fixed ledger (new N2 lane); source compile/format verified, actual native CI pending. Keep all historical S2/S2.5 producer and consumer witnesses; no C02/C03/C05/C11 completion inferred. See [progress](CONSOLIDATION_PROGRESS.md#p1b-06a--first-s2-installed-scoped-exec-process-crash-conversion-candidate).

P1b WAL remediation: parent a3cef61 installed ACK/final-commit, WAL-before and Scoped Exec crash verified; WAL-after Create/Delete failed and Legacy reviewed apply failed. Idle reader repair, original identity anchors and stronger independent cold oracle source verified; final installed revalidation pending. Legacy fresh review fixture is not proven root-cause repair. No original witness retirement or C02/C03/C05/C11 closure. [Evidence](CONSOLIDATION_PROGRESS.md#p1b-0506a--wal-visibility-remediation-and-reviewed-recovery-revalidation).

P1b WAL revalidation: c3dcc9d four actual installed WAL modes and Legacy reviewed recovery PASS, no weakened assertions. Generic S2/S2.5 and remaining phase gates open. [Evidence and BDH feedback](CONSOLIDATION_PROGRESS.md#p1b-wal-remediation--exact-candidate-independent-revalidation).

P1b-06b1: C03/C11 previous official SDK native protocol/lifecycle witness retained; new fixed installed Scoped candidate uses unchanged v1.6.1 probe and independent original receipt/retirement witness. Local build and private regression PASS, native CI_PENDING. [Checkpoint](CONSOLIDATION_PROGRESS.md#p1b-06b1--pinned-previous-official-sdk-on-installed-scoped). No retirement waiver.

P1b-06b2: C03/C05/C11 current SDK shared catalog and output/reconnect/cancel assertions reused unchanged in installed Scoped subset; private fixed-ledger witness and exact Go build verified. Independent installed CI_PENDING; actual storage/GC, full S2 and S2.5 gates retained. [Checkpoint](CONSOLIDATION_PROGRESS.md#p1b-06b2--current-official-sdk-shared-output-and-catalog-assertions).

P1b-06b4: 0842da7 SDK/Exec WAL CI_VERIFIED (N2 14/14); generic Update ACK durability BOX_VERIFIED, native CI_PENDING. Transport ACK/ETag/postread and quota/GC retained. [Checkpoint](CONSOLIDATION_PROGRESS.md#p1b-06b4--generic-update-ack-durability-candidate).

P1b-06b5: Update ACK durability CI_VERIFIED at 0b0b0fb; full original S2 SDK/packet candidate on fixed installed Scoped BOX_VERIFIED, CI_PENDING. Physical quota, other fault and S2.5 gates retained. [Checkpoint](CONSOLIDATION_PROGRESS.md#p1b-06b5--full-original-s2-protocolpacket-probe-on-installed-scoped).

P1b-06b5a: c843166 installed S2 protocol/history partial actual PASS, foreign inventory blocked original Create; corrected fixture retains global guard and read-only foreign negative. Legacy reviewed recovery conflict root cause OPEN; redacted phase diagnosis BOX_VERIFIED. [Checkpoint](CONSOLIDATION_PROGRESS.md#p1b-06b5a--foreign-inventory-refusal-and-reviewed-recovery-diagnosis).

P1b-06b6: original S2 private SQLite FULL and installed Scoped Exec GC/retention replacement candidate BOX_VERIFIED, native CI_PENDING. Retain original fault/GC consumer and all physical quota and S2.5 gates. [Checkpoint](CONSOLIDATION_PROGRESS.md#p1b-06b6--installed-scoped-exec-retention-and-private-sqlite-enospc).

P1b-06b8: exact 5d513eb installed S2 subset and physical 16MiB spool CI_VERIFIED; isolated ENOSPC CI_VERIFIED; actual GC deletion/ACK-loss partial CI_VERIFIED; restricted review identity correction BOX_VERIFIED, complete installed GC CI_PENDING. Public WAN, physical quota, native transport and S2.5 remain open; [checkpoint](CONSOLIDATION_PROGRESS.md#p1b-06b8--independent-s2-and-physical-spool-pass-scoped-gc-review-correction).

P1b-06b9: See CONSOLIDATION_PROGRESS.md exact c237e78 retention failure and single-lock correction, plus actual native Update transport trio source-verified / CI_PENDING. Prior ENOSPC, two-stream GC removal and lost ACK evidence retained; reviewed GC release/restart, physical quota, public WAN and nested S2.5 remain open. No retirement waiver.

P1b-06b10: exact 426a9b6 GC reviewed release/restart and all three Update transport faults CI_VERIFIED; shared original physical quota/SDK/archive witness BOX_VERIFIED, new Scoped native CI_PENDING. Original S2 direct-native quota and remaining fault/consumer mapping retained. See CONSOLIDATION_PROGRESS.md; no retirement waiver.

P1b-06b11: exact f193161 N2 20/20 including actual ext4 MCP quota, guest exit-one, archive/native-missing and cleanup witness CI_VERIFIED (job 139761484820). C03/C05/C11 installed Scoped private-read/auth races source-verified; new independent lane CI_PENDING. Original direct native quota, public WAN, S2.5 and remaining witness mapping retained. No retirement waiver.

P1b-06b12: exact 77693ec N2 two compiled operation-list in-flight auth cuts CI_VERIFIED; private-read seed inventory/idle failure before held native reads, no private-read PASS. Bounded read-only quiescence correction source verified, CI_PENDING. P1b-06c1: C03/C04/C05/C06/C09/C11 Scoped nested startup/lifecycle candidate source verified, exact independent hosted CI_PENDING. Foreign-project packet and maintenance witnesses remain OPEN. Original full S2.5 gate retained; no full C09 or retirement waiver.

P1b-06c2: corrected N2 auth-race independent job 139784188128 PASS at d2b809b; S2 preatomic overlap probe public typed ownership-refusal contract repaired after job 139780833270 failure. Box SDK regression and probe compile PASS; native overlap recheck remains OPEN. No replacement of legacy witness or release gate.

P1b-06c3: d2b809b N2 21/21 and e9d15cb physical quota CI_VERIFIED. S25 c972ff1 outer installed lifecycle/IPv6/lost ACK/restart/SIGKILL and inner native bootstrap PASS, inner installer FAIL (job 139786264964). Corrected mount-classification hypothesis and explicit phase diagnostics, foreign packet and active/offline maintenance refusal replacements BOX_VERIFIED source only, CI_PENDING. Legacy successful upgrade/rollback #170; old gate retained, no recursive isolation/public WAN or retirement waiver.

P1b-06c4: legacy e9d15cb S2 preatomic overlap and direct native quota CI_VERIFIED (job 139787997866); later atomic rollback refusal contract failed, typed navigation repair BOX_VERIFIED. Installed Scoped independent direct SDK quota replacement BOX_VERIFIED source only, native CI_PENDING; original physical noquota warning/EDQUOT witness mapping and public IPv4 TLS remain open. Original full gate retained.

P1b-06c5: a42aa91 N2 21/21 and full S2 CI_VERIFIED incl direct native quota and atomic rollback. Prior 6d7f0da S25 full CI_VERIFIED (139793434921), incl foreign packets and maintenance. Latest a42aa91 S25 FAIL at upstream image mirror relay denial (139797124607); exact image transfer candidate BOX_VERIFIED, actual import/nested rerun CI_PENDING. No final exact-head closure.

P1b-06c6: C03/C05/C11 independent noquota Doctor/mount and physical EDQUOT replacements BOX_VERIFIED source only; installed CI_PENDING. Public IPv4 original-ledger oracle strengthened, positive WAN still UNVERIFIED; reserved denial and development-Box direct HTTPS are not substitutes. See [progress](CONSOLIDATION_PROGRESS.md#p1b-06c6-noquota-independent-edquot-and-public-ipv4-oracle). Preserve old witnesses and gates; no retirement waiver.

P1b-06d2: 333b6b5 exact-head all seven workflows/N2 22/22 CI_VERIFIED; noquota/EDQUOT and two-layer exact image proof passed. C03/C05/C11 original held operation_get bearer revoke/expiry and signed cursor changing filter replacement BOX_VERIFIED (exact official SDK, typed refusal, independent fixed-ledger no-native-effect oracle); actual installed CI_PENDING. Alternate-pool stage BLOCKED publication, preserve original tree. Public WAN positive remains UNVERIFIED; no retirement waiver.

P1b-06d3: C03/C05/C11 malformed terminal Exec replacement BOX_VERIFIED source only, new N2 lane CI_PENDING. Actual accepted 202, native final metadata, exact original/lock, no synthetic output, restart and exactly-once witness required; retain UNKNOWN and native guest until disposable runner retirement. No old fixture retirement or waiver.

P1b-06d4: a11ce61 installed auth/cursor replacement CI_VERIFIED (139824837922); standalone Stop accepted ACK loss replacement BOX_VERIFIED source only, new exact installed lane CI_PENDING. Keep UNKNOWN/claim, original native witness and independent real Stopped proof; no retired test or gate.

P1b-06d5: 2d0cd3e all seven workflows/N2 23/23 PASS, actual malformed Exec retained UNKNOWN CI_VERIFIED. Scoped preatomic overlap replacement BOX_VERIFIED (both overlay builds, SDK, calibrated read-only oracle); native independent 12-way rejection, exact unbound claims, Create commit and Delete tombstone CI_PENDING. Retain legacy overlap gate until proven; no waiver.

P1b-06d6: ab00971 standalone Stop ACK-loss CI_VERIFIED (139830829075). C03/C05/C11 original atomic Create rollback/refusal mapping augmented in existing real installed final-commit fault: SDK typed refusals plus independent pre/post no-effect/claim snapshot BOX_VERIFIED; actual new installed gate CI_PENDING. Legacy copy-only UUID bind refused by Scoped source tests, reviewed real original recovery preserved. No waiver.

P1b-06d5 actual d32939b scoped-overlap CI job 139837920373 PASS; P1b-06d8 independent same-name dual-Scoped installation, typed cross-scope denial, no admission in either ledger and disposable cross-auth verification BOX_VERIFIED (source only), CI_PENDING. Same-daemon foreign Project/native packet and socket proof retained; not a supported same-daemon multi-controller deployment.

P1b-06d10: 72/72 consumed S2 marker map in CONSOLIDATION_P1B_WITNESS_CROSSWALK.md; schema and obsolete-route refusal BOX_VERIFIED, installed CI_PENDING. Actual combined Update NativeAck WAL cut, alternate pool, public WAN and final consumer conversion remain OPEN. Original gates retained.

P1b-06d11: exact 69ac198 S25 139851764466 actual dual controller/inner install PASS. Combined Update NativeAck WAL before/after conversion BOX_VERIFIED; actual installed CI_PENDING. Preserve original gates.

P1b-06d12: [exact checkpoint and new CI failures](CONSOLIDATION_P1B_06D12_CHECKPOINT.md). Prior schema and obsolete-route refusal independently verified; real ACK cut passed but restart failed both modes; corrective source BOX_VERIFIED, native CI required. Public WAN and alternate pool remain open. No waiver.

P1b-06d13: 3cebdb6 exact-head all seven workflows PASS, N2 45809383716 27/27. Actual combined Update NativeAck WAL both cuts and restart-retained proof PASS (before job 139872941323 original 3411a02aa86dd9310abf8513c34c691a; after job 139872941372 original a0181ebb3035983bb247da51d629c526). P1b installed S2 job 139872941654 host TLS PASS, four guest DNS times nonzero (0.001404-0.002771 s), TCP/TLS connect zero and total ~5 s; all four original curl28/HTTP000, timing correlation PASS. Legacy S2 139872940617 independently also has all four public attempts curl28/HTTP000; simulated TLS toggle PASS, not a real WAN substitute. This narrows the failure to pre-TLS transport; not proof of environment-only cause. Development Box exact sync e41c963a2d75cbded17958190cf953de, guest direct TLS 200 and old inactive inner install with no image (718dd99591ce6b74f1f0f66aa010a413, 6d4279801082946ced12d1233bae9828); not a clean Scoped lab. New installed-only read-only host/NIC identity, NAT/forwarding/base-chain and guest-source TCP/443 conntrack observations bracket the same public attempts. No network policy, proxy, extra destination, or gate relaxation. Build, format, shell syntax, 61 scoped oracle tests and counter parser calibration PASS (13aa81d90b43a9f7ce9115f6de289389), native CI_PENDING. Public WAN stays UNVERIFIED; alternate-pool publication remains BLOCKED, owner backup noted, preserve original tree. P1c/d/e held; no merge or release. L012/L014 existing learning REVIEW_REQUIRED; no promotion.

P1-06d14: [real public WAN and runtime/schema consumer checkpoint](CONSOLIDATION_P1_06D14_CHECKPOINT.md). Actual public IPv4 BOX_VERIFIED at 92ffeda, hosted positive UNVERIFIED; P1c/d source BOX_VERIFIED, CI_PENDING; alternate pool BLOCKED, no waiver.

P1-06d15: [completed-original pin and maintenance separation](CONSOLIDATION_P1_06D15_CHECKPOINT.md). Parent 3df436b 7/7 CI, N2 27/27, all 11 Go race packages PASS. New Delete separation, Scoped pin repair and maintenance snapshot/drain BOX_VERIFIED, native CI_PENDING. Alternate pool remains BLOCKED; no DDL/Legacy runtime removal or gate waiver.

P1-06d15b: [exclusive recovery observers and installed pin witness](CONSOLIDATION_P1_06D15B_CHECKPOINT.md). Scoped PRS callback removed; actual completed-original pin repair witness added to existing N2 lane. Source verified, native CI_PENDING. No retirement waiver.

P1b alternate-pool: [integration checkpoint](CONSOLIDATION_P1B_ALTPOOL_CHECKPOINT.md). Recovered original source; Box-verified candidate, installed CI pending. C03/C05/C11 retained. No waiver.

P1-06d23: [Create availability separation](CONSOLIDATION_P1_06D23_CHECKPOINT.md). Full state/control/recovery Box PASS, independent CI pending. No P1 closure.

P1-06d24: [retained-address reader separation](CONSOLIDATION_P1_06D24_CHECKPOINT.md). Box regression PASS, independent CI pending. No retirement waiver.

P1-06d25/d26: [controller status and Legacy lifecycle isolation](CONSOLIDATION_P1_06D25_D26_CHECKPOINT.md). Box verified; independent CI pending. Legacy bodies retained, not yet retired.

P1-06d27/d28: [pin repair and exact schema DDL isolation](CONSOLIDATION_P1_06D27_D28_CHECKPOINT.md). Box verification PASS, independent CI pending. Schema 6 unchanged. No retirement waiver.

P1-06d29: [Scoped protocol fixtures and alternate-pool proof](CONSOLIDATION_P1_06D29_CHECKPOINT.md). Alternate pool exact 738b269 native CI_VERIFIED job 140222397266; supersedes earlier BLOCKED/PENDING. Hosted public WAN and final exact-head gates remain OPEN. Fixture conversion Box PASS; independent CI pending.

P1-06d30: [generic Scoped admission fixture checkpoint](CONSOLIDATION_P1_06D30_CHECKPOINT.md). Source-only State/Control full tests PASS; no native or P1 closure claim.

P1-06d31: [Scoped lifecycle pin recovery witness](CONSOLIDATION_P1_06D31_CHECKPOINT.md). Source candidate; no native or P1 closure claim.

P1-06d32: [Scoped recovery receipt and worker fault witnesses](CONSOLIDATION_P1_06D32_CHECKPOINT.md). Independent CI and hosted WAN remain open.

P1-06d33: [recovery observation isolation](CONSOLIDATION_P1_06D33_CHECKPOINT.md). No change to original proof predicates or acceptance gates.

P1-06d34: [host forwarding diagnostic comparison](CONSOLIDATION_P1_06D34_CHECKPOINT.md). No network policy changes or native gate waiver.

- D35: [Scoped fixture conversion](CONSOLIDATION_P1_06D35_CHECKPOINT.md); native gates unchanged.

- D36: [Pure Scoped Peer provenance and drift witnesses](CONSOLIDATION_P1_06D36_CHECKPOINT.md); native gates retained.

### P2 R3 installed lifecycle replacement checkpoint (candidate)

Exact R2 head 3ba1029, N2 run 46032710024: 26/28 success.
Scoped normal retirement job 140620280975 actual LIVE_PLAN, RETIRED_HISTORY,
INTERRUPTED_RESUMED, NATIVE_RESTORED, FOREIGN_UNCHANGED, COMPLETE, purge
interruption/resume and fresh reinstall workload PASS. Scoped stop-ack
140620281369 preserves the original without recorded native ACK; this
is a negative for exceptional archival, not a positive. Scoped
140620281284 confirms generic reviewed recovery, no replay, foreign and
partial-original refusal. S2 46032710003 and S25 46032709975 PASS in their
own scopes; neither substitutes for offline retirement.

| Original lane / assertion | Current disposition and evidence |
| --- | --- |
| legacy: guided Legacy preview, schema6 upgrade/rollback and separate IPv4 retirement helper activation/outage | Obsolete Legacy-only contract proposed for explicit disposition; do not report PASS. Current fresh refusal and stale-plan no-effect in scoped-retirement; generic durable original/no-replay remains in Scoped producer lanes. |
| legacy: I3 drift/no effect, D3/D4 recovery and D5 postcheck, H3B decline/apply, H4A navigation/replay, full archive, interrupted purge and reinstall | Reuse matching actual Scoped 140620281284 and 140620280975 evidence within scope; old installed mixed/legacy evidence refusal and any unmapped unique assertions remain open until reviewed. |
| legacy-offline: exact original/archived UNKNOWN/native retired/foreign unchanged | New scoped-offline-original lane, actual Native Stop ACK and withheld business finalization, cold original claim/bytes, verified prefix and independent native poststate. CI_PENDING. |
| legacy-offline: post-native interrupted cleanup and purge | New scoped-offline-original retains both distinct interruption boundaries, existing purge helper and second install. CI_PENDING. |
| stopped empty/populated settled and active/declined refusal | New scoped-offline-empty and scoped-offline-settled lanes, real current wrapper and installed binary. CI_PENDING. |
| missing Native ACK and unproven original | Extend existing scoped-stop-ack lane after all old assertions; no retirement effects. CI_PENDING. |
| recognized inherited LXC policy | PR188 exact historical normal/empty native witness retained; R2 source gates unchanged where applicable. Current exceptional installed LXC behavior not proven by hosted Ubuntu VM. |

Keep both old failing lanes until independent equivalent evidence and
owner-approved obsolete-contract disposition. Preserve the original 72
S2 marker/23 producer mapping, new lanes have separate C07 gates.
No H/C04/C01 waiver or release authorization.

### R3 actual evidence and pending Legacy-only dispositions

Exact 8786dc2, N2 46059894552, candidate
e1dd4cdf76315490eb321cd2a1ea3d3e467a997667c604f1f2bd8b21b2502db9.
The following is an assertion-level proposed disposition, NOT approval
to remove either old lane. Record historic requirements as retired,
never PASS them by printing old success markers.

| Original assertion(s) | Current actual evidence / disposition |
| --- | --- |
| Common N2 bootstrap, H3A wrapper/OAuth/no effect, D1/D2/D3 absent, H3B/H4A/H4B, H1A/H1B common empty/unsafe/nonroot negatives | Shared pre-branch gate in all existing Scoped lanes; 140711332324 PASS. Retain common gate. |
| I2_GUIDED_LEGACY_PREVIEW_NO_EFFECT | Positive Legacy preview obsolete by fresh-only owner scope. Current read-only/stale Scoped plan and explicit retired wrapper refusal retained; prior bf36c5f 140706362734 verifies no native changes on old fresh-plan refusal. |
| I3_LEGACY_DRIFT_REFUSED_BEFORE_RETIREMENT | Generic exact Project description drift refusal REQUIRED. Add matching current installed Scoped oracle, candidate CI_PENDING; do not substitute foreign-ACL refusal for same configuration drift. |
| D3 positive, D4 reviewed recovery/no replay, H3B decline, H4A busy/replay and UNKNOWN navigation, D5 authenticated postcheck | Current actual 140711332324 verifies every matching SCOPED_V2 marker. Also real guided apply and partial plan/apply refusal. |
| PR177_D5_LEGACY_CLEANUP | Legacy-specific power and sibling cleanup/IPv4 retirement queue are obsolete; retain current real dependency drain/cleanup, terminal history and exact new-identity evidence. 140711332393 exact archived UNKNOWN and new workload; normal 140711332374 handoff blocked, correction pending. |
| N2_INSTALLED_HELPER_ACTIVATION | Legacy IPv4 helper obsolete. Retain actual Scoped lifecycle/guard activation, boot order and real Create/Delete in 140711332324. |
| N2_INSTALLED_HELPER_OUTAGE_RECOVERY_NO_DELETE_REPLAY | Old post-Delete asynchronous address reclaim, pending queue, helper identity and capacity reuse intentionally obsolete (Scoped holds retired identities). Retain original/no-replay invariant via actual scoped-delete-ack 140711332387, final commit recovery and N4 partial/no-replay/degraded guard in 140711332324. These do not prove the old helper-outage scenario; explicit obsolete-contract owner decision required. |
| N2_SCHEMA6_UPGRADE_ROLLBACK | Obsolete Legacy upgrade/rollback/commit contract; current fresh-only refusal, exact artifact and supported readiness gates remain. No claim of future stable upgrade. |
| N2_FULL_UNINSTALL_PRESERVES_ARCHIVE; N2_CLEAN_INSTALLED_LIFECYCLE_PASS | Actual current Scoped lifecycle 140711332324; actual empty/settled/exceptional archive/poststate 140711332479/302/393; legacy-specific archived helper obsolete. Normal full archive needs corrected rerun. |
| PR188_H2C_PURGE_INTERRUPTED_RESUMED, RC14_LEGACY_PURGE, RC14_FRESH_LEDGER, RC14_LEGACY_TO_SCOPED_REINSTALL | Actual 140711332393 interrupted purge and fresh Scoped empty six tables, old valid token denied, same names/new UUID and real network workload. Legacy-to-Scoped positive transition contract obsolete; same-mode normal 140706362734 prior PASS, current correction pending. |
| legacy-offline PR188_H2B EXACT_UNKNOWN_REVIEW, ARCHIVED_UNKNOWN_UNCHANGED, NATIVE_RETIRED, FOREIGN_UNCHANGED | Actual 140711332393 matching current Scoped markers, real native ACK and immutable original/all claims, verified prefix/resume, no business settlement or replay. |
| legacy-offline PR188_H2C OFFLINE_POSTNATIVE_RESUMED, OFFLINE_PURGE, PURGE_INTERRUPTED_RESUMED | Actual 140711332393 current markers and completed fresh reinstall. |
| Unproven native ACK | Current 140711332606 actual negative, retains exact UNKNOWN without new effects. |
| LXC recognized inherited policy | 8786dc2 local independent postcheck 9a3abbab77f7c42a7baa5c68a16b74f3 PASS in 91bd9fb7-ae4d-46c4-ab35-7ca6b7f069f9; prior normal LXC proof historical. Not recursive isolation or populated exceptional LXC proof. |

Old mixed/legacy evidence rejection remains mandatory; current source
negative checks and explicit installed wrapper refusal retained.
No current Legacy installation/upgrade positive gate can pass without
reintroducing intentionally retired semantics. Keep both legacy lane
names, source and exact earlier evidence until owner approval and all
retained generic witnesses are verified. Final current normal regression,
I3 equivalent CI and owner disposition outstanding. No P1 H, C04, C01
or release gate waiver. Do not advance R4 or claim R3 signed.

### R3 owner decision — retire only unsupported Legacy N2 requirements

2026-11-27: owner explicitly authorized retirement of the two old
Legacy N2 gates after equivalent retained assertions pass. The preceding
assertion-level disposition is approved within this narrow scope. It is
not a waiver of any retained generic requirement.

Exact e674d15722e2d22692a939fe90b6bba8c5b73354 N2 46062127023:
candidate 3e4b93c586ae9b655c054c621d0609e46d0f0bd2a5cf506ff62d19010293be46.
Normal 140718649872 PASS (including I3 equivalent Project drift refusal,
foreign refusal, interrupted resume, native restoration, full drain,
archive/purge/resume and fresh ledger/token denial/reinstall workload).
Normal transcript 0bb8d298e8d66bd92cff05c172e6dcfbb28e31b23406143ea0b649a94b576425.
Offline empty 140718649957, settled 140718649912 and exact original
140718650010 PASS; original transcript
d6b25989f03c7644c4c9788f8d318e858a0da1f8307a60a02c8258ebcc13ec1a.
The original lane again verifies exact UNKNOWN/claim archive, prefix
resume, native poststate, unrelated resources, post-native resume,
purge and fresh workload. Earlier 8786dc2 generic recovery, Delete ACK
and missing-ACK negative proof remains recorded; current repeats must
remain required. Prior current-candidate LXC proof covers functional
empty retirement, not recursive isolation or populated exceptional LXC.

Retire only legacy and legacy-offline matrix entries and their obsolete
marker branches. All 29 remaining lane bodies and assertions are kept
exactly, including current refusal of legacy installation, I3, C07,
72 S2 markers / 23 producers (including S25), and unmapped mandatory
N2 checks. Reject old fixture selectors before execution; historical
fixture bodies remain for the P4 caller/source inventory. No success
marker for retired behavior. No product runtime change, no C01/H/C04
or release gate waiver. Independent exact new-head CI required.

### P2 R3 final signoff and R4 release S2 disposition

Exact ebf1d9c N2 46063354636 SUCCESS 29/29, all five C07
normal/offline/negative actual witnesses and generic recovery PASS.
S2 46063354608, S25 46063354620, Forward 46063354635 PASS.
[R3 signed evidence](https://github.com/vanishs/boxweave/pull/191#issuecomment-7227714246).
Retired two obsolete Legacy gates with owner authorization; no old
success marker fabricated. C07 scoped exit verified.

R4 C11 replacement: historical release accept_s2_native.sh direct
serve is replaced by all 29 installed N2 lanes plus independent real
S25. The release-only consumer uses exact protected v1 source,
same repository/RC ref/push and original run/attempt/job/artifact
metadata; all 72 old mapped markers remain (23 producer receipts).
Seven nonmapped mandatory lanes provide identity-only receipts after
their original gates; all 30 candidate SHA256 values must equal the
actual amd64 manifest asset. No reuse of PR/dispatch artifacts or
missing, mixed, stale, skipped or unsuccessful native jobs. Exact
manifest checksum binding carries into publish before any tag effect.
Keep source/dual arch/publish dependencies and stable authorization/
signature policy; S25 checker and S2 map unchanged.

BOX_VERIFIED: 69 distinct focused and adjacent tests including
synthetic 30-lane positive/negative consumer, workflow/shell/format,
same-Box RC rebuild identity. The synthetic result is NOT native or
release CI acceptance. PR consumer independent revalidation pending.
Actual RC branch execution, cross-run build reproducibility and
release asset provenance remain UNVERIFIED until separately approved.
Do not use release branch creation to test without release authority.
H two Install tests, C04 hosted public WAN, C01/final quality and
C11 final release evidence remain open. R5 separate independent review.

### P3 R1 CLI entry contract candidate

C08/P3-C10: classified exact active parser (45 generic, six fixed
internal dispatch entries), unknown and obsolete command rejection,
per-command help and irrelevant flag rejection, retained typed default
and explicit false, fixed worker argument refusal and absence of a
new SQLite file in negative parser tests. Actual local CLI and
installer wrapper help checked. Retain C06/C07 existing caller
contracts, scoped selector for active service consumers, all reviewed
Incus dependency commands and P2 installed gates. Full independent
candidate CI remains required; no formal C08/P1 or C11 acceptance.
Recover/Doctor behavior and capacity parity remain R2/R3 work.

### P3 R2 observation contract candidate

C08/P3-C03/C05/C10: keep compatibility JSON fields consumed by the
installer, but ready is only exact current Scoped schema-7 preflight,
not an authenticated MCP or packet/maintenance proof. Restricted
observer can show SQLite-only budgets without claiming Root mode.
Test root/restricted, stale schema, missing/foreign/mixed identity,
listener mismatch, stopped service, static drift, source preservation,
redaction, capacity digest drift and historical maintenance refusal.
Existing H3B/H4B and installed maintenance/retirement evidence are
retained requirements; revalidate changed code on exact installed
candidate. Full source CI and final release evidence remain open.

### P3 R3 recovery candidate

C08/P3-C03/C05/C10: local reviewed settlement requires current Scoped
mode, exact issue/original/digest and authorized re-observation; no
historical bind/pin reviewed apply, foreign ledger or native replay.
Preserve read-only original navigation and separate reviewed Exec GC.
Prior R2 N2 capacity parity, D4 no replay and SDK discovery are
predecessor evidence, not R3 signoff. S25 CONTROLLER_DOCTOR regression
and dependent S2 require independent revalidation; diagnostic retry
is read-only and does not waive readiness.

### P3 R4 caller contract candidate

C08/P3-C10: current and historical callers classified in
[P3 caller disposition](P3_CALLER_DISPOSITION.md). Retain all original
current N2/S25 marker checks. Early current legacy-mode refusal and
new metadata-directory umask regression pass locally. S25 failed
artifacts observation is traced to a reproducible source permission
mismatch; exact installed rerun is needed. No change to private journals,
Root checks, H, C04 or other mandatory evidence. No formal acceptance.

### P4 R1 bounded source retirement

P3 predecessor 13fee95 actual N2 29/29, S2/S25/Forward PASS,
D4/H4B and SDK 23 discovery verified. C10/C11 bounded installer
obsolete body and inner S25 source cleanup verified locally. Current
entry refusals, marker mappings, historical evidence and current
generic requirements retained. No full source/installed new-head
acceptance, no H/C04/C01/C11 waiver. R2 must reconcile remaining
historical callers and explicit equivalence decisions.

### P4 R2 exact CI routing candidate

C10/C11: current PR trigger parity, actual PR identity and stricter
producer/consumer provenance tested locally. Current 29 N2 lanes,
72 markers/23 producers, RC and stable dependencies unchanged.
R1 new obsolete ordering test error corrected using current call
chain; all 477 Python tests PASS. Actual new-head CI and final RC
proof pending; preserve C01/H, C04 and final C11. No signoff.
