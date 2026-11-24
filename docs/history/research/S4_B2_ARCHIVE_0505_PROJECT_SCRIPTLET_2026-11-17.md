# S4-B2 - Archive v0.50.5 versus Incus Project and Scriptlet authorization

**2026-11-17 UTC.** Historical source inspection and a disposable L0-only pure scope-function test. This report is not installed B2 authorization acceptance. **Stage: `S4_B2_PARTIAL`, B3 not started.**

## Proven historical scope

- Exact read-only historical ref: [`archive/0.50.5@306589cb6bf49c8c819f6cebeaaa618f0087f17b`](https://github.com/vanishs/boxweave/tree/306589cb6bf49c8c819f6cebeaaa618f0087f17b). Clone was a separate Python/Containarium/SSH application generation, not the v1 official Go Incus SDK implementation.
- Full tracked-file `git grep -in` over that ref found **zero** matches for `scriptlet`, `tls-restricted`, `authorization.client`, `openfga`, `Starlark`, `NetworkACL` and `network_acls`. This ref therefore offers **no direct Scriptlet implementation or native Scriptlet/Project coexistence acceptance**. Do not reinterpret lack of results as proof about other historical branches.
- Reusable *design principle*: [`bwmini/coordination_runtime.py`](https://github.com/vanishs/boxweave/blob/306589cb6bf49c8c819f6cebeaaa618f0087f17b/bwmini/coordination_runtime.py) implements `scope_digest(provider, backend, project)` and `canonical_box_identity` from **controller-verified**, not caller-provided, scope. A pure Python call in disposable L0 confirmed identical provider/backend with different project strings yields different digests. This is a metadata identity fence, **not** TLS authorization, native UUID proof or cross-Project network isolation.
- Legacy [`bwmini/host_cli.py`](https://github.com/vanishs/boxweave/blob/306589cb6bf49c8c819f6cebeaaa618f0087f17b/bwmini/host_cli.py) uses native `incus --force-local --project` only for operator inspection. The Incus CLI Project selector is **not** a client authorization boundary.

## Actual Incus authorization model

Authoritative upstream references: [Incus Authorization](https://linuxcontainers.org/incus/docs/main/authorization/), [Project isolation](https://linuxcontainers.org/incus/docs/main/explanation/projects/), [Project confining](https://linuxcontainers.org/incus/docs/main/howto/projects_confine/) and [authorization options](https://linuxcontainers.org/incus/docs/main/server_config/).

1. A restricted Incus Project (`restricted=true`, managed NIC/network access, etc.) is a workload/device/resource isolation boundary. An authorized client still needs a request authorization decision. The Project name in a CLI flag or MCP input never authenticates its caller.
2. Restricted TLS certificates normally route to the built-in `tls` authorization driver, which checks the certificate's permitted Project list. Scriptlet can be installed at the same time as the TLS/OpenFGA drivers, **but each request routes to one driver based on client class**.
3. `authorization.client.tls-restricted=scriptlet` routes **every** restricted-TLS certificate to Scriptlet; the built-in TLS certificate Project list is **not consulted** for those requests. This is daemon-global for that client class, not a switch for one certificate. A Scriptlet must reproduce the old Project/read/write restrictions explicitly, using trusted certificate data and precise object/entitlement/method checks; deny unknown cases.
4. Hence **Incus Project and Scriptlet can coexist** as workload separation + request authorization, but there is **no implicit simultaneous built-in TLS Project enforcement and Scriptlet authorization** for one restricted TLS request. Do not claim that merely adding a Project, choosing `--project`, or installing `authorization.scriptlet` alone makes the exact shared ACL writable.
5. Root-over-Unix admin access, restricted Unix `incus` Project access, and dedicated restricted TLS ACL-write authorization are different actor classes. Never grant the public MCP runtime a host-root socket or unrestricted TLS certificate to avoid Scriptlet migration.

## v1 product selection and evidence

- Existing approved v1 design: one installer-precreated exact shared ACL in its effective network Project (native tests use `default`), with owned Box NICs attached through an independently confined managed-only Box Project; a **distinct restricted TLS certificate** may GET/PUT only this exact ACL via Scriptlet; the normal non-root restricted Project actor retains lifecycle, Files and Exec.
- The selected native ACL/Scriptlet work is recorded in [single shared ACL acceptance](../verification/SINGLE_SHARED_ACL_SCRIPTLET_ACCEPTANCE_2026-11-08.md), the [S4-B plan](../development/S4_B_SIX_STAGE_PLAN_2026-11-16.md) and the [B2 actual experiment log](../verification/S4_B2_STAGE1_EXPERIMENTS_2026-11-16.md). This v1 evidence, **not** `archive/0.50.5`, establishes the Scriptlet design.
- [Disposable Incus 7.6 CI #44054768739](https://github.com/vanishs/boxweave/actions/runs/44054768739) verified legacy restricted TLS Project/Profile/Files and **original Exec final SUCCESS** before routing, after Scriptlet, and after rollback; foreign Project Exec/Files remained denied and the dedicated writer was limited to exact ACL GET/PUT. This is meaningful compatibility evidence, but it did not install the v1 production Scriptlet controller or prove every legacy entitlement.

## B2 implementation implication / decision required

- **No new owner decision is necessary** to study or implement within the already accepted global Scriptlet design. No evidence in `archive/0.50.5` justifies changing that architecture or copying legacy Python/Containarium code.
- Move beyond repeated research: implement the host-local **approved** Scriptlet inventory, native TLS-client parity inventory, exact policy derivation/validation, durable original configuration journal, ordered apply and rollback, and fail-closed refusal on unmergeable pre-existing policy. Verify actual installed non-root MCP actor, original Incus/SQLite IDs, kernel per-NIC ACL convergence and stale conntrack revoke behavior. A root-side read-only bridge snapshot alone must not authorize a later grant.
- If an actual target daemon has pre-existing Scriptlet semantics that cannot be preserved or audited, **block activation and ask the owner for a host-specific policy decision**. Do not silently overwrite authorization or enable broad fallback grants. L0 shared Incus global authorization must remain unchanged.

**Disposition:** Historical audit complete for this ref; B2 still needs production engineering + installed independent acceptance. **`S4_B2_PARTIAL`; `S4_B3_NOT_STARTED`**.
