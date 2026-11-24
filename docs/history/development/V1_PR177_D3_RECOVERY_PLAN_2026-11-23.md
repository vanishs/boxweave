# PR #177 D3 - read-only evidence-bound recovery plan

**D4 candidate update (2026-11-23 PT):** [Reviewed recovery execution and exact Box evidence](V1_PR177_D4_REVIEWED_RECOVERY_2026-11-23.md). D2/D3 independent exact-head CI all four passed. D4 code, three focused Race rounds, full verify and fresh scoped installed positive recovery passed in Box. D4 independent CI is required before D5; legacy installed D4 and scoped partial apply refusal remain CI gates. Earlier staged-status notes below are historical, not instructions to redo accepted stages. Keep Draft; no v1 merge, release or physical L0 modification.

Status: Box-verified candidate for independent CI. D2 installed CI correction included. No D4 apply, production install, v1 merge or release.

## D2 reconciliation
Exact head ad51a1ced4948984ed9cf21699d2dd7a549510a3: source 45345918542, quality 45345918532 and architecture 45345918540 passed. Installed 45345918529 failed in scoped and legacy jobs 138280305679 and 138280306013: Python KeyError inventory_scanned. Actual Doctor JSON field is entire_project_inventory_scanned. Correct the assertion, add TestPR177D2InventoryJSONContract and retain all acceptance checks. Subsequent installed gates were not reached. D2 is not signed off.

## D3 contract
- Use doctor, then recover --plan --issue <issue-id> -project <project>. Fresh installation, bounded ledger and exact original observation. Stable issue ID is not review digest or idempotency key.
- Versioned JSON binds project, proven mode, installed identity, original revision, affected Box identities and resource references, original credential status, ledger/native proof digests, observer UID, typed leaf, postconditions and two-minute review window. Private evidence hashed.
- REVIEW_READY is a supported existing leaf evidenced for review, not approval. Original settlement without native replay, legacy identity binding and legacy pin ledger repair are eligible. ApplyAvailable always false. D4 must independently revalidate evidence and authorization at commit.
- Separate read-only snapshot/recovery observer; no writable Store, challenge, maintenance lock, recovery receipt, native mutation or state transition. Fixed-scope restricted actor handoff. No architecture guessing.
- Scoped Root explicit reconcile-approved read-only selector restricted to original UNKNOWN; phase and canonical proof rechecked server side. Normal dispatch distinct. Old helper may refuse; no fallback. Existing public MCP recovery uses explicit selector; no public schema/count change.
- Block changed, foreign, expired/revoked, unsupported or unproven originals. Scoped Create/Delete repair unsupported. No reset, compaction, automatic address reuse, broad unlock or replay. Preserve resource evidence even when credential inactive.

## Box evidence
Development Box ac585d2c-37ae-4879-a2e6-3cecc338d4d7, exact D2 base. Initial observer a867378a9ea73836ad60ecf5c3647252 and integrated plan 753833e8431ddddbb8a953d19c779159 passed. Precondition compile issue corrected; three Race rounds d4795fdea70846495755a09451ae39b2 and focused candidate 8fb763257e324629e29a5a88c047e985 guest exit 0:
- D3_SCOPED_READ_ONLY_ORIGINAL_HISTORY_AND_PROOF_PASS
- D3_SHARED_OBSERVER_STALE_AND_PRECONDITION_PASS
- D3_REVIEW_DIGEST_SCOPE_REVISION_EXPIRY_AND_UNSUPPORTED_PASS
- D3_EXPLICIT_ROOT_SELECTOR_NEVER_DISPATCHES_PASS
- D3_CLI_NO_SOURCE_CREATION_NO_APPLY_PASS

Full make verify 160ed55cbdac3042827078b6cc537b12 guest exit 0: format, YAML, module checks, vet, all Go Race packages, 16 MCP tests, 42 uninstall tests, 10 stable gate tests, 7 witness tests. Full run precedes final evidence-output ordering fix.

Historical scoped lab 8d93d497-95e7-46af-925c-c6e08f08eb5a: noninstalled candidate, original 53254acbb11eeaf598faaca6eadf1778. Observation 055bc4c82019714ea1a5ae947e920f84 correctly blocked expired credential but omitted resource evidence. Move refusal after evidence population. Retest 9c4dd6547d964d4daa2e69c9fdfaf392 guest exit 0: BLOCKED_UNPROVEN, SCOPED_ORIGINAL_CREDENTIAL_NOT_ACTIVE, mode UNVERIFIED, five claims and two affected Boxes, no apply/digest/retry, unchanged observed active operations and service PID. No installed binary or original credential changed. Final corrected regression 8d2acbec2e3dedc754b7804d5ce3319e guest exit 0: three Race rounds diagnostic/recovery/CLI, including five claims and two Boxes on refusal.

## CI and D4
Exact submitted head source/quality/architecture/installed CI required. Both installed modes check D2 corrected JSON assertion and D3 absent issue/no effect. Scoped degraded original retains five claims, blocks plan without new recovery receipt, reruns existing native degraded acceptance. Synthetic positive proof tests are not native acceptance; fresh installed positive planning with valid original credential and complete native proof remains an explicit validation gap. D4 must revalidate current evidence, permissions and expiry, never accept caller-supplied plan as authority. Preserve original reconciliation/no replay, unrelated Box connections and scoped history limits. No effect work by weakening refusal.
