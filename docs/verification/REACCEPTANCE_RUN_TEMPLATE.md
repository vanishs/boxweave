# Full reacceptance run ledger and next-Agent handoff template

**Status:** workflow template only; not a claim that any test has run. For each invocation of [the full-project reacceptance runbook](../process/FULL_PROJECT_REACCEPTANCE.md), create a concise version-scoped report in `docs/verification/` (and sanitized test artifacts in existing approved locations). Keep prior reports immutable in meaning; do not overwrite historic PASS, UNKNOWN or incident evidence.

## 1. Run manifest

| Field | Required value |
| --- | --- |
| Run ID / start (UTC) / coordinator | Exact |
| Source revision / dirty state / intended candidate | Exact SHA, diff identification |
| Published tag + artifact SHA-256 / installed / running app-worker-Gateway | Record each **separately**; use U if unknown |
| Provider daemon/MCP and Incus versions | Observed or U |
| Outer project/backend and disposable Box UUID + createdAt | Immutable identity; no guessed ownership |
| Authorized operations / exclusions / tested policy mode | Named and scoped; R-02/R-04 opt-ins separately |
| Finder / Fixer / Verifier identities and independence | Actual Agents or **same-Agent sequential fallback** |
| Observation oracles / sanitized evidence location | Named source, command/receipt and stable reference |

## 2. Dynamic surface inventory (mandatory)

Produce a complete row per **MCP tool and CLI leaf in each actual installed policy/transport** from each available client context, or a detailed machine-generated manifest with a human-readable checked-in summary and immutable digest. A top-level help page or the outer ChatGPT MCP tool list is not a complete installed inventory. Record the discovered total and the unable-to-discover contexts. No hardcoded claim of a fixed MCP tool or Web route count.

| ID | Surface / action | Source / installed seen? | Transport/policy | Read/write/risk | Test case ID or explicit defer reason | State / evidence |
| --- | --- | --- | --- | --- | --- | --- |
| MCP-S-001 | Actual tool name | Source S; running I confirmed/U | SSH stdio + auth/policy | Read / P1 | SC-... | PARTIAL / S,I |
| HTTP-... | Actual HTTP MCP tool | ... | Auth/client mode | ... | ... | ... |
| CLI-... | Actual installed CLI action | ... | Host/app permissions | ... | ... | ... |
| WEB-... | Actual route + method | ... | Cookie/CSRF/auth | ... | ... | ... |
| REST-... | Actual upstream interaction | ... | Authenticated Provider | ... | ... | ... |

Surface classification must cover **all** observed entries, even when not safe to execute. Entries exposed only upstream/directly should be marked **outside BoxWeave enforcement**; distinguish them from BoxWeave-owned routes. A missing catalog is a BLOCKED *discovery context*, not proof of an empty catalog.

## 2A. Required **per-action MCP/CLI adversarial execution ledger**

**Every discovered MCP tool and every installed CLI leaf command must appear here individually**, per real installed SSH/HTTP MCP transport, identity/policy and host/app CLI context when different. Enumerate recursively (including hidden `boxweave http`, `boxweave task`, `boxweave selftest` dispatch and `python -m bwmini` modes). The outer ChatGPT-exposed MCP catalog is not a valid replacement for inner installed catalog discovery. **No sampling exception** for low-risk/read-only/forwarded tools.

| ID / function | Actual runtime MCP tool or installed CLI leaf, version and policy | Positive real call/output | Invalid, missing, boundary-input result | Unauthorized/stale/wrong target denial or justified N/A | Independent postcondition, exact exit/receipt and cleanup | PASS/FAIL/PARTIAL/BLOCKED/SKIPPED/UNKNOWN, evidence and next safe step |
| --- | --- | --- | --- | --- | --- | --- |
| MCP-... | Exact tool, SSH/HTTP mode, current catalog | Sanitized request, actual reply, case ID | Named case + typed error | Case + observed zero forbidden side effects | Provider UUID/createdAt, guest, receipt, distinct observer | Scope-specific status and proof |
| CLI-... | Exact executable, subcommand and flags | Actual stdout/stderr, exit and behavior | Exit/error and named case | Auth/permission/target invariant where meaningful | Separate state observation or honest unavailable reason | Status and proof |

For every **safe/authorized/runnable** entry, deliberately attempt real installed positive, malformed/boundary and relevant denial cases, rather than only `tools/list`, `--help`, a single successful call, or fixtures. For read-only tools, positive + absent/invalid target; for mutators verify success and **no forbidden side effect** on denial. A dimension not meaningful for that command is `NOT_APPLICABLE` with a specific reason, not an invented denial. Any individual unsafe/unsupported/unavailable operation receives a **separate named** BLOCKED/SKIPPED/UNKNOWN row with exact authority/dependency blocker and next command. Low priority does not justify skipping. Discovery-only cannot be PASS. An unknown dispatched mutation is preserved, never retried merely to fill the table.

## 2B. Required **owner-facing detailed functional acceptance table**

Use the [consolidated prior-scheme functional baseline](REACCEPTANCE_FUNCTIONAL_COVERAGE_BASELINE.md), [historical 39-feature evidence depth](EVIDENCE_LAYERS_0449.md), [call-chain/R-02/R-04 plan](../design/RUNTIME_CALL_CHAINS_0449.md), [test strategy](TEST_STRATEGY.md), feature matrix and new installed surfaces. Reconcile **all historical F01–F39** plus current/new features; historical PASS never grants current PASS. Every tool/CLI leaf must map to one or more named functions and every function back to actual implementing commands/routes.

| Feature ID + understandable purpose | Actual MCP tools/CLI leaves/Web/REST entries | Concrete positive scenario tried/result | Concrete malformed, denied, boundary, concurrency or cross-interface scenario/result | Expected vs **independently observed** effect and receipt | Status, exact installed version and evidence S/T/I/H/X | Untested paths, bug/issue references and smallest next-agent test |
| --- | --- | --- | --- | --- | --- | --- |
| Fxx / human-readable feature name | Exact names (no counts-only) | Actual case, command and exit | Actual cases or detailed non-execution reason | Provider/guest/task/Incus independent read | Scoped PASS / PARTIAL / FAIL / BLOCKED / SKIPPED / UNKNOWN | Missing subpaths and next action |
| Xyy / newly discovered feature | Real entrypoints | Test or explicit not tested | Test or explicit not tested | Real oracle or unavailable | Same | Actual blocker |

**Acceptance of the report requires every named feature row, including passed, failed, blocked, skipped and untested features.** Each row explains **what the feature does, what was intentionally challenged, what occurred and what remains unverified**, so the owner can judge the intended coverage. A tally such as “15/68 tested” or a list of only failures is insufficient. If large, commit the full human-readable table plus the tool/action ledger, link them in the final response, and still display a readable grouped feature summary covering all areas (not only totals).

## 3. Scenario evidence (black-box-first)

For **every safely runnable discovered MCP tool and CLI leaf**, the required adversarial dimensions are **positive + meaningful authorization/identity denial + malformed/boundary input**, or exact per-action unavailable/unsupported reasoning; combine the entries into risk-weighted scenarios from earlier acceptance schemes. Real authenticated Web positive tests are not replaced by fixtures or a 401. Test-owned mutation requires original identity and independent readback.

| Case | Boundary / workflow | Expected response and remote effect | Actual response/exit | Independent oracle | Source / installed / policy | Result + evidence class |
| --- | --- | --- | --- | --- | --- | --- |
| SC-... | Tool/API/CLI chain | Explicit success or failure; side effect count | Actual | UUID/createdAt, guest exit, Provider read, receipt | Exact | PASS / I, FAIL / I, or U |

Pairwise/stateful scenarios additionally name the original ordering and competing actor/transport. Fault scenarios name the bounded injected failure and what the harness should detect. A **known-bad negative control** must be marked DETECTED or MISSED; MISSED creates a HARNESS_DEFECT and downgrades related acceptance.

## 4. Issue packet (one per finding, including SUSPECT)

```text
Issue ID: RACC-<run>-<sequence>
Class: CONFIRMED_BUG | SUSPECT/NEEDS_REPRO | UPSTREAM | ENVIRONMENT | HARNESS_DEFECT
Priority: P0 | P1 | P2
Boundary / call chain:
Original exact source SHA / installed release+digest / running versions:
Outer/backend/project / original Box UUID+createdAt / policy mode:
Actor/transport / timestamp / sanitised exact request or command:
Expected output, exit, receipt and independent remote state:
Observed output, exit, receipt and independent remote state:
Evidence location, trace/operation/task/claim ID (redacted):
Reproduction attempts and limits; why the observation may be ambiguous:
Threat to data/integrity and no-replay disposition:
Ownership: Finder -> Fixer -> independent Verifier
Next smallest safe investigation or recheck (exact command/condition):
Grant/resource/dependency needed, if blocked:
Fix commit / candidate installed artifact / original-repro verification result:
```

**Do not** put credentials, JWT, private keys, raw secret config, private DBs or unsanitized stderr in reports. An UNKNOWN operation is never converted to FAIL or retried merely to complete the table. Recovered information is appended without erasing the original uncertain observation.

## 5. Final decision record

| Metric | Required report |
| --- | --- |
| Discovered surfaces | Totals by SSH MCP, HTTP MCP, host/app CLI, Web, REST/Provider; available and blocked contexts |
| Evaluated capabilities | **Full named owner-facing functional table** and separate **every MCP/CLI leaf** black-box ledger; positive / denial / invalid / unsupported / not-executed details per action and feature, with counts only supplemental |
| Real-world depth | T fixture vs I/L installed vs H operator host vs X independent client, scoped to exact version |
| State-machine + fault coverage | Named sequences, independent oracle and injected failures, not just test counts |
| Known-bad harness check | DETECTED / MISSED / NOT RUN and implication |
| Outcomes | CONFIRMED_BUG fixed/open; SUSPECT queued; UPSTREAM; HARNESS_DEFECT; UNKNOWN preserved |
| Fix provenance | Finder original repro -> Fixer commit -> Verifier independent installed result |
| Shipping gate | Source/CI, release asset, clean install, upgrade, runtime acceptance **separately** |
| Operator exclusions | Deferred R-02 reboot, owner-host R-04 policy, original UNKNOWN, stable selector |
| Ownership and cleanup | Exact surviving test Box identities, approved cleanup evidence |
| Next-Agent packet(s) | Executable next action, evidence and blocker, no vague TODO |

Update [FEATURE_MATRIX](FEATURE_MATRIX.md), [REMEDIATION_TEST_QUEUE](REMEDIATION_TEST_QUEUE.md), [OPEN_RISKS](OPEN_RISKS.md), [PHASE_STATUS](../process/PHASE_STATUS.md), and [FINAL_HANDOFF](../process/FINAL_HANDOFF.md) **only for new evidence or work state actually changed**. This template itself does not change acceptance state, request a release or authorize host mutation.
