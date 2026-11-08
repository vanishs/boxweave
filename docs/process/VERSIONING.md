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
- CI should build and verify reproducibly and automatically publish a new immutable tag/Release for a successful build on `v1` with an unpublished VERSION and exact checked assets; a source-CI Release does not imply full installed acceptance.
- Treat Git tag/asset publication as immutable. Never retarget a published tag or replace published bytes silently.
- Feature and patch increments follow the owner's `1.X.Y` policy, with a separate decision for any future major version change.
- **Accepted owner policy:** for a verified new-version candidate in `v1` with real deliverables and green CI, publish the exact immutable GitHub source release automatically, without requesting release-by-release approval. The immutable tag/release must not be recreated for an unchanged VERSION.
- Explicit operator approval is still needed to promote an installer stable selector or deploy/change a real host.
- Source artifacts and installed Go binaries must report the same version. Pin one version source of truth and include the exact dependency manifest (Go toolchain/MCP SDK/Incus SDK versions, observed runtime Incus and required API extension baseline) in release assets and local installation ledger. Do not conflate CI auto-publication with host dependency auto-upgrades during a separately initiated local install/update.

## Branch policy

`main` remains the v0.x development line until the owner instructs otherwise. `v1` is independent in **contents**, using an entirely new source tree. Preserve old work in `main`; do not merge or automatically migrate its implementation into `v1`.

Use scoped feature branches/PRs when concurrent Agent work may conflict, and preserve the current branch's review/CI policy. No workflow in the v1 design-only branch should auto-publish a source release just because `VERSION` exists.
