# Containarium Tenant networking candidate

This directory contains the complete A+B source patch against upstream
v0.114.2 commit `8107ea2724dd12786bbd73ff5371e8314f3d35d9`.
It includes generated protobuf/gRPC/REST/Swagger files and tests; no generation
step is required to apply it. [manifest.json](manifest.json) records the patch
SHA256 and actual isolated Linux amd64 build checksums.

This is an unpublished development candidate. The normal BoxWeave installer pin
is unchanged. Do not install it over an existing daemon as part of patch checking.

## Reproduce in an isolated Linux checkout

Use a new directory with the exact upstream source and a compatible Go toolchain
(the module selects Go 1.26.10). Keep dependencies/builds inside the assigned LXC.

```bash
git clone --branch v0.114.2 https://github.com/FootprintAI/Containarium.git containarium-tenant-candidate
cd containarium-tenant-candidate
test "$(git rev-parse HEAD)" = 8107ea2724dd12786bbd73ff5371e8314f3d35d9
# Verify sha256sum of the patch against manifest.json before applying it.
git apply --check /absolute/boxweave/patches/containarium/v0.114.2-tenant-network.patch
git apply /absolute/boxweave/patches/containarium/v0.114.2-tenant-network.patch
bash /absolute/boxweave/patches/containarium/build-candidate.sh "$PWD" /absolute/candidate-output
```

The script refuses another base or clean/unpatched source. It only builds, tests
no runtime configuration, and installs nothing. Build checksums vary with paths,
toolchains and environment; the manifest hashes identify this actual build and
are not a reproducibility or upstream-signature claim.

Run the relevant tests as an ordinary Linux user:

```bash
go test -p 2 ./pkg/core/container ./pkg/core/box/lxc ./pkg/core/incus \
  ./internal/tenantguard ./internal/server ./internal/mcp \
  ./internal/client ./internal/cmd ./internal/auth
```

For real transaction coverage, supply TEST_TENANT_PG_DSN pointing only to a
dedicated disposable PostgreSQL database: its test creates/drops a unique schema.
Without the variable, that test explicitly skips. Never point it at a production
database. The recorded run used an isolated loopback fixture, not a mock SQL driver.

The tests include fake Incus pre-start ordering/fail-closed tests, actual generated
REST/gRPC and an MCP stdio protocol exchange. They do not establish physical or
nested real TCP isolation. See [deployment/rollback and live acceptance gate](../../docs/TENANT-UPSTREAM.md)
and [recorded results](../../docs/WEB-TESTING.md).

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
