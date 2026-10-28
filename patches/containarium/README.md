# Upstream native Tenant source profile

Native network Tenant support was merged upstream in
[PR #3165](https://github.com/FootprintAI/Containarium/pull/3165). The reviewed
profile now selects the official repository's main commit
`d4aad1a8a8cd748ab6211c67a3417f173316f9e1`, recorded in
[native-network-tenant.json](native-network-tenant.json).
The authenticated wire contract is `native-network-tenant/v1`.

This is a source compatibility baseline, not an assertion that an official
release containing it is already published. The normal installer continues to
select official MCP releases and does not replace an installed Provider daemon.
Old or unknown Providers remain capability-gated. Upgrade daemon, CLI and MCP
compatibly when the corresponding official release is available, reconnect MCP,
and let independent Web token rotation rediscover policy scopes.

## Build inside the assigned Linux development LXC

```bash
git clone https://github.com/FootprintAI/Containarium.git containarium-tenant-source
cd containarium-tenant-source
git checkout --detach d4aad1a8a8cd748ab6211c67a3417f173316f9e1
bash /absolute/boxweave/patches/containarium/build-candidate.sh "$PWD" /absolute/build-output
```

The helper reads the exact commit and contract from the profile, requires a
clean checkout, checks the upstream contract constant, and builds daemon, CLI
and Host MCP with an explicit `0.0.0-source.<commit>` version. SHA256SUMS includes
the source profile. It installs nothing; these are development artifacts, not
signed upstream release binaries. Run the cross-repository REST fixture with
`BOXWEAVE_PROVIDER_SOURCE` pointing at that clean checkout. The fixture uses real
generated policy RPCs and synthetic infrastructure; it does not test real NIC
ACLs, kernel eBPF or TCP isolation.

## Deployment and compatibility boundaries

Provider deployment remains a separate operator action. Back up compatible
binaries/configuration and private Web data before a reviewed update, reconnect
MCP and verify an authenticated handshake and read-only call. No host upgrade,
credential edits, service restart or automatic rollback is part of this source
adaptation. Current capabilities, not a version string, enable Web writes.

Shared membership uses exact Provider keys; `account:<user>` and `<user>` remain
separate enforcement identities. Account tenancy is operator opt-in and default
off. Local backend/pool placement is supported; resolved remote placements and
VM/Windows/K8s shared membership are refused. New shared creates still require
synchronous enforced NIC attachment before Start. Creation capability is separate
from another existing box's attachment failure; verified Guard evidence and
policy application receipts retain their stricter checks.

The old v0.114.2 patch, manifest.json and historical fork verification in
WEB-TESTING are retained as historical evidence only. Do not apply that patch to
current upstream or treat its downstream contract id as supported by this
adapter. Existing Box reassignment remains unsupported.
