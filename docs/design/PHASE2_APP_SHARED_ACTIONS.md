# P1-B2 / R-04 — app BoxActions shared execution claim

**Source candidate, opt-in only.** This slice connects the existing app Web/worker **BoxActions** dispatch, including configured agent-box write tools and auto-sleep toggles, to the same previously provisioned private coordination SQLite used by the SSH/HTTP gateway tracked `boxweave_task_submit`. The app's original durable Box action receipt remains authoritative; the coordinator never stores commands, file contents or stdout.

## Activation contract

Only a reviewed app-owned runtime JSON property `"coordination":{"enabled":true}` enables the app BoxActions guard. Any malformed option or fixture opt-in fails closed. No user-controlled endpoint can set it. Before dispatch, the app verifies the root-published Provider/backend/project scope in the 45-second host registry, exact pinned app Target identity and the two common `box:` and `name:` resource keys. It acquires a claim with the existing receipt ID, records its original app receipt as dispatching, commits shared dispatch intent, then calls the existing BoxAgent adapter. No SQLite transaction spans remote operations.

If resources are busy, the registry is stale/mismatched or the shared SQLite is unavailable, the action is marked failed **before any remote write**. If the original dispatch outcome is UNKNOWN, or an auto-sleep / long-running process returns only acknowledged status, the shared claim remains held for later exact receipt-based reconciliation. A succeeded terminal BoxAgent action releases its claim only after the original app receipt is durably committed. Neither crashes nor abandoned leases cause automatic replay.

## Important limits

This is **not** a turnkey global R-04 guarantee. Standard installations without the explicit app opt-in preserve the old Web worker; legacy direct gateway `connect(exec)` and raw Provider mutations remain unprotected. All named mutation entrypoints must either join the same guard or fail closed *before* any installed blanket multi-Agent promise. A real installed source-matching Web worker and SSH/HTTP gateway sharing the same service UID/private SQLite have not been observed. Avoid automatically changing installed host config, ACL, service account or stable installer.

## Test/acceptance gates

Source fixture tests cover app/gateway same-Box race; independent Box; committed Web receipt before claim release; post-dispatch transport UNKNOWN; stale/mismatched root scope; missing DB predispatch denial; app config opt-in validation; no command disclosure. Full project CI and an authorized disposable Linux Box unit run are required before source merge. Real installed provider and both live transport gates, multi-version helper restart, provision/upgrade/rollback and full lifecycle/access write coverage remain U.
