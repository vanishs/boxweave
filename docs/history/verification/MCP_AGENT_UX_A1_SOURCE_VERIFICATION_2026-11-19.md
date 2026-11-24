# BoxWeave v1 — MCP Agent UX A1 source verification (2026-11-19)

**Verdict: `A1_SOURCE_AND_OFFICIAL_SDK_PASS`; `REAL_CHATGPT_BEHAVIOR_NOT_TESTED`; `A2_INTERRUPTION_RECOVERY_NOT_TESTED`; `ACTIVITY_B_NOT_IMPLEMENTED`.**

Repository: [BoxWeave Draft PR #153](https://github.com/vanishs/boxweave/pull/153). Source with final A1 code+test changes: [`8b1b8a8a40d4972b5bb51a5dfcc98c7d80f5c34b`](https://github.com/vanishs/boxweave/commit/8b1b8a8a40d4972b5bb51a5dfcc98c7d80f5c34b); implementation code first committed at [`6335fbae`](https://github.com/vanishs/boxweave/commit/6335fbaea9683440a131e9b1b4ac63255d5dbf4b). The independent [design/plan](../development/V1_MCP_AGENT_UX_AND_ACTIVITY_DESIGN_2026-11-18.md) controls remaining scope. This report is not an S4/S5 native, release, or ChatGPT behavior signoff.

## Changed behavior (presentation only)

- The pinned official MCP Go SDK `v1.8.0` supports `ServerOptions.Instructions` and places the text in `InitializeResult.Instructions`; `internal/control/server.go` now configures it from `serverContinuityInstructions`. The first 493-character paragraph covers owned Project, durable asynchronous Operations, original idempotency keys, `operation_list/get`, UNKNOWN safety, and the distinction between Operation receipts and Agent task completion.
- Secondary instructions clarify untrusted work/agent labels, independent Actor idempotency, `operation_output_get` guest exit status, `box_get` native Running vs guest readiness, and the absence of built-in Agent plans/Git checkpoints.
- Targeted Tool descriptions now lead with their actual purpose. The nine mutating Tools retain the same shared fallback tail, reduced from 294 to 127 literal characters, without changing typed input schemas or native safety boundaries. The public Tool inventory remains the same 17 actual tools on this candidate (not a claim about future S4-C).
- Strict-schema rejection, authenticated read paths, operation journal, original native dispatch, OAuth, resource locks, Project confinement, and SQLite Schema v4 are unchanged by A1.

## Reproducible source and test evidence

- [Required quality gate #44558082579](https://github.com/vanishs/boxweave/actions/runs/44558082579) for exact source `8b1b8a8a`: **SUCCESS**, including format, YAML, whitespace, module checks, `go vet`, complete `go test -race -count=1 ./...` and Go build steps.
- `TestOfficialMCPHandshakeAndS2ReadTools` uses a real official SDK HTTP client against a bearer-protected local MCP test server, checks the exact initialize Instructions, first-paragraph length/content, original 17-tool inventory, and exactly nine mutating Tool descriptions with compact hints.
- `TestS2AllPublicToolSchemasRejectUnknownArguments` still proves the strict public input schemas and invalid-project-style arguments are refused; its recovery-hint assertions now correctly check the six shared safeguards in the **initialize Instructions**, and the compact fallback in **tools/list** separately.
- Original [quality run #44557179195](https://github.com/vanishs/boxweave/actions/runs/44557179195) on code commit `6335fbae` **FAILED** because an older test required the six repeated recovery words on *every* mutating Tool Description. The targeted test-only correction was committed as `8b1b8a8a`; subsequent full Race passed. Preserve this chronology, do not relabel the original failed run.
- Static literal count (not tokenizer measurement): the nine hints save `(294 - 127) × 9 = 1,503` characters; seven revised core descriptions save a net additional `130` characters. Approximate Tool-description reduction `1,633` characters, ignoring JSON/tool schema encoding. Server Instructions add an initialization field of roughly `1,040` characters. **Do not treat character counts as actual LLM tokens or proven model uptake.**

## Remaining independent gates

1. **A2 realistic single-Agent original-receipt recovery:** test one authenticated Agent client after lost MCP reply/reconnect, server restart, original UNKNOWN, old work_ref, and a nonzero guest Exec exit. Verify exact original Operation/Native ID, no blind new-key replay, honest missing evidence, revoked Token refusal; use disposable authorized native resources.
2. **Client/model behavior and token budget:** compare prior descriptions, compact-only, and compact-plus-Instructions on actual clients. Measure full `tools/list`, actual tokenizer Tokens, call count/time and false retries. Confirm whether real ChatGPT/Codex receives/uses initialization Instructions; SDK return alone is insufficient.
3. **Activity B:** owner decides if `box_activity_get` becomes another public Tool; then independently build/verify full `overview/issues/evidence`, bounded old UNKNOWN and blocker visibility, cross-Layer authorization, and pre-return revocation checks. Database changes only if query data proves necessity.
4. **Later S5:** TC-015/016 full independent A→B task handoff and checkpoint conflict detection remain explicitly separate. Stage-specific full native and source-exact final release gates remain unchanged.

**Safety:** no shared L0–L5 host modification, no RC publishing, no database or installed OAuth-client reset under this A1 source change.
