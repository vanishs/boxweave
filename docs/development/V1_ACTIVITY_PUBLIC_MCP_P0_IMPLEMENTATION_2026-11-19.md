# BoxWeave v1 — public Activity P0 implementation and verification (2026-11-19)

**Status: IMPLEMENTED / Go + official MCP SDK + CI SOURCE PASS on Draft PR #153.** No merge, public RC, installed Host deployment or real ChatGPT post-interruption behavioral acceptance follows from this report.

**Pinned last production/test code source:** `378bd4d6d74058c27b081e939ce9744aeaf84bff`. Documentation-only commits after this SHA do not constitute new native-code evidence. [Approved product exception](../decisions/PRODUCT_BOUNDARY.md) · [Source lineage audit](V1_MCP_TO_DB_LOG_LINEAGE_SOURCE_AUDIT_AND_IMPLEMENTATION_DESIGN_2026-11-19.md).

## Delivered

1. **One new public read-only `box_activity_get` Tool** — registered by `internal/control/server.go`, typed handler in `internal/control/activity_tools.go`, closed JSON Schema with the three valid `view` values, read-only annotation and short recovery guidance in MCP Instructions. Current official SDK Tool list contains **18**, including the unchanged previous 17. The full planned catalog, after S4-B4–B6 and the approved S4-C readiness Tool, is **22**, **not** 22 implemented.
2. **`overview` and `issues` signed keyset pages** — `internal/state/activity_query.go` uses a private SQLite cursor secret, authenticated Layer + TokenID + view/filters/limit binding and initial admission rowid watermark. `next_cursor` can be followed safely without changing scope. Each page is a new SQLite read-only transaction; changing network rows between pages is not an immutable database snapshot. Response truthfully exposes `truncated` and `attention_complete=false`.
3. **Complete current SQLite candidate families for `issues`:** unresolved admitted/dispatch/running/verifying/UNKNOWN originals, including Root Peer Project barrier unaffected by generic recovery RELEASED; pending Exec GC deletion or REVIEW_REQUIRED; confirmed/unresolved directional Peer; non-RELEASED network address fences; non-REVOKED forward claims. Box UUID filtering checks primary Box and multi-resource claim targets plus network source/target. Current network dependencies are distinguished from actual failed Operations. Labels are untrusted narrowers, never authentication.
4. **`evidence`** — exact original Receipt, ordered safe `operation_events`, bounded reviewer decisions without evidence digest, resource claims, Exec exit/GC metadata, ownership state and latest verified Peer original. No raw Root journal, arbitrary guest command/env/file bytes/stdout, unjournaled reads or external Git/Agent plan.
5. **Real Incus Box UUID acceptance** — `box_id` accepts canonical hyphenated UUIDs and the existing 32-hex operation fixture format; malformed arbitrary IDs are rejected. All public reads start from Bearer principal's configured Layer and use fresh TokenID liveness checks before/after SQLite query and again at controller return boundary.

**Schema v5** peer `last_verified_operation_id` and operation evidence indexes were implemented in the preceding stage; the new public Activity P0 does not add another database version or alter existing mutation behavior.

## Evidence and test status

| Verification | Source/evidence | Status |
|---|---|---|
| Public official MCP Go SDK `tools/list`, 18 tools, read-only metadata | `internal/control/server_test.go`, `operation_tools_test.go`, `activity_tools_test.go` | **PASS** |
| Authenticated SDK `overview/issues/evidence`, separate same-Layer Actor, invalid view/limit/unknown inputs | `TestActivityOfficialMCPThreeViewsAndSameLayerTakeover` | **PASS** |
| SQLite historical UNKNOWN, released generic vs Root Peer UNKNOWN, event order and evidence safe omission | `internal/state/activity_query_test.go` | **PASS** |
| Address fence, confirmed Peer and Forward dependencies, real hyphen UUID, foreign Project rejection | `internal/state/activity_query_pagination_test.go` | **PASS** |
| Keyset pagination, same-time resource keys, HMAC tampering, cross-Layer/filter cursor refusal and admission watermark | `internal/state/activity_query_pagination_test.go` | **PASS** |
| Full `go test ./internal/state ./internal/control -count=1`, Go 1.26.9 / CGO, disposable L0-owned Box | Code `378bd4d6` | **PASS** |
| Full `go test -race`, gofmt, vet, module validation and Go build | [required quality #44588051079](https://github.com/vanishs/boxweave/actions/runs/44588051079) on `378bd4d6` | **PASS** |
| amd64, arm64 native builds and source manifest (without RC publication) | [candidate build #44588051025](https://github.com/vanishs/boxweave/actions/runs/44588051025) on `378bd4d6` | **PASS** |

**Not accepted / next stage:** actual ChatGPT seeing/using the new public Tool on a deployed Schema v5 controller; one real Agent losing a response and recovering an original/UNKNOWN; post-deploy OAuth grant refresh/reauthorization/revoke races; adversarial large-history `issues` query-plan benchmarks; independent exact-source live S4-B3 native/Root kernel confirmation; later full S5 two-Agent TC-015/016. SQLite `attention_complete=false` is a deliberate truthful boundary, **not** a hidden PASS for every physical/native dependency. The fixed same-Project Activity projection is not a persistent task journal or safe permission to replay UNKNOWN.

## Compatibility / deployment gate

**Current installed Schema v4 SQLite is incompatible with Schema v5 and must fail closed.** There is no automatic migration. Do not attach new binary to existing live controller DB expecting credentials, OAuth grants, connectors, Boxes, unknown originals or Root journal evidence to transfer. Keep the original receipts and native state protected; any fresh Schema v5 controller/credentials/OAuth/ChatGPT connection setup and disposable Project reset require separate approved safe-environment scope. Read [explicit step-by-step rebuild warning](V1_ACTIVITY_LINEAGE_IMPLEMENTATION_AND_REBUILD_NOTICE_2026-11-19.md).

The work was confined to the existing Draft PR #153 and an isolated owned development Box. No PR merge, RC release or shared physical Host/Root/PRS modification is authorized here.

## Disposable resource cleanup and separate native failure

The **only** development Box created for this P0 campaign was `36e79888-4f9e-4afd-a22d-058f95f071c9`, under `work_ref=activity-p0-20261119`. Its stop original `81f8339d48343b43d873a5e12b1359e4` and delete original `c2e97f5d7d0b0449f9f5613bd4caa67f` both reached **SUCCEEDED**. A following authenticated `box_list` returned **zero Boxes**. No other native Host/Root/PRS resources were intentionally modified.

A **separate** [S4-A installed native workflow #44588050791](https://github.com/vanishs/boxweave/actions/runs/44588050791) on the exact code source `378bd4d6` ended **FAIL/NOT_SIGNED** after reporting multiple earlier native PASS checkpoints; the summarized GitHub step ends at `S4_A_INSTALLED_REAL_NATIVE_MCP=FAIL_OR_BLOCKED` without independently identifying the root failure. Its predecessor also failed before Activity P0, so **do not attribute it to Activity or dismiss it as flaky without isolated investigation**. Neither Go/SDK Activity PASS nor S4-B2 prior PASS overrides this separate S4-A signoff gate.
