# R1 closeout: workflow retirement and remaining coverage queue

Exact reviewed source `2d505657cb8cd4194862ae4c3d00b683cb6e0849`, PR #201. R1 route classification is complete; final CI and native acceptance are not implied.

| Route | RC15 baseline | Current |
| --- | ---: | ---: |
| Historical only | 36 | 20 |
| Core | 8 | 8 |
| Optional manual | 2 | 2 |
| Total | 46 | 30 |
| Workflow lines | 10241 | 7736 |
| Historical workflow lines | 9114 | 6609 |

16 historical copies and 2505 lines retired. All 213 script files and 296 Go test files unchanged. The per-file decisions and immutable old-source references are in [R1A](CI_TEST_CONSOLIDATION_R1.md), [R1B](CI_TEST_CONSOLIDATION_R1B.md), [R1C](CI_TEST_CONSOLIDATION_R1C.md). Prior three slices verified retained workflow bytes and unchanged runtime/scripts. No claimed runner-time savings.

## Remaining 20 historical routes grouped for R3

All remain guarded to historical refs, without PR triggers. Keep until explicit per-assertion equivalence or obsolete-contract disposition. The following are reuse/investigation candidates, not new job requirements.

| Group | Historical basenames (without .yml) | Current constituent evidence and unresolved mapping |
| --- | --- | --- |
| Exec (3) | v1-s25-bh018-installed-candidate, v1-s25-bh018-native, v1-s5a1-installed | Current installed exec-before/after WAL, cold SQLite and crash one-effect checks; Go special-exit tests. Original installed 126/127 independent post-restart receipt, direct SDK 27-case matrix and BH019 provider comparison not shown equivalent. Review s2_bh018_atomic_mcp_probe.go and bh018_native_exec_probe.go for minimal reuse. |
| File/Git (2) | v1-s3-file-installed, v1-s3-file-native | Existing S3 real native SFTP integration and file/state source tests; current S25 explicitly rejects S3 phase2 selector in Scoped mode. Guest Git repair is not file tools proof. Map exclusive 12-writer winner, hostile types, bounded listing and independent original/Git bytes; reuse s3_file_installed_probe.go. |
| Identity/OAuth/recovery (2) | v1-s4-a-installed, v1-s5a2-installed | Current OAuth HTTP replay/revoke tests, installed auth-races, overlap, ACK/WAL and recovery lanes. Independent raw terminal/SQLite correlation, actual installed OAuth refresh no redispatch, supported local CLI/MCP recovery parity and post-delete evidence need mapping. |
| Peer/network (11) | v1-s4-b1-native, v1-s4-b2-bridge-ownership-native, v1-s4-b2-controller-root-installed, v1-s4-b2-dedicated-native, v1-s4-b2-native, v1-s4-b2-quiescent-native, v1-s4-b3-final-closure-native, v1-s4-b3-final-gates-native, v1-s4-b3-foreign-revoke-native, v1-s4-b3-grant-native, v1-s4-b3-stale-etag-native | Current N3/N4 installed directional/live-flow/UDP/partial/identity/boot and modeled CAS/port guards; S25 foreign packets. Old global PRS/shared ACL and address reuse positives obsolete. Map real ETag competition, foreign NIC denial and established cross-project continuity, actual ledger/native interleaving, internet=false direction/gateway denial, native service-cycle fencing and linked Delete. Modeled cases are not installed proof. |
| Forward/combined lifecycle (2) | v1-s4-b4-forward-native, v1-s5b-peer-production | Current N2 real Forward/partial/concurrency/ACK/lifecycle and Forward policy. Listener collision, removal/new-flow denial, dependent Delete and combined active Peer/Forward/pending Exec restart need precise mapping. |

Preserve R1C real missing-link/current mutation composition question and existing C04/H limitations. Some obligations may already have current evidence outside inspected routes; search existing evidence before developing any test. Keep two optional manual routes. Do not waive applicable assertions.

## Evidence and status

Existing Box 73d2ecae-3932-46e6-b4c6-3feecf7fba15 at clean exact parent. Exec d716e1882133350a4e74495c091a6de5 exit 0 (current crosswalk and S25 selector); e24e0ff4cd99dcc85f59da2887a4f3ee exit 0 (WAL producer anchors and tracked script inventory; some guessed file paths absent); b4339ee0fc554ee9cf8e2de09150d6bc exit 0 (20/20 historical job guards, no PR trigger, existing file/OAuth/BH018 test anchors). Static guard check only, not live dispatch. Current s2_native_probe.go explicitly checks exits 7, 124, 0, not the original installed 126/127 matrix.

First and second deletion slices all five respective CI runs passed; third slice CI pending at last read (PR 46129680226, 46129680257, 46129680268; push 46129675969, 46129676003). Each deletion slice 88 focused tests passed; Box/remote tree matched. No new native run in this closeout. Required gate identities unchanged.

R1 bounded implementation/classification complete; acceptance pending CI, remaining generic behavior addressed by R3. Do not repeatedly classify the same old wrappers without new evidence. Proceed R2: consolidate active shared preparation/build/receipt helpers with unchanged assertions and independent isolation. R3: resolve five groups using existing evidence or minimal missing witnesses before further removal. R4/R5 unchanged. No merge, publication, ruleset or host change.
