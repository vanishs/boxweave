# BoxWeave v1.0.0 — formal stable publication plan

Date: 2026-11-21 PT. Owner-selected order: **stable candidate → evidence matrix → independent dual signatures → protected dry-run → stable publication**. Status: PLAN, not release signoff or authorization to change a physical Host. This supersedes historical planning checkpoints, not their original evidence or safety requirements. [Post-release environment reuse](V1_POST_RELEASE_ENVIRONMENT_REUSE_PLAN_2026-11-21.md) must not enter or block this candidate.

## 1. Baseline and feature freeze

At review, protected v1 HEAD was 3e55a4bae822eb3618edb8d734bf8e6cfdbbea72 (merged [PR #169](https://github.com/vanishs/boxweave/pull/169)); VERSION=1.0.0, V1_PUBLISHED_RELEASE=UNPUBLISHED. Neither stable v1.0.0 nor RC12 Release was found. GitHub default branch is v1 and both stable workflows are active. Recheck these facts when executing. Freeze the exact current protected v1 HEAD **after this planning commit**; the above SHA is provenance, not the final candidate.

- Published [RC11](https://github.com/vanishs/boxweave/releases/tag/v1.0.0-rc.11), source d7087d4c9a7e571ca71bdc785274e4534f944941, amd64 SHA256 54d16100e06b543568c4efdcb36cbd4bc07ae9412aeb7f271924f71f0bdce218; all eight [release jobs](https://github.com/vanishs/boxweave/actions/runs/44995638227) PASS. Physical L0 RC11, Schema5, Incus 7.6, ready/mcp_ready, unchanged unit and original authenticated connection; RC10 rollback retained. Quota attestation remained unverified.
- Subsequent merged [#168 guided upgrade](https://github.com/vanishs/boxweave/pull/168#issuecomment-7075896828), [44997672394](https://github.com/vanishs/boxweave/actions/runs/44997672394): real RC10→RC11→candidate→rollback, original credential and unit/drop-in preserved.
- Subsequent merged [#169 Project capabilities and Forward diagnostics](https://github.com/vanishs/boxweave/pull/169#issuecomment-7076208565), [44999974362](https://github.com/vanishs/boxweave/actions/runs/44999974362): reviewed fresh capability, permission denial, actual wildcard packet, cleanup/rollback and legacy preservation.
- #166/#167 are already included in RC11. Source and artifact delta review is mandatory; prior RCs and scoped S5B evidence do not prove final stable artifact acceptance.

Only release-blocking corrections are admissible during freeze. No new feature/tool/dependency or schema experiment. An additional RC is optional, not needed to generate installable test bytes. If HEAD moves (even docs-only), a candidate run attempt changes or assets expire/change, rebind the exact candidate and assess affected evidence; new signatures and dry-run are mandatory. Never retag or replace a released RC.

## 2. Execution gates and accountability

| Step | Work | Owner of exit | Required exit |
| --- | --- | --- | --- |
| R0 | Freeze exact source and supported scope | Release engineer + owner | Protected SHA, real inventory/delta and prerequisites reviewed |
| R1 | Build stable candidate, no publish | CI | Four successful exact-source jobs, six verified assets, run/attempt |
| R2 | Final evidence crosswalk and targeted acceptance | Test engineer + independent verifier | All mandatory supported-scope rows resolved; genuine native/installed evidence and independent review |
| R3 | Sign reviewed evidence | Owner + independent verifier | Two distinct trusted signatures over identical raw signoff; separate immutable evidence commit |
| R4 | Protected dry-run | CI + approved reviewer | STABLE_RELEASE_GATE_PASS, DRY_RUN_NO_PUBLICATION |
| R5 | Protected publication/readback | Owner authorizes; CI writes; verifier reads | PUBLISHED_EXACT_READBACK; exact tag/commit/assets; non-draft/stable |
| R6 | Closeout and stable selector | Owner-approved PR; operator | Separate pointer review, immutable release record, supported compatibility baseline |

No protected gate, reviewer, approval or native safety refusal may be bypassed. Continue independent safe work when a row is blocked.

## 3. R1 — exact stable candidate

Manually dispatch [v1-stable-candidate.yml](../../.github/workflows/v1-stable-candidate.yml) on the reviewed v1 branch once freeze is approved:

```sh
gh workflow run v1-stable-candidate.yml --repo vanishs/boxweave --ref v1
```

Verify actual repository, workflow_dispatch, head_branch=v1, exact source and successful latest attempt with exactly stable-candidate-source, stable-candidate-build-amd64, stable-candidate-build-arm64, stable-candidate-package. Retain the actual run ID, attempt and artifact ID/name v1-stable-assets-<attempt>. Six files: both boxweave-v1.0.0-linux-<arch> binaries, release.json, SHA256SUMS, install.sh, uninstall.sh. Native CGO builds, architecture, build-info, checksums, dependency pins and exact standalone script source must match [v1_stable_gate.py](../../scripts/v1_stable_gate.py). Record both binary and manifest/checksum SHA256, package digest, candidate source and all job IDs. 30-day artifact retention; do not let it expire. STABLE_CANDIDATE_PACKAGED_NOT_PUBLISHED is packaging only. Use these exact bytes in installed acceptance; no rebuild at publication.

## 4. R2 — evidence matrix and delta acceptance

Read [S5 canonical phase](phases/05_RELEASE_ACCEPTANCE.md), [A6 gate](S5A6_RELEASE_GATE_2026-11-20.md), [RC10 closeout](S5B_RC10_CLOSEOUT_2026-11-21.md), [formal review Section 9](S5B_RC10_FORMAL_ACCEPTANCE_REVIEW_2026-11-21.md#9-owner-approved-combination-evidence-decision-and-final-rc10-technical-review-2026-11-21-pt), and [#169 capability contract](V1_PROJECT_CAPABILITIES_2026-11-21.md). Reconcile [ACCEPTANCE](../verification/ACCEPTANCE.md) A-001–082 and SDK-G01–G08, [BRIDGEHEAD](../verification/BRIDGEHEAD_ACCEPTANCE.md) BH-001–019, [CONTINUITY](../verification/CONTINUITY_ACCEPTANCE.md) applicable TC cases including TC-019 actual installed Tool/CLI crosswalk and TC-020 calibration, and all S5A1–A6/B1–B6. Discover actual current catalog, schemas and CLI leaves; older role/count text is historical. Do not create all-PASS rows from counts or summaries.

For each row record: original case and latest supported requirement/owner decision; actual native/installed/independent evidence and immutable ref + SHA256; observed source, binary, environment, original BoxWeave/native IDs where applicable; result and evidence class (exact candidate bytes, exact-source lab build, historical native, simulation, or scope/refusal); delta from witness to stable candidate; affected regressions, remaining gap and reviewer. Preserve historical negatives and explicit unsupported behavior. No missing mandatory case may be labelled PASS. Unsupported/excluded historical scope needs its actual owner decision and truthful refusal or limitation, not a simulated positive.

Reuse approved valid evidence first. Do not repeat already accepted recovery groups, ChatGPT handoff or LAN merely to collect another green log. Independently accept exact stable candidate bytes in supported native environments; candidate packaging CI alone does not run S2/S2.5. Run applicable installed native and nested regression on separate clean environments, verify current tool/CLI inventory and adversarial negative controls. Target the actual source/contract delta and unresolved required cases:

- #166/#169: Forward loopback management, explicit external ACK, fresh native permission versus legacy preserved unit/overrides, wildcard/collision/IPv4/IPv6, dependent cleanup and restart/UNKNOWN, observed native denial vs controller refusal. Packet proof cannot be inferred from config.
- #167: compiled operation_list post-query authorization boundary; two independent revoke/expire cuts and unchanged original receipt. Historical failing assertion not conclusively established.
- #168/#169: exact verified local/package install, service-only vs native preparation, one-command upgrade, correct approval/plan, non-TTY/refusal, previous recovery-window closure, same-byte no-op, corrupt backup refusal, retained schema/service/credentials/unit and real rollback. Test exact final assets; no implicit permission expansion or native preparation on upgrade.
- Physical evidence: verify latest S5B chained LAN temporary Host permission restoration if not already confirmed; do not repeat A-049 without a relevant delta. Quota witness is time/boot-bound. Real physical tests, deployment or ending existing RC10 rollback on L0 require separate exact-target owner consent.
- Existing supported-scope decisions: three RC10 residual PRS/generic/GC positives accepted via historical/native/exact-source/affected-regression combination, not literal final download bytes; review deltas before reuse. #151 historical cause unresolved under approved bounded retrieval/refusal/regression. #150 strong hostile deep recursion excluded; conditional trusted nesting and truthful refusal remain. Actual ChatGPT automatic refresh/raw refusal not witnessed; independent backend + actual two-grant/handoff substitution accepted, no credential extraction or tunnel repetition. Simulated routed IPv6 remains simulation; A-049 actual chained TCP is not UDP or current Host state.

S5B RC10 scoped PASS is not full stable qualification. Keep actual missing/blocked cases explicit, obtain a precise owner/verifier decision on any changed supported scope or targeted tests as necessary. The signed gate accepts **all twelve S5 gates with status PASS** only after mandatory supported-scope evidence is reconciled. No PARTIAL, BLOCKED, SIMULATED_PASS, WAIVED or missing gate can pass. Scope decisions do not convert historical untested positives into observations.

## 5. R3 — independent dual signatures

Follow [A6 gate contract](S5A6_RELEASE_GATE_2026-11-20.md). Owner configures and verifies protected GitHub environment v1-stable-release, v1-only deployment policy, independent required reviewers, no self-review or bypass, separately pinned Ed25519 public PEMs STABLE_OWNER_PUBLIC_KEY and STABLE_VERIFIER_PUBLIC_KEY. Verify actual trust setup, do not infer from active workflows. Private keys remain with distinct trusted custodians outside Agents, CI, Boxes, repository and logs. Test keys or Agent-authored approval cannot sign a real release.

Assemble raw release-signoff/v1.0.0/signoff.json: exact format=1, purpose=BOXWEAVE_STABLE_PUBLICATION, repository=vanishs/boxweave, source, tag, candidate_run_id, candidate_run_attempt, release_json_sha256, sha256sums_sha256, gates (exact S5A1–A6/B1–B6 each status PASS and nonempty evidence list of {ref,sha256}), scope_decisions (same evidence ref/digest shape). Use strict field/size rules in gate; no fixture signoff. Both sign the identical raw bytes independently, detached binary 64-byte owner.sig and verifier.sig. Store these three files and reachable evidence on a separate immutable evidence commit, not on frozen v1. Record commit SHA and signoff digest. Reviewer must verify actual external evidence, not only cryptographic/JSON shape.

## 6. R4–R5 — dry-run and publication

Dispatch [v1-stable-release.yml](../../.github/workflows/v1-stable-release.yml) on v1 with actual candidate_run_id and immutable evidence_commit, publish=false. Its [driver](../../scripts/v1_stable_release_ci.py) verifies actual protected HEAD/run/attempt, exact artifacts and both signatures; tag and release must be absent. Require successful environment review, STABLE_RELEASE_GATE_PASS with publication=DRY_RUN_NO_PUBLICATION. No write permission in dry-run. Recheck unchanged source, attempt, assets, evidence and trust. Only after separate owner publication authorization, dispatch same inputs with publish=true. CI must reverify before publishing once; require PUBLISHED_EXACT_READBACK, exact lightweight tag source and six verified downloaded assets, live non-draft/non-prerelease Release. If partial publication leaves tag/upload uncertain, stop and reconcile; never automatically retry, overwrite, delete/retag or substitute rebuilt bytes.

## 7. R6 — closeout and next phase

Record exact tag/source, candidate and evidence identity, dry-run/publish job IDs, readback digests, supported claims/limitations and reviewer. V1_PUBLISHED_RELEASE stays UNPUBLISHED until a separate reviewed owner-approved PR promotes it to the actual verified stable tag. This does not upgrade any Host. Physical deployment remains separate exact-target approval. Freeze supported stable MCP/CLI/schema/data compatibility per [versioning](../process/VERSIONING.md); retain original UNKNOWN and recovery history, no destructive reset.

Only after v1.0.0 publication/readback and recorded release closeout may the [post-release environment plan](V1_POST_RELEASE_ENVIRONMENT_REUSE_PLAN_2026-11-21.md) begin implementation (prefer stable-pointer review complete). Next coherent accepted feature increment normally 1.1.0; no version bump for an unimplemented plan.

**Next action:** confirm the final protected v1 SHA and approve R1 candidate dispatch. At this checkpoint no stable candidate, complete final matrix, trusted signer setup/signatures, stable dry-run or publication is asserted. Keep new features deferred.
