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

Produce a complete row per deployed callable action from each available policy/client context, or a compact machine-generated manifest with a checked-in **summary and immutable digest**. Record the discovered total and the unable-to-discover contexts. No hardcoded claim of a fixed MCP tool or Web route count.

| ID | Surface / action | Source / installed seen? | Transport/policy | Read/write/risk | Test case ID or explicit defer reason | State / evidence |
| --- | --- | --- | --- | --- | --- | --- |
| MCP-S-001 | Actual tool name | Source S; running I confirmed/U | SSH stdio + auth/policy | Read / P1 | SC-... | PARTIAL / S,I |
| HTTP-... | Actual HTTP MCP tool | ... | Auth/client mode | ... | ... | ... |
| CLI-... | Actual installed CLI action | ... | Host/app permissions | ... | ... | ... |
| WEB-... | Actual route + method | ... | Cookie/CSRF/auth | ... | ... | ... |
| REST-... | Actual upstream interaction | ... | Authenticated Provider | ... | ... | ... |

Surface classification must cover **all** observed entries, even when not safe to execute. Entries exposed only upstream/directly should be marked **outside BoxWeave enforcement**; distinguish them from BoxWeave-owned routes. A missing catalog is a BLOCKED *discovery context*, not proof of an empty catalog.

## 3. Scenario evidence (black-box-first)

For each risk-critical family, the target is **positive + authorization/identity denial + invalid input**, or precise unavailable/unsupported reasoning. Real authenticated Web positive tests are not replaced by fixtures or a 401. Test-owned mutation requires original identity and independent readback.

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
| Evaluated capabilities | Positive / denial / invalid / unsupported / not executed counts **by family** |
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
