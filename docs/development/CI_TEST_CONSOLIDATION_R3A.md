# R3A: special Exec original receipt migration

Parent b081bbe1f331143403b0a8aef9fa3ce83c1c2422, PR #201. Reused exact existing Box 73d2ecae-3932-46e6-b4c6-3feecf7fba15. R3 starts with the Exec group from R1 closeout.

## Existing evidence and change

The old installed BH018 candidate and S5A1 workflows require actual special exits 126/127, restart readback, original idempotency and cleanup. Current installed Exec WAL/crash lanes already address terminal commit faults, independent SQLite receipts and one-effect checks, but do not themselves prove the complete 126/127 restart readback. Existing old/current SDK output lane covered 0/7/124.

Reuse the unchanged official Go SDK s2_bh018_atomic_mcp_probe.go in the existing scoped-sdk-compat lane. No additional workflow or matrix job. Compile in the existing native runner, after the old and current SDK scenarios run the special probe start, independently read fixed Schema 7 SQLite and exact native UUID, require no unsettled operation before controlled service restart, prove controller process replacement without Incus/Root lifecycle or guard change, compare original rows, resume the same probe and verify independent readback, same-key dedupe and concurrent original Delete. Check seven durable originals (Create, both special Exec, both fresh readbacks, Stop, Delete), exact exit codes, distinct tracking/native identities, same actor/project/Box incarnation, tombstone and released binding, no native Box/extra ACL or unsettled operation. Preserve private checkpoint on failure; no replay of UNKNOWN or forced cleanup. Require original probe markers and independent fixed-ledger marker in the current lane. No product runtime or release gate bypass; no invocation on the development host.

## Actual verification

- Inventory 7b99303c4b3119ae1f0e20f617b1f480 exit 0: three old workflows, current lane and source anchors. Existing direct 27-case native provider/BH019 nesting A/B is not equivalent to this change.
- First oversized input rejected at admission; work-ref operation list inspected before smaller edits.
- Focused test operation f39724ffb3cf730b8a17b66e0467c86c exit 0: 97 tests PASS (88 core focused, 6 SDK witness, 3 Exec WAL oracle). New SDK oracle test includes wrong exit, native ID, UNKNOWN, foreign project/incarnation, duplicate, wrong tombstone and binding negatives. Shell syntax, four-file format and diff checks passed. Staged code-only tree 05e7be6ca391d26a2ef59d5d853febbfd6e2826e, export 1ebad6eecc34f0bef6885da79d9022a9 exit 0.
- Development Box has no Go compiler. The reused probe is unchanged but new lane integration has not run against the real installed candidate. Existing 30-minute native lane timeout remains unchanged; actual combined duration must be checked in CI. Previous head Forward CI succeeded, other R2 checks were still pending at last read.

## Disposition and next action

R3 partial. Keep all 20 historical-only routes for now; in particular retain all three Exec wrappers until exact-current native PASS and remaining direct-provider diagnostic/S5A1 repeated-run and raw evidence mapping. No assertion that this replaces all 27 direct SDK cases or BH019 provider comparison. Once native evidence is available, decide whether old installed/S5A1 wrapper obligations are fully mapped; do not retire merely because a new test exists. Next review File/Git group and existing S3 integration fixtures while CI runs. Identity/OAuth/recovery, Peer/network, Forward/combined lifecycle groups remain open. Workflow count 30, scripts count 215 unchanged, code/test lines +119 net in this slice. No measured runner saving, merge, publication or host mutation.
