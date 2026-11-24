# S2.5 RC6 engineering checkpoint — 2026-11-15 PT

**State: PHASE ONE INCOMPLETE; disposable L0 build Box created, no RC6 and no L1 service upgrade, BH-019 security blocked.** This is a code-continuity record, not an acceptance certificate. PR #146 remains Draft.

## Code already submitted

- `670df08019d8f0a408e46340108d249d97662c29`: 13 typed new native-dotted optional inputs shared by Create/Update, concise MCP field descriptions, omission/explicit-false preservation, restricted-Project and dangerous-option fail-closed admission, native allowlisted postread, and initial negative tests. Existing `security.nesting` remains a separate native boolean.
- `3c702b99db2586f326b1923e1a2f0c020359c7be`: rename MCP input field from legacy `nesting` to exact `security.nesting` and update affected source SDK probes; Box readback `nesting` remains a status property, not a configuration key.
- `2d790728026f1bf752cde6dda4a7da07ec49cc85`: verify exactly 14 schema properties, descriptions and dotted decode; restrict `lxc.apparmor.profile` to the four documented enum strings in strict MCP JSON Schema.
- `97f4b0842d124933aea865d4783213420025c26f`: owner-approved opt-in `restricted.containers.lowlevel=allow` enables a server-generated, exact `raw.lxc` AppArmor translation for `unchanged`/`generated` and generated-profile nesting only; `unconfined` and arbitrary raw text remain denied. An inherited `unchanged` request fails unless the BoxWeave parent process actually has `(enforce)`. Unit tests cover invalid and permitted raw translations.
- `0fa072ded622f821983bbd64578b5352031725f3`, `90c5f1e792d1d3e2f1d1868de499c92730fbd9fd`: cleanup to gofmt-conformant native/control source; the latter is the latest compiled/tested code SHA in the one-use lab.
- No `INCUS_SECURITY_APPARMOR` parameter was added. Inner Incus daemon service setup remains an explicit guest-side agent task.

## Genuine security blockers, no automatic fallback

1. **Owner change 2026-11-15:** owner explicitly accepts the wider `restricted.containers.lowlevel=allow` authority for an approved Project. This affects *all* instances in that Project; it is not a per-Box grant. The current patch maps only exact allowed `unchanged`/`generated` LXC values under this flag and refuses `unconfined`/arbitrary raw text. Native per-depth enforcement and complete non-bypass isolation are **not** yet proven. The existing nine-Box L1 Project was **not** modified. For a real installed test, opt in only on the new one-use lab Project after reviewing native rights. L0 physical-host Incus Project remains unchanged.
2. An actual UID/GID lease allocator with persistent SQLite reservations, verified parent **both** ID pools, sibling/native occupancy, overflow/exhaustion and UNKNOWN lifecycle reconciliation remains to implement. The current patch declines positive explicit size/base/isolation requests until that exists. Do not construe this as implemented ID map allocation.
3. Guest-only Ubuntu AppArmor service compatibility, dynamic/link-local IPv6 adversarial isolation, positive independent full BoxWeave L5 daemon deployment and full failure-injection are not complete.

## Evidence and preservation

Prior native Incus 7.6 labs reached functional L5 with a shared enforced ancestor Profile; they do not prove per-depth independent AppArmor isolation or BoxWeave-managed recursion. Original operation IDs remain in the prior reports. A **new** L0-owned disposable build lab `rc6-dev-l1-1115`, UUID `8267272d-a1d7-412d-9a5d-dd32dfa1d6ee`, was created, original BW create `8061953d4b52d8bc7862c586768ad09d`, native `de12ef2b-f71a-4eda-9fb6-efb0ee9dd1c4`, SUCCESS. It installed Go 1.26.9, cloned PR #146 and compiled a **development-only** CLI candidate. No shared Box, existing L1 service, host/AppArmor policy, L0 Project security setting or historical UNKNOWN was changed.

As of this checkpoint, previous candidate `670df080` has PASS results for the isolated BH-018 native suite and installed HTTP MCP workflow (GitHub runs 43701862947 and 43701862896). These are regression checks, **not** BH-019 integrated acceptance. Quality/race/native option-specific and current exact-head checks are NOT YET ACCEPTED. Recheck every exact-head workflow result instead of reusing an ancestor's PASS.

## 2026-11-15 disposable build evidence (not release evidence)

- Tested source SHA: `90c5f1e792d1d3e2f1d1868de499c92730fbd9fd`, Go 1.26.9, Ubuntu 24.04.5 in new L1 build Box. All `go test -count=1 ./...` packages PASS: `cmd/boxweave`, `internal/control`, `internal/install`, `internal/native`, `internal/state`. `go build -o /root/boxweave-dev-candidate ./cmd/boxweave` exit0; `go vet ./...` exit0. Native/control option regression tests exit0; pertinent changed files gofmt clean.
- Development binary SHA256 `ad868975ac2286c66bc9966bdc77e699fc327077701e4933e18564a5762b1ffe`, compiled without release ldflags: **build-info reports `0.0.0-s1-dev`/`unverified`**, therefore **not** a traceable RC6 artifact. CLI `status` on fresh disposable 0700 SQLite dir returned `ready=false`, `REFUSE_ROOT_RUNTIME` as expected from root. No installed HTTP MCP or native managed recursion tested for this SHA.
- Original BW/native operations from this lab: compile `153e21f5845bc4020c97ec1b0d25e90e` / `80e42b54-7b63-4dd5-b19f-c4bbf3f290be`; all-Go test `64a2c69a3e46ca5f20c46fbfab00fb01` / `f93fcd76-21dd-4823-8e66-b2a3bd4e309e`; vet `4ac16d802f3ee228b397d633608dd36b` / `7d1dc254-ee1c-4cb6-b068-4fd3f1a80a29`; CLI smoke `e0dd40736ce355e1f787bf36368cae45` / `e74a5db2-ee7a-49bb-8107-b7d3019adf28`. All showed original BW SUCCEEDED and Guest exit 0, separate from full security acceptance.
- Existing L1 RC4 service retained with 9 pre-existing Boxes; no in-place upgrade, no lowlevel switch on its shared `user-1001` Project. The one-use lab is available for further isolated work.

## Next implementation gate

Finish review of all dotted input call sites and verify no legacy aliases remain; finish SQLite UID/GID allocator and actual Project-safe exact AppArmor adapter; add unit/race and native negative tests, format and compile. Only after the entire integrated feature patch passes source gates should a **new one-use L0** smoke deployment begin. RC6 must be built only from the exact source SHA accepted in that smoke and explicitly labeled test candidate. Do not widen Project authority or mark phase one PASS while either P1 blocker exists.
