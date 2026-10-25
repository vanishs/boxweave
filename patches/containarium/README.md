# Native Tenant Provider source profile

The authoritative candidate is the fork branch
[feat/native-network-tenant-core](https://github.com/vanishs/Containarium/tree/feat/native-network-tenant-core),
pinned to `c13582b5016e7471d83d50d1e09a49a2fe661395` in
[native-network-tenant.json](native-network-tenant.json).
It contains native pre-Start shared creation (A) and independent atomic
Tenant ingress policy updates (B), based on the current fork schema.

## Build inside the assigned Linux development LXC

```bash
git clone --branch feat/native-network-tenant-core https://github.com/vanishs/Containarium.git containarium-tenant-candidate
cd containarium-tenant-candidate
git checkout --detach c13582b5016e7471d83d50d1e09a49a2fe661395
bash /absolute/boxweave/patches/containarium/build-candidate.sh "$PWD" /absolute/candidate-output
```

The script requires the exact clean reviewed commit and only builds daemon,
CLI and Host MCP with candidate version metadata. It installs nothing.
Run the documented Provider regression tests, including the optional dedicated
PostgreSQL fixture. No live Incus or TCP isolation result is implied.

The v0.114.2 patch and `manifest.json` are historical evidence only. Their hashes,
field numbers and built artifacts do not describe this candidate. Do not apply
the old patch to this branch or to a newer Provider checkout.

## Operator deployment checklist

After review and explicit deployment approval, back up matching binaries/config
and private Web data, identify the exact test host and services, and update daemon,
CLI and Host MCP together using that host's established packaging procedure.
Keep originals available. Reconnect MCP and prove initialize/tools/list plus an
authenticated read-only call. Wait for normal independent Web token rotation,
then verify capability contract and fresh Guard. Only an approved dedicated live
plan may create workloads or grant/revoke policies. No automated host installer,
service restart, credential edit, real ACL mutation or rollback is included here.

The candidate does not support existing-Box reassignment, VM/Windows/K8s or
cross-host native shared membership. Non-admin local creation now uses reserved
account:<username> membership; review this intentional security compatibility
change before deployment. Older Provider binaries do not advertise the contract.
