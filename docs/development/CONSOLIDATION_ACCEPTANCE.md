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
