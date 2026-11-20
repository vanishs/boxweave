# S5A1 — diagnostic implementation and acceptance checkpoint

Status: PARTIAL; development and scoped source tests, not installed-native signoff.

## Historical #151
Run [43776449453 attempt 1](https://github.com/vanishs/boxweave/actions/runs/43776449453/job/133003716492), exact original code 9daa7006d8d172eaac955d1a2aec583511014144, reported a newly admitted fresh readback a529cc5a3bb2f64714455d0612161a80 after successful original 126/127 receipts, service restart, reconnect and original replay. The verifier exited 2, class OTHER_VERIFIER_ASSERTION. No terminal/native ID or SQLite snapshot for this fresh original remains in the fetched log; run artifact list empty. Original outcome remains unclassified/evidence unavailable, not a diagnosed transport fault or a proven product defect. Attempt 2 success does not erase this. #151 remains OPEN and release-blocking absent reviewed resolution. Never replay it.

## Implementation
Additive operation_get diagnostic: SQLite durable evidence source, bounded wait outcome, recorded native ID presence, safe cause domain, verified Guest exit and recorded-output metadata when original Exec SUCCEEDED. No raw paths, live scan, mutation, schema change, or inference of available bytes. Worker preserves fixed Exec observation/deadline/missing receipt, evidence and SQLite-finalization classifications as UNKNOWN with existing locks. Probe distinguishes client-call errors, tool rejection and malformed reply; emits only fixed classes and original identifiers. Independent native witness compares each fresh post-restart readback against its own SQLite original and native terminal using the existing isolated event observer.

## Evidence
Development Box 11bbc73e-2bce-4cf9-857f-799f768305f8; Create 2ed7ddf52d1e8846330b182def75ba1f / native 6816e4c7-d8df-4fd7-a98a-c2edd80aee2a. Clone/toolchain bootstrap 9bddfb8317b883a065fe98b5468a5d53 guest exit 0. A genuine nonzero guest exit 127 during initial clone attempt (git unavailable) retained a SUCCEEDED receipt and separate exit metadata; this does not classify #151. Earlier short keys rejected before dispatch; corrected keys used for new development work.

Focused Go tests and format/diff checks original 66c277f9f1628c70866ceb9758b8639c / native ddb18dd6-c693-4d7c-ad93-4725d38ba423 guest exit 0. Expanded BH018/S4 OAuth and diagnostic regression plus verifier build 5231e5719226716bfa6dcc398b0c2495 / native b84369a4-5cd5-4fab-abb4-ae323fd3ca97 guest exit 0. Final shell syntax/diff checks 5df6e28d01f2c2a3be887eb233f293fd / native 002bec35-c3ea-488b-82ed-133e49c77cc0 exit 0. Source files transferred with matching Git blob hashes. The isolated native workflow runs two fresh CI runners, preserving distinct evidence; results pending at this checkpoint. No physical host install or restart. Preserve development Box.

#148 stable grant remains source-implemented; exact Box HTTP tests passed, real ChatGPT refresh/revoke trace remains separate. SDK v1.8.0 supports new and legacy elicitation, but current stateless HTTP connector does not expose negotiated capability or verifiable human approval assertions to this agent; actual remote feasibility unproved. No A3 mechanism selected and no high-risk authorization enabled.

Next: inspect both independent native runs; classify failures without replay, review the scoped SDK feasibility substitute below and #151 evidence-loss disposition before A1 PASS. A2–A6 not started.

## A1 approval feasibility — Box substitute acceptance, 2026-11-20

Owner directed an alternative verification inside the development Box, using an independent programmatic client. No A3 architecture is selected. The test-only `TestS5A1ElicitationFeasibility` uses the exact pinned Go MCP SDK v1.8.0, local real HTTP StreamableTransport, inert tool and response, no actual credentials or Box/host mutation. For the current server topology (stateless, JSON response, no propagated request cancellation), it checks both protocol `2026-07-28` and `2025-11-25`; stateful HTTP is a positive control, not a proposed runtime change.

| Scenario | Observed transport result |
| --- | --- |
| 2026-07-28, stateless, form, handler present | available |
| 2026-07-28, stateless, URL, handler present | available |
| 2026-07-28, stateless, form, no handler | unavailable |
| 2025-11-25, stateless, form, handler present | unavailable |
| 2025-11-25, stateless, URL, handler present | unavailable |
| 2025-11-25, stateful, form, handler present | available |
| 2025-11-25, stateful, URL, handler present | available |

All available cases returned an entirely program-generated accept, with no human presence or approval assertion. URL accept occurred without visiting the inert URL. This is direct negative evidence against treating elicitation accept itself or a client capability as independently verifiable consent. The unavailable cases are asserted, not passed by suppressing unexpected errors. This covers SDK/transport feasibility and trust-boundary review for A1; it does NOT establish ChatGPT-specific negotiated capabilities or user UI behavior. Actual supported client/user proof remains A3/B4, and a new approval cannot be enabled until its independent authorization gate passes.

Box `11bbc73e-2bce-4cf9-857f-799f768305f8`; exact production baseline `259db881715755878b71273d998d4729c986d2d2`; initial test original `72cbb0b4e01ba9aac8e08673b1418824` / native `db45a8ef-7fba-497c-9620-8e86dddaa637`, guest exit 0. Strict expected-result rerun three times, 21/21: original `8ad348326942bd88a273bbd0e0c4c846` / native `f4a951d2-a94e-4710-9f10-a75f5e59853c`, guest exit 0. Test blob `f302add5f0e79173b7765d0c0a3bc53846487203`. No production code changed in this follow-up.

Independent installed native matrix [44811599102](https://github.com/vanishs/boxweave/actions/runs/44811599102), source `259db881715755878b71273d998d4729c986d2d2`: runner 1 PASS; fresh 126 readback BW 31211ef7831f945926e2dfac20c3e0bd / native abbf6dea-e707-46de-a9e7-7f70786897c4; fresh 127 readback BW 98ea4a6f0dc9cd47e66790450e8918a7 / native fff9008d-e1c4-40ce-ae87-35a6e2101c05. Both independent SQLite/native witnesses PASS. Runner 2 in progress at this observation. The earlier #151 original remains unclassified; A1 overall remains PARTIAL.
