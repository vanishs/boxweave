# Agent-operated reacceptance simplification: approved release trust model

**Status: OWNER APPROVED, implemented in source (PR #90).** The owner explicitly authorized the default trust change on 2026-11-05 PT: distinct independent Agent + separate authorized Box real evidence, submitted and checked by an authenticated GitHub account through official workflows, with no mandatory extra signer, self-hosted Runner or protected environment approval. Independent actual remote testing, complete scenario evidence and candidate-byte parity remain mandatory. Implementation does not itself prove any installed case passed or publish a Release.

This document records the approved replacement and tradeoffs. The [active runbook](../process/FULL_PROJECT_REACCEPTANCE.md) controls execution and preserves prior host/UNKNOWN authorization boundaries. [The capability audit](REACCEPTANCE_TOOL_CAPABILITY_AUDIT_2026-11-05.md) separates runnable helpers from incomplete automation. The original draft patch files below remain historical review artifacts, not the controlling workflow source.

## Active owner-facing contract

1. Coordinator pins current source/runtime and exact approved test targets, discovers installed MCP/CLI/Web/Provider surfaces, and schedules real workflows plus invalid/denied/recovery cases.
2. Finder preserves original failed request, stdout/stderr/exit and an independent native state oracle before repair. Source fixtures and catalog discovery remain separate from behavioral acceptance.
3. Fixer repairs the root cause, runs focused regressions, follows version policy for runnable changes, and obtains the exact successful main CI candidate. No independently rebuilt substitute or changed bytes under an old tag.
4. A different Verifier Agent installs those same candidate bytes in a separate authorized Box, reruns the original reproductions and current applicable functional cases, and records its own observed output/state. Existing Agent collaboration and MCP/CLI tools perform the orchestration; missing automatic test drivers do not prevent manual execution.
5. If required behavior is untested/failed, observations disagree, candidate safety is UNKNOWN, or a release-blocking defect remains, repair and independently verify again. Preserve uncertain originals without replay, unlock, adoption or deletion. Historical #88 targets do not need to be destroyed to test a new candidate safely.
6. Once actual acceptance passes, the Agent records bounded sanitized original/native evidence through an ordinary repository-authorized GitHub workflow, then invokes the official publication workflow with the original candidate and evidence run IDs.
7. Official publication verifies source/provenance, all assets and original case files, creates an immutable tag/Release and downloads assets for byte parity. Stable installation promotion remains separately authorized. Clean successful approved resources and retain failed/UNKNOWN scenes.

No routine repeated owner choices are required inside existing grants. Production/operator-host modifications, Provider upgrades, SSH/trust/credential changes, public exposure, original UNKNOWN recovery and owner-host reboot keep their existing specific permission boundaries.

## Owner-approved change in release trust

| Area | Previous signed-gate behavior | Approved default |
| --- | --- | --- |
| Test execution | Incomplete self-hosted install/inspection workflow always exits PARTIAL/2 | Independent Agents execute actual tests through existing tools; CI only records/checks resulting evidence |
| Runner | Dedicated self-hosted test Box Runner expected | No mandatory Runner registration; ordinary Box control through existing authorized MCP/CLI |
| Submission identity | External signer key and protected deployment environment | Existing repository-authorized workflow caller; no additional human approver or long-lived signing key |
| Independent acceptance | Distinct actors/Boxes and original/native evidence required | Retained; report submission authentication is not cryptographic proof of remote execution |
| Case integrity | Exact source/artifact, required case graphs and hash-bound bytes | Retained; Agents may execute graph nodes manually; all canonical scenarios accounted for |
| UNKNOWN | Durable intent and frozen original incarnation | Retained without replay/unlock/delete |
| Publication | Official workflow, immutable versions, same candidate and draft/public byte checks | Retained; no force flag or manual release bypass |

**Tradeoff:** the approved default trusts the repository-authorized Agent to report genuinely independent execution truthfully. It removes the additional cryptographic signing identity and GitHub environment approval boundary. Raw evidence and hashes support inspection and tamper detection; they cannot prove that the reporter did not fabricate observations. An independent signing path can remain an optional advanced choice, not a default infrastructure requirement.

## Implemented code and workflow changes

- Added the data-only evidence carrier `tools/reacceptance/box_acceptance.py` and 13 targeted tests (archived [review patch](reacceptance-draft/box_acceptance_helper.patch)). It packs/checks original receipts and actual case JSON, retains original input, derives only missing hashes, rejects contradictory hashes and unsafe/oversized archives, and stamps only authoritative workflow metadata. Every result states `release_authorized=false`; it does not sign, run tests or publish.
- Replaced `.github/workflows/independent-verifier.yml` with GitHub-hosted evidence recording and validation (archived [review draft](reacceptance-draft/independent-verifier.yml.txt)). Use GitHub-hosted CI to validate submitted sanitized evidence against the exact main candidate. A successful recording run does not imply CI itself performed the Box tests.
- Applied the [reviewed CI patch](reacceptance-draft/release_workflow.patch) to remove the mandatory environment/key checks and invoke the same strict byte/provenance/semantic checks through the evidence carrier. Preserve original candidate/run identity, complete asset binding, immutable published releases and downloaded-byte comparisons.
- Updated the active runbook/AGENTS/status references after explicit owner approval. Former signed/Runner designs remain knowledge references. Runnable workflow/tool changes require passing PR CI and merge before the new gate runs on `main`; neither a product Release nor an independent acceptance claim follows from this change alone.

Inputs are sanitized test evidence only, never credentials, signing keys, JWTs, production data or unrestricted shell commands. Use structured request files and bounded stdin. Original private logs remain in their originating Box; do not silently truncate or publicly export them. Data size/transport failures remain explicit blockers rather than permission to fabricate a smaller PASS.

## What documents alone can already accomplish

The Agent can already discover tools, allocate specifically authorized test Boxes, spawn real independent Agent roles where collaboration is available, execute missing supported cases manually, preserve evidence and loop over fixes. No new test scheduler is necessary. Existing helpers have partial scopes, which must remain honestly labelled. Required actual tests, safety, current runtime identities, original repros and per-action accounting are retained.

The documentation-only limitation has been addressed by the source workflow/helper changes in PR #90. Repository-admin rights and ordinary GitHub-hosted Actions are sufficient for the newly approved recording path; dedicated Runner and signer setup are not required. A real independent test and correctly authenticated evidence submission are still necessary. PR CI and merge must complete before this is available on `main`.

## Useful advanced references to retain as knowledge

- [System automation design](../design/REACCEPTANCE_SYSTEM_BUG_HUNT_IMPLEMENTATION.md): scenario scheduling, dependency maps and future RACC-SYS engineering ideas.
- [Execution orchestration design](../design/REACCEPTANCE_EXECUTION_ORCHESTRATION_PLAN.md): future adapters and optional Proof Pack integration.
- [v0.49.1 source/evidence audit](../verification/RACC_0491_RELEASE_GATE_2026-11-05.md): historical signed-gate implementation and real source/CI evidence; do not retroactively rewrite those results.
- [Issue #88 summary](../verification/RACC_0491_CREATE_LATENCY_2026-11-05.md): unresolved original acknowledgement and deadline boundaries; proposed policy changes do not resolve it.
- `tools.reacceptance.release_gate`, `signing_preflight`, `verifier_worker`: useful optional formal signing and one-attempt lab-install components. The former policy was used by historical workflow revisions; signed checks remain optional advanced code paths, not the default active gate.
