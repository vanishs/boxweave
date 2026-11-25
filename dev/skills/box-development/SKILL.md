---
name: box-development
description: Choose, inspect, reuse or prepare an authorized Box-first development environment, or a conventional fallback; diagnose missing dependency and fixture capability problems.
---

# Box development

- Recover existing task workspace before provisioning. Read exact UUID, worktree, branch, outstanding operations and running services. Do not restart or delete unknown/shared resources. For docs-only work, direct repository editing is an acceptable fallback.
- Match environment to task: source tests need pinned language tools/Git; real native tests need the explicitly authorized nested capabilities and separate candidate controller; clean install tests need a clean fixture; experiments and evals need isolated reproducible starting state.
- Before tests check actual toolchain versions, Git availability, repository root, required base ref or exact authorized SHA, free resources and required capabilities. Record unknowns. Check native extension/config/authority and actual fixture state when relevant. Do not infer readiness from a version string.
- Use existing setup and acceptance scripts only after reading their effects and prerequisites. Never borrow parent sockets, bypass parent confinement, expose publicly, modify physical/shared host or remove existing resources without exact approval.
- Record environment recipe, versions, exact test commands, failures and evidence; reuse verified caches and worktree. A populated lab is not a clean runner. If capability is missing, use a suitable authorized local/VM/CI lane and record the reason.
- Retain environment after work; cleanup follows the existing owner-approval policy. Save state outside it before handoff.

References: [workflow](../../../docs/development/AGENT_WORKFLOW.md), [nesting](../../../docs/design/NESTING.md), [H5 historical evidence](../../../docs/development/PR188_H5_DELIVERY.md), [task-resume](../task-resume/SKILL.md). Git/base and fixture checks derive from actual H5 failures; this is guidance, not an implemented preflight utility.

Optional v0.5: use bwdev profile or start --profile to observe task-specific prerequisites from [profiles](../../profiles.json). Presence of Incus/systemctl is not native readiness. Use existing authorized recipes and independent capability checks; the profile does not install or provision.

## Bounded guest source transfer

If large inline argv is refused at admission, distinguish an explicit refusal with no receipt from a lost acknowledgement. Read any original Operation before further effects; never replay uncertain work or bypass admission. For a new patch or source file in the owned isolated guest, use exclusive box_file_apply create, confirm terminal success and verified file readback, then execute a short explicit argv referencing the reviewed file in the correct checkout. Check actual guest exit and output, and inspect the changed files. Do not infer an exact size limit, overwrite a pre-existing file, or apply this as authorization on a Host. See [BDH-L014](../../learning/BDH-L014/00000003.json).

For repeated or multi-part source transfers, prefer the candidate
[offline transfer planner](../../transfer_plan.py) when a reviewed Python
checkout is available. It prepares explicit exclusive-create request JSON,
a stable bounded manifest and initial intent; it never dispatches MCP.
Keep the original intent and per-call receipt/readback journal outside Git
if source bytes are sensitive. Use assess to reconcile; missing/unknown
or conflicting evidence stops retries. Submit only authorized exact
requests, inspect original Operations and verify readback. Assemble with
the reviewed helper in the owned guest only after all part and manifest
receipts/readbacks are verified; confirm guest exit zero and independently
hash all final readback pages. Existing target, missing/corrupt chunks,
unproven outcome or unsupported exclusive publication must stop. The
manual procedure remains the fallback, not authorization to bypass
admission. The helper is a candidate until independent review and real
ordinary-task benefit evidence.
