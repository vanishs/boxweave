# P1-B2/R-04 — root-policy Gateway mutation classification

**Source candidate v0.39.0. NOT enabled on the installed host.** This is an optional strict mode for the same standalone forced SSH and HTTP MCP Gateway. It never grants a new agent identity or RBAC scope.

## Boundary

The Gateway reads a root-owned, non-group/other-writable `/var/lib/boxweave-default/public/coordination-policy.json` file with strict schema `{"version":1,"enabled":true}`. An absent marker means legacy behavior for existing installations; an existing but malformed, symlinked, untrusted or unsafe policy fails closed. The file is **not** generated, installed or enabled by this release. No client MCP method can set it. A future reviewed root CLI activation must verify the app's coordination opt-in, private shared DB ownership, both current Gateway helpers and rollback barriers before writing this marker. Manual file placement is not an accepted deployment workflow.

With protection on, the Gateway advertises only known Provider reads (`list_containers`, `get_container`, `get_system_info`), reviewed gateway-only read/diagnostic tools, `boxweave_nesting` for read-only `plan`, and the original R-02 `boxweave_task_submit/get/list` surface. The Gateway denies legacy direct `connect(exec)` or human SSH instructions, unknown/forwarded Provider methods, create/resize/delete and `boxweave_nesting apply` **before** forwarding to the Provider. The tracked submit retains its existing independent exact Box trust, original durable task receipt, private shared Box/name claims and UNKNOWN no-replay behavior. Provider text metadata and readOnlyHint alone never whitelist a tool.

Both SSH and OAuth-authenticated HTTP MCP use this identical gateway source. Protected-mode filters must execute on every `tools/list` and `tools/call`, including existing long-running SSH sessions; failed policy reads do not silently fall back to legacy mode.

## What is NOT done

Web app opt-in `coordination.enabled=true` exists for BoxActions only. Web lifecycle, access, direct external SSH outside this Gateway and any unreviewed old helper remain outside the shared boundary. No all-entrypoint conflict-safety guarantee is made. The root policy is deliberately **not activated** and no host permissions, SSH/Incus, Provider or app config were modified. E2/E3/E4 snapshot/restore/clone remain unsupported without upstream contracts.

## Acceptance

Source tests: `tests/test_gateway_protection.py` covers bad root ownership/modes/symlinks/schema, missing-marker legacy compatibility, no Provider dispatch for direct/unclassified mutations, hidden mutator catalogue and verified tracked/read allowlist. Whole project CI and standalone packaged helper imports must succeed. Real dual-client SSH/HTTP with root-owned policy preflight, app Web lifetime, old helper, restart and failure rollback remain separate LIVE U checkpoints. Do not market this source candidate as protected production operation.
