# S2.5 RC6 engineering checkpoint — 2026-11-15 PT

**State: PHASE ONE INCOMPLETE, NO RC6, NO L0 deployment, BH-019 security blocked.** This is a code-continuity record, not an acceptance certificate. PR #146 remains Draft.

## Code already submitted

- `670df08019d8f0a408e46340108d249d97662c29`: 13 typed new native-dotted optional inputs shared by Create/Update, concise MCP field descriptions, omission/explicit-false preservation, restricted-Project and dangerous-option fail-closed admission, native allowlisted postread, and initial negative tests. Existing `security.nesting` remains a separate native boolean.
- `3c702b99db2586f326b1923e1a2f0c020359c7be`: rename MCP input field from legacy `nesting` to exact `security.nesting` and update affected source SDK probes; Box readback `nesting` remains a status property, not a configuration key.
- `2d790728026f1bf752cde6dda4a7da07ec49cc85`: verify exactly 14 schema properties, descriptions and dotted decode; restrict `lxc.apparmor.profile` to the four documented enum strings in strict MCP JSON Schema.
- No `INCUS_SECURITY_APPARMOR` parameter was added. Inner Incus daemon service setup remains an explicit guest-side agent task.

## Genuine security blockers, no automatic fallback

1. Official Incus Project documentation confirms that `restricted.containers.lowlevel=allow` grants general `raw.lxc`, `raw.idmap`, `volatile.*` authority rather than permitting just the exact audited `lxc.apparmor.profile = unchanged` line. Existing BoxWeave's restricted Project and `SafePerUserProfileConfig` forbid arbitrary raw config. Enabling general low-level admission contradicts the owner's security constraint. A narrowly scoped, independently reviewed trusted adapter or upstream Incus capability with exact per-key admission is necessary before positive managed-recursion tests. The current code intentionally denies `unchanged`/`generated`/unconfined requests rather than accepting but failing to enforce the requested profile.
2. An actual UID/GID lease allocator with persistent SQLite reservations, verified parent **both** ID pools, sibling/native occupancy, overflow/exhaustion and UNKNOWN lifecycle reconciliation remains to implement. The current patch declines positive explicit size/base/isolation requests until that exists. Do not construe this as implemented ID map allocation.
3. Guest-only Ubuntu AppArmor service compatibility, dynamic/link-local IPv6 adversarial isolation, positive independent full BoxWeave L5 daemon deployment and full failure-injection are not complete.

## Evidence and preservation

Prior native Incus 7.6 labs reached functional L5 with a shared enforced ancestor Profile; they do not prove per-depth independent AppArmor isolation or BoxWeave-managed recursion. Their original operation IDs are kept intact in the existing L5 and comprehensive lab reports. No new L0 Box or native Incus operations were created during this code pass; L0 `host_status` was read-only. No shared Boxes, host AppArmor or historical UNKNOWN were changed.

As of this checkpoint, previous candidate `670df080` has PASS results for the isolated BH-018 native suite and installed HTTP MCP workflow (GitHub runs 43701862947 and 43701862896). These are regression checks, **not** BH-019 integrated acceptance. Quality/race/native option-specific and current exact-head checks are NOT YET ACCEPTED. Recheck every exact-head workflow result instead of reusing an ancestor's PASS.

## Next implementation gate

Finish review of all dotted input call sites and verify no legacy aliases remain; finish SQLite UID/GID allocator and actual Project-safe exact AppArmor adapter; add unit/race and native negative tests, format and compile. Only after the entire integrated feature patch passes source gates should a **new one-use L0** smoke deployment begin. RC6 must be built only from the exact source SHA accepted in that smoke and explicitly labeled test candidate. Do not widen Project authority or mark phase one PASS while either P1 blocker exists.
