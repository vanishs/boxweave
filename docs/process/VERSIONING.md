# v1 versioning and publication

**Version:** `1.0.0`, recorded in the root [VERSION](../../VERSION) file.

## Owner-defined scheme

Use `1.X.Y` where:

- `1` is the major line for this from-scratch BoxWeave design.
- `X` is a feature increment. Completing a new feature slice increments X by 1 and resets Y to 0 (`1.0.0` -> `1.1.0`, or `1.3.2` -> `1.4.0`).
- `Y` is a patch/fix increment for existing functionality (`1.3.0` -> `1.3.1`).
- Do not bump a version for an unimplemented promise. One coherent release contains verified changes and the corresponding docs/acceptance state.

## This initial branch

`v1` contains **only the new specification**, no inherited v0.x runtime, build, updater or tests. Therefore `1.0.0` is the **source design baseline**, not a claim that a production-ready 1.0.0 executable exists. No `v1.0.0` Git tag, functional release assets, stable installer pointer update or installed acceptance is justified by these **design-only** documents. When an actual verified v1.0.0 application artifact is first available, the branch CI may automatically issue the initial `v1.0.0` Release if the immutable tag has not been published yet.

## Later releases

- Record exact source revision, immutable build/artifact identities and SHA-256, current version, change scope, native integration tests, denial tests and known risks.
- CI should build and verify reproducibly and automatically publish a new immutable tag/Release for a successful build on `v1` with an unpublished VERSION and exact checked assets. **A successful release CI gate must include real candidate-installed native/nested regression evidence and independent verification of exact candidate artifact hashes**, not merely a unit test pass or an untrusted `PASS` marker. A GitHub Release does not imply a running host was upgraded.
- Treat Git tag/asset publication as immutable. Never retarget a published tag or replace published bytes silently.
- Feature and patch increments follow the owner's `1.X.Y` policy, with a separate decision for any future major version change.
- **Accepted owner policy:** for a verified new-version candidate in `v1` with real deliverables and green CI, publish the exact immutable GitHub source release automatically, without requesting release-by-release approval. The immutable tag/release must not be recreated for an unchanged VERSION.
- Explicit operator approval is still needed to promote an installer stable selector or deploy/change a real host.
- Source artifacts and installed Go binaries must report the same version. Pin one version source of truth and include the exact dependency manifest (Go toolchain/MCP SDK/Incus SDK versions, observed runtime Incus and required API extension baseline) in release assets and local installation ledger. Do not conflate CI auto-publication with host dependency auto-upgrades during a separately initiated local install/update.

## Candidate -> native acceptance -> automatic formal release

For a bug in an actually installed first `1.0.0` release, the correction is normally `1.0.1` (Y patch) rather than a new feature X. This does **not** authorize tagging today's documents-only VERSION.

1. Before formal publication, produce an **installable test candidate** from an exact commit with VERSION, binary digests and dependency manifest. It may come from CI candidate/test artifacts or locally built, independently verified bytes; it must **not require an already published GitHub Release** to be installed in an isolated nested Box.
2. Reproduce the original native failure on the actual known-bad baseline where safe, then install the candidate **inside** a separate nesting-capable Box as an independent controller. Test the original mutation with dependencies, HTTP/real host-port behavior, native delete post-state, injected mid-delete fault/UNKNOWN recovery and all changed security boundaries. Use a guest-local MCP test client or separately operator-configured transport to reach an inner localhost MCP; no automatic cross-layer management API.
3. An **independent Verifier** checks original reproduction and fix, exact source SHA and the **exact binary bytes** proposed for publishing, actual Incus/OS capabilities and negative side-effect evidence. CI must validate trustworthy real-test evidence bound to the candidate commit/digest (or execute those tests on a suitable isolated native/nested runner); do not accept an unverified user-supplied `passed: true` report. If mandatory nested/native tests cannot run, mark BLOCKED and **do not auto-publish a formal version**.
4. **Only after** this expanded CI release gate is green does the owner's standing **automatic tag/GitHub Release** rule apply. Publish the verified immutable artifact/digest once, with traceable acceptance evidence and known limitations. Neither publication nor verification upgrades the first-version L0 host or repairs its pre-existing failed operations automatically.
5. The physical host's administrator may **separately** upgrade L0 after formal publication, verify the installed digest, then safely reconcile the original failed Box/delete/forward under its original Operation evidence. No installer stable pointer is moved implicitly.

Use [fault-driven scenario](../verification/FAILURE_DRIVEN_RELEASE_SCENARIOS.md) for detailed stages, baseline/candidate distinction and simulation versus actual evidence. Real native acceptance is part of the **release gate**, not post-release work.

## Release metadata and workflow input safety

The tag/version naming path is a security boundary: validate VERSION/tag input before release, carry externally influenced branch/tag strings into workflow steps as **data** (e.g. fixed environment variables), not unescaped source fragments inside a generated shell script. The published assets must report the same tagged VERSION and exact build SHA/digest as the independently verified candidate. Keep source, binary and release identities aligned even for documentation-only changes; **do not publish** the design-only `1.0.0` baseline. This hardens the existing release workflow without adding a signing/orchestration platform. See [Containarium incident review #1602/#549](../research/CONTAINARIUM_MAIN_HISTORY_2026-11-07.md).

## Branch policy

`main` remains the v0.x development line until the owner instructs otherwise. `v1` is independent in **contents**, using an entirely new source tree. Preserve old work in `main`; do not merge or automatically migrate its implementation into `v1`.

Use scoped feature branches/PRs when concurrent Agent work may conflict, and preserve the current branch's review/CI policy. No workflow in the v1 design-only branch should auto-publish a source release just because `VERSION` exists.
