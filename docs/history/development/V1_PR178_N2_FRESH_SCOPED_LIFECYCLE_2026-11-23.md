# PR 178 N2: fresh scoped installation and lifecycle candidate

Status: local installed Create/Delete acceptance passed; exact-head CI required. N3 remains disabled. Do not merge or release solely on N2 evidence.

## Installed contract

The explicit fresh-only candidate flag is `--scoped-peer-fresh`. It does not migrate legacy databases, profiles, ACLs or guests. The legacy installer remains available for regression jobs, not the new architecture. A scoped journal fences legacy stage, upgrade, rollback, backup commit and uninstall. Scoped uninstall and address reuse are deferred.

Root derives the dedicated Project and bridge, validates both address families and rejects existing native objects or ambiguous paths. It creates the initial no-grant bridge Guard exclusively, reads back the kernel policy, journals its identity, prepares an explicit deny-all quarantine ACL and pins the default profile. The installed Guard unit is required before Incus startup. The lifecycle helper accepts only the original Operation ID from the authenticated restricted controller; native and SQLite identities are independently derived.

A restricted actor cannot choose volatile.uuid. Create reserves addresses and creates a stopped, autostart-disabled quarantine instance. Root derives the dedicated ACL from the actual UUID, creates it exclusively and verifies exact attachment before restoring the caller's boot flag and Internet choice. Controller start is conditional on the caller's request. Final ownership, reservation and ACL claims commit together. Never-started guests may lack static DHCP leases; exact native device and address collision checks precede boot, followed by real lease proof on requested start.

Delete first confirms exact native instance absence. Root verifies the original Create receipt, detached ACL and unaffected inventory, and deletes only that ACL. Controller settles the original Delete only after the Root receipt. Unknown effects remain fenced; no automatic replay. Released addresses remain reserved against reuse. N3 must handle validated scoped deleted history before enabling grants; legacy Peer and retirement sweep are disabled in scoped mode.

## Local evidence

- Development Box: ac585d2c-37ae-4879-a2e6-3cecc338d4d7.
- Native-only laboratory: 8f288a5f-e42c-4609-92ef-426aff10140d.
- Clean installed laboratory: 0a82c2e1-96bd-4b53-b144-5fc022b080d6.
- Root initial Guard: 6447f4822626c21e734815045712c8e4, exit 0.
- Quarantine preparation: e01b9281784fdbbc08ed705b4e33fe1d, exit 0.
- Real two-phase native Create: d3653756f02e8a65777cdddf4bdb2650 and c20011e82f8b48b750a853a24de99d94, exit 0.
- Clean install through install.sh: c9af9c32ca4800506cbd85416678207d, exit 0; controller/Guard/lifecycle active, scoped manifest settled.
- Installed MCP lifecycle: 4604f6dea210106acb96a08c47616253, exit 0. Two independent ACLs, default true and explicit false, both creates and both deletes succeeded. Deleting alpha left beta running with its exact ACL. Final native ACL list contained only quarantine. Legacy Peer refused.
- Full Go suite before final serve-mode hardening: 0cc715d2bfea0398c3c5f78995497992, exit 0. Targeted final serve-mode test: e28b334f8e52a311f334ffc07eaa3d51, exit 0.

- After both Deletes, a new running/default-autostart Box and a never-started explicit-false Box were created successfully: 79d0cbc3ad63743bd1c3826e30c0e112, exit 0.
- Actual isolated Host cold restart: L0 stop 21a10725db0f32ccc85e811cf0f17ed9 and start 0e7a2ed12e60f5418c218d8f7bde4a62, both SUCCEEDED. Postflight b7c342f7c1b7dee28ff4e6e2a87c2a7a, exit 0: all three units active, Guard active-enter 237102901495 before Incus 237105256827; default-true guest Running, explicit-false guest Stopped, exact UUID/ACL retained. Guard boot command verified the expected kernel policy. This was an actual isolated Host stop/start, not a systemd-only simulation.

- Post-reboot MCP stop/delete and never-started Delete, both original released ledger bindings and final quarantine-only ACL inventory: 8c6e12651f5206fd91c19b40c7a668d8, exit 0. Direct legacy upgrade refused before backup creation; controller stayed active.
- Final source gate: cf3e53f661aea31f1a851ed04102aa4a, exit 0; full Go suite, guided upgrade (9) and uninstall (31) tests, 41-file format and diff checks.

## Final CI

The existing installed lifecycle workflow now runs legacy/scoped jobs. Scoped checks exact-head native bootstrap, explicit scoped installation, upgrade refusal, real MCP create/stop/delete, distinct ACLs, effective boot dependencies, boot flags, historical-address holds and non-target survival. Legacy retains upgrade/rollback and uninstall coverage. N3 grant/revoke, N4 traffic and fault matrix, and N5 release acceptance remain separate gates.
