# PR188 nested end-to-end follow-up

Real nested fresh install and workloads PASS; live and empty retirement BLOCKED by inherited effective service policy. No positive nested reinstall claim. Separate clean-VM same-mode roundtrip CI added.

Base b8c0caf6de5e502ceabed9358ac98cf2195381dc; all five H5 workflows passed, installed lifecycle 45535512133. Disposable Box pr188-roundtrip-lab c5a60ef1-9ab0-4e05-bfe1-60a74e08b631, original Create cf11cc3b8ba5f750edfa5b9f7f1d17d6. No physical L0 changes, privileged mode or AppArmor override.

## Actual evidence

- Preflight 4e65f8c4f82214a99c567fb0514022fb exit 0: empty Ubuntu 24.04.5, nested, inherited /run/systemd/system/service.d/zzz-lxc-service.conf. Exact H5 binary SHA256 a4790be1b69cd60deb9c8df00bafee397ba4d8e17ef6df2e3bd107c61001d37f. Signed dependencies 573667bee9d8dfd60a79947f950cda75 exit 0. Private transfer grant 4f9e3efa7ac74c21c882ed6be2086585, fetch 8cba413112a304a077a22913056e98cd, revoke 811071d3bc22149fc14195e9651f34c2, listener cleanup 78071c5854652f3c95e5e87863c81d87 all succeeded.
- Native bootstrap 35318eba55583dbe63080de0de6f8157 exit 0: Incus 7.6, restricted Project, /21, nominal 2045 IPv4. Interactive Scoped install 3ae078cf60dcdab8007957901a2c3cd2 exit 0: three services active, root Doctor healthy, image copied, OAuth deferred.
- 64679828255ee8b5f86af43337547b9d exit 0: N2/N3 real create/delete, busy/replay, retention, IPv4/IPv6 directional and unrelated flows, active policy lifecycle and boot policy.
- 2fffc057272ccec5f1e824952babad2f exit 0: Forward HTTP 200; live full plan exit 2 with EFFECTIVE_DROPIN_UNREVIEWED on controller, guard and lifecycle units. Binary, manifest, drop-in and PIDs unchanged; no retirement fence/audit.
- 79b70ef9ad950b0353eef4053cefd7fb exit 1: three Peer revokes succeeded, harness incorrectly included box_id in Forward Delete. Strict schema refusal; inspected originals, no replay. Corrected cleanup 0c11eb7eb21068092ab8f50ca1464b38 exit 0: Forward and four Boxes removed via MCP, 8 retired, zero live. Empty plan exit 1, unreviewed systemd drop-ins. No native retirement effects.
- Actual new network helper 4a67c24ad476ab71d66d0c74e4701f09 exit 0: two fresh guests, actual guest HTTP, default and reverse deny (checked guest exit), grant, Forward HTTP 200, all cleanup, 10 retained identities, zero live, test token revoked and receipt removed. Retain outer lab for review.

## Changes and remaining gate

Not-found units report empty DropInPaths before installation even though type-wide policy applies later (e1c8b16c7fef914e301d6b1ce79a3b83). Inherited LXC file changes security-related settings. Installer emits advisory before fresh approval; direct uninstaller refusal identifies unit and maintenance navigation. No override, removal or whitelist of inherited policy. Verified nested retirement compatibility needs separate effective-unit design and proof; healthy runtime alone is insufficient.

15 guided and 57 uninstall tests PASS (0c3a40eb30170af1560aa68c64bfaaa4); 4 fixture guard tests plus actual preflight advisory PASS (dd622f031e5aa07d7f1d62ffda1a2b9a). Format/YAML/syntax PASS.

Clean-VM scoped-retirement lane previously ended after purge. Add actual HTTP before retirement, same-mode interactive reinstall, fresh empty ledger, denial of still-unexpired old credential, same names/keys with new UUIDs, real second workload. Preserve foreign sentinel proof, then remove only exact CI-created sentinels before second exclusive install. Incus PID unchanged. Positive reinstall remains independent CI gate, not claimed in nested C. No merge or release.

Follow-up: [real nested empty and live automatic retirement compatibility](PR188_LXC_LIVE_RETIREMENT.md). This earlier report remains historical.
