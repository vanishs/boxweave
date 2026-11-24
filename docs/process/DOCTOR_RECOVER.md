# Doctor and Recover: operator guide

This guide describes the merged v1 implementation from PR #177, not the
published RC13 executable. D0–D5 passed exact-head CI and were merged on
2026-11-24. Use the exact installed build's help and manifest. One binary,
two local CLI entries, unchanged 23-tool MCP catalog. See [current status](../INDEX.md).

## Diagnose without modifying the installation

Use the exact confined Project, normally user-UID for the dedicated
boxweave service user. Prefer MCP for ordinary managed-Box work.

- On reconnect, use `operation_list` or `box_activity_get` to find the
  original Operation, then `operation_get` and, for Exec, the original
  `operation_output_get` exit code. Do not retry UNKNOWN as a new action.
- `boxweave doctor -project user-UID` reports JSON with independent
  source coverage, installed architecture, ledger and native observations,
  typed issues and stable issue IDs. Run locally as root for Root-only
  evidence; a restricted service-user report is intentionally incomplete
  for scoped Root authority. Keep the report private.
- A healthy service or a successful native config observation is not a
  packet, listener, authenticated OAuth, tunnel or storage quota witness.
  A bounded or moving inventory is not an atomic all-source snapshot.
  Historical failures and retained tombstones are not necessarily blockers.

For an UNKNOWN original, inspect the existing MCP `operation_recover`
with mode `inspect`. Its independently authorized reconcile route is
preferred when supported. Host-local repair requires the appropriate
local authority; do not give the Agent general Host access.

## Review one issue and apply only a supported plan

From a local operator terminal, replace the variables with the actual
installed Project, stable issue ID, reviewer and separately chosen durable
recovery key. Preserve the private exact plan:

```sh
umask 077
sudo boxweave doctor -project "$project" > doctor.json
sudo boxweave recover --plan --issue "$issue" -project "$project" > plan.json
```

Inspect the full JSON, even if the command returns nonzero. A root plan
must be version 2, REVIEW_READY, apply_available=true, with the exact
scope, original, leaf, fresh evidence and review_sha256. The plan is not
approval. It expires after two minutes for a new effect; obtain and
review a fresh plan if the evidence changes or it expires. Never apply a
historical or blocked plan. After explicit local approval, substitute the
reviewed digest, not a recomputed digest of altered JSON:

```sh
sudo boxweave recover --apply --issue "$issue" -project "$project" \
  --approve-local --reviewer "$reviewer" --recovery-key "$key" \
  --confirm-review-sha256 "$review_sha" < plan.json
```

The exact committed key and review can recover the durable receipt after
a lost response, including after expiry; a different key cannot reuse
stale evidence. The command never redispatches the original native
action. An identity binding is not settlement of UNKNOWN. Review the
receipt and durable postcheck, then run doctor again and independently
inspect the same original through MCP Operation and Activity. Native
postcheck is NOT_REPROBED_AFTER_LEDGER_COMMIT: verify current native
state or actual packet paths separately if required.

## When to stop and preserve evidence

If a plan is BLOCKED_UNPROVEN, a proof is incomplete or changed, a
native journal cannot be verified, a resource is outside the exact
scope, or the required local authority is unavailable, preserve the
original claims and evidence. Do not use SQL edits, native replay,
blanket unlock, policy resets, reinstall or unrelated Box restarts to
force progress. A reviewed settlement is not a generic repair tool.
Scoped fresh v2 does not support legacy upgrade, rollback or uninstall.
Legacy guided retirement/resume has its own exact review and audit flow.

See [the D4 execution contract](../development/V1_PR177_D4_REVIEWED_RECOVERY_2026-11-23.md)
and [the D5 acceptance record](../development/V1_PR177_D5_DELIVERY_2026-11-24.md).
