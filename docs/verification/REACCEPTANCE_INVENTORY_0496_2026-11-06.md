# v0.49.6 reacceptance inventory, mapping and fixture reliability

**Scope:** development-only reacceptance runner under `tools/reacceptance/`, a source-only feature mapping manifest, and ordinary automated fixture regressions. No production MCP/Web/CLI permission changes, additional listener, resident service, auto-repair, or new customer diagnostics component. **Source CI is not independent installed acceptance.**

## Root-cause review of PR #100's partial v0.49.4 ledger

| Defect | Actual cause | Remediation | Acceptance boundary |
| --- | --- | --- | --- |
| False HTTP source routes, e.g. `GET Authorization` / `GET code_verifier` | `source_web_routes` walked **every** `.get/.post` call, including request header/form dictionary reads | Inspect only Flask `@app.get/post/put/patch/delete/route` function decorators; validate path and methods | Source declaration is **not** installed route and still needs authenticated HTTP test |
| Inflated CLI help tree, e.g. `boxweave http configure configure` or `setup text https` | A regex took any `{…}` from help, including optional argument **values**; fallback help could repeat parent syntax | Consume exact command prefix and only a **top-level positional** choice group; mark fallback as unconfirmed and exclude it from advertised installed callable leaf rows | Help or syntax discovery never equals positive/denied functional acceptance |
| Incorrect F mapping, e.g. every Web route marked F18/F19 and every remote MCP route F22/F23 | Broad overlapping dependency globs were combined indiscriminately, regardless of action semantics | Explicit mapping for actual source Web/HTTP routes, 36 app CLI actions and 68 known upstream MCP names. Most-specific rule takes precedence, unknown future actions remain `UNMAPPED` | Feature tagging is a **selection** input, never evidence of PASS; issue unknowns as mapping drift |
| False functional failures from missing Box preconditions | The installed read-only runner sent `{}` to optional target-taking tools or accepted a caller-provided username as evidence of a usable guest | Block target-dependent probes until native-attested, test-owned UUID/createdAt are available. Preserve the actual error as **SUSPECT/BLOCKED** pending a semantic oracle; do not claim tested positive | A blocked tool is not a product bug and does not count as passed |

The historical PR #100 report remains immutable evidence about the original run. Re-running against the exact new candidate must generate fresh per-tool and per-feature inventories; do not silently edit old numbers (68 actual installed MCP, 77 help observations, 178 previously source-counted entries).

## Safe fixture preparation path (explicit opt-in only)

The existing source-only `tools.reacceptance.lifecycle` tool can now accept `--with-readonly-breadth`. It still requires its original **private**, current-host/outer-UUID/createdAt/expiry-bound grant, a unique `bwracc-…` child, exact user confirmation, and nested native Incus access. The sequence is:

1. Verify the exact host, private approved grant, absent target, then **record a pre-dispatch intent** and perform at most one MCP create.
2. Read the child from native Incus and pin its exact UUID and createdAt. No `RUNNING`/logical-name-only shortcut.
3. With the **same** valid child, perform only allowlisted read-only MCP positives/invalid probes, saving original private evidence. A missing schema prerequisite remains BLOCKED, not fake FAIL.
4. Re-read the native incarnation and state. On any replacement or UNKNOWN, stop all subsequent writes; never replay, adopt or delete a changed target.
5. When original state and identity remain valid, continue the previously approved stop/start/delete sequence with native oracles and no residue.

This is an **operator-invoked test-only harness**, not a production BoxWeave feature. Source CI uses fixtures only; it cannot establish installed native correctness, OAuth/public access, full MCP triads or a separate Verifier. An Agent performing the full command must still secure a unique isolated test parent and maintain the original UNKNOWN freeze policy. **The owner host is never auto-provisioned.**

## Operator workflow

In an approved disposable nested Provider host with the current external reacceptance source checkout, create the private lifecycle grant using the existing documented exact-host procedure, then run the previous lifecycle command with this optional flag:

```sh
python -m tools.reacceptance.cli lifecycle \
  --grant /private/exact-owner-grant.json \
  --confirm I_AUTHORIZE_TEST_OWNED_LIFECYCLE \
  --mcp-argv-json /private/verified-mcp-argv.json \
  --out /private/new-racc-run \
  --with-readonly-breadth
```

This call is **mutating**, requires separately approved test-only scope, and must never be run blindly on a normal production host. Without the flag, the previous 4-operation lifecycle contract is unchanged. The installed read-only breadth utility does not independently accept an arbitrary `--read-only-target` as native identity proof.

## Required follow-up evidence

- Exact candidate SHA/version and source-derived routes/CLI help inventory; no `GET Authorization`, `GET code_verifier`, repeated argument-choice CLI paths, or blanket F18/F19/F22/F23 mappings.
- Fixture test cases for source route decorators, CLI opt-values/fallback, precise action tagging, missing-target false failures and bounded test-owned read-only lifecycle.
- Actual installed CLI help and MCP catalog on the correct **inner** test Provider, not an outer ChatGPT tool list.
- Independent native status/oracle for any test-owned child, explicit original dispatch and cleanup, all suspicious tool errors retained privately.
- Per-tool remaining positive/invalid/denied cases and separate independent-Agent installed acceptance remain PARTIAL/BLOCKED until actually exercised.

**Release policy:** normal main CI builds and publishes the immutable v0.49.6 source release only after full source verification. It does not move the stable installer pin or assert a complete system-level PASS.
