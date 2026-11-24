# S2 P1 — HTTP privacy, two Projects, Stop/Delete 202 ACK loss

**Latest accepted source:** `52d4f07690ef6eb15671ffabb4ba58640172375c`; candidate SHA256 `e02b0826f8eca1eca0967ce5029ad9565a80c50d85debdd3f3cddb46700a1539`. [Complete disposable real Incus 7.6 + official authenticated SDK](https://github.com/vanishs/boxweave/actions/runs/42974549026), [quality](https://github.com/vanishs/boxweave/actions/runs/42974554716), [S1/Go race](https://github.com/vanishs/boxweave/actions/runs/42974554715) all SUCCESS.

| Test | Concrete proof | Disposition |
|---|---|---|
| Authenticated MCP HTTP cache | SDK overrode early header with merely `no-cache, no-transform`; fixed in `privateMCPWriter` at WriteHeader/Write/Flush (and Vary: Authorization), now actual response contains `private, no-store`. Tested denial and streaming paths | PASS |
| All 12 S2 typed methods | Official SDK rejected bad JSON types for all 12 S2 tools; no accidental native Create | PASS for typed-invalid cases |
| Separate real Incus Project | CI admin created a separate native Project, exact same-name foreign stopped instance with distinct UUID; restricted nonroot user could not inspect it; real own-Project MCP Create→Delete completed, foreign original UUID survived unchanged and was only then explicitly removed by independent CI admin | PASS for this scenario |
| Stop accepted but 202 ACK lost | One original native Stop genuinely accepted, independent admin observed exact original UUID now STOPPED. Original UNKNOWN, receipt, no redispatch survived same SQLite restart | PASS |
| Delete accepted but 202 ACK lost | One original native Delete genuinely accepted, independent admin saw original UUID absent. Original UNKNOWN and no redispatch survived same SQLite restart | PASS |
| Status changed during cursor pagination | Go/race `TestS2CursorStatusChangeDuringPagingKeepsIdentityAndFilter`: signed cursor stays scoped while a not-yet-read original is cancelled; new query sees CANCELLED and unrelated actor sees nothing | PASS **source test only** |

**Exact CI markers:** `S2_P1_REAL_AUTHENTICATED_HTTP_PRIVATE_ZERO_TTL_CACHE_CONTROL`, `S2_P1_TWO_REAL_PROJECTS_SAME_NAME_ISOLATED_FOREIGN_UUID_PRESERVED`, `S2_P1_REAL_OFFICIAL_SDK_ALL_TWELVE_TYPED_TOOL_NEGATIVES_NO_NATIVE_EFFECT`, `S2_P1_REAL_NATIVE_STOP_202_LOST_ACK_SAFE_RESTART`, `S2_P1_REAL_NATIVE_DELETE_202_LOST_ACK_SAFE_RESTART`, `S2_P1_REAL_STOP_AND_DELETE_NATIVE_202_ACK_LOSS_NO_REPLAY`, and final `S2_NATIVE_SDK_MCP_BLACKBOX_REAL_INCUS` all =PASS.

**Failed attempts retained:** first naïve HTTP header set was overwritten by the SDK and failed actual native HTTP check; corrected by response-writer wrapper. Initial cursor-status test changed the already-visited row, not an unvisited row; corrected and passed Go race. Source formatting failures were separate from product faults. Only the exact successful source above is accepted.

**Still unverified:** older SDK versions, token-expiration *during* in-flight MCP calls, cache/cursor access with two simultaneous actors, distinct-key Create/Delete races, malformed Exec metadata and stalled output I/O, explicit Guest Ready contract, custom pool alternatives, public online WAN baseline and actual power outage. **S4 peer/forward-dependent Delete is outside S2.** No shared Incus, historical UNKNOWN, protected v1 merge or production deployment occurred.
