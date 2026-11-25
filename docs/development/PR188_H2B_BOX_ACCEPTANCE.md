# H2B bounded Legacy offline retirement candidate

## Scope and evidence

Base ed72b4c49e8be633d11c340c9c93d24df52865c0. Box A is the retained nested RC13 installation (9d374e18-196a-493d-bf1f-2d582589f04d). No physical L0 changes. No original database modification or outcome settlement.

- Actual RC13 offline plan: operation cd11409a4e5dce42e38c37e5af6c4011, exit 2 as expected; PINNED_RC13, SQLITE_EXCLUSIVE_COLD_COPY, BOUNDED_SOURCES_OBSERVED, exact preparation and base ACL observed. Binary, manifest and ledger SHA256 unchanged. No offline fence created.
- Two inherited systemd drop-ins and missing original serve lock block actual destructive execution in Box A. Do not bypass them.
- Exact native bridge and ACL consumer checks against RC13: ddfbc94159981ef8a6184b0c96512462, exit zero.
- Go H1A/H1B/H2A/H2B state regression including exact original UNKNOWN claim witness: 562509390998d15f7e966fedfc417657, exit zero. Go install/control/cmd compile: d5662580f30bfed7de64c387e8c82c80.
- Four new offline scope tests: 35311348901e312ef3629618e14e3411, PASS. Existing 53 uninstall, 6 maintenance and 6 scoped drain tests: e1813a0b65d1c10ef5679bc00c88dc4c, PASS. Shell/Python syntax and diff checks pass.

## Explicit scope and pending acceptance

The executable offline leaf is deliberately limited to exact stopped Legacy installations with the original baseline ACL, verified native bindings, no unsettled network/Forward effects, and only exact owned stopped-box UNKNOWN power-stop originals with retained claims and native receipt. It preserves the cold database byte-for-byte, archives before native effects, checks native inventory prefixes, records durable checkpoints, and uses the existing native rollback. Offline fence refuses ordinary install/uninstall and controller admission. It is not original recovery authority.

The new clean VM legacy-offline CI lane creates a real Box and a native stop whose ACK is left unresolved by the existing installed D4 fixture, then reviews and retires without settling the original. It must verify archive SHA256 and UNKNOWN claim, native retirement, unchanged Incus PID and unrelated project. No positive installed destructive result claimed before this CI.

Historical RC13 with unresolved Peer Root journal, active ACL delta, Forward dependencies, ambiguous leases, and partial Scoped installations remain explicit blockers. Full H2B contract is not signed off. H2C offline archive purge and interrupted resume are not implemented; keep audit/fence intact. H2A prior CI failed after approval at the first Scoped drain operation; current patch changes original probe to private snapshot and adds bounded diagnostics. The actual cause is not yet proven and regression CI remains required.
