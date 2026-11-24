# S2 P1 protocol, identity and same-key Create race — exact black-box evidence

**Verified source:** `36336403c03338553cac2497fc5e386f3060c60b`; compiled binary SHA256 `f14a1e5772f8f973309e3b1f78b18466b6949c49c49fc39ecd4d55d938c9e7c7`. [Pinned real Incus 7.6 and official Go MCP SDK CI](https://github.com/vanishs/boxweave/actions/runs/42967338261), [required quality](https://github.com/vanishs/boxweave/actions/runs/42967343347), [S1/race](https://github.com/vanishs/boxweave/actions/runs/42967343390) SUCCESS.

| Case | Real installed/SDK evidence | Result |
|---|---|---|
| TC-001/002 | Malformed `_meta`: work map, agent boolean, newline/control redaction sentinel, untrusted URL, and contradiction against legacy work argument all rejected by real MCP HTTP; receiving middleware log omitted the sentinel | PASS for tested negatives |
| TC-006/008 | Viewer-provided fake admin work/agent `_meta` could not create a Box; invalid CPU JSON type rejected; scoped `box_list` remained empty before the normal lifecycle | PASS for tested negatives |
| TC-007 | No-metadata authenticated legacy JSON TextContent and StructuredContent returned the same original durable `operation_id` | PASS for tested fallback |
| TC-004 / A-007 | Twelve independent official MCP SDK sessions concurrently replayed the identical actor/key/intent for a real Create; all received one original `operation_id`, and the full native lifecycle created the expected single UUID | PASS for same-key replay |

Required CI markers: `S2_P1_REAL_SDK_HOSTILE_META_VIEWER_AND_SCHEMA_NO_NATIVE_EFFECT=PASS`, `S2_P1_REAL_SDK_LEGACY_TEXT_AND_STRUCTURED_RECEIPT_SAME_ID=PASS`, `S2_P1_NATIVE_TWELVE_INDEPENDENT_SDK_CREATE_SAME_KEY_ONE_RECEIPT=PASS`, `S2_P1_INSTALLED_RECEIVING_MIDDLEWARE_REDACTION_AND_TYPED_NEGATIVES=PASS`, and final `S2_NATIVE_SDK_MCP_BLACKBOX_REAL_INCUS=PASS`.

**Boundary:** This does NOT prove real second Incus Project separation, a different-key Create/Delete race, Stop/Delete native lost ACK, older-version SDK compatibility across every tool, cross-token cache/expiry in-flight races, complete schema fuzz, WAN online positive control, explicit Guest readiness or physical hardware power loss. Those TC/A cases remain PARTIAL. This is exclusively disposable CI; historical UNKNOWN, protected `v1`, shared Incus and production were untouched.

**Failure history:** Initial new probe commit failed gofmt only; corrected with isolated CI formatter. `3b3434c` and `3633640` then passed full real-native black-box tests. Formatter failure was not a product-authentication defect.
