# PR #178 — Root native default-Project ACL provenance correction (2026-11-23)

**STATUS: EXACT INCUS 7.6 NATIVE EVIDENCE + STRICT READ-ONLY VALIDATOR CORRECTION. No network writer, migration or MCP Root feature is enabled.**

## Real cause and experiment

The earlier `NativePeerGuardValidateScopedInventory` test modeled both the root ACL *object* and the tenant Box/UsedBy as belonging to `user-1001`. That model was wrong in the actual root-native ACL architecture.

In the disposable inner Incus 7.6 test Box `pr178-lab-1123`, operation `8ec33e789409c326b02b72514a85bfc0` (guest exit **0**) created an **empty temporary Project and an independent ACL**, attached the root-created ACL to that Project's `default` profile, queried root Incus's actual JSON, then cleaned up the temporary device/Project/ACL by the shell trap. **No new running guest was created; the four existing A/B/C/D remained unchanged.**

Observed:
- The ACL's `Project` is **`default`**, even though it is consumed by a tenant Project.
- Its exact `UsedBy` is `/1.0/profiles/default?project=pr178-guard-probe` (tenant-scoped reference).
- The bridge network `Project` is **`default`**, and `Network.UsedBy` includes `/1.0/profiles/default` (root default profile), tenant references with a canonical `?project=`, and exact native Instance references. This is *expected for this shared experimental bridge only*. A **dedicated** production bridge must continue to refuse foreign/root default profile consumers; do not indiscriminately whitelist them.
- In the earlier all-default Project A/B/C/D fixture, ACL `UsedBy` lacks any Project query because those instances truly belonged to `default`. It is NOT evidence that tenant-owned Instance `UsedBy` lacks `?project=user-1001`. Tenant Instance references must be independently tested later.
- The first probe command `07996f50531e276b10388692af417d4a` had Guest exit 1 due solely to an incorrect `incus query` JSON envelope assumption (`metadata` nested vs direct). The corrected read-only query `f8f4e0d8f5d911d88c85526643783f4f` exited 0; no network mutation occurred there.

## Corrected Root inventory validator and tests

- `internal/install/network_peer_scoped_inventory.go`: require exact native ACL `Project=="default"` (root-managed ACL), independent **tenant** Instance identity/Project, exclusive one exact tenant Instance `ACL.UsedBy`, trusted native ETag, and the complete root network `UsedBy` whitelist. No inference from ACL name or Project alone.
- Require **canonical URL** `RawQuery=="project="+url.QueryEscape(project)`, rejecting alternate encodings that parse to the same Project and rejecting opaque/userinfo/raw-path/extra query/foreign Project.
- `internal/install/network_peer_scoped_inventory_test.go`: positive root-default ACL+tenant-Instance projection; negative native ACL incorrectly put in tenant Project, noncanonical `%75ser-1001` alias and other 20+ foreign/identity/lease/ACL cases. Targeted `5ac5e8c26d3db284a54c6f52b26ce591` SUCCEEDED, Guest exit **0**.
- Complete isolated Box `make verify` `165e02d1a31c8e471d6d2fd9c16e2815` SUCCEEDED, **Guest exit 0**. Includes Go race, state, native, vet, mod, Python smoke, uninstall and stable gate. Temporary Project and ACL absence was checked before this full run.
- Exact commit GitHub CI must be checked separately before signing release checks.

## What remains BLOCKED

This is a necessary correction for *future* root-native collection, not an installed dispatcher. Still no per-Box ACL migration, protected original Root actor/Token read, root-owned exact socket mediation, secure guard boot order, A/B-only conntrack in the full Operation, no-replay complete SQLite journal, C/D all-protocol/Forward and Host reboot tests. Do not loosen Root `Network.UsedBy` whitelist to allow foreign default Profile on the owner-dedicated bridge; that would widen the attack surface. No changes to L0 Host, #177 Doctor or v1 release. PR #178 stays Draft.
