# Task checkpoint (copy, do not edit this template)

Use the authoritative [client-owned contract](../../docs/development/TASK_CHECKPOINT_CONTRACT.md). Keep a canonical versioned checkpoint in an authorized remote branch/PR or durable store; reference it from the active PR. This illustrative YAML is not a server record, lock or automatic resume.

```yaml
format_version: 1
work_ref: example/task
attempt_ref: example-attempt
updated_at_utc: "YYYY-MM-DDTHH:MM:SSZ"
objective: "Owner-approved task"
stage: verify
workspace:
  kind: box
  box_name: "example"
  box_uuid: "example-owned-uuid"
  layer: "L1"
  path: "/example/worktree"
  deletion_approval: not_approved
source:
  repository: "owner/repository"
  commit: "exact-commit-sha"
  worktree: unknown
verified_completed: []
current_uncertainty: []
original_operations: []
do_not_repeat: []
next_safe_action: "Verify original workspace, Git and Operation state"
links: []
```

Include exact native/BoxWeave IDs and evidence refs where applicable, not secrets or full logs. Record dirty work and how it is durably preserved (or explicitly not preserved), remote PR/head, candidate digest, required CI and blocker. Compare revision before updating; a conflict requires reconciliation, not overwrite. Unavailable data stays unknown. Never infer a current state from this template or from a stale checkpoint.

## bwdev v0.1 JSON alternative

The fields above remain the contract. A generated JSON record is an alternative
serialization, not a second authoritative copy. Use a fresh file outside the
worktree and explicitly persist a sanitized copy in the PR or authorized store.

```sh
python3 dev/bwdev --json checkpoint capture /tmp/task-attempt.json \
  --work-ref example/task --attempt-ref first --objective "Validate task" \
  --stage verify --repository owner/repository \
  --next-safe-action "Inspect original state"
python3 dev/bwdev --json checkpoint check /tmp/task-attempt.json
```

For Box use, also supply `--kind box --box-uuid EXACT_UUID --layer L1`,
and `--operation-id ORIGINAL_ID` for each relevant operation. Capture records
observed Git state, not the uncommitted changes themselves. Check exit 1
means review required, not failed native operation. Always independently
read original Operations and exact Box/remote state before continuing.

For budget pressure, follow the [early checkpoint guidance](../skills/task-resume/SKILL.md#early-checkpoint-before-call-budget-pressure-candidate). In the existing notes/links record observed call count or estimate and scope, actual persistence location, any unsaved bytes, and next safe action. Do not invent remaining quota.
